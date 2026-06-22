// Package capacity is agent-manager's adopter of the platform capacity broker
// (plan §7 Phase 7 ollama adopter, §8.4 CLI contract). agent-manager's only
// ollama usage is recommendation extraction (resource-ollama gateway generate);
// this package lets the extractor hold a short, op-scoped capacity claim around
// that generate so ollama's GPU usage shows as CLAIMED (not unclaimed) in
// `vrooli capacity reconcile` and the broker won't try to reclaim ollama's VRAM
// mid-generation. It speaks the sanctioned `vrooli capacity …` CLI so it never
// couples to the broker's storage internals. It is strictly advisory: claim
// failures never block extraction.
package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// OllamaExtractVRAMEstimateBytes is the coarse VRAM estimate used as the op
// claim's preferred amount for a recommendation-extraction generate. Tunable
// lever (plan §2): the resolved code.local model footprint varies, so this is a
// deliberate middle estimate for ledger visibility — the claim is advisory, so
// an imperfect estimate never changes whether extraction runs.
const OllamaExtractVRAMEstimateBytes int64 = 2 << 30 // 2 GiB

// claimTTL bounds the op claim's liveness without a heartbeat. A generate is
// bounded (seconds), and the extractor releases in a defer; the TTL is the
// crash-safety net so an abandoned claim is swept by the broker rather than
// leaking a reservation.
const claimTTL = "5m"

// Lease is the broker's response to an op-scoped claim. The zero value (empty
// ClaimID) means "no claim recorded" — the caller proceeds regardless (advisory).
type Lease struct {
	ClaimID  string
	Warnings []string
}

// Broker arbitrates an op-scoped ollama claim against the host capacity ledger.
// It is optional everywhere it is used: a nil Broker means no arbitration and
// the caller behaves exactly as before.
type Broker interface {
	// Claim records an op-scoped, service-priority VRAM claim for ownerID. It is
	// advisory: the returned error is operational (the caller logs and proceeds).
	Claim(ctx context.Context, ownerID string, preferredBytes int64) (Lease, error)
	// Release frees a prior claim; a no-op on the empty claim id.
	Release(ctx context.Context, claimID string)
}

// CLIBroker is the production Broker. It shells the `vrooli capacity` CLI.
type CLIBroker struct {
	// GPUIndex is the GPU ollama contends for (default 0).
	GPUIndex int
	// Exec runs `vrooli capacity …` and returns stdout. Injectable for tests;
	// nil resolves the real vrooli binary on PATH.
	Exec func(ctx context.Context, args ...string) ([]byte, error)
}

func (b *CLIBroker) run(ctx context.Context, args ...string) ([]byte, error) {
	if b.Exec != nil {
		return b.Exec(ctx, args...)
	}
	return exec.CommandContext(ctx, "vrooli", args...).Output()
}

// claimResponse mirrors the `vrooli capacity claim --json` envelope (only the
// fields this adopter needs).
type claimResponse struct {
	Verdict struct {
		Warnings []string `json:"warnings,omitempty"`
	} `json:"verdict"`
	Claim struct {
		ClaimID string `json:"claim_id"`
	} `json:"claim"`
}

// Claim records an op-scoped, service-priority VRAM claim for ownerID. The
// owner id should be "ollama:<scope>" so the colon-prefix matches ollama's
// container in reconcile's owner matching.
func (b *CLIBroker) Claim(ctx context.Context, ownerID string, preferredBytes int64) (Lease, error) {
	out, err := b.run(ctx,
		"capacity", "claim",
		"--owner-kind", "op",
		"--owner-id", ownerID,
		"--resource-kind", "vram",
		"--gpu-index", strconv.Itoa(b.GPUIndex),
		"--preferred", strconv.FormatInt(preferredBytes, 10),
		"--floor", "0",
		"--priority", "service",
		"--ttl", claimTTL,
		"--json",
	)
	if err != nil {
		return Lease{}, fmt.Errorf("capacity claim: %w", err)
	}
	var resp claimResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return Lease{}, fmt.Errorf("capacity claim: decode verdict: %w", err)
	}
	return Lease{ClaimID: resp.Claim.ClaimID, Warnings: resp.Verdict.Warnings}, nil
}

// Release frees a prior op claim. Best-effort: a failed release only delays the
// broker's TTL sweep, so errors are swallowed (advisory contract).
func (b *CLIBroker) Release(ctx context.Context, claimID string) {
	if claimID == "" {
		return
	}
	_, _ = b.run(ctx, "capacity", "release", "--claim-id", claimID, "--json")
}
