package optimization

import (
	"context"
	"errors"
	"time"

	"network-manager/internal/adapters"
	"network-manager/internal/snapshot"
)

const TimeFormat = time.RFC3339Nano

var (
	ErrNotFound          = errors.New("optimization run not found")
	ErrCandidateNotFound = errors.New("optimization candidate not found")
	ErrBaselineRequired  = errors.New("baseline snapshot required before optimization")
	ErrApprovalRequired  = errors.New("optimization approval required")
	ErrManualRequired    = errors.New("manual recovery or adapter action required")
)

type Run struct {
	ID                 string
	Status             string
	ScoringProfile     string
	BaselineSnapshotID string
	Candidates         []Candidate
	Recommendation     string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Candidate struct {
	ID                  string
	RunID               string
	Description         string
	Status              string
	Score               float64
	Evidence            []string
	ApprovalRequired    bool
	RollbackSupported   bool
	RollbackHandle      string
	BaselineSnapshotID  string
	CandidateSnapshotID string
	AfterSnapshotID     string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ApprovalRecord struct {
	ID          string
	RunID       string
	CandidateID string
	Approved    bool
	Note        string
	CreatedAt   time.Time
}

type RollbackRecord struct {
	ID          string
	RunID       string
	CandidateID string
	Status      string
	Details     []string
	CreatedAt   time.Time
}

type Repository interface {
	SaveRun(ctx context.Context, run Run) (Run, error)
	GetRun(ctx context.Context, id string) (Run, error)
	UpdateRun(ctx context.Context, run Run) (Run, error)
	SaveCandidate(ctx context.Context, candidate Candidate) (Candidate, error)
	UpdateCandidate(ctx context.Context, candidate Candidate) (Candidate, error)
	SaveApproval(ctx context.Context, approval ApprovalRecord) (ApprovalRecord, error)
	SaveRollback(ctx context.Context, rollback RollbackRecord) (RollbackRecord, error)
}

type CapabilitySource interface {
	ListCapabilities(ctx context.Context) ([]adapters.Capability, error)
}

type SnapshotRunner interface {
	Run(ctx context.Context, profile string, dryRun bool) (snapshot.Snapshot, error)
}

type SnapshotReader interface {
	List(ctx context.Context) ([]snapshot.Snapshot, error)
}

type ApplyResult struct {
	Evidence       []string
	RollbackHandle string
}

type RollbackResult struct {
	Evidence []string
}

type Applier interface {
	Apply(ctx context.Context, run Run, candidate Candidate) (ApplyResult, error)
	Rollback(ctx context.Context, run Run, candidate Candidate) (RollbackResult, error)
}
