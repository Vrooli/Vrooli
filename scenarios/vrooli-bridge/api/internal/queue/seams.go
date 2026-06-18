package queue

import "context"

// Pusher is the channel-push seam: deliver a job to the node's held dial-out
// channel. The handler adapter translates a Job into a channel.JobPush
// ServerFrame and pushes it via the presence hub. delivered is the number of
// live connections the frame reached (0 means the node dropped). The scheduler
// stays proto-free behind this seam.
type Pusher interface {
	Push(ctx context.Context, job Job) (delivered int, err error)
}

// Aborter is the run-abort seam: mark a run terminal-aborted (used when a queued
// job cannot be delivered at promotion time because its node went away). The
// handler adapter wraps runs.Service.Abort.
type Aborter interface {
	Abort(ctx context.Context, runID, reason string) error
}
