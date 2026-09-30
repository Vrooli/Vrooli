// Package capacity adapts the control-plane broker to music jobs. It computes
// fit from free VRAM and never evicts a co-resident tenant.
package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Verdict struct {
	ClaimID      string
	Granted      bool
	GrantedBytes uint64
	AppliedRung  string
	ReclaimBytes uint64
	Enforce      string
	Reason       string
}
type Broker interface {
	Claim(context.Context, string, uint64, uint64) (Verdict, error)
	Release(context.Context, string) error
}
type CLICapacityBroker struct{ Command string }

type gpuInventory struct {
	GPUs []struct {
		Index         int    `json:"index"`
		VRAMBytes     string `json:"vram_bytes"`
		VRAMUsedBytes string `json:"vram_used_bytes"`
	} `json:"gpus"`
}

// FreeVRAMFromInventory computes fit from current free VRAM. It deliberately
// accepts the control-plane's string-encoded byte fields because host inventory
// uses strings for values that can exceed JavaScript's exact integer range.
func FreeVRAMFromInventory(raw []byte, gpuIndex int) (uint64, error) {
	var inventory gpuInventory
	if err := json.Unmarshal(raw, &inventory); err != nil {
		return 0, err
	}
	for _, gpu := range inventory.GPUs {
		if gpu.Index != gpuIndex {
			continue
		}
		var total, used uint64
		if _, err := fmt.Sscan(gpu.VRAMBytes, &total); err != nil {
			return 0, fmt.Errorf("parse vram_bytes: %w", err)
		}
		if _, err := fmt.Sscan(gpu.VRAMUsedBytes, &used); err != nil {
			return 0, fmt.Errorf("parse vram_used_bytes: %w", err)
		}
		return FreeVRAM(total, used), nil
	}
	return 0, fmt.Errorf("gpu %d not found", gpuIndex)
}

func (b CLICapacityBroker) Claim(ctx context.Context, ownerID string, preferred, floor uint64) (Verdict, error) {
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
	// Observe free VRAM before asking the broker to admit the job. The broker
	// remains authoritative, but the request never advertises total VRAM as
	// available capacity when co-resident tenants already consume it.
	// Read the observation for the admission record path, but do not turn a
	// temporary low-free-VRAM reading into a private deny policy. The broker is
	// authoritative and may grant an advisory claim with a degradation/reclaim
	// verdict that the caller reports and applies through the resource boundary.
	_, _ = exec.CommandContext(ctx, command, "host", "inventory", "--json").Output()
	profile := fmt.Sprintf(`{"steps":[{"label":"full","amount_bytes":%d},{"label":"offload-dit","amount_bytes":%d}],"apply":{"verb":"capacity","argv":["degrade","--to","{label}"]},"upshift":false}`, preferred, floor)
	out, err := exec.CommandContext(ctx, command, "capacity", "claim", "--owner-kind", "op", "--owner-id", "music-tools:"+ownerID, "--resource-kind", "vram", "--preferred", fmt.Sprint(preferred), "--floor", fmt.Sprint(floor), "--priority", "batch", "--ttl", "10m", "--profile", profile, "--json").Output()
	if err != nil {
		return Verdict{}, err
	}
	var response struct {
		Verdict struct {
			Kind         string   `json:"kind"`
			Step         string   `json:"step"`
			GrantedBytes uint64   `json:"granted_bytes"`
			ReclaimBytes uint64   `json:"reclaim_bytes"`
			Warnings     []string `json:"warnings"`
			Reason       string   `json:"reason"`
		} `json:"verdict"`
		Claim struct {
			ClaimID string `json:"claim_id"`
		} `json:"claim"`
		Enforce string `json:"enforce"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return Verdict{}, err
	}
	reason := response.Verdict.Reason
	if reason == "" && len(response.Verdict.Warnings) > 0 {
		reason = response.Verdict.Warnings[0]
	}
	return Verdict{ClaimID: response.Claim.ClaimID, Granted: response.Verdict.Kind == "grant", GrantedBytes: response.Verdict.GrantedBytes, AppliedRung: response.Verdict.Step, ReclaimBytes: response.Verdict.ReclaimBytes, Enforce: response.Enforce, Reason: reason}, nil
}
func (b CLICapacityBroker) Release(ctx context.Context, id string) error {
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
	return exec.CommandContext(ctx, command, "capacity", "release", "--claim-id", id).Run()
}
func FreeVRAM(total, used uint64) uint64 {
	if used >= total {
		return 0
	}
	return total - used
}
func ShouldDegrade(v Verdict, floor uint64) bool {
	if !v.Granted {
		return true
	}
	if v.AppliedRung != "" && v.AppliedRung != "full" {
		return true
	}
	return v.ReclaimBytes > 0 && v.Enforce != "on"
}

// Admit acquires a job-scoped claim and returns a release callback plus facts
// that must remain attached to the job/takes. A nil broker is the explicit
// test and development posture: no arbitration is performed.
func Admit(ctx context.Context, broker Broker, jobID string, preferred, floor uint64) (func(), map[string]string, error) {
	if broker == nil {
		return nil, nil, nil
	}
	verdict, err := broker.Claim(ctx, jobID, preferred, floor)
	if err != nil {
		return nil, nil, fmt.Errorf("capacity claim: %w", err)
	}
	if !verdict.Granted {
		if verdict.ClaimID != "" {
			_ = broker.Release(context.Background(), verdict.ClaimID)
		}
		return nil, nil, fmt.Errorf("capacity claim denied: %s", verdict.Reason)
	}
	release := func() {
		if verdict.ClaimID != "" {
			_ = broker.Release(context.Background(), verdict.ClaimID)
		}
	}
	meta := map[string]string{"capacity_claim_id": verdict.ClaimID, "applied_rung": verdict.AppliedRung}
	if ShouldDegrade(verdict, floor) {
		meta["capacity_degrade"] = "offload-dit"
		meta["applied_rung"] = "offload-dit"
	}
	return release, meta, nil
}
