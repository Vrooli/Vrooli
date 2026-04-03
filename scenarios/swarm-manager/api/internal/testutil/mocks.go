package testutil

import "swarm-manager/internal/dispatch"

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
