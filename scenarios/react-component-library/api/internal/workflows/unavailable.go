package workflows

import (
	"context"
	"errors"
)

// UnavailableDispatcher is the explicit graceful-degradation seam used until
// lifecycle discovery resolves Agent Manager. A durable workflow still records
// the attempted request; browser callers never fall back to direct dispatch.
type UnavailableDispatcher struct{ Reason string }

func (d UnavailableDispatcher) Dispatch(context.Context, StartInput) (DispatchResult, error) {
	if d.Reason == "" {
		d.Reason = "agent-manager is unavailable"
	}
	return DispatchResult{}, errors.New(d.Reason)
}
func (d UnavailableDispatcher) Snapshot(context.Context, string, int64) (RunSnapshot, error) {
	return RunSnapshot{}, errors.New(d.Reason)
}
func (d UnavailableDispatcher) Stop(context.Context, string) (RunSnapshot, error) {
	return RunSnapshot{}, errors.New(d.Reason)
}
