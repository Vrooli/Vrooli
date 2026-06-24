package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// sdGPUVRAMEstimateBytes is the VRAM an fp16 GPU image generation is expected
// to need for its op-scoped claim's preferred amount; the floor is 0 (the CPU
// step). Tunable lever (plan §2 control-surface-tunable-levers-design): a
// coarse, honest over-estimate that biases toward degrading to CPU under
// contention rather than OOMing the GPU. Per-model VRAM metadata can refine
// this later without changing the seam.
const sdGPUVRAMEstimateBytes int64 = 6 << 30 // 6 GiB

// capacityClaimTTL bounds an op claim's liveness without a heartbeat. A
// generation is bounded (seconds–minutes) and run() releases the claim in a
// defer; the TTL is the crash-safety net so an abandoned claim is swept by the
// broker rather than leaking a VRAM reservation forever.
const capacityClaimTTL = "10m"

// CapacityLease is the broker's verdict on an op-scoped GPU claim. The zero
// value means "proceed on GPU unchanged" — the broker is advisory and never
// blocks a generation.
type CapacityLease struct {
	ClaimID      string
	DegradeToCPU bool
	Warnings     []string
}

// CapacityBroker arbitrates op-scoped GPU VRAM for a single generation against
// the host capacity ledger (plan §7 Phase 7, §8.4 CLI contract). It is optional
// in Deps: a nil broker means no arbitration and the engine runs exactly as
// before. Declared behind a seam so tests fake it without shelling out.
type CapacityBroker interface {
	// Claim reserves preferredBytes of GPU VRAM for ownerID with an fp16-gpu→cpu
	// degrade profile, returning whether the broker degraded the job to CPU.
	Claim(ctx context.Context, ownerID string, preferredBytes int64) (CapacityLease, error)
	// Release frees a prior claim; a no-op on the empty claim id.
	Release(ctx context.Context, claimID string)
}

// CLICapacityBroker is the production CapacityBroker. It speaks the sanctioned
// `vrooli capacity …` CLI contract (plan §8.4) so image-tools never couples to
// the broker's SQLite storage internals.
type CLICapacityBroker struct {
	// GPUIndex is the GPU the op contends for (default 0).
	GPUIndex int
	// Exec runs `vrooli capacity …` and returns stdout. Injectable for tests;
	// nil resolves the real vrooli binary on PATH.
	Exec func(ctx context.Context, args ...string) ([]byte, error)
}

func (b *CLICapacityBroker) run(ctx context.Context, args ...string) ([]byte, error) {
	if b.Exec != nil {
		return b.Exec(ctx, args...)
	}
	return exec.CommandContext(ctx, "vrooli", args...).Output()
}

// claimResponse mirrors the `vrooli capacity claim --json` envelope (the
// fields this adopter needs).
type claimResponse struct {
	Verdict struct {
		Kind     string   `json:"kind"`
		Step     string   `json:"step"`
		Warnings []string `json:"warnings,omitempty"`
	} `json:"verdict"`
	Claim struct {
		ClaimID string `json:"claim_id"`
	} `json:"claim"`
}

// Claim reserves an op-scoped fp16-gpu→cpu VRAM claim for ownerID. A non-grant
// verdict (or a granted "cpu" step) means the GPU can't host this op now → the
// caller should run on CPU. Any returned claim id must be released.
func (b *CLICapacityBroker) Claim(ctx context.Context, ownerID string, preferredBytes int64) (CapacityLease, error) {
	profile := fmt.Sprintf(
		`{"steps":[{"label":"fp16-gpu","amount_bytes":%d},{"label":"cpu","amount_bytes":0}],"apply":{"verb":"noop","argv":[]},"upshift":false}`,
		preferredBytes,
	)
	out, err := b.run(ctx,
		"capacity", "claim",
		"--owner-kind", "op",
		"--owner-id", ownerID,
		"--resource-kind", "vram",
		"--gpu-index", strconv.Itoa(b.GPUIndex),
		"--preferred", strconv.FormatInt(preferredBytes, 10),
		"--floor", "0",
		"--priority", "batch",
		"--ttl", capacityClaimTTL,
		"--profile", profile,
		"--json",
	)
	if err != nil {
		return CapacityLease{}, fmt.Errorf("capacity claim: %w", err)
	}
	var resp claimResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return CapacityLease{}, fmt.Errorf("capacity claim: decode verdict: %w", err)
	}
	lease := CapacityLease{ClaimID: resp.Claim.ClaimID, Warnings: resp.Verdict.Warnings}
	// GPU only when the broker granted the top (fp16-gpu) step. Any degrade /
	// queue / deny — or a granted "cpu" step — means the GPU can't host this op
	// right now, so the caller runs on CPU.
	if resp.Verdict.Kind != "grant" || resp.Verdict.Step == "cpu" {
		lease.DegradeToCPU = true
	}
	return lease, nil
}

// Release frees a prior op claim. Best-effort: a generation that completed has
// already produced its output; a failed release only delays the broker's TTL
// sweep, so errors are swallowed (advisory contract).
func (b *CLICapacityBroker) Release(ctx context.Context, claimID string) {
	if claimID == "" {
		return
	}
	_, _ = b.run(ctx, "capacity", "release", "--claim-id", claimID, "--json")
}
