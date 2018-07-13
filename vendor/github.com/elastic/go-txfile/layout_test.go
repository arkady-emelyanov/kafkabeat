package txfile

import (
	"testing"
	"unsafe"
)

func TestMetaPage(t *testing.T) {
	assert := newAssertions(t)

	assert.Run("fail to validate", func(assert *assertions) {
		// var buf metaBuf
		buf := make([]byte, unsafe.Sizeof(metaPage{}))
		hdr := castMetaPage(buf[:])
		assert.Error(hdr.Validate())
		hdr.magic.Set(magic)
		assert.Error(hdr.Validate())
		hdr.version.Set(version)
		assert.Error(hdr.Validate())
	})

	assert.Run("fail if checksum not set", func(assert *assertions) {
		// var buf metabuf
		buf := make([]byte, unsafe.Sizeof(metaPage{}))
		hdr := castMetaPage(buf[:])
		hdr.Init(0, 4096, 1<<30)
		assert.Error(hdr.Validate())
	})

	assert.Run("with checksum", func(assert *assertions) {
		// var buf metabuf
		buf := make([]byte, unsafe.Sizeof(metaPage{}))
		hdr := castMetaPage(buf[:])
		hdr.Init(0, 4096, 1<<30)
		hdr.Finalize()
		assert.NoError(hdr.Validate())
	})

	assert.Run("check if contents changed", func(assert *assertions) {
		// var buf metabuf
		buf := make([]byte, unsafe.Sizeof(metaPage{}))
		hdr := castMetaPage(buf[:])
		hdr.Init(0, 4096, 1<<30)
		hdr.Finalize()
		buf[4] = 0xff
		assert.Error(hdr.Validate())
	})
}
