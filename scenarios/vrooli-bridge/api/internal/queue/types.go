// Package queue is the per-node job scheduler (OT-P1-004): each node runs at
// most a small bounded number of jobs at once (default 1, mirroring test-genie's
// one-run-per-scenario discipline so a gate fan-out never thrashes a node), and
// additional dispatched jobs QUEUE in fair FIFO order, pulled as running slots
// free.
//
// The scheduler sits transparently on the dispatch → channel-push path: it
// satisfies dispatch's existing job-push seam (Submit returns the same
// (delivered, error) contract), so a dispatch whose target node has a free slot
// is pushed immediately and one whose node is busy is held queued (its durable
// run stays QUEUED until a slot frees). When a run reaches a terminal status the
// runs domain calls Complete, which frees the slot and promotes the next queued
// job. The durable source of truth for a job is still its Run (runs domain);
// this scheduler holds only the live, in-memory scheduling state, surfaced
// read-only by the QueueService.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs: the channel push (Pusher) and the run-abort (Aborter). The
// scheduler imports no sibling domain and no proto.
package queue

import "time"

// DefaultConcurrencyLimit is a node's default maximum concurrent running jobs.
// One mirrors test-genie's one-run-per-scenario discipline; a node can be given
// a larger bound when its hardware warrants it.
const DefaultConcurrencyLimit = 1

// State is a scheduled job's live state.
type State int

const (
	// StateUnspecified is the zero value.
	StateUnspecified State = 0
	// StateQueued — waiting for a running slot on its node to free.
	StateQueued State = 1
	// StateRunning — occupying a running slot (pushed to the node; running until
	// its run reaches a terminal status).
	StateRunning State = 2
)

// String renders the state as a short lowercase label.
func (s State) String() string {
	switch s {
	case StateQueued:
		return "queued"
	case StateRunning:
		return "running"
	default:
		return "unspecified"
	}
}

// Job is the scheduler's proto-free job DTO (a dispatched, allowlist-validated
// job bound to a durable run).
type Job struct {
	RunID                string
	NodeID               string
	Scenario             string
	Verb                 string
	Args                 []string
	TimeoutSeconds       int64
	Outputs              []Output
	CredentialInjections []CredentialInjection
}

type CredentialInjection struct {
	LogicalID string
	Field     string
	EnvName   string
}

// Output is a typed artifact declaration carried to the node.
type Output struct {
	Name       string
	MediaType  string
	OutputFlag string
	MaxBytes   int64
}

// Entry is one job's line in a node's live queue view.
type Entry struct {
	Job
	State State
	// Position is the 0-based index among the node's QUEUED jobs; running jobs
	// report -1.
	Position int
	// EnqueuedAt is when the job entered the scheduler.
	EnqueuedAt time.Time
	// StartedAt is when the job was pushed to the node (set once State ==
	// StateRunning).
	StartedAt        time.Time
	PushedAt         time.Time
	AckedAt          time.Time
	LeaseExpiresAt   time.Time
	DeliveryAttempts int
	Acked            bool
}

// NodeQueue is one node's slice of the live queue: its running + queued jobs and
// its concurrency bound.
type NodeQueue struct {
	NodeID           string
	ConcurrencyLimit int
	Running          int
	Queued           int
	// Entries are running first, then queued in FIFO order.
	Entries []Entry
}

// Outcome is what Submit reports: whether the job was pushed to the node now or
// held queued.
type Outcome int

const (
	// OutcomePushed — a running slot was free; the job was pushed immediately.
	OutcomePushed Outcome = 1
	// OutcomeQueued — the node was at its bound; the job is queued.
	OutcomeQueued Outcome = 2
)
