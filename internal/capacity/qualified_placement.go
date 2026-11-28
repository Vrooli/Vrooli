package capacity

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// PlacementQualification is a suite-backed measurement for one model role at
// one rung. A role may only move to a rung whose quality suite and latency
// objective passed for the exact model digest and device. These receipts are
// measurements, never an authorization to reclaim an active/protected claim.
type PlacementQualification struct {
	ReceiptID       string        `json:"receipt_id,omitempty"`
	ClaimID         string        `json:"claim_id"`
	ClaimGeneration int64         `json:"claim_generation"`
	Role            string        `json:"role"`
	Model           ModelIdentity `json:"model"`
	Rung            string        `json:"rung"`
	Device          string        `json:"device"`
	Suite           string        `json:"suite"`
	SuiteDigest     string        `json:"suite_digest"`
	Cases           int           `json:"cases"`
	Passed          bool          `json:"passed"`
	P95LatencyMS    float64       `json:"p95_latency_ms"`
	MaxLatencyMS    float64       `json:"max_latency_ms"`
	ObservedVRAM    int64         `json:"observed_vram_bytes"`
	SampledAt       time.Time     `json:"sampled_at"`
	RunID           string        `json:"run_id"`
	ArtifactURI     string        `json:"artifact_uri"`
	ArtifactSHA     string        `json:"artifact_sha256"`
}

// PlacementQualificationRepository stores immutable, content-addressed suite
// receipts and returns them for planners and placement reports.
type PlacementQualificationRepository interface {
	RecordPlacementQualification(context.Context, PlacementQualification) (PlacementQualification, error)
	ListPlacementQualifications(context.Context, PlacementQualificationFilter) ([]PlacementQualification, error)
}

type PlacementQualificationFilter struct {
	ClaimID         string
	ClaimGeneration int64
	ClaimIDs        []string
	Role            string
	ModelDigest     string
	Rung            string
	Device          string
	Since           *time.Time
}

// QualifiedPlacementRequest describes an already-admitted higher-priority
// workload's capacity deficit. Candidate moves are advisory plans; execution
// must still use the canonical owner actuator and exact claim generation.
type QualifiedPlacementRequest struct {
	RequesterPriority int
	DeficitBytes      int64
	Claims            []CapacityClaim
	Evidence          []PlacementQualification
	Policy            Policy
	Now               time.Time
	MaxEvidenceAge    time.Duration
}

// PlanQualifiedDownsize selects only idle, unprotected, lower-priority claims
// with fresh passing evidence for the exact model and target rung. The planner
// prefers the least disruptive qualified rung that covers the remaining
// deficit; when multiple roles are needed it returns a deterministic plan.
// It never preempts and never treats a missing/invalid receipt as permission.
func PlanQualifiedDownsize(req QualifiedPlacementRequest) EscalationPlan {
	plan := EscalationPlan{Deficit: req.DeficitBytes}
	if req.DeficitBytes <= 0 {
		plan.Satisfied = true
		return plan
	}
	maxAge := req.MaxEvidenceAge
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}
	eligible := make([]CapacityClaim, 0, len(req.Claims))
	for _, claim := range req.Claims {
		if claim.ResourceKind == ResourceKindVRAM && reclaimEligibleFor(claim, req.RequesterPriority, req.Policy, req.Now) {
			eligible = append(eligible, claim)
		}
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Priority != eligible[j].Priority {
			return eligible[i].Priority < eligible[j].Priority
		}
		if !IdleSince(eligible[i]).Equal(IdleSince(eligible[j])) {
			return IdleSince(eligible[i]).Before(IdleSince(eligible[j]))
		}
		return eligible[i].ClaimID < eligible[j].ClaimID
	})

	for _, claim := range eligible {
		need := req.DeficitBytes - plan.FreedBytes
		action, ok := qualifiedDegradeAction(claim, need, req.Evidence, req.Now, maxAge)
		if !ok {
			continue
		}
		plan.Actions = append(plan.Actions, action)
		plan.FreedBytes += action.FreesBytes
		if plan.FreedBytes >= req.DeficitBytes {
			break
		}
	}
	plan.Satisfied = plan.FreedBytes >= req.DeficitBytes
	return plan
}

// PlanQualifiedDownsizeFromStore makes persisted, verified receipt history the
// planner input. It is the preferred seam for admission owners once their real
// quality suite artifacts are registered in capacity.
func PlanQualifiedDownsizeFromStore(ctx context.Context, req QualifiedPlacementRequest, repo PlacementQualificationRepository) (EscalationPlan, error) {
	claimIDs := make([]string, 0, len(req.Claims))
	for _, claim := range req.Claims {
		if claim.ResourceKind == ResourceKindVRAM && !claim.Protected && strings.TrimSpace(claim.ClaimID) != "" {
			claimIDs = append(claimIDs, claim.ClaimID)
		}
	}
	if len(claimIDs) == 0 {
		return PlanQualifiedDownsize(req), nil
	}
	receipts, err := repo.ListPlacementQualifications(ctx, PlacementQualificationFilter{ClaimIDs: claimIDs})
	if err != nil {
		return EscalationPlan{}, fmt.Errorf("load placement qualification receipts: %w", err)
	}
	req.Evidence = receipts
	return PlanQualifiedDownsize(req), nil
}

// RunQualifiedDownsize sends the suite-qualified plan through the established
// generation-checked adopter actuator. Advisory/off modes return the plan only;
// they do not resize a resource. This is the execution seam for callers that
// have per-role qualification receipts available.
func RunQualifiedDownsize(ctx context.Context, req QualifiedPlacementRequest, store ClaimRepository, exec ApplyExecutor, enforce string) (EscalationPlan, ActuationResult, error) {
	plan := PlanQualifiedDownsize(req)
	if enforce != EnforceOn || len(plan.Actions) == 0 {
		return plan, ActuationResult{}, nil
	}
	result, err := Actuate(ctx, plan, store, exec, req.Policy, req.Now)
	return plan, result, err
}

func RunQualifiedDownsizeFromStore(ctx context.Context, req QualifiedPlacementRequest, store ClaimRepository, repo PlacementQualificationRepository, exec ApplyExecutor, enforce string) (EscalationPlan, ActuationResult, error) {
	plan, err := PlanQualifiedDownsizeFromStore(ctx, req, repo)
	if err != nil || enforce != EnforceOn || len(plan.Actions) == 0 {
		return plan, ActuationResult{}, err
	}
	result, err := Actuate(ctx, plan, store, exec, req.Policy, req.Now)
	return plan, result, err
}

func qualifiedDegradeAction(claim CapacityClaim, need int64, receipts []PlacementQualification, now time.Time, maxAge time.Duration) (EscalationAction, bool) {
	if claim.DegradeProfile == nil || len(claim.DegradeProfile.Steps) == 0 || strings.TrimSpace(claim.DegradeProfile.Role) == "" || !claim.DegradeProfile.Model.Complete() {
		return EscalationAction{}, false
	}
	type candidate struct {
		step  DegradeStep
		proof PlacementQualification
	}
	var candidates []candidate
	for _, step := range claim.DegradeProfile.Steps {
		if step.AmountBytes < claim.FloorBytes || step.AmountBytes >= claim.AmountBytes || step.AmountBytes < 0 {
			continue
		}
		for _, proof := range receipts {
			if proof.ClaimID != claim.ClaimID || proof.ClaimGeneration != claim.Generation || proof.Role != claim.DegradeProfile.Role || proof.Rung != step.Label || proof.Model != claim.DegradeProfile.Model || !qualificationValid(proof, step, now, maxAge) {
				continue
			}
			candidates = append(candidates, candidate{step: step, proof: proof})
		}
	}
	if len(candidates) == 0 {
		return EscalationAction{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].step.AmountBytes > candidates[j].step.AmountBytes
	})
	chosen := candidates[len(candidates)-1]
	for _, c := range candidates {
		if claim.AmountBytes-c.step.AmountBytes >= need {
			chosen = c
			break
		}
	}
	frees := claim.AmountBytes - chosen.step.AmountBytes
	return EscalationAction{
		ClaimID: claim.ClaimID, Generation: claim.Generation, OwnerID: claim.OwnerID,
		Action: ActionRequestDegrade, ToStep: chosen.step.Label, FreesBytes: frees,
		Reason: fmt.Sprintf("role %q passed %q at %q (p95 %.1f ms, %s VRAM); exact model-qualified rung frees %s", chosen.proof.Role, chosen.proof.Suite, chosen.step.Label, chosen.proof.P95LatencyMS, humanBytes(chosen.proof.ObservedVRAM), humanBytes(frees)),
	}, true
}

func qualificationValid(proof PlacementQualification, step DegradeStep, now time.Time, maxAge time.Duration) bool {
	if strings.TrimSpace(proof.Role) == "" || strings.TrimSpace(proof.Suite) == "" || strings.TrimSpace(proof.SuiteDigest) == "" || proof.Cases < 20 || !proof.Passed {
		return false
	}
	if proof.SampledAt.IsZero() || proof.SampledAt.After(now) || now.Sub(proof.SampledAt) > maxAge {
		return false
	}
	if proof.P95LatencyMS <= 0 || proof.MaxLatencyMS <= 0 || proof.P95LatencyMS > proof.MaxLatencyMS || proof.ObservedVRAM < 0 {
		return false
	}
	// For a zero-byte CPU/unloaded rung, the device label still distinguishes
	// CPU residency from absent weights. CPU placement must prove no VRAM use.
	if step.AmountBytes == 0 {
		if proof.Device != ResidencyBackendCPU && proof.Device != ResidencyBackendUnloaded {
			return false
		}
		return proof.ObservedVRAM == 0
	}
	if !validResidencyBackend(proof.Device) || proof.Device == ResidencyBackendCPU || proof.Device == ResidencyBackendUnloaded {
		return false
	}
	// The claim rung must cover the measured live footprint with 20% margin.
	return float64(step.AmountBytes) >= float64(proof.ObservedVRAM)*1.2
}

// PlanQualifiedUpshiftAll keeps the existing shared headroom accounting and
// only returns an upshift if the exact target rung has a fresh passing suite
// receipt for the claim's model. The underlying planner remains responsible
// for hysteresis, idle/protection gates, and per-GPU anti-overcommit.
func PlanQualifiedUpshiftAll(claims []CapacityClaim, snapshot hostinventory.Snapshot, policy Policy, evidence []PlacementQualification, now time.Time, maxAge time.Duration) EscalationPlan {
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}
	plan := PlanUpshiftAll(claims, snapshot, policy, now)
	qualified := make([]EscalationAction, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		var claim *CapacityClaim
		for i := range claims {
			if claims[i].ClaimID == action.ClaimID && claims[i].Generation == action.Generation {
				claim = &claims[i]
				break
			}
		}
		if claim == nil || claim.DegradeProfile == nil || !claim.DegradeProfile.Model.Complete() {
			continue
		}
		step, ok := findStep(claim.DegradeProfile, action.ToStep)
		if !ok {
			continue
		}
		for _, proof := range evidence {
			if proof.ClaimID == claim.ClaimID && proof.ClaimGeneration == claim.Generation && proof.Role == claim.DegradeProfile.Role && proof.Model == claim.DegradeProfile.Model && proof.Rung == step.Label && qualificationValid(proof, step, now, maxAge) {
				qualified = append(qualified, action)
				break
			}
		}
	}
	plan.Actions = qualified
	return plan
}

func findStep(profile *DegradeProfile, label string) (DegradeStep, bool) {
	for _, step := range profile.Steps {
		if step.Label == label {
			return step, true
		}
	}
	return DegradeStep{}, false
}
