// Package capacity adapts the control-plane broker to music jobs. It computes
// fit from free VRAM and never evicts a co-resident tenant.
package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
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
	Heartbeat(context.Context, string) error
}
type CLICapacityBroker struct{ Command string }

func (b CLICapacityBroker) Claim(ctx context.Context, ownerID string, preferred, floor uint64) (Verdict, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
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
	// A reduced profile rung is an admission too; queue and deny are not.
	granted := response.Verdict.Kind == "grant" || response.Verdict.Kind == "degrade"
	return Verdict{ClaimID: response.Claim.ClaimID, Granted: granted, GrantedBytes: response.Verdict.GrantedBytes, AppliedRung: response.Verdict.Step, ReclaimBytes: response.Verdict.ReclaimBytes, Enforce: response.Enforce, Reason: reason}, nil
}

func (b CLICapacityBroker) Release(ctx context.Context, id string) error {
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
	return exec.CommandContext(ctx, command, "capacity", "release", "--claim-id", id).Run()
}

func ShouldDegrade(v Verdict) bool {
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
func Admit(ctx context.Context, cancel context.CancelCauseFunc, broker Broker, jobID string, preferred, floor uint64) (func(), map[string]string, error) {
	if broker == nil {
		return nil, nil, nil
	}
	verdict, err := broker.Claim(ctx, jobID, preferred, floor)
	if err != nil {
		return nil, nil, fmt.Errorf("capacity claim: %w", err)
	}
	if !verdict.Granted {
		if verdict.ClaimID != "" {
			releaseClaim(broker, verdict.ClaimID)
		}
		return nil, nil, fmt.Errorf("capacity claim denied: %s", verdict.Reason)
	}
	if verdict.ClaimID == "" {
		return nil, nil, fmt.Errorf("capacity admission omitted claim identity")
	}
	if verdict.AppliedRung != "full" && verdict.AppliedRung != "offload-dit" {
		releaseClaim(broker, verdict.ClaimID)
		return nil, nil, fmt.Errorf("capacity admission returned unsupported rung %q", verdict.AppliedRung)
	}
	release := maintain(ctx, cancel, broker, verdict.ClaimID, 2*time.Minute)
	meta := map[string]string{"capacity_claim_id": verdict.ClaimID, "capacity_admitted_rung": verdict.AppliedRung}
	if ShouldDegrade(verdict) {
		meta["capacity_degrade"] = "offload-dit"
		meta["capacity_admitted_rung"] = "offload-dit"
	}
	return release, meta, nil
}

func (b CLICapacityBroker) Heartbeat(ctx context.Context, id string) error {
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
	return exec.CommandContext(ctx, command, "capacity", "heartbeat", "--claim-id", id, "--ttl", "10m", "--json").Run()
}

func releaseClaim(broker Broker, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := broker.Release(ctx, id); err != nil {
		log.Printf("music capacity release %s failed: %v", id, err)
	}
}

func maintain(ctx context.Context, cancel context.CancelCauseFunc, broker Broker, id string, interval time.Duration) func() {
	if id == "" {
		return func() {}
	}
	leaseCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticker.C:
				request, end := context.WithTimeout(leaseCtx, 5*time.Second)
				err := broker.Heartbeat(request, id)
				end()
				if err != nil {
					if leaseCtx.Err() == nil {
						cancel(fmt.Errorf("capacity lease renewal failed: %w", err))
					}
					return
				}
			}
		}
	}()
	return func() { once.Do(func() { stop(); <-done; releaseClaim(broker, id) }) }
}
