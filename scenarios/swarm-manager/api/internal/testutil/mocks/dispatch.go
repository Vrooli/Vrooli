package mocks

import "swarm-manager/internal/dispatch"

type NoopInvalidator struct{}

func (NoopInvalidator) DispatchInvalidate(...string) {}

var _ dispatch.Invalidator = NoopInvalidator{}

type NoopNodeDispatcher struct{ NoopInvalidator }

func (NoopNodeDispatcher) DispatchNodeUpdate(string, string, any) {}

var _ dispatch.NodeDispatcher = NoopNodeDispatcher{}

type RecordingInvalidator struct {
	Calls [][]string
}

func (r *RecordingInvalidator) DispatchInvalidate(lenses ...string) {
	r.Calls = append(r.Calls, lenses)
}

var _ dispatch.Invalidator = &RecordingInvalidator{}
