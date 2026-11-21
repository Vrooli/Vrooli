package toolfacts

import "context"

// EventStream exposes the run's event stream to the external tests, which
// check the shape of the events the invocation readers consume.
func (r *Reader) EventStream(ctx context.Context, runID string) (eventStream, error) {
	return r.events(ctx, runID)
}
