package txfile

import "testing"

func TestIDList(t *testing.T) {
	assert := newAssertions(t)

	assert.Run("query nil list", func(assert *assertions) {
		var l idList
		assert.Nil(l.ToSet())
		assert.Nil(l.Regions())
		assert.NotPanics(l.Sort)
	})

	assert.Run("query empty list", func(assert *assertions) {
		l := idList{}
		assert.Nil(l.ToSet())
		assert.Nil(l.Regions())
		assert.NotPanics(l.Sort)
	})

	assert.Run("sort", func(assert *assertions) {
		l := idList{10, 6, 23, 1}
		l.Sort()
		assert.Equal(idList{1, 6, 10, 23}, l)
	})

	assert.Run("add to nil list", func(assert *assertions) {
		var l idList
		l.Add(2)
		l.Add(10)
		l.Add(1)
		assert.Equal(idList{2, 10, 1}, l)
	})

	assert.Run("transformers", func(assert *assertions) {
		l := idList{1, 2, 10}

		set := l.ToSet()
		for _, id := range l {
			assert.True(set.Has(id))
		}
		assert.False(set.Has(3))

		assert.Equal(regionList{{1, 2}, {10, 1}}, l.Regions())
	})
}
