package queue

import (
	"context"
	"time"
)

// Pusher is the channel-push seam: deliver a job to the node's held dial-out
// channel. The handler adapter translates a Job into a channel.JobPush
// ServerFrame and pushes it via the presence hub. delivered is the number of
// live connections the frame reached (0 means the node dropped). The scheduler
// stays proto-free behind this seam.
type Pusher interface {
	Push(ctx context.Context, job Job) (delivered int, err error)
}

type Availability interface {
	IsAvailable(nodeID string) bool
}

// Aborter is the run-abort seam: mark a run terminal-aborted (used when a queued
// job cannot be delivered at promotion time because its node went away). The
// handler adapter wraps runs.Service.Abort.
type Aborter interface {
	Abort(ctx context.Context, runID, reason string) error
}

// DurableStore is the queue's persistence projection. The queue package stays
// independent of the runs domain while production can rebuild its live view
// from server-owned rows at boot.
type DurableStore interface {
	Load(ctx context.Context) ([]DurableEntry, error)
	MarkQueued(ctx context.Context, runID string, at time.Time, detail ...string) error
	MarkPushed(ctx context.Context, runID string, at, leaseExpiresAt time.Time) error
	MarkFailedDelivery(ctx context.Context, runID, reason string, at time.Time) error
}

type DurableEntry struct {
	Job              Job
	State            State
	EnqueuedAt       time.Time
	StartedAt        time.Time
	PushedAt         time.Time
	AckedAt          time.Time
	LeaseExpiresAt   time.Time
	DeliveryAttempts int
	Acked            bool
}
