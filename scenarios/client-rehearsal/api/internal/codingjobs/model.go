package codingjobs

import (
	"context"
	"time"
)

type State string

const (
	StateReady      State = "ready"
	StateSolving    State = "solving"
	StateScoring    State = "scoring"
	StateReviewable State = "reviewable"
	StateFailed     State = "failed"
)

type Job struct {
	ID                     string    `json:"id"`
	ReplayKey              string    `json:"replay_key"`
	CaseID                 string    `json:"case_id"`
	Project                string    `json:"project"`
	Title                  string    `json:"title"`
	Task                   string    `json:"task"`
	State                  State     `json:"state"`
	SourceSealRef          string    `json:"source_seal_ref"`
	SolverRunRef           string    `json:"solver_run_ref"`
	EvaluatorReceiptRef    string    `json:"evaluator_receipt_ref"`
	EvaluatorReceiptSHA256 string    `json:"evaluator_receipt_sha256"`
	Score                  string    `json:"score"`
	Revision               int64     `json:"revision"`
	Events                 []Event   `json:"events"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type Event struct {
	Sequence       uint64    `json:"sequence"`
	Stage          string    `json:"stage"`
	Actor          string    `json:"actor"`
	EvidenceRef    string    `json:"evidence_ref,omitempty"`
	EvidenceSHA256 string    `json:"evidence_sha256,omitempty"`
	Note           string    `json:"note,omitempty"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type CreateInput struct {
	ReplayKey     string
	CaseID        string
	Project       string
	Title         string
	Task          string
	SourceSealRef string
}

type StageInput struct {
	ID               string
	Stage            string
	Actor            string
	EvidenceRef      string
	EvidenceSHA256   string
	Note             string
	ExpectedRevision int64
}

type EvaluationInput struct {
	ID               string
	ReceiptRef       string
	ReceiptSHA256    string
	Score            string
	ExpectedRevision int64
}

type Repository interface {
	Create(context.Context, Job) (Job, error)
	Get(context.Context, string) (Job, error)
	GetByReplayKey(context.Context, string) (Job, error)
	List(context.Context, int) ([]Job, error)
	Update(context.Context, Job, int64) (Job, error)
}

type Service interface {
	Create(context.Context, CreateInput) (Job, error)
	RecordStage(context.Context, StageInput) (Job, error)
	RecordEvaluation(context.Context, EvaluationInput) (Job, error)
	Get(context.Context, string) (Job, error)
	List(context.Context, int) ([]Job, error)
}

type Clock func() time.Time
