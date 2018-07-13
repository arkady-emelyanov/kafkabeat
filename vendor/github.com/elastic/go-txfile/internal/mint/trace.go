package mint

type TestLogTracer struct {
	backends []backend
}

type backend struct {
	print  func(...interface{})
	printf func(string, ...interface{})
}

func NewTestLogTracer(loggers ...interface{}) *TestLogTracer {
	type testLogger interface {
		Log(...interface{})
		Logf(string, ...interface{})
	}

	type tracer interface {
		Println(...interface{})
		Printf(string, ...interface{})
	}

	bs := make([]backend, 0, len(loggers))
	for _, logger := range loggers {
		var to backend

		switch v := logger.(type) {
		case testLogger:
			to = backend{print: v.Log, printf: v.Logf}
		case tracer:
			to = backend{print: v.Println, printf: v.Printf}
		}

		if to.print != nil {
			bs = append(bs, to)
		}
	}

	return &TestLogTracer{bs}
}

func (t *TestLogTracer) Println(vs ...interface{}) {
	for _, b := range t.backends {
		b.print(vs...)
	}
}

func (t *TestLogTracer) Printf(fmt string, vs ...interface{}) {
	for _, b := range t.backends {
		b.printf(fmt, vs...)
	}
}
