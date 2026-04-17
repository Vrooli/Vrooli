package testutil

import (
	"errors"
	"net/http"
	"swarm-manager/internal/dispatch"
)

// NoopInvalidator is a no-op implementation of dispatch.Invalidator for tests.
type NoopInvalidator struct{}

func (NoopInvalidator) DispatchInvalidate(...string) {}

// Verify interface compliance
var _ dispatch.Invalidator = NoopInvalidator{}

// NoopNodeDispatcher is a no-op implementation of dispatch.NodeDispatcher for tests.
type NoopNodeDispatcher struct{ NoopInvalidator }

func (NoopNodeDispatcher) DispatchNodeUpdate(string, string, any) {}

var _ dispatch.NodeDispatcher = NoopNodeDispatcher{}

// RecordingInvalidator captures DispatchInvalidate calls for assertions.
type RecordingInvalidator struct {
	Calls [][]string
}

func (r *RecordingInvalidator) DispatchInvalidate(lenses ...string) {
	r.Calls = append(r.Calls, lenses)
}

var _ dispatch.Invalidator = &RecordingInvalidator{}

// ErrorWriter is an http.ResponseWriter that always fails on Write,
// for testing JSON encoding error paths.
type ErrorWriter struct {
	header   http.Header
	Statuses []int
}

func (e *ErrorWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}
	return e.header
}

func (e *ErrorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func (e *ErrorWriter) WriteHeader(statusCode int) {
	e.Statuses = append(e.Statuses, statusCode)
}

// HasStatus returns true if the given status code was written.
func (e *ErrorWriter) HasStatus(code int) bool {
	for _, s := range e.Statuses {
		if s == code {
			return true
		}
	}
	return false
}
