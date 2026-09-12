package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CapacityVerdict is the stable, provider-neutral result of consulting the
// platform capacity broker about a local route candidate. It is recorded on
// route candidates and route evidence and surfaced to operators.
type CapacityVerdict string

const (
	// CapacityNotEvaluated means capacity was not consulted (no adapter, remote
	// candidate, or no declared footprint requirement).
	CapacityNotEvaluated CapacityVerdict = ""
	CapacityFit          CapacityVerdict = "fit"
	CapacityInsufficient CapacityVerdict = "insufficient_capacity"
	// CapacityUnknown means the broker could not be consulted (unavailable/errored)
	// or no footprint was declared. It is advisory and never blocks routing.
	CapacityUnknown CapacityVerdict = "unknown_capacity"
	// CapacityReclaimRequired means the broker would admit the route but only by
	// reclaiming capacity from other holders, and enforcement is on.
	CapacityReclaimRequired CapacityVerdict = "reclaim_required"
	// CapacityAdvisoryReclaimUnavailable means admission needs reclaim but
	// enforcement is advisory, so the reclaim will not actually happen — the local
	// route is treated as unavailable rather than risking an OOM.
	CapacityAdvisoryReclaimUnavailable CapacityVerdict = "advisory_reclaim_unavailable"
)

// blocksLocal reports whether a verdict should keep a local candidate out of the
// eligible set. Unknown/fit/reclaim_required are admissible; insufficient and
// advisory-reclaim-unavailable are not.
func (v CapacityVerdict) blocksLocal() bool {
	return v == CapacityInsufficient || v == CapacityAdvisoryReclaimUnavailable
}

// CapacityRequest is the footprint a local route wants to reserve.
type CapacityRequest struct {
	OwnerID       string
	RequiredBytes int64
	AllowReclaim  bool
}

// CapacityEvaluation is the broker's verdict plus the acquired claim (if any).
type CapacityEvaluation struct {
	Verdict         CapacityVerdict
	RequiredBytes   int64
	GrantedBytes    int64
	ReclaimRequired bool
	ClaimID         string
	Warnings        []string
}

// CapacityAdapter consults the platform capacity broker for local-route
// eligibility. It is a seam: production wires CLICapacityAdapter (the sanctioned
// `vrooli capacity` CLI contract), tests wire a fake. A nil adapter disables
// capacity gating entirely — local routes proceed exactly as before.
type CapacityAdapter interface {
	// Claim asks whether a local route of req.RequiredBytes fits, acquiring an
	// op-scoped claim. Any returned ClaimID must be released. An operational error
	// (broker unavailable) is returned so the caller can degrade to unknown.
	Claim(ctx context.Context, req CapacityRequest) (CapacityEvaluation, error)
	// Release frees a prior claim; a no-op on the empty claim id.
	Release(ctx context.Context, claimID string)
}

const capacityClaimTTL = "5m"

// CLICapacityAdapter speaks the `vrooli capacity …` CLI so AI Gateway never
// couples to the broker's storage internals. It never probes GPU/RAM directly.
type CLICapacityAdapter struct {
	GPUIndex int
	// Exec runs `vrooli capacity …` and returns stdout. Injectable for tests;
	// nil resolves the real vrooli binary on PATH.
	Exec func(ctx context.Context, args ...string) ([]byte, error)
}

func (a *CLICapacityAdapter) run(ctx context.Context, args ...string) ([]byte, error) {
	if a.Exec != nil {
		return a.Exec(ctx, args...)
	}
	return exec.CommandContext(ctx, "vrooli", args...).Output()
}

type capacityClaimResponse struct {
	Verdict struct {
		Kind         string   `json:"kind"`
		Step         string   `json:"step"`
		Warnings     []string `json:"warnings,omitempty"`
		ReclaimBytes int64    `json:"reclaim_bytes,omitempty"`
		GrantedBytes int64    `json:"granted_bytes,omitempty"`
	} `json:"verdict"`
	Claim struct {
		ClaimID string `json:"claim_id"`
	} `json:"claim"`
	Enforce string `json:"enforce"`
}

func (a *CLICapacityAdapter) Claim(ctx context.Context, req CapacityRequest) (CapacityEvaluation, error) {
	priority := "service"
	if !req.AllowReclaim {
		priority = "batch"
	}
	out, err := a.run(ctx,
		"capacity", "claim",
		"--owner-kind", "op",
		"--owner-id", req.OwnerID,
		"--resource-kind", "vram",
		"--gpu-index", strconv.Itoa(a.GPUIndex),
		"--preferred", strconv.FormatInt(req.RequiredBytes, 10),
		"--floor", "0",
		"--priority", priority,
		"--ttl", capacityClaimTTL,
		"--json",
	)
	if err != nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: req.RequiredBytes}, fmt.Errorf("capacity claim: %w", err)
	}
	var resp capacityClaimResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: req.RequiredBytes}, fmt.Errorf("capacity claim: decode verdict: %w", err)
	}
	eval := CapacityEvaluation{
		RequiredBytes:   req.RequiredBytes,
		GrantedBytes:    resp.Verdict.GrantedBytes,
		ReclaimRequired: resp.Verdict.ReclaimBytes > 0,
		ClaimID:         resp.Claim.ClaimID,
		Warnings:        resp.Verdict.Warnings,
	}
	eval.Verdict = classifyCapacity(resp)
	return eval, nil
}

func classifyCapacity(resp capacityClaimResponse) CapacityVerdict {
	if resp.Verdict.Kind != "grant" || resp.Verdict.Step == "cpu" {
		return CapacityInsufficient
	}
	if resp.Verdict.ReclaimBytes > 0 {
		if strings.EqualFold(resp.Enforce, "on") {
			return CapacityReclaimRequired
		}
		return CapacityAdvisoryReclaimUnavailable
	}
	return CapacityFit
}

func (a *CLICapacityAdapter) Release(ctx context.Context, claimID string) {
	if strings.TrimSpace(claimID) == "" {
		return
	}
	_, _ = a.run(ctx, "capacity", "release", "--claim-id", claimID, "--json")
}

// capacityRequirementBytes resolves the local footprint a request declares.
// Source of truth is explicit request metadata (a caller-declared constraint) —
// never inferred from model names. Accepted keys, first match wins. A zero
// result means "no declared footprint" → capacity is reported as unknown, not
// enforced.
func capacityRequirementBytes(metadata map[string]string) int64 {
	for _, key := range []string{"required_vram_bytes", "local_vram_bytes", "required_bytes"} {
		if raw, ok := metadata[key]; ok {
			if n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
