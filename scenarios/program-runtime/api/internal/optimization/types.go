package optimization

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("optimization run not found")
	ErrRevision         = errors.New("optimization run revision conflict")
	ErrInvalidRequest   = errors.New("invalid optimization request")
	ErrQueueFull        = errors.New("optimization queue has reached its active-run limit")
	ErrRetryNotReady    = errors.New("optimization retry is not due yet")
	ErrNoAdapter        = errors.New("optimization target adapter is unavailable")
	ErrOwnerUnavailable = errors.New("optimization target owner is not configured")
)

// Stage is monotone. An owner operation is repeated only with the same run and
// stage id; adapters must make each operation idempotent for that identity.
type Stage string

const (
	StagePreflightRights Stage = "preflight_rights"
	StagePreflightDisk   Stage = "preflight_disk"
	StageAcquire         Stage = "acquire"
	StageEvaluate        Stage = "evaluate"
	StageRights          Stage = "rights"
	StageDisk            Stage = "disk"
	StagePromote         Stage = "promote"
	StageCleanup         Stage = "cleanup"
	StageComplete        Stage = "complete"
	StageRefused         Stage = "refused"
)

// Request contains the immutable work identity. Target-specific details remain
// opaque to the engine and are interpreted only by the named owner adapter.
type Request struct {
	RunID         string         `json:"run_id"`
	Target        string         `json:"target"`
	Subject       map[string]any `json:"subject"`
	Candidates    []Candidate    `json:"candidates"`
	PredecessorID string         `json:"predecessor_id,omitempty"`
	SuiteID       string         `json:"suite_id"`
	CaseIDs       []string       `json:"case_ids"`
	MinimumCases  int            `json:"minimum_cases"`
}

type Candidate struct {
	ID                string `json:"id"`
	ArtifactDigest    string `json:"artifact_digest"`
	PredecessorDigest string `json:"predecessor_digest,omitempty"`
}

type Proof struct {
	Decision       string    `json:"decision"`
	ReceiptID      string    `json:"receipt_id"`
	SubjectDigest  string    `json:"subject_digest"`
	ExpiresAt      time.Time `json:"expires_at"`
	RequiredBytes  int64     `json:"required_bytes,omitempty"`
	AvailableBytes int64     `json:"available_bytes,omitempty"`
}

type Evidence struct {
	Stage       Stage          `json:"stage"`
	ReceiptID   string         `json:"receipt_id,omitempty"`
	Disposition string         `json:"disposition,omitempty"`
	Proof       *Proof         `json:"proof,omitempty"`
	Details     map[string]any `json:"details,omitempty"`
	RecordedAt  time.Time      `json:"recorded_at"`
}

type Record struct {
	Request       Request       `json:"request"`
	Stage         Stage         `json:"stage"`
	Revision      uint64        `json:"revision"`
	WinnerID      string        `json:"winner_id,omitempty"`
	Evidence      []Evidence    `json:"evidence"`
	Failure       string        `json:"failure,omitempty"`
	LastError     string        `json:"last_error,omitempty"`
	RetryAt       time.Time     `json:"retry_at,omitempty"`
	StageAttempts map[Stage]int `json:"stage_attempts,omitempty"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type Store interface {
	Create(context.Context, Record) (Record, error)
	Load(context.Context, string) (Record, error)
	CompareAndSwap(context.Context, uint64, Record) (Record, error)
	ListActive(context.Context, int) ([]Record, error)
}

// Adapter is an owner boundary. Every mutating method receives a stable run
// identity so a retry after process loss returns the original receipt instead
// of repeating an acquisition, promotion, or deletion.
type Adapter interface {
	Target() string
	Validate(Request) error
	Acquire(context.Context, string, Request) (Evidence, error)
	Evaluate(context.Context, string, Request, []Evidence) (Evidence, error)
	CheckRights(context.Context, string, Stage, Request, []Evidence) (Evidence, error)
	CheckDisk(context.Context, string, Stage, Request, []Evidence) (Evidence, error)
	Promote(context.Context, string, Request, []Evidence) (Evidence, error)
	Cleanup(context.Context, string, Request, []Evidence) (Evidence, error)
}

type Engine struct {
	Store      Store
	Adapters   map[string]Adapter
	Now        func() time.Time
	RetryDelay func(int) time.Duration
	locks      runLockSet
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
