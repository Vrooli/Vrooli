package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Submit(context.Context, SubmitInput) (Job, bool, error)
	Get(context.Context, string) (Job, error)
	Start(context.Context, string, ExecutionClaimInput, int64) (Job, error)
	Heartbeat(context.Context, string, string, int64) (Job, error)
	SaveCheckpoint(context.Context, string, string, Checkpoint, QuiescenceProof, int64) (Job, error)
	Complete(context.Context, string, string, CompletionEvidence, int64) (Job, error)
	Fail(context.Context, string, string, string, int64) (Job, error)
	RequestCancel(context.Context, string, int64) (Job, error)
	ConfirmCancelled(context.Context, string, string, QuiescenceProof, int64) (Job, error)
	Settle(context.Context, string, SettlementState, int64) (Job, error)
	Recoveries(context.Context, time.Duration) ([]Recovery, error)
}
type service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) Service {
	if now == nil {
		now = time.Now
	}
	return &service{repo: repo, now: now}
}

var _ Service = (*service)(nil)

func (s *service) Submit(ctx context.Context, in SubmitInput) (Job, bool, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" || len(key) > 200 {
		return Job{}, false, fmt.Errorf("%w: idempotency key required and limited to 200 bytes", ErrInvalid)
	}
	canonical, fp, err := CanonicalInput(in.Input)
	if err != nil {
		return Job{}, false, err
	}
	now := s.now().UTC()
	j := Job{ID: uuid.NewString(), IdempotencyKey: key, Input: canonical, InputFingerprint: fp, State: Queued, Version: 1, CreatedAt: now, UpdatedAt: now, Settlement: func() SettlementState {
		if canonical.AccountingRequired {
			return SettlementPending
		}
		return SettlementNotRequired
	}(), Attempts: []Attempt{{ID: uuid.NewString(), Number: 1, State: AttemptQueued, CreatedAt: now}}}
	return s.repo.Submit(ctx, j)
}
func (s *service) Get(ctx context.Context, id string) (Job, error) {
	if strings.TrimSpace(id) == "" {
		return Job{}, fmt.Errorf("%w: job id required", ErrInvalid)
	}
	return s.repo.Get(ctx, id)
}
func (s *service) Start(ctx context.Context, id string, owner ExecutionClaimInput, version int64) (Job, error) {
	j, err := s.repo.Get(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if j.Version != version {
		return Job{}, ErrConcurrentTransition
	}
	if strings.TrimSpace(owner.AttemptID) == "" || owner.AttemptNumber <= 0 || strings.TrimSpace(owner.OwnerID) == "" || strings.TrimSpace(owner.ScopeID) == "" || strings.TrimSpace(owner.ArtifactID) == "" || validDigest(owner.SHA256) != nil {
		return Job{}, fmt.Errorf("%w: worker owner must provide a durable execution claim reference", ErrInvalid)
	}
	switch j.State {
	case Queued:
		a, ok := currentAttempt(j)
		if !ok || a.State != AttemptQueued || owner.AttemptID != a.ID || owner.AttemptNumber != a.Number {
			return Job{}, ErrInvalidTransition
		}
	case Checkpointed:
		prev, ok := currentAttempt(j)
		if !ok || prev.State != AttemptCheckpointed || prev.Checkpoint == nil || prev.CheckpointQuiescence == nil || validateCheckpoint(j, prev, *prev.Checkpoint) != nil || validateQuiescence(j, prev, *prev.CheckpointQuiescence) != nil || prev.CheckpointQuiescence.ObservedAt.Before(prev.Checkpoint.CreatedAt) {
			return Job{}, ErrCheckpointMismatch
		}
		if attemptIDExists(j, owner.AttemptID) || executionScopeExists(j, owner.ScopeID) || owner.AttemptNumber != prev.Number+1 {
			return Job{}, ErrInvalidTransition
		}
	default:
		return Job{}, transitionErr(&j, "start")
	}
	now := s.now().UTC()
	claim := ExecutionClaim{ID: uuid.NewString(), JobID: j.ID, AttemptID: owner.AttemptID, AttemptNumber: owner.AttemptNumber, InputFingerprint: j.InputFingerprint, ToolProfileDigest: j.Input.ToolProfileDigest, SourceClockDigest: j.Input.SourceClockDigest, OwnerID: strings.TrimSpace(owner.OwnerID), ScopeID: strings.TrimSpace(owner.ScopeID), ArtifactID: strings.TrimSpace(owner.ArtifactID), SHA256: owner.SHA256, ClaimedAt: now}
	return s.repo.Start(ctx, claim, now.UnixNano(), version)
}
func (s *service) Heartbeat(ctx context.Context, id, attempt string, version int64) (Job, error) {
	return s.repo.Heartbeat(ctx, id, attempt, s.now().UTC().UnixNano(), version)
}
func (s *service) SaveCheckpoint(ctx context.Context, id, attempt string, c Checkpoint, proof QuiescenceProof, version int64) (Job, error) {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = s.now().UTC()
	}
	if proof.ObservedAt.IsZero() {
		proof.ObservedAt = s.now().UTC()
	}
	return s.repo.Checkpoint(ctx, id, attempt, c, proof, version)
}
func (s *service) Complete(ctx context.Context, id, attempt string, c CompletionEvidence, version int64) (Job, error) {
	if c.CompletedAt.IsZero() {
		c.CompletedAt = s.now().UTC()
	}
	return s.repo.Complete(ctx, id, attempt, c, version)
}
func (s *service) Fail(ctx context.Context, id, attempt, reason string, version int64) (Job, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Job{}, fmt.Errorf("%w: failure reason required", ErrInvalid)
	}
	return s.repo.Fail(ctx, id, attempt, reason, s.now().UTC().UnixNano(), version)
}
func (s *service) RequestCancel(ctx context.Context, id string, version int64) (Job, error) {
	return s.repo.RequestCancel(ctx, id, s.now().UTC().UnixNano(), version)
}
func (s *service) ConfirmCancelled(ctx context.Context, id, attempt string, p QuiescenceProof, version int64) (Job, error) {
	if p.ObservedAt.IsZero() {
		p.ObservedAt = s.now().UTC()
	}
	if !p.NoOwnedExecution || strings.TrimSpace(p.ScopeID) == "" || strings.TrimSpace(p.Reason) == "" || strings.TrimSpace(p.OwnerID) == "" || strings.TrimSpace(p.JobID) == "" || strings.TrimSpace(p.AttemptID) == "" || strings.TrimSpace(p.InputFingerprint) == "" || strings.TrimSpace(p.ProofArtifactID) == "" || validDigest(p.ProofSHA256) != nil {
		return Job{}, fmt.Errorf("%w: cancellation requires an attempt-bound owner proof artifact", ErrInvalid)
	}
	return s.repo.ConfirmCancelled(ctx, id, attempt, p, version)
}
func (s *service) Settle(ctx context.Context, id string, state SettlementState, version int64) (Job, error) {
	if state != SettlementSettled && state != SettlementReconciliation {
		return Job{}, fmt.Errorf("%w: settlement must be settled or reconciliation_required", ErrInvalid)
	}
	return s.repo.Settle(ctx, id, state, s.now().UTC().UnixNano(), version)
}
func (s *service) Recoveries(ctx context.Context, lease time.Duration) ([]Recovery, error) {
	if lease <= 0 {
		return nil, fmt.Errorf("%w: recovery lease must be positive", ErrInvalid)
	}
	jobs, err := s.repo.ListNonterminal(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	out := make([]Recovery, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, classify(j, now, lease))
	}
	return out, nil
}
func classify(j Job, now time.Time, lease time.Duration) Recovery {
	r := Recovery{JobID: j.ID}
	switch j.State {
	case Queued:
		r.Class = RecoveryQueued
	case CancelRequested:
		r.Class = RecoveryCancel
	case Checkpointed:
		r.Class = RecoveryReconcile
		if a, ok := currentAttempt(j); ok {
			r.AttemptID = a.ID
			if a.Checkpoint != nil && a.CheckpointQuiescence != nil && validateCheckpoint(j, a, *a.Checkpoint) == nil && validateQuiescence(j, a, *a.CheckpointQuiescence) == nil && !a.CheckpointQuiescence.ObservedAt.Before(a.Checkpoint.CreatedAt) {
				r.Class = RecoveryResume
				r.Checkpoint = a.Checkpoint
			}
		}
	case Running:
		r.Class = RecoveryReconcile
		if a, ok := currentAttempt(j); ok {
			r.AttemptID = a.ID
			if a.HeartbeatAt != nil && now.Sub(*a.HeartbeatAt) <= lease {
				r.Class = RecoveryLive
			}

		}
	default:
		r.Class = RecoveryTerminal
	}
	return r
}
func executionScopeExists(j Job, scope string) bool {
	for _, a := range j.Attempts {
		if a.Execution != nil && a.Execution.ScopeID == scope {
			return true
		}
	}
	return false
}
func attemptIDExists(j Job, id string) bool {
	for _, a := range j.Attempts {
		if a.ID == id {
			return true
		}
	}
	return false
}
func currentAttempt(j Job) (Attempt, bool) {
	if len(j.Attempts) == 0 {
		return Attempt{}, false
	}
	a := j.Attempts[len(j.Attempts)-1]
	return a, true
}
