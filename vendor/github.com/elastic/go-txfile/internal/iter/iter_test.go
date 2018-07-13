package iter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-txfile/internal/iter"
)

func TestIterForward(t *testing.T) {
	L := 10
	var is []int
	for i, end, next := iter.Forward(L); i != end; i = next(i) {
		is = append(is, i)
	}
	for i, value := range is {
		assert.Equal(t, i, value)
	}
}

func TestIterReversed(t *testing.T) {
	L := 10
	var is []int
	for i, end, next := iter.Reversed(L); i != end; i = next(i) {
		is = append(is, i)
	}
	for i, value := range is {
		assert.Equal(t, L-i-1, value)
	}
}
