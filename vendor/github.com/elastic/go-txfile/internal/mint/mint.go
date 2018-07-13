package mint

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// T wraps the testing, assert, and quick packages in order to provide
// some unified functionality.
type T struct {
	T *testing.T
	*assert.Assertions
	wrap Wrap

	defaultGenerators []interface{}
}

// Wrap function will be called before sub-tests are run.
// The wrapper function must return a 'teardown' function.
type Wrap func(*T) func()

// New returns a new T for testing.
func New(t *testing.T) *T {
	return &T{
		T:          t,
		Assertions: assert.New(t),
	}
}

// NewWith returns a new T with custom test wrapper for testing.
func NewWith(t *testing.T, w Wrap) *T {
	return &T{
		T:          t,
		Assertions: assert.New(t),
		wrap:       w,
	}
}

func (t *T) Run(name string, fn func(t *T)) bool {
	var ok bool
	t.T.Run(name, func(std *testing.T) {
		sub := NewWith(std, t.wrap)
		sub.defaultGenerators = t.defaultGenerators

		if sub.wrap != nil {
			teardown := sub.wrap(sub)
			defer teardown()
		}

		fn(sub)
		ok = !t.Failed()
	})
	return ok
}

func (t *T) Log(vs ...interface{}) {
	t.T.Helper()
	t.T.Log(vs...)
}

func (t *T) Logf(fmt string, vs ...interface{}) {
	t.T.Helper()
	t.T.Logf(fmt, vs...)
}

func (t *T) Fatal(v interface{}) {
	t.T.Helper()
	t.T.Fatal(v)
}

func (t *T) FatalOnError(err error, msgAndArgs ...interface{}) {
	t.T.Helper()
	if err != nil {
		t.T.Fatal(fmt.Sprintf("error: %v\n%v", err, messageFromMsgAndArgs(msgAndArgs)))
	}
}

func (t *T) Skip(args ...interface{}) {
	t.T.Helper()
	t.T.Skip(args...)
}

func (t *T) Failed() bool {
	return t.T.Failed()
}

func messageFromMsgAndArgs(msgAndArgs []interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		return msgAndArgs[0].(string)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}
