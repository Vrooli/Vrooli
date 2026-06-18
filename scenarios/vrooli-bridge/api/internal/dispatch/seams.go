package dispatch

import "context"

// NodeReader is the registry read seam: dispatch projects a node down to the
// TargetNode it needs (scopes + revocation). The handler adapter wraps the
// registry service. A missing node surfaces as ErrNodeNotFound.
type NodeReader interface {
	GetTarget(ctx context.Context, id string) (TargetNode, error)
}

// Presence is the live online/offline seam (the presence hub satisfies it).
type Presence interface {
	IsOnline(nodeID string) bool
}

// RunController is the runs domain seam dispatch needs: create a durable run and
// (on a delivery failure) abort it. The handler adapter wraps runs.Service.
type RunController interface {
	// CreateRun persists a new QUEUED run for the already-validated job and
	// returns its server-owned id.
	CreateRun(ctx context.Context, in CreateRunInput) (runID string, err error)
	// AbortRun marks a run terminal-aborted (used when delivery fails after the
	// run was created).
	AbortRun(ctx context.Context, runID, reason string) error
}

// CreateRunInput is the dispatch-local DTO for run creation.
type CreateRunInput struct {
	NodeID         string
	Scenario       string
	Verb           string
	Args           []string
	TimeoutSeconds int64
}

// AuditSink is the append-only accountability seam (the audit store satisfies
// it via the handler adapter). Every dispatch — accepted or rejected — is
// recorded.
type AuditSink interface {
	Record(ctx context.Context, e Entry) error
}

// Entry is the dispatch-local audit DTO. Accepted distinguishes a dispatched
// job from a denied one; Detail carries the rejection reason or acceptance note.
type Entry struct {
	Actor    string
	NodeID   string
	Scenario string
	Verb     string
	Args     []string
	Accepted bool
	Detail   string
	RunID    string
}

// JobPusher is the channel push seam: deliver the typed job to the node's held
// dial-out channel. The handler adapter translates PushedJob into a
// channel.JobPush ServerFrame and calls the presence hub's push. delivered is
// the number of live connections the frame reached (0 means the node dropped
// between the online check and the push).
type JobPusher interface {
	PushJob(ctx context.Context, nodeID string, job PushedJob) (delivered int, err error)
}

// PushedJob is the dispatch-local DTO for the pushed job (proto-free).
type PushedJob struct {
	RunID          string
	Scenario       string
	Verb           string
	Args           []string
	TimeoutSeconds int64
}
