package capacity

import (
	"context"
	"errors"
	"fmt"
	"time"

	engine "github.com/vrooli/vrooli/internal/capacity"
)

// PlacementCapture is the result emitted by a real owner suite after it has
// measured the requested rung. Claim identity is intentionally omitted: the
// service resolves it from the live ledger at capture time so a suite cannot
// write evidence for a stale generation or a different role/model.
type PlacementCapture struct {
	ClaimID      string  `json:"claim_id"`
	Rung         string  `json:"rung"`
	Device       string  `json:"device"`
	Suite        string  `json:"suite"`
	SuiteDigest  string  `json:"suite_digest"`
	Cases        int     `json:"cases"`
	Passed       bool    `json:"passed"`
	P95LatencyMS float64 `json:"p95_latency_ms"`
	MaxLatencyMS float64 `json:"max_latency_ms"`
	ObservedVRAM int64   `json:"observed_vram_bytes"`
	SampledAt    string  `json:"sampled_at"`
	RunID        string  `json:"run_id"`
	ArtifactURI  string  `json:"artifact_uri"`
	ArtifactSHA  string  `json:"artifact_sha256"`
}

// CapturePlacementQualification binds an owner-suite result to the current
// active claim generation and its declared role/model/rung before persisting
// it. The capacity store independently verifies the artifact digest and all
// qualification constraints in the same write path.
func (s Service) CapturePlacementQualification(ctx context.Context, capture PlacementCapture) (engine.PlacementQualification, error) {
	if capture.ClaimID == "" {
		return engine.PlacementQualification{}, errors.New("placement capture requires a claim id")
	}
	store, err := s.openStore(ctx)
	if err != nil {
		return engine.PlacementQualification{}, err
	}
	defer store.Close()
	repo, ok := store.(engine.PlacementQualificationRepository)
	if !ok {
		return engine.PlacementQualification{}, errors.New("capacity store does not support placement qualifications")
	}
	claim, err := store.GetClaim(ctx, capture.ClaimID)
	if err != nil {
		return engine.PlacementQualification{}, fmt.Errorf("load placement capture claim: %w", err)
	}
	if !engine.IsActiveClaimStatus(claim.Status) {
		return engine.PlacementQualification{}, errors.New("placement capture requires an active claim")
	}
	if claim.DegradeProfile == nil || !claim.DegradeProfile.Model.Complete() || claim.DegradeProfile.Role == "" {
		return engine.PlacementQualification{}, errors.New("placement capture claim has no complete role/model profile")
	}
	sampledAt, err := time.Parse(time.RFC3339Nano, capture.SampledAt)
	if err != nil {
		return engine.PlacementQualification{}, fmt.Errorf("parse placement capture sample time: %w", err)
	}
	qualification := engine.PlacementQualification{
		ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation,
		Role: claim.DegradeProfile.Role, Model: claim.DegradeProfile.Model,
		Rung: capture.Rung, Device: capture.Device, Suite: capture.Suite,
		SuiteDigest: capture.SuiteDigest, Cases: capture.Cases, Passed: capture.Passed,
		P95LatencyMS: capture.P95LatencyMS, MaxLatencyMS: capture.MaxLatencyMS,
		ObservedVRAM: capture.ObservedVRAM, SampledAt: sampledAt.UTC(), RunID: capture.RunID,
		ArtifactURI: capture.ArtifactURI, ArtifactSHA: capture.ArtifactSHA,
	}
	return repo.RecordPlacementQualification(ctx, qualification)
}
