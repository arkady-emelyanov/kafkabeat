package txfile

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/go-txfile/internal/cleanup"
)

type testFile struct {
	*File
	path   string
	assert *assertions
	opts   Options
}

func TestTxFile(t *testing.T) {
	assert := newAssertions(t)

	sampleContents := []string{
		"Hello World",
		"The quick brown fox jumps over the lazy dog",
		"Lorem ipsum dolor sit amet, consectetur adipisici elit, sed eiusmod tempor incidunt ut labore et dolore magna aliqua.",
	}

	assert.Run("mmap sizing", func(assert *assertions) {
		const (
			_ = 1 << (10 * iota)
			KB
			MB
			GB
		)

		testcases := []struct {
			min, max, expected uint64
		}{
			{64, 0, 64 * KB},
			{4 * KB, 0, 64 * KB},
			{100 * KB, 0, 128 * KB},
			{5 * MB, 0, 8 * MB},
			{300 * MB, 0, 512 * MB},
			{1200 * MB, 0, 2 * GB},
			{2100 * MB, 0, 3 * GB},
		}

		for _, test := range testcases {
			min, max, expected := test.min, test.max, test.expected
			title := fmt.Sprintf("min=%v,max=%v,expected=%v", min, max, expected)

			assert.Run(title, func(assert *assertions) {
				if expected > uint64(maxUint) {
					assert.Skip("unsatisfyable tests on 32bit system")
				}

				sz, err := computeMmapSize(uint(min), uint(max), 4096)
				assert.NoError(err)
				assert.Equal(int(expected), int(sz))
			})
		}
	})

	assert.Run("open/close", func(assert *assertions) {
		path, teardown := setupPath(assert, "")
		defer teardown()

		f, err := Open(path, os.ModePerm, Options{
			MaxSize:  10 * 1 << 20, // 10MB
			PageSize: 4096,
		})
		assert.FatalOnError(err)
		assert.NoError(f.Close())

		// check if we can re-open the file:

		f, err = Open(path, os.ModePerm, Options{})
		assert.FatalOnError(err)
		assert.NoError(f.Close())
	})
	if assert.Failed() {
		// if we find a file can not even be created correctly, no need to run more
		// tests, that rely on file creation during test setup
		return
	}

	assert.Run("start and close readonly transaction without reads", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		tx := f.BeginReadonly()
		assert.NotNil(tx)
		assert.True(tx.Readonly())
		assert.False(tx.Writable())
		assert.True(tx.Active())
		assert.NoError(tx.Close())
		assert.False(tx.Active())
	})

	assert.Run("start and close read-write transaction without reads/writes", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		tx := f.Begin()
		assert.NotNil(tx)
		assert.False(tx.Readonly())
		assert.True(tx.Writable())
		assert.True(tx.Active())
		assert.NoError(tx.Close())
		assert.False(tx.Active())
	})

	assert.Run("readonly transaction can not allocate pages", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		tx := f.BeginReadonly()
		defer tx.Close()

		page, err := tx.Alloc()
		assert.Nil(page)
		assert.Error(err)
	})

	assert.Run("write transaction with modifications on new file with rollback", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		tx := f.Begin()
		defer tx.Close()

		ids := pageSet{}

		page, err := tx.Alloc()
		assert.FatalOnError(err)
		if assert.NotNil(page) {
			ids.Add(page.ID())
			assert.NotEqual(PageID(0), page.ID())
		}

		pages, err := tx.AllocN(5)
		assert.FatalOnError(err)
		assert.Len(pages, 5)
		for _, page := range pages {
			if assert.NotNil(page) {
				assert.NotEqual(PageID(0), page.ID())
				ids.Add(page.ID())
			}
		}

		// check we didn't get the same ID twice:
		assert.Len(ids, 6)

		f.Rollback(tx)
	})

	assert.Run("comitting write transaction without modifications", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		tx := f.Begin()
		defer tx.Close()
		f.Commit(tx)
	})

	assert.Run("committing write transaction on new file with page writes", func(assert *assertions) {
		contents := sampleContents
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		var ids idList
		func() {
			tx := f.Begin()
			defer tx.Close()
			ids = f.txAppend(tx, contents)
			tx.SetRoot(ids[0])
			f.Commit(tx, "failed to commit the initial transaction")
		}()

		traceln("transaction page ids: ", ids)

		f.Reopen()
		tx := f.BeginReadonly()
		defer tx.Close()
		assert.Equal(ids[0], tx.Root())
		assert.Equal(contents, f.readIDs(ids))
	})

	assert.Run("read contents after reopen and page contents has been overwritten", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		writeSample := func(msg string, id PageID) (to PageID) {
			f.withTx(true, func(tx *Tx) {
				to = id
				if to == 0 {
					page, err := tx.Alloc()
					assert.FatalOnError(err)
					to = page.ID()
				}

				f.txWriteAt(tx, to, msg)
				f.Commit(tx)
			})
			return
		}

		msgs := sampleContents[:2]
		id := writeSample(msgs[0], 0)
		writeSample(msgs[1], id)

		// reopen and check new contents is really available
		f.Reopen()
		assert.Equal(msgs[1], f.read(id))
	})

	assert.Run("check allocates smallest page possible", func(assert *assertions) {
		bools := [2]bool{false, true}

		for _, reopen := range bools {
			for _, withFragmentation := range bools {
				reopen := reopen
				withFragmentation := withFragmentation
				title := fmt.Sprintf("reopen=%v, fragmentation=%v", reopen, withFragmentation)
				assert.Run(title, func(assert *assertions) {

					f, teardown := setupTestFile(assert, Options{})
					defer teardown()

					// allocate 2 pages
					var id PageID
					f.withTx(true, func(tx *Tx) {
						page, err := tx.Alloc()
						assert.FatalOnError(err)

						// allocate dummy page, so to ensure some fragmentation on free
						if withFragmentation {
							_, err = tx.Alloc()
							assert.FatalOnError(err)
						}

						id = page.ID()
						f.Commit(tx)
					})

					if reopen {
						f.Reopen()
					}

					// free first allocated page
					f.withTx(true, func(tx *Tx) {
						page, err := tx.Page(id)
						assert.FatalOnError(err)
						assert.FatalOnError(page.Free())
						f.Commit(tx)
					})

					if reopen {
						f.Reopen()
					}

					// expect just freed page can be allocated again
					var newID PageID
					f.withTx(true, func(tx *Tx) {
						page, err := tx.Alloc()
						assert.FatalOnError(err)

						newID = page.ID()
						f.Commit(tx)
					})

					// verify
					assert.Equal(id, newID)
				})
			}
		}

	})

	assert.Run("file open old transaction if verification fails", func(assert *assertions) {
		msgs := sampleContents[:2]
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		// start first transaction with first message

		writeSample := func(msg string, id PageID) PageID {
			tx := f.Begin()
			defer tx.Close()

			if id == 0 {
				page, err := tx.Alloc()
				assert.FatalOnError(err)
				id = page.ID()
			}

			f.txWriteAt(tx, id, msg)
			tx.SetRoot(id)
			f.Commit(tx)
			return id
		}

		id := writeSample(msgs[0], 0)
		writeSample(msgs[1], id) // overwrite contents

		// get active meta page id
		metaID := PageID(f.File.metaActive)
		metaOff := int64(metaID) * int64(f.allocator.pageSize)

		// close file and write invalid contents into last tx meta page
		f.Close()

		func() {
			tmp, err := os.OpenFile(f.path, os.O_RDWR, 0777)
			assert.FatalOnError(err)
			defer tmp.Close()

			_, err = tmp.WriteAt([]byte{1, 2, 3, 4, 5}, metaOff)
			assert.FatalOnError(err)
		}()

		// Open db file again and check recent transaction contents is still
		// available.
		f.Open()
		assert.Equal(msgs[0], f.read(id))
	})

	assert.Run("concurrent read transaction can not access not comitted contents", func(assert *assertions) {
		orig, modified := sampleContents[0], sampleContents[1:]
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		// commit original contents first
		id := f.append([]string{orig})[0]

		current := orig
		for _, newmsg := range modified {
			func() {
				// create write transaction, used to update the contents
				writer := f.Begin()
				defer writer.Close()
				f.txWriteAt(writer, id, newmsg)

				// check concurrent read will not have access to new contents
				reader := f.BeginReadonly()
				defer reader.Close()
				assert.Equal(current, f.txRead(reader, id))
				f.Commit(reader) // close reader transaction

				// check new reader will read modified contents
				reader = f.BeginReadonly()
				defer reader.Close()
				assert.Equal(current, f.txRead(reader, id), "read unexpected message")
				f.Commit(reader)

				// finish transaction
				f.Commit(writer)
				current = newmsg
			}()
		}
	})

	assert.Run("execute wal checkpoint", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		// first tx with original contents
		ids := f.append(sampleContents)

		// overwrite original contents
		overwrites := append(sampleContents[1:], sampleContents[0])
		f.writeAllAt(ids, overwrites)
		assert.Len(f.wal.mapping, 3, "expected wal mapping entries")

		// run wal checkpoint
		f.withTx(true, func(tx *Tx) {
			assert.FatalOnError(tx.CheckpointWAL())
			f.Commit(tx)
		})
		assert.Len(f.wal.mapping, 0, "mapping must be empty after checkpointing")
	})

	assert.Run("force remap", func(assert *assertions) {
		N, i := 64/3+1, 0
		contents := make([]string, 0, N)
		for len(contents) < N {
			contents = append(contents, sampleContents[i])
			if i++; i == len(sampleContents) {
				i = 0
			}
		}

		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		ids := f.append(contents) // write enough contents to enforce an munmap/mmap
		assert.Equal(len(contents), len(ids))

		// check we can still read all contents
		assert.Equal(contents, f.readIDs(ids))
	})

	assert.Run("inplace update", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		id := f.append(sampleContents[:1])[0]

		func() {
			tx := f.Begin()
			defer tx.Close()

			page, err := tx.Page(id)
			assert.FatalOnError(err)
			assert.FatalOnError(page.Load())
			buf, err := page.Bytes()
			assert.FatalOnError(err)

			// validate contents buffer
			assert.Len(buf, 4096)
			L := int(castU32(buf).Get())
			assert.Equal(len(sampleContents[0]), L)
			assert.Equal(sampleContents[0], string(buf[4:4+L]))

			// overwrite buffer
			castU32(buf).Set(uint32(len(sampleContents[1])))
			copy(buf[4:], sampleContents[1])
			assert.FatalOnError(page.MarkDirty())

			// commit
			f.Commit(tx)
		}()

		// read/validate new contents
		assert.Equal(sampleContents[1], f.read(id))
	})

	assert.Run("multiple inplace updates", func(assert *assertions) {
		f, teardown := setupTestFile(assert, Options{})
		defer teardown()

		id := f.append(sampleContents[:1])[0]
		assert.Equal(sampleContents[0], f.read(id))

		f.writeAt(id, sampleContents[1])
		assert.Equal(sampleContents[1], f.read(id))

		f.writeAt(id, sampleContents[2])
		assert.Equal(sampleContents[2], f.read(id))
	})
}

func setupTestFile(assert *assertions, opts Options) (*testFile, func()) {
	// if opts.MaxSize == 0 {
	// 	opts.MaxSize = 10 * 1 << 20 // 10 MB
	// }
	if opts.PageSize == 0 {
		opts.PageSize = 4096
	}

	ok := false
	path, teardown := setupPath(assert, "")
	defer cleanup.IfNot(&ok, teardown)

	tf := &testFile{path: path, assert: assert, opts: opts}
	tf.Open()

	ok = true
	return tf, func() {
		tf.Close()
		teardown()
	}
}

func (f *testFile) Reopen() {
	f.Close()
	f.Open()
}

func (f *testFile) Close() {
	if f.File != nil {
		f.assert.NoError(f.File.Close(), "close failed on reopen")
		f.File = nil
	}
}

func (f *testFile) Open() {
	tmp, err := Open(f.path, os.ModePerm, f.opts)
	f.assert.FatalOnError(err, "reopen failed")
	f.File = tmp

	f.checkConsistency()
}

func (f *testFile) Commit(tx *Tx, msgAndArgs ...interface{}) {
	needsCheck := tx.Writable()
	f.assert.FatalOnError(tx.Commit(), msgAndArgs...)
	if needsCheck {
		if !f.checkConsistency() {
			f.assert.Fatal("inconsistent file state")
		}
	}
}

func (f *testFile) Rollback(tx *Tx, msgAndArgs ...interface{}) {
	needsCheck := tx.Writable()
	f.assert.FatalOnError(tx.Rollback(), msgAndArgs...)
	if needsCheck {
		if !f.checkConsistency() {
			f.assert.Fatal("inconsistent file state")
		}
	}
}

// checkConsistency checks on disk file state is correct and consistent with in
// memory state.
func (f *testFile) checkConsistency() bool {
	if err := f.getMetaPage().Validate(); err != nil {
		f.assert.Error(err, "meta page validation")
		return false
	}

	meta := f.getMetaPage()
	ok := true

	// check wal:
	walMapping, walPages, err := readWAL(f.mmapedPage, meta.wal.Get())
	f.assert.FatalOnError(err, "reading wal state")
	ok = ok && f.assert.Equal(walMapping, f.wal.mapping, "wal mapping")
	ok = ok && f.assert.Equal(walPages.Regions(), f.wal.metaPages, "wal meta pages")

	// validate meta end markers state
	maxSize := meta.maxSize.Get() / uint64(meta.pageSize.Get())
	dataEnd := meta.dataEndMarker.Get()
	metaEnd := meta.metaEndMarker.Get()
	ok = ok && f.assert.True(maxSize == 0 || uint64(dataEnd) <= maxSize, "data end marker in bounds")

	// compare alloc markers and counters with allocator state
	ok = ok && f.assert.Equal(dataEnd, f.allocator.data.endMarker, "data end marker mismatch")
	ok = ok && f.assert.Equal(metaEnd, f.allocator.meta.endMarker, "meta end marker mismatch")
	ok = ok && f.assert.Equal(uint(meta.metaTotal.Get()), f.allocator.metaTotal, "meta area size mismatch")

	// compare free lists
	var metaList, dataList regionList
	flPages, err := readFreeList(f.mmapedPage, meta.freelist.Get(), func(isMeta bool, r region) {
		if isMeta {
			metaList.Add(r)
		} else {
			dataList.Add(r)
		}
	})
	optimizeRegionList(&metaList)
	optimizeRegionList(&dataList)
	f.assert.FatalOnError(err, "reading freelist")
	ok = ok && f.assert.Equal(metaList, f.allocator.meta.freelist.regions, "meta area freelist mismatch")
	ok = ok && f.assert.Equal(dataList, f.allocator.data.freelist.regions, "data area freelist mismatch")
	ok = ok && f.assert.Equal(flPages.Regions(), f.allocator.freelistPages, "freelist meta pages")

	return ok
}

func (f *testFile) withTx(write bool, fn func(tx *Tx)) {
	tx := f.BeginWith(TxOptions{Readonly: !write})
	defer func() {
		f.assert.FatalOnError(tx.Close())
	}()
	fn(tx)
}

func (f *testFile) append(contents []string) (ids idList) {
	f.withTx(true, func(tx *Tx) {
		ids = f.txAppend(tx, contents)
		f.assert.FatalOnError(tx.Commit())
	})
	return
}

func (f *testFile) txAppend(tx *Tx, contents []string) idList {
	ids := make(idList, len(contents))
	pages, err := tx.AllocN(len(contents))
	f.assert.FatalOnError(err)
	for i, page := range pages {
		buf := make([]byte, tx.PageSize())
		data := contents[i]
		castU32(buf).Set(uint32(len(data)))
		copy(buf[4:], data)
		f.assert.NoError(page.SetBytes(buf))

		ids[i] = page.ID()
	}
	return ids
}

func (f *testFile) read(id PageID) (s string) {
	f.withTx(false, func(tx *Tx) { s = f.txRead(tx, id) })
	return
}

func (f *testFile) txRead(tx *Tx, id PageID) string {
	page, err := tx.Page(id)
	f.assert.FatalOnError(err)

	buf, err := page.Bytes()
	f.assert.FatalOnError(err)

	count := castU32(buf).Get()
	return string(buf[4 : 4+count])
}

func (f *testFile) readIDs(ids idList) (contents []string) {
	f.withTx(false, func(tx *Tx) { contents = f.txReadIDs(tx, ids) })
	return
}

func (f *testFile) txReadIDs(tx *Tx, ids idList) []string {
	contents := make([]string, len(ids))
	for i, id := range ids {
		contents[i] = f.txRead(tx, id)
	}
	return contents
}

func (f *testFile) writeAt(id PageID, msg string) {
	f.withTx(true, func(tx *Tx) {
		f.txWriteAt(tx, id, msg)
		f.assert.FatalOnError(tx.Commit())
	})
}

func (f *testFile) txWriteAt(tx *Tx, id PageID, msg string) {
	page, err := tx.Page(id)
	f.assert.FatalOnError(err)
	buf := make([]byte, tx.PageSize())
	castU32(buf).Set(uint32(len(msg)))
	copy(buf[4:], msg)
	f.assert.NoError(page.SetBytes(buf))
}

func (f *testFile) writeAllAt(ids idList, msgs []string) {
	f.withTx(true, func(tx *Tx) {
		f.txWriteAllAt(tx, ids, msgs)
		f.assert.FatalOnError(tx.Commit())
	})
}

func (f *testFile) txWriteAllAt(tx *Tx, ids idList, msgs []string) {
	for i, id := range ids {
		f.txWriteAt(tx, id, msgs[i])
	}
}
