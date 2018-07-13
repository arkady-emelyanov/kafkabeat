package txfile

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestOSFileSupport(t *testing.T) {
	assert := newAssertions(t)

	setupFile := func(assert *assertions, file string) (vfsFile, func()) {
		path, teardown := setupPath(assert, file)

		f, err := openOSFile(path, os.ModePerm)
		if err != nil {
			teardown()
			assert.Fatal(err)
		}

		return f, func() {
			f.Close()
			teardown()
		}
	}

	assert.Run("file size", func(assert *assertions) {
		file, teardown := setupFile(assert, "")
		defer teardown()

		_, err := file.WriteAt([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0)
		assert.FatalOnError(err)

		sz, err := file.Size()
		assert.NoError(err)
		assert.Equal(10, int(sz))
	})

	assert.Run("lock/unlock succeed", func(assert *assertions) {
		file, teardown := setupFile(assert, "")
		defer teardown()

		err := file.Lock(true, false)
		assert.NoError(err)

		err = file.Unlock()
		assert.NoError(err)
	})

	assert.Run("locking locked file fails", func(assert *assertions) {
		f1, teardown := setupFile(assert, "")
		defer teardown()

		f2, err := openOSFile(f1.Name(), os.ModePerm)
		assert.FatalOnError(err)
		defer f2.Close()

		err = f1.Lock(true, false)
		assert.NoError(err)

		err = f2.Lock(true, false)
		assert.Error(err)

		err = f1.Unlock()
		assert.NoError(err)
	})

	assert.Run("mmap file", func(assert *assertions) {
		f, teardown := setupFile(assert, "")
		defer teardown()

		var buf = [10]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		n, err := f.WriteAt(buf[:], 0)
		assert.Equal(len(buf), n)
		assert.NoError(err)

		mem, err := f.MMap(len(buf))
		assert.FatalOnError(err)
		defer func() {
			assert.NoError(f.MUnmap(mem))
		}()

		assert.Equal(buf[:], mem[:len(buf)])
	})
}

func setupPath(assert *assertions, file string) (string, func()) {
	dir, err := ioutil.TempDir("", "")
	assert.FatalOnError(err)

	if file == "" {
		file = "test.dat"
	}
	return path.Join(dir, file), func() {
		os.RemoveAll(dir)
	}
}
