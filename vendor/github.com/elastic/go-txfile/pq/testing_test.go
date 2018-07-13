package pq

import (
	"math/rand"

	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/cleanup"
	"github.com/elastic/go-txfile/internal/mint"
	"github.com/elastic/go-txfile/txfiletest"
)

type testQueue struct {
	t      *mint.T
	config config
	*Queue
	*txfiletest.TestFile
}

type config struct {
	File  txfile.Options
	Queue Settings
}

type testRange struct {
	min, max int
}

func exactly(n int) testRange        { return testRange{n, n} }
func between(min, max int) testRange { return testRange{min, max} }

func setupQueue(t *mint.T, cfg config) (*testQueue, func()) {
	tf, teardown := txfiletest.SetupTestFile(t.T, cfg.File)

	ok := false
	defer cleanup.IfNot(&ok, teardown)

	tq := &testQueue{
		t:        t,
		config:   cfg,
		TestFile: tf,
		Queue:    nil,
	}

	tq.Open()
	ok = true
	return tq, func() {
		tq.Close()
		teardown()
	}
}

func (q *testQueue) Reopen() {
	q.Close()
	q.Open()
}

func (q *testQueue) Open() {
	if q.Queue != nil {
		return
	}

	q.TestFile.Open()

	d, err := NewStandaloneDelegate(q.TestFile.File)
	if err != nil {
		q.t.Fatal(err)
	}

	tmp, err := New(d, q.config.Queue)
	if err != nil {
		q.t.Fatal(err)
	}

	q.Queue = tmp
}

func (q *testQueue) Close() {
	if q.Queue == nil {
		return
	}

	q.t.FatalOnError(q.Queue.Close())
	q.TestFile.Close()
	q.Queue = nil
}

func (q *testQueue) len() int {
	return int(q.Reader().Available())
}

func (q *testQueue) append(events ...string) {
	w := q.Queue.Writer()
	for _, event := range events {
		_, err := w.Write([]byte(event))
		q.t.FatalOnError(err)
		q.t.FatalOnError(w.Next())
	}
}

// read reads up to n events from the queue.
func (q *testQueue) read(n int) []string {
	var out []string
	if n > 0 {
		out = make([]string, 0, n)
	}

	for n < 0 || len(out) < n {
		sz, err := q.Reader().Next()
		q.t.FatalOnError(err)
		if sz <= 0 {
			break
		}

		buf := make([]byte, sz)
		_, err = q.Reader().Read(buf)
		q.t.FatalOnError(err)

		out = append(out, string(buf))
	}
	return out
}

func (q *testQueue) flush() {
	err := q.Queue.Writer().Flush()
	q.t.NoError(err)
}

func (q *testQueue) ack(n uint) {
	err := q.Queue.ACK(n)
	q.t.NoError(err)
}

func (r testRange) rand(rng *rand.Rand) int {
	if r.min == r.max {
		return r.min
	}
	return rng.Intn(r.max-r.min) + r.min
}
