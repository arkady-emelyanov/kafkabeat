package cleanup_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-txfile/internal/cleanup"
)

func TestIfBool(t *testing.T) {
	testcases := []struct {
		title   string
		fn      func(*bool, func())
		value   bool
		cleanup bool
	}{
		{
			"IfNot runs cleanup",
			cleanup.IfNot, false, true,
		},
		{
			"IfNot does not run cleanup",
			cleanup.IfNot, true, false,
		},
		{
			"If runs cleanup",
			cleanup.If, true, true,
		},
		{
			"If does not run cleanup",
			cleanup.If, false, false,
		},
	}

	for _, test := range testcases {
		test := test
		t.Run(test.title, func(t *testing.T) {
			executed := false
			func() {
				v := test.value
				defer test.fn(&v, func() { executed = true })
			}()

			assert.Equal(t, test.cleanup, executed)
		})
	}
}
