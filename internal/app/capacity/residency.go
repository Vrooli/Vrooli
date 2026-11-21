package capacity

import (
	"context"
	"fmt"
	"strings"
	"time"

	engine "github.com/vrooli/vrooli/internal/capacity"
)

// ResidencyRequest carries an owner-authored transition and the exact claim
// generation it was admitted against. Revision is required for completion.
type ResidencyRequest struct {
	OwnerID                string                     `json:"owner_id"`
	ClaimID                string                     `json:"claim_id"`
	ClaimGeneration        int64                      `json:"claim_generation"`
	CurrentClaimGeneration int64                      `json:"current_claim_generation,omitempty"`
	Revision               int64                      `json:"revision,omitempty"`
	Transition             engine.ResidencyTransition `json:"transition"`
}

// ResidencyBegin persists the immutable start evidence for one resource
// transition. It does not perform or authorize a physical lifecycle action.
func (s Service) ResidencyBegin(ctx context.Context, req ResidencyRequest) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	claim, err := store.GetClaim(ctx, req.ClaimID)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	if claim.OwnerKind != "resource" || strings.TrimSpace(req.OwnerID) == "" || claim.OwnerID != req.OwnerID {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("residency owner does not match the exact resource claim")
	}
	return repo.BeginResidencyTransition(ctx, req.ClaimID, req.ClaimGeneration, req.Transition)
}

// ResidencyComplete records the resource owner's observed result by revision
// CAS. Unknown and partial effects are preserved by the repository.
func (s Service) ResidencyComplete(ctx context.Context, req ResidencyRequest) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	claim, err := store.GetClaim(ctx, req.ClaimID)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	if claim.OwnerKind != "resource" || strings.TrimSpace(req.OwnerID) == "" || claim.OwnerID != req.OwnerID {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("residency owner does not match the exact resource claim")
	}
	return repo.CompleteResidencyTransition(ctx, req.ClaimID, req.ClaimGeneration, req.Revision, req.Transition)
}

// ResidencyReconcile refreshes owner observations on an already-completed
// transition after prior evidence ages out. The request must remain bound to
// the same active claim generation and immutable start facts.
func (s Service) ResidencyReconcile(ctx context.Context, req ResidencyRequest) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	claim, err := store.GetClaim(ctx, req.ClaimID)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	if claim.OwnerKind != "resource" || strings.TrimSpace(req.OwnerID) == "" || claim.OwnerID != req.OwnerID {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("residency owner does not match the exact resource claim")
	}
	return repo.ReconcileResidencyTransition(ctx, req.ClaimID, req.ClaimGeneration, req.Revision, req.Transition)
}

// ResidencyRecover reconciles a started transition after a process restart.
// It writes an append-only recovery event, and unknown placement remains open.
func (s Service) ResidencyRecover(ctx context.Context, req ResidencyRequest) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	return repo.RecoverResidencyTransition(ctx, req.OwnerID, req.ClaimID, req.ClaimGeneration, req.CurrentClaimGeneration, req.Revision, req.Transition)
}

// ResidencyHistory exposes immutable transition event snapshots for audit and
// restart diagnosis.
func (s Service) ResidencyHistory(ctx context.Context, transitionID string) ([]engine.ResidencyTransitionEvent, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return nil, fmt.Errorf("capacity store does not support residency transitions")
	}
	return repo.ListResidencyTransitionEvents(ctx, transitionID)
}

type ResidencyReleaseRequest struct {
	OwnerID         string               `json:"owner_id"`
	ClaimID         string               `json:"claim_id"`
	ClaimGeneration int64                `json:"claim_generation"`
	TransitionID    string               `json:"transition_id"`
	Revision        int64                `json:"revision"`
	Model           engine.ModelIdentity `json:"model"`
}

// ResidencyGet replays exact durable transition state without inferring
// whether the intended physical effect occurred.
func (s Service) ResidencyGet(ctx context.Context, transitionID string) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	return repo.GetResidencyTransition(ctx, transitionID)
}

type ResidencyObservation struct {
	Claim      engine.CapacityClaim
	Bytes      int64
	ObservedAt time.Time
}

// ResidencyObserveClaim takes an explicit fresh host snapshot and attributes
// it against the current full claim cohort. Sensing, attribution, incomplete
// process provenance, or concurrent claim changes fail closed. It does not
// update claim state; the owner persists the returned evidence with its
// transition completion/recovery event.
func (s Service) ResidencyObserveClaim(ctx context.Context, claimID string) (ResidencyObservation, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return ResidencyObservation{}, err
	}
	defer store.Close()
	target, err := store.GetClaim(ctx, claimID)
	if err != nil {
		return ResidencyObservation{}, err
	}
	cohort, err := store.ListClaims(ctx, engine.ClaimFilter{Statuses: engine.ActiveClaimStatuses()})
	if err != nil {
		return ResidencyObservation{}, err
	}
	snapshot, err := s.source().Snapshot(ctx)
	if err != nil {
		return ResidencyObservation{}, fmt.Errorf("collect fresh GPU process inventory: %w", err)
	}
	currentCohort, err := store.ListClaims(ctx, engine.ClaimFilter{Statuses: engine.ActiveClaimStatuses()})
	if err != nil {
		return ResidencyObservation{}, err
	}
	if !sameResidencyCohort(cohort, currentCohort) {
		return ResidencyObservation{}, fmt.Errorf("active capacity claim cohort changed during physical residency sampling")
	}
	now := s.now()
	sample, err := engine.MeasureClaimResidency(ctx, cohort, target, snapshot, s.attributor(), now)
	if err != nil {
		return ResidencyObservation{}, err
	}
	current, err := store.GetClaim(ctx, claimID)
	if err != nil {
		return ResidencyObservation{}, err
	}
	if current.Generation != target.Generation || current.Status != target.Status || current.OwnerKind != target.OwnerKind || current.OwnerID != target.OwnerID || current.ResourceKind != target.ResourceKind || !sameGPUIndex(current.GPUIndex, target.GPUIndex) || !current.UpdatedAt.Equal(target.UpdatedAt) || !sameOptionalTime(current.LastHeartbeatAt, target.LastHeartbeatAt) || !sameOptionalTime(current.LastActiveAt, target.LastActiveAt) {
		return ResidencyObservation{}, fmt.Errorf("claim changed during physical residency sampling")
	}
	return ResidencyObservation{Claim: current, Bytes: sample.Bytes, ObservedAt: sample.ObservedAt}, nil
}

func sameGPUIndex(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameResidencyCohort(left, right []engine.CapacityClaim) bool {
	if len(left) != len(right) {
		return false
	}
	byID := make(map[string]engine.CapacityClaim, len(left))
	for _, claim := range left {
		byID[claim.ClaimID] = claim
	}
	for _, current := range right {
		prior, ok := byID[current.ClaimID]
		if !ok || prior.Generation != current.Generation || prior.Status != current.Status || prior.OwnerKind != current.OwnerKind || prior.OwnerID != current.OwnerID || prior.ResourceKind != current.ResourceKind || !sameGPUIndex(prior.GPUIndex, current.GPUIndex) || !prior.UpdatedAt.Equal(current.UpdatedAt) || !sameOptionalTime(prior.LastHeartbeatAt, current.LastHeartbeatAt) || !sameOptionalTime(prior.LastActiveAt, current.LastActiveAt) {
			return false
		}
	}
	return true
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

// ResidencyGetOpenForOwner replays the single unresolved transition for an
// exact resource so its owner can reconcile after process/control restart.
func (s Service) ResidencyGetOpenForOwner(ctx context.Context, ownerID string) (engine.ResidencyTransitionRecord, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.ResidencyTransitionRecord{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.ResidencyTransitionRecord{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	return repo.GetUnresolvedResidencyTransitionForOwner(ctx, ownerID)
}

// ResidencyRelease delegates to the repository's verified-unload release gate.
func (s Service) ResidencyRelease(ctx context.Context, req ResidencyReleaseRequest) (engine.CapacityClaim, error) {
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.CapacityClaim{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.ResidencyRepository)
	if !ok {
		return engine.CapacityClaim{}, fmt.Errorf("capacity store does not support residency transitions")
	}
	return repo.ReleaseClaimAfterResidency(ctx, req.OwnerID, req.ClaimID, req.ClaimGeneration, req.TransitionID, req.Revision, req.Model)
}
