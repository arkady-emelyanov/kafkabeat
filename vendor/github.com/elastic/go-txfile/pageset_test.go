package txfile

import "testing"

func TestPageSet(t *testing.T) {
	assert := newAssertions(t)

	assert.Run("query nil pageSet", func(assert *assertions) {
		var s pageSet
		assert.False(s.Has(1))
		assert.True(s.Empty())
		assert.Equal(0, s.Count())
		assert.Nil(s.IDs())
		assert.Nil(s.Regions())
	})

	assert.Run("query empty pageSet", func(assert *assertions) {
		s := pageSet{}
		assert.False(s.Has(1))
		assert.True(s.Empty())
		assert.Equal(0, s.Count())
		assert.Nil(s.IDs())
		assert.Nil(s.Regions())
	})

	assert.Run("pageSet modifications", func(assert *assertions) {
		var s pageSet

		s.Add(1)
		s.Add(2)
		s.Add(10)
		assert.False(s.Empty())
		assert.Equal(3, s.Count())

		assert.True(s.Has(1))
		assert.True(s.Has(2))
		assert.False(s.Has(3))
		assert.False(s.Has(4))
		assert.True(s.Has(10))

		ids := s.IDs()
		ids.Sort() // ids might be unsorted
		assert.Equal(idList{1, 2, 10}, ids)

		regions := s.Regions()
		assert.Equal(regionList{{1, 2}, {10, 1}}, regions)
	})
}
