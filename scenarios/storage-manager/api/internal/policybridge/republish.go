package policybridge

// Republisher lets another subsystem (the deletion-grant resolver) ask the
// publish loop to refresh the snapshot immediately instead of waiting for the
// next PublishInterval tick. A just-approved grant takes effect in seconds
// rather than up to half an hour.
//
// The trigger is a size-1 buffered channel used as a coalescing signal: Trigger
// never blocks, and a burst of approvals collapses into at most one pending
// refresh (the publish reads the current grant set, so one pass covers them
// all). The periodic ticker remains the backstop, so a dropped trigger only
// delays a refresh, never loses one.
type Republisher struct {
	ch chan struct{}
}

// NewRepublisher builds a Republisher. A nil *Republisher is safe to pass to
// Start and to Trigger (both treat it as "no on-demand refresh").
func NewRepublisher() *Republisher {
	return &Republisher{ch: make(chan struct{}, 1)}
}

// Trigger requests a refresh without blocking. If one is already pending it is
// a no-op (the pending refresh will observe the latest state).
func (r *Republisher) Trigger() {
	if r == nil {
		return
	}
	select {
	case r.ch <- struct{}{}:
	default:
	}
}

// signal returns the channel the publish loop selects on, or nil when there is
// no republisher (a nil receive blocks forever, which is the intended "ticker
// only" behaviour in a select).
func (r *Republisher) signal() <-chan struct{} {
	if r == nil {
		return nil
	}
	return r.ch
}
