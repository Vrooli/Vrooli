package homeintegration

import "context"

type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, Event) error { return nil }

type CapturingPublisher struct {
	Events []Event
	Err    error
}

func (p *CapturingPublisher) Publish(_ context.Context, event Event) error {
	p.Events = append(p.Events, event)
	return p.Err
}
