package txfile

import (
	"errors"
	"testing"
	"time"
)

type (
	testIOOperations []testIOOp

	testIOOp interface {
		Type() opType
		At() int64
		Contents() []byte
	}

	testWriteAtOP struct {
		at       int64
		contents []byte
	}

	testSyncOP struct{}

	opType uint8
)

type testWriterTarget struct {
	writeAt func([]byte, int64) (int, error)
	sync    func() error
}

const (
	opUndefined opType = iota
	opWriteAt
	opSync
)

func TestFileWriter(t *testing.T) {
	assert := newAssertions(t)

	assert.Run("start stop", func(assert *assertions) {
		var ops testIOOperations
		_, teardown := newTestWriter(recordWriteOps(&ops), 64)

		time.Sleep(10 * time.Millisecond)
		if assert.NoError(teardown()) {
			assert.Len(ops, 0)
		}
	})

	assert.Run("write and sync", func(assert *assertions) {
		var ops testIOOperations
		w, teardown := newTestWriter(recordWriteOps(&ops), 10)
		defer mustSucceed(assert, teardown)

		var tmp [10]byte
		sync := newTxWriteSync()
		w.Schedule(sync, 0, tmp[:])
		w.Sync(sync)

		if !assert.NoError(waitFn(1*time.Second, sync.Wait)) {
			assert.FailNow("invalid writer state")
		}

		if assert.Len(ops, 2) {
			assert.Equal(opWriteAt, ops[0].Type())
			assert.Equal(int64(0), ops[0].At())
			assert.Len(ops[0].Contents(), 10)

			assert.Equal(syncOp, ops[1])
		}
	})

	assert.Run("write after sync", func(assert *assertions) {
		var ops testIOOperations
		b, target := blockingTarget(recordWriteOps(&ops))
		b.Block()

		w, teardown := newTestWriter(target, 10)
		defer mustSucceed(assert, teardown)

		var tmp [10]byte
		sync := newTxWriteSync()
		w.Schedule(sync, 0, tmp[:])
		w.Schedule(sync, 1, tmp[:])
		w.Sync(sync)
		w.Schedule(sync, 2, tmp[:])
		w.Schedule(sync, 3, tmp[:])
		w.Schedule(sync, 4, tmp[:])
		w.Sync(sync)
		w.Schedule(sync, 5, tmp[:])
		w.Sync(sync)

		// unblock writer, so scheduled write commands will be executed
		b.Unblock()

		if !assert.NoError(waitFn(1*time.Second, sync.Wait)) {
			assert.FailNow("invalid writer state")
		}

		expectedOps := []opType{opWriteAt, opWriteAt, opSync, opWriteAt, opWriteAt, opWriteAt, opSync, opWriteAt, opSync}
		expectedOffsets := []int64{0, 10, -1, 20, 30, 40, -1, 50, -1}
		offsets := make([]int64, len(ops))
		actual := make([]opType, len(ops))
		for i, op := range ops {
			t := op.Type()
			off := int64(-1)
			if t == opWriteAt {
				off = op.At()
			}

			actual[i] = t
			offsets[i] = off
		}
		assert.Equal(expectedOps, actual)
		assert.Equal(expectedOffsets, offsets)
	})

	assert.Run("ordered writes", func(assert *assertions) {
		var ops testIOOperations
		b, target := blockingTarget(recordWriteOps(&ops))
		b.Block()

		w, teardown := newTestWriter(target, 10)
		defer mustSucceed(assert, teardown)

		var tmp [10]byte
		sync := newTxWriteSync()
		w.Sync(sync) // start with sync, to guarantee writer is really blocked
		w.Schedule(sync, 3, tmp[:])
		w.Schedule(sync, 2, tmp[:])
		w.Schedule(sync, 1, tmp[:])
		w.Sync(sync)

		// unblock writer, so scheduled write commands will be executed
		b.Unblock()

		if !assert.NoError(waitFn(1*time.Second, sync.Wait)) {
			assert.FailNow("invalid  writer state")
		}

		expected := []int64{10, 20, 30}
		var actual []int64
		for _, op := range ops {
			if op.Type() == opWriteAt {
				actual = append(actual, op.At())
				assert.Equal(10, len(op.Contents()))
			}
		}
		assert.Equal(expected, actual)
	})

	assert.Run("fail on sync", func(assert *assertions) {
		expectedErr := errors.New("ups")

		var ops testIOOperations
		target := recordWriteOps(&ops)
		recordSync := target.sync
		target.sync = func() error {
			recordSync()
			return expectedErr
		}

		w, teardown := newTestWriter(target, 10)
		defer mustSucceed(assert, teardown)

		var tmp [10]byte
		sync := newTxWriteSync()
		w.Schedule(sync, 1, tmp[:])
		w.Sync(sync)
		w.Schedule(sync, 2, tmp[:])
		w.Sync(sync)

		err := waitFn(1*time.Second, sync.Wait)
		if expectedErr != err {
			assert.FailNow("unexpected error")
		}

		// writer should stop on first error and ignore all following commands
		expectedOps := []opType{opWriteAt, opSync}
		actual := make([]opType, len(ops))
		for i, op := range ops {
			actual[i] = op.Type()
		}
		assert.Equal(expectedOps, actual)
	})

	assert.Run("fail on write", func(assert *assertions) {
		expectedErr := errors.New("ups")

		var ops testIOOperations
		target := recordWriteOps(&ops)
		recordWrites := target.writeAt
		target.writeAt = func(p []byte, off int64) (int, error) {
			recordWrites(p, off)
			return 0, expectedErr
		}

		w, teardown := newTestWriter(target, 10)
		defer mustSucceed(assert, teardown)

		var tmp [10]byte
		sync := newTxWriteSync()
		w.Schedule(sync, 1, tmp[:])
		w.Schedule(sync, 2, tmp[:])
		w.Sync(sync)
		w.Schedule(sync, 3, tmp[:])
		w.Schedule(sync, 4, tmp[:])
		w.Sync(sync)

		err := waitFn(5*time.Second, sync.Wait)
		assert.Error(err)
		assert.Equal(expectedErr, err)

		// writer should stop on first error and ignore all following commands
		expectedOps := []opType{opWriteAt}
		actual := make([]opType, len(ops))
		for i, op := range ops {
			actual[i] = op.Type()
		}
		assert.Equal(expectedOps, actual)
	})
}

func newTestWriter(to writable, pagesize uint) (*writer, func() error) {
	w := &writer{}
	w.Init(to, pagesize)

	cw := makeCloseWait(1 * time.Second)
	cw.Add(1)

	go func() {
		defer cw.Done()
		w.Run()
	}()

	return w, func() error {
		w.Stop()
		if !cw.Wait() {
			return errors.New("test writer shutdown timeout")
		}
		return nil
	}
}

func (w *testWriterTarget) Sync() error { return w.sync() }
func (w *testWriterTarget) WriteAt(p []byte, off int64) (n int, err error) {
	return w.writeAt(p, off)
}

var ignoreOps = &testWriterTarget{
	writeAt: func(p []byte, _ int64) (int, error) { return len(p), nil },
	sync:    func() error { return nil },
}

func recordWriteOps(ops *testIOOperations) *testWriterTarget {
	return &testWriterTarget{
		writeAt: func(p []byte, off int64) (n int, err error) {
			*ops = append(*ops, &testWriteAtOP{at: off, contents: p})
			return len(p), nil
		},
		sync: func() error {
			*ops = append(*ops, syncOp)
			return nil
		},
	}
}

func blockingTarget(w *testWriterTarget) (*blocker, *testWriterTarget) {
	b := newBlocker()
	return b, &testWriterTarget{
		writeAt: func(p []byte, off int64) (n int, err error) {
			b.Wait()
			return w.writeAt(p, off)
		},
		sync: func() error {
			b.Wait()
			return w.sync()
		},
	}
}

func (op *testWriteAtOP) Type() opType     { return opWriteAt }
func (op *testWriteAtOP) At() int64        { return op.at }
func (op *testWriteAtOP) Contents() []byte { return op.contents }

var syncOp = (*testSyncOP)(nil)

func (op *testSyncOP) Type() opType     { return opSync }
func (op *testSyncOP) At() int64        { panic("sync op") }
func (op *testSyncOP) Contents() []byte { panic("sync op") }

func mustSucceed(assert *assertions, fn func() error) {
	assert.NoError(fn())
}
