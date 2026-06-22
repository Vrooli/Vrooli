// Package sttcapacity reports STT transcription activity to the platform
// capacity broker (plan §7 Phase 7 audio-tools adopter).
//
// audio-tools holds NO VRAM capacity claim of its own — the backing STT
// resource (whisper / kyutai-stt) declares and holds the claim, so audio-tools
// never double-reserves. What audio-tools uniquely knows is *when a
// transcription is in flight*, which the third-party model-server containers
// cannot report for themselves. This package bridges that: at the start of a
// streaming session on a local-resource engine it marks the resource's claim
// activity=active (which protects it from idle reclaim per plan §8.3), and at
// session end marks it idle again.
//
// It is strictly advisory and best-effort: a missing vrooli binary, a resource
// with no live claim, or any CLI error is swallowed. Capacity reporting NEVER
// affects transcription.
package sttcapacity

import (
	"context"
	"encoding/json"
	"os/exec"
)

// allowedResources is the whitelist of backing STT resources whose capacity
// claim this reporter will ever touch. Defence-in-depth: the engine registry
// already constrains engine.Resource, this is a second guard.
var allowedResources = map[string]struct{}{
	"whisper":    {},
	"kyutai-stt": {},
}

// Reporter marks a backing STT resource's capacity claim active during a
// transcription session and idle afterwards. The zero/nil reporter is a no-op.
type Reporter interface {
	// Active marks the resource's active capacity claim as activity=active and
	// returns the claim id to later pass to Idle. Returns "" when there is no
	// applicable claim or anything failed (best-effort).
	Active(ctx context.Context, resource string) string
	// Idle marks the given claim activity=idle. No-op on an empty id.
	Idle(ctx context.Context, claimID string)
}

// CLIReporter is the production Reporter. It speaks the sanctioned
// `vrooli capacity …` CLI (plan §8.4) so audio-tools never couples to the
// broker's storage internals.
type CLIReporter struct {
	// vrooliBin is the absolute path to the vrooli binary, or "" when absent.
	vrooliBin string
	// exec runs vrooli with the given args and returns stdout. Injectable for
	// tests; nil uses vrooliBin via os/exec.
	exec func(ctx context.Context, bin string, args ...string) ([]byte, error)
}

// NewCLIReporter resolves the vrooli binary once at startup. The returned
// reporter is safe even when the binary is missing (every method no-ops).
func NewCLIReporter() *CLIReporter {
	bin, err := exec.LookPath("vrooli")
	if err != nil {
		return &CLIReporter{}
	}
	return &CLIReporter{vrooliBin: bin}
}

func (r *CLIReporter) run(ctx context.Context, args ...string) ([]byte, error) {
	if r.exec != nil {
		return r.exec(ctx, r.vrooliBin, args...)
	}
	return exec.CommandContext(ctx, r.vrooliBin, args...).Output()
}

type listResponse struct {
	Claims []struct {
		ClaimID string `json:"claim_id"`
		OwnerID string `json:"owner_id"`
	} `json:"claims"`
}

// Active resolves the resource's active capacity claim and marks it
// activity=active, returning the claim id (or "" if none/failed).
func (r *CLIReporter) Active(ctx context.Context, resource string) string {
	if r.vrooliBin == "" || resource == "" {
		return ""
	}
	if _, ok := allowedResources[resource]; !ok {
		return ""
	}
	out, err := r.run(ctx, "capacity", "list", "--owner", resource, "--active", "--json")
	if err != nil {
		return ""
	}
	var resp listResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return ""
	}
	for _, c := range resp.Claims {
		if c.OwnerID == resource && c.ClaimID != "" {
			// activity auto-resolves the generation server-side, so no --generation.
			if _, err := r.run(ctx, "capacity", "activity", "--claim-id", c.ClaimID, "--state", "active", "--json"); err != nil {
				return ""
			}
			return c.ClaimID
		}
	}
	return ""
}

// Idle marks the given claim activity=idle (best-effort).
func (r *CLIReporter) Idle(ctx context.Context, claimID string) {
	if r.vrooliBin == "" || claimID == "" {
		return
	}
	_, _ = r.run(ctx, "capacity", "activity", "--claim-id", claimID, "--state", "idle", "--json")
}
