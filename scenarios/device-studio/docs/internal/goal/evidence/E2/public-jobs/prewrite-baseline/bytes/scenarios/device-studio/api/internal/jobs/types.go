// Package jobs owns durable local render-job state. It records authoritative
// transitions and receipts but deliberately does not launch or stop workers.
package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type State string

const (
	Queued          State = "queued"
	Running         State = "running"
	Checkpointed    State = "checkpointed"
	CancelRequested State = "cancel_requested"
	Succeeded       State = "succeeded"
	Failed          State = "failed"
	Cancelled       State = "cancelled"
)

type AttemptState string

const (
	AttemptQueued       AttemptState = "queued"
	AttemptRunning      AttemptState = "running"
	AttemptCheckpointed AttemptState = "checkpointed"
	AttemptSucceeded    AttemptState = "succeeded"
	AttemptFailed       AttemptState = "failed"
	AttemptCancelled    AttemptState = "cancelled"
)

type SettlementState string

const (
	SettlementNotRequired    SettlementState = "not_required"
	SettlementPending        SettlementState = "pending"
	SettlementSettled        SettlementState = "settled"
	SettlementReconciliation SettlementState = "reconciliation_required"
)

var (
	ErrNotFound             = errors.New("job not found")
	ErrAlreadyTerminal      = errors.New("job is already terminal")
	ErrInvalidTransition    = errors.New("invalid job transition")
	ErrIdempotencyConflict  = errors.New("idempotency key was used with different immutable inputs")
	ErrCheckpointMismatch   = errors.New("checkpoint identity does not match the pinned job and attempt")
	ErrInvalid              = errors.New("invalid job input")
	ErrConcurrentTransition = errors.New("job changed during transition")
)

type ArtifactBinding struct {
	Role       string `json:"role"`
	ArtifactID string `json:"artifact_id"`
	RevisionID string `json:"revision_id"`
	SHA256     string `json:"sha256"`
}

// Input is the immutable execution closure. Binary bytes remain in the blob
// owner; every referenced artifact is bound by revision and digest here.
type Input struct {
	ProjectID             string            `json:"project_id"`
	ShotRevisionID        string            `json:"shot_revision_id"`
	ShotRevision          int64             `json:"shot_revision"`
	Bindings              []ArtifactBinding `json:"bindings"`
	SourceClockDigest     string            `json:"source_clock_digest"`
	ToolProfileDigest     string            `json:"tool_profile_digest"`
	RendererVersion       string            `json:"renderer_version"`
	OutputProfile         string            `json:"output_profile"`
	AccountingOperationID string            `json:"accounting_operation_id,omitempty"`
	AccountingRequired    bool              `json:"accounting_required"`
}

type Job struct {
	ID                string           `json:"id"`
	IdempotencyKey    string           `json:"idempotency_key"`
	Input             Input            `json:"input"`
	InputFingerprint  string           `json:"input_fingerprint"`
	State             State            `json:"state"`
	Version           int64            `json:"version"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	CancelRequestedAt *time.Time       `json:"cancel_requested_at,omitempty"`
	Quiescence        *QuiescenceProof `json:"quiescence,omitempty"`
	Failure           string           `json:"failure,omitempty"`
	Settlement        SettlementState  `json:"settlement"`
	Attempts          []Attempt        `json:"attempts"`
}
type Attempt struct {
	ID                   string              `json:"id"`
	Number               int                 `json:"number"`
	State                AttemptState        `json:"state"`
	CreatedAt            time.Time           `json:"created_at"`
	StartedAt            *time.Time          `json:"started_at,omitempty"`
	HeartbeatAt          *time.Time          `json:"heartbeat_at,omitempty"`
	FinishedAt           *time.Time          `json:"finished_at,omitempty"`
	Execution            *ExecutionClaim     `json:"execution,omitempty"`
	Checkpoint           *Checkpoint         `json:"checkpoint,omitempty"`
	CheckpointQuiescence *QuiescenceProof    `json:"checkpoint_quiescence,omitempty"`
	ResumeFrom           *Checkpoint         `json:"resume_from,omitempty"`
	Completion           *CompletionEvidence `json:"completion,omitempty"`
	Failure              string              `json:"failure,omitempty"`
}

// ExecutionClaim is an immutable reference to the worker owner's persisted
// execution scope. It is bound to one job attempt and closure before work runs.
// The job domain stores the reference; it cannot establish that external bytes
// or process controls actually exist.
type ExecutionClaim struct {
	ID                string    `json:"id"`
	JobID             string    `json:"job_id"`
	AttemptID         string    `json:"attempt_id"`
	AttemptNumber     int       `json:"attempt_number"`
	InputFingerprint  string    `json:"input_fingerprint"`
	ToolProfileDigest string    `json:"tool_profile_digest"`
	SourceClockDigest string    `json:"source_clock_digest"`
	OwnerID           string    `json:"owner_id"`
	ScopeID           string    `json:"scope_id"`
	ArtifactID        string    `json:"artifact_id"`
	SHA256            string    `json:"sha256"`
	ClaimedAt         time.Time `json:"claimed_at"`
}

// ExecutionClaimInput is supplied by the actual worker/process owner. The
// service binds it to the admitted job and attempt before persistence.
type ExecutionClaimInput struct {
	AttemptID     string `json:"attempt_id"`
	AttemptNumber int    `json:"attempt_number"`
	OwnerID       string `json:"owner_id"`
	ScopeID       string `json:"scope_id"`
	ArtifactID    string `json:"artifact_id"`
	SHA256        string `json:"sha256"`
}
type Checkpoint struct {
	AttemptID         string    `json:"attempt_id"`
	AttemptNumber     int       `json:"attempt_number"`
	ExecutionClaimID  string    `json:"execution_claim_id"`
	InputFingerprint  string    `json:"input_fingerprint"`
	ToolProfileDigest string    `json:"tool_profile_digest"`
	SourceClockDigest string    `json:"source_clock_digest"`
	Unit              int64     `json:"unit"`
	ArtifactID        string    `json:"artifact_id"`
	SHA256            string    `json:"sha256"`
	CreatedAt         time.Time `json:"created_at"`
}
type OutputRef struct {
	Kind       string `json:"kind"`
	ArtifactID string `json:"artifact_id"`
	SHA256     string `json:"sha256"`
	MIMEType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
}

// CompletionEvidence is supplied by the worker/adapter, never synthesized by
// the job service. The adapter must validate bytes before invoking Complete.
type CompletionEvidence struct {
	AttemptID         string      `json:"attempt_id"`
	ExecutionClaimID  string      `json:"execution_claim_id"`
	InputFingerprint  string      `json:"input_fingerprint"`
	Outputs           []OutputRef `json:"outputs"`
	ReceiptArtifactID string      `json:"receipt_artifact_id"`
	ReceiptSHA256     string      `json:"receipt_sha256"`
	CompletedAt       time.Time   `json:"completed_at"`
}
type QuiescenceProof struct {
	JobID            string    `json:"job_id"`
	AttemptID        string    `json:"attempt_id"`
	InputFingerprint string    `json:"input_fingerprint"`
	ExecutionClaimID string    `json:"execution_claim_id,omitempty"`
	OwnerID          string    `json:"owner_id"`
	ScopeID          string    `json:"scope_id"`
	ProofArtifactID  string    `json:"proof_artifact_id,omitempty"`
	ProofSHA256      string    `json:"proof_sha256,omitempty"`
	ObservedAt       time.Time `json:"observed_at"`
	NoOwnedExecution bool      `json:"no_owned_execution"`
	PreDispatch      bool      `json:"pre_dispatch,omitempty"`
	Reason           string    `json:"reason"`
}
type RecoveryClass string

const (
	RecoveryQueued    RecoveryClass = "queued_dispatch_eligible"
	RecoveryLive      RecoveryClass = "lease_live"
	RecoveryResume    RecoveryClass = "checkpoint_resume_candidate"
	RecoveryReconcile RecoveryClass = "worker_outcome_reconciliation_required"
	RecoveryCancel    RecoveryClass = "cancellation_quiescence_required"
	RecoveryTerminal  RecoveryClass = "terminal"
)

type Recovery struct {
	JobID      string
	Class      RecoveryClass
	AttemptID  string
	Checkpoint *Checkpoint
}

type SubmitInput struct {
	IdempotencyKey string
	Input          Input
}

func CanonicalInput(in Input) (Input, string, error) {
	in.ProjectID = strings.TrimSpace(in.ProjectID)
	in.ShotRevisionID = strings.TrimSpace(in.ShotRevisionID)
	in.RendererVersion = strings.TrimSpace(in.RendererVersion)
	in.OutputProfile = strings.TrimSpace(in.OutputProfile)
	if in.ProjectID == "" || in.ShotRevisionID == "" || in.ShotRevision <= 0 || in.RendererVersion == "" || in.OutputProfile == "" {
		return Input{}, "", fmt.Errorf("%w: project, shot revision, renderer and output profile are required", ErrInvalid)
	}
	if in.AccountingRequired && strings.TrimSpace(in.AccountingOperationID) == "" {
		return Input{}, "", fmt.Errorf("%w: accounting operation id is required when settlement is tracked", ErrInvalid)
	}
	if !in.AccountingRequired && strings.TrimSpace(in.AccountingOperationID) != "" {
		return Input{}, "", fmt.Errorf("%w: accounting operation requires settlement tracking", ErrInvalid)
	}
	in.AccountingOperationID = strings.TrimSpace(in.AccountingOperationID)
	if err := validDigest(in.SourceClockDigest); err != nil {
		return Input{}, "", fmt.Errorf("%w: source clock digest: %v", ErrInvalid, err)
	}
	if err := validDigest(in.ToolProfileDigest); err != nil {
		return Input{}, "", fmt.Errorf("%w: tool profile digest: %v", ErrInvalid, err)
	}
	if len(in.Bindings) == 0 {
		return Input{}, "", fmt.Errorf("%w: at least one immutable artifact binding is required", ErrInvalid)
	}
	in.Bindings = append([]ArtifactBinding(nil), in.Bindings...)
	for i := range in.Bindings {
		b := &in.Bindings[i]
		b.Role, b.ArtifactID, b.RevisionID = strings.TrimSpace(b.Role), strings.TrimSpace(b.ArtifactID), strings.TrimSpace(b.RevisionID)
		if b.Role == "" || b.ArtifactID == "" || b.RevisionID == "" {
			return Input{}, "", fmt.Errorf("%w: artifact role, id, and revision are required", ErrInvalid)
		}
		if err := validDigest(b.SHA256); err != nil {
			return Input{}, "", fmt.Errorf("%w: artifact %s digest: %v", ErrInvalid, b.ArtifactID, err)
		}
	}
	sort.Slice(in.Bindings, func(i, j int) bool { return in.Bindings[i].Role < in.Bindings[j].Role })
	for i := 1; i < len(in.Bindings); i++ {
		if in.Bindings[i-1].Role == in.Bindings[i].Role {
			return Input{}, "", fmt.Errorf("%w: duplicate artifact role %q", ErrInvalid, in.Bindings[i].Role)
		}
	}
	data, err := json.Marshal(in)
	if err != nil {
		return Input{}, "", err
	}
	sum := sha256.Sum256(data)
	return in, hex.EncodeToString(sum[:]), nil
}
func validDigest(s string) error {
	if len(s) != 64 {
		return errors.New("must be 64 lowercase hexadecimal characters")
	}
	if strings.ToLower(s) != s {
		return errors.New("must be lowercase hexadecimal")
	}
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return errors.New("must be 64 lowercase hexadecimal characters")
	}
	return nil
}
func validateExecutionClaim(j Job, a Attempt, c ExecutionClaim) error {
	if c.ID == "" || c.JobID != j.ID || c.AttemptID != a.ID || c.AttemptNumber != a.Number || c.InputFingerprint != j.InputFingerprint || c.ToolProfileDigest != j.Input.ToolProfileDigest || c.SourceClockDigest != j.Input.SourceClockDigest || c.OwnerID == "" || c.ScopeID == "" || c.ArtifactID == "" || validDigest(c.SHA256) != nil || c.ClaimedAt.IsZero() {
		return ErrInvalid
	}
	return nil
}
func validateQuiescence(j Job, a Attempt, p QuiescenceProof) error {
	if !p.NoOwnedExecution || p.JobID != j.ID || p.AttemptID != a.ID || p.InputFingerprint != j.InputFingerprint || p.ScopeID == "" || p.OwnerID == "" || p.Reason == "" || p.ObservedAt.IsZero() {
		return fmt.Errorf("%w: quiescence proof is not bound to this attempt", ErrInvalid)
	}
	if p.PreDispatch {
		if (a.State != AttemptQueued && a.State != AttemptCancelled) || a.Execution != nil || p.ExecutionClaimID != "" || p.OwnerID != "jobs" || p.ScopeID != "pre-dispatch:"+j.ID || p.ProofArtifactID != "" || p.ProofSHA256 != "" {
			return fmt.Errorf("%w: invalid pre-dispatch proof", ErrInvalid)
		}
		return nil
	}
	if a.Execution == nil || p.ExecutionClaimID != a.Execution.ID || p.OwnerID != a.Execution.OwnerID || p.ScopeID != a.Execution.ScopeID || p.ProofArtifactID == "" || validDigest(p.ProofSHA256) != nil || p.ObservedAt.Before(a.Execution.ClaimedAt) {
		return fmt.Errorf("%w: proof does not match persisted execution ownership", ErrInvalid)
	}
	return nil
}
func validateCheckpoint(j Job, a Attempt, c Checkpoint) error {
	if c.AttemptID != a.ID || c.AttemptNumber != a.Number || c.ExecutionClaimID == "" || a.Execution == nil || c.ExecutionClaimID != a.Execution.ID || c.InputFingerprint != j.InputFingerprint || c.ToolProfileDigest != j.Input.ToolProfileDigest || c.SourceClockDigest != j.Input.SourceClockDigest || c.Unit < 0 || c.ArtifactID == "" || validDigest(c.SHA256) != nil || c.CreatedAt.IsZero() {
		return ErrCheckpointMismatch
	}
	return nil
}
func validateCompletion(j Job, a Attempt, c CompletionEvidence) error {
	if a.Execution == nil || c.AttemptID != a.ID || c.ExecutionClaimID != a.Execution.ID || c.InputFingerprint != j.InputFingerprint || c.CompletedAt.IsZero() || c.ReceiptArtifactID == "" || validDigest(c.ReceiptSHA256) != nil || len(c.Outputs) == 0 {
		return fmt.Errorf("%w: completion evidence is incomplete or bound to another execution", ErrInvalid)
	}
	seen := map[string]bool{}
	for _, o := range c.Outputs {
		if o.Kind == "" || o.ArtifactID == "" || seen[o.Kind] || validDigest(o.SHA256) != nil || o.MIMEType == "" || o.SizeBytes <= 0 {
			return fmt.Errorf("%w: output reference is incomplete or duplicated", ErrInvalid)
		}
		seen[o.Kind] = true
	}
	return nil
}
func isTerminal(s State) bool { return s == Succeeded || s == Failed || s == Cancelled }
