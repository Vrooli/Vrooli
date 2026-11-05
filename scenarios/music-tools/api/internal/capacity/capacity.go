// Package capacity adapts the control-plane broker to music jobs. It computes
// fit from free VRAM and never evicts a co-resident tenant.
package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClaimRef struct {
	ClaimID    string
	Generation int64
}

func (r ClaimRef) valid() bool { return strings.TrimSpace(r.ClaimID) != "" && r.Generation > 0 }

type Verdict struct {
	Generation   int64
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
	Release(context.Context, ClaimRef) error
	Heartbeat(context.Context, ClaimRef) error
}
type CLICapacityBroker struct{ Command string }

func (b CLICapacityBroker) Claim(ctx context.Context, ownerID string, preferred, floor uint64) (Verdict, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	profile := fmt.Sprintf(`{"steps":[{"label":"full","amount_bytes":%d},{"label":"offload-dit","amount_bytes":%d}],"apply":{"verb":"capacity","argv":["degrade","--to","{label}"]},"upshift":false}`, preferred, floor)
	out, err := b.output(ctx, "claim", "--owner-kind", "op", "--owner-id", "music-tools:"+ownerID, "--resource-kind", "vram", "--preferred", fmt.Sprint(preferred), "--floor", fmt.Sprint(floor), "--priority", "batch", "--ttl", "10m", "--profile", profile, "--json")
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
			ClaimID    string `json:"claim_id"`
			Generation int64  `json:"generation"`
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
	return Verdict{ClaimID: response.Claim.ClaimID, Generation: response.Claim.Generation, Granted: granted, GrantedBytes: response.Verdict.GrantedBytes, AppliedRung: response.Verdict.Step, ReclaimBytes: response.Verdict.ReclaimBytes, Enforce: response.Enforce, Reason: reason}, nil
}

// Release retains the acquired reference, but the canonical owner protocol is
// deliberately ID-only: its CLI has no generation flag and its store does not
// guard release by generation. Never resolve another/current claim to release.
func (b CLICapacityBroker) Release(ctx context.Context, ref ClaimRef) error {
	if !ref.valid() {
		return fmt.Errorf("capacity release requires captured claim ID and positive generation")
	}
	_, err := b.output(ctx, "release", "--claim-id", ref.ClaimID)
	return err
}

// Keep diagnostics independently bounded; exec.Output otherwise hides stderr
// behind exit status. This does not alter the CLI's output/ownership protocol.
type boundedStderr struct {
	data      []byte
	truncated bool
}

func (w *boundedStderr) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 4096 - len(w.data)
	if n > remaining {
		w.truncated = true
		p = p[:remaining]
	}
	w.data = append(w.data, p...)
	return n, nil
}

func (b CLICapacityBroker) output(ctx context.Context, verb string, args ...string) ([]byte, error) {
	command := b.Command
	if command == "" {
		command = "vrooli"
	}
	cmd := exec.CommandContext(ctx, command, append([]string{"capacity", verb}, args...)...)
	var stderr boundedStderr
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		diagnostic := strings.TrimSpace(string(stderr.data))
		if stderr.truncated {
			diagnostic += " [stderr truncated]"
		}
		if diagnostic != "" {
			return nil, fmt.Errorf("capacity %s: %w; stderr: %s", verb, err, diagnostic)
		}
		return nil, fmt.Errorf("capacity %s: %w", verb, err)
	}
	return out, nil
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
	ref := ClaimRef{ClaimID: verdict.ClaimID, Generation: verdict.Generation}
	if verdict.ClaimID != "" && !ref.valid() {
		return nil, nil, fmt.Errorf("capacity admission omitted valid claim generation")
	}
	if !verdict.Granted {
		if verdict.ClaimID != "" {
			releaseClaim(broker, ref)
		}
		return nil, nil, fmt.Errorf("capacity claim denied: %s", verdict.Reason)
	}
	if verdict.ClaimID == "" {
		return nil, nil, fmt.Errorf("capacity admission omitted claim identity")
	}
	if verdict.AppliedRung != "full" && verdict.AppliedRung != "offload-dit" {
		releaseClaim(broker, ref)
		return nil, nil, fmt.Errorf("capacity admission returned unsupported rung %q", verdict.AppliedRung)
	}
	release := maintain(ctx, cancel, broker, ref, 2*time.Minute)
	meta := map[string]string{"capacity_claim_id": verdict.ClaimID, "capacity_claim_generation": strconv.FormatInt(verdict.Generation, 10), "capacity_admitted_rung": verdict.AppliedRung}
	if ShouldDegrade(verdict) {
		meta["capacity_degrade"] = "offload-dit"
		meta["capacity_admitted_rung"] = "offload-dit"
	}
	return release, meta, nil
}

func (b CLICapacityBroker) Heartbeat(ctx context.Context, ref ClaimRef) error {
	if !ref.valid() {
		return fmt.Errorf("capacity heartbeat requires captured claim ID and positive generation")
	}
	_, err := b.output(ctx, "heartbeat", "--claim-id", ref.ClaimID, "--generation", strconv.FormatInt(ref.Generation, 10), "--ttl", "10m", "--json")
	return err
}

func releaseClaim(broker Broker, ref ClaimRef) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := broker.Release(ctx, ref); err != nil {
		log.Printf("music capacity release %s generation %d failed: %v", ref.ClaimID, ref.Generation, err)
	}
}

func maintain(ctx context.Context, cancel context.CancelCauseFunc, broker Broker, ref ClaimRef, interval time.Duration) func() {
	if !ref.valid() {
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
				err := broker.Heartbeat(request, ref)
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
	return func() { once.Do(func() { stop(); <-done; releaseClaim(broker, ref) }) }
}
