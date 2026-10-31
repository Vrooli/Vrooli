package vroolicli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGPUCapacityLeaseBindsFreshUUIDAndRenewsExactGeneration(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":3,"name":"RTX","uuid":"GPU-exact","vram_bytes":"17179869184"}]}`)},
		{output: []byte(`{"verdict":{"kind":"grant"},"claim":{"claim_id":"claim-1","generation":7,"owner_id":"ds:render","gpu_index":3,"resource_kind":"vram","amount_bytes":2147483648,"status":"granted"}}`)},
		{output: []byte(`{"gpus":[{"index":3,"name":"RTX","uuid":"GPU-exact","vram_bytes":"17179869184"}]}`)},
		{output: []byte(`{"claim_id":"claim-1","generation":7,"status":"granted"}`)},
		{output: []byte(`{"claim_id":"claim-1"}`)},
	}}
	client := New(WithRunner(runner))
	lease, err := client.AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds:render", GPUUUID: "GPU-exact", VRAMBytes: 2 << 30})
	if err != nil {
		t.Fatal(err)
	}
	if got := lease.Identity(); got.Index != 3 || got.UUID != "GPU-exact" || got.Name != "RTX" {
		t.Fatalf("identity = %+v", got)
	}
	if err := lease.Heartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !equalArgs(runner.calls[1].args, []string{"capacity", "claim", "--owner-kind", "op", "--owner-id", "ds:render", "--resource-kind", "vram", "--gpu-index", "3", "--preferred", "2147483648", "--floor", "2147483648", "--priority", "interactive", "--no-reclaim", "--protected", "--ttl", "15m0s", "--json"}) {
		t.Fatalf("claim args = %v", runner.calls[1].args)
	}
	if !equalArgs(runner.calls[3].args, []string{"capacity", "heartbeat", "--claim-id", "claim-1", "--generation", "7", "--ttl", "15m0s", "--json"}) {
		t.Fatalf("heartbeat args = %v", runner.calls[3].args)
	}
	if !equalArgs(runner.calls[4].args, []string{"capacity", "release", "--claim-id", "claim-1", "--json"}) {
		t.Fatalf("release args = %v", runner.calls[4].args)
	}
}

func TestGPUCapacityRefusesUnknownUUIDAndCleansDeniedClaim(t *testing.T) {
	unknown := &stubRunner{responses: []stubResponse{{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)}}}
	if _, err := New(WithRunner(unknown)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds", GPUUUID: "GPU-2", VRAMBytes: 1}); err == nil {
		t.Fatal("unknown UUID admitted")
	}
	if len(unknown.calls) != 1 {
		t.Fatalf("unknown UUID invoked capacity claim: %d calls", len(unknown.calls))
	}

	denied := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"verdict":{"kind":"deny","reason":"insufficient free VRAM"},"claim":{"claim_id":"claim-denied","generation":1,"owner_id":"ds","gpu_index":0,"resource_kind":"vram","amount_bytes":1,"status":"granted"}}`)},
		{output: []byte(`{"claim_id":"claim-denied"}`)},
	}}
	if _, err := New(WithRunner(denied)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds", GPUUUID: "GPU-1", VRAMBytes: 1}); err == nil {
		t.Fatal("denied GPU claim returned a lease")
	}
	if len(denied.calls) != 3 || denied.calls[2].args[1] != "release" {
		t.Fatalf("denied claim was not cleaned up: calls=%+v", denied.calls)
	}
}

func TestGPUCapacityReleaseFailureRemainsRetryable(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"verdict":{"kind":"grant"},"claim":{"claim_id":"claim-1","generation":1,"owner_id":"ds","gpu_index":0,"resource_kind":"vram","amount_bytes":1,"status":"granted"}}`)},
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{err: errors.New("release unavailable")},
		{output: []byte(`{"claim_id":"claim-1"}`)},
	}}
	lease, err := New(WithRunner(runner)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds", GPUUUID: "GPU-1", VRAMBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err == nil {
		t.Fatal("first release unexpectedly succeeded")
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatalf("retry release: %v", err)
	}
	if len(runner.calls) != 5 {
		t.Fatalf("release attempts = %d", len(runner.calls))
	}
	if err := lease.Heartbeat(context.Background()); err == nil {
		t.Fatal("heartbeat succeeded after release")
	}
}

func TestGPUCapacityCleanupSurvivesCancelledContextAndLostHeartbeatIsRefused(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"verdict":{"kind":"grant"},"claim":{"claim_id":"claim-1","generation":2,"owner_id":"ds","gpu_index":0,"resource_kind":"vram","amount_bytes":1,"status":"granted"}}`)},
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"claim_id":"claim-1","generation":2,"status":"released"}`)},
		{output: []byte(`{"claim_id":"claim-1"}`)},
	}}
	lease, err := New(WithRunner(runner)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds", GPUUUID: "GPU-1", VRAMBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Heartbeat(context.Background()); err == nil {
		t.Fatal("heartbeat accepted a released claim")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lease.Release(ctx); err != nil {
		t.Fatalf("release after owner cancellation: %v", err)
	}
	if !runner.calls[4].hasDeadline {
		t.Fatal("cleanup release did not use a bounded context")
	}
}

func TestGPUCapacityDeniedCleanupFailureIsVisible(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"verdict":{"kind":"deny","reason":"no room"},"claim":{"claim_id":"claim-denied","generation":1,"owner_id":"ds","gpu_index":0,"resource_kind":"vram","amount_bytes":1,"status":"granted"}}`)},
		{err: errors.New("ledger unavailable")},
	}}
	_, err := New(WithRunner(runner)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "ds", GPUUUID: "GPU-1", VRAMBytes: 1})
	if err == nil || !strings.Contains(err.Error(), "cleanup claim claim-denied failed") || !strings.Contains(err.Error(), "ledger unavailable") {
		t.Fatalf("denied cleanup failure was not reported: %v", err)
	}
}

func TestGPUCapacityPolicyAndAdvisoryReclaim(t *testing.T) {
	for _, tc := range []struct {
		name      string
		allow     bool
		enforce   string
		reclaim   int64
		wantGrant bool
	}{
		{"no-reclaim-independent-of-service-priority", false, "advisory", 0, true},
		{"explicit-reclaim-enforced", true, "on", 1024, true},
		{"advisory-reclaim-is-unavailable", true, "advisory", 1024, false},
		{"unexpected-reclaim-is-unavailable", false, "on", 1024, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)
			claim := []byte(fmt.Sprintf(`{"verdict":{"kind":"grant","reclaim_bytes":%d},"claim":{"claim_id":"claim-1","generation":1,"owner_id":"test","gpu_index":0,"resource_kind":"vram","amount_bytes":2147483648,"status":"granted"},"enforce":%q}`, tc.reclaim, tc.enforce))
			responses := []stubResponse{{output: host}, {output: claim}}
			if tc.wantGrant {
				responses = append(responses, stubResponse{output: host})
			} else {
				responses = append(responses, stubResponse{output: []byte(`{"claim_id":"claim-1"}`)})
			}
			runner := &stubRunner{responses: responses}
			lease, err := New(WithRunner(runner)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "test", GPUUUID: "GPU-1", VRAMBytes: 2 << 30, Priority: "service", AllowReclaim: tc.allow})
			if tc.wantGrant {
				if err != nil || lease.ReclaimBytes() != tc.reclaim {
					t.Fatalf("grant=%v %v", lease, err)
				}
			} else if !errors.Is(err, ErrGPUCapacityUnavailable) {
				t.Fatalf("not a classified denial: %v", err)
			}
			args := strings.Join(runner.calls[1].args, " ")
			if !strings.Contains(args, "--priority service") || strings.Contains(args, "--no-reclaim") == tc.allow {
				t.Fatalf("priority/reclaim policy lost: %s", args)
			}
		})
	}
}

func TestGPUCapacityIndexReorderReturnsClaimBeforeRefusing(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte(`{"gpus":[{"index":0,"uuid":"GPU-1"}]}`)},
		{output: []byte(`{"verdict":{"kind":"grant"},"claim":{"claim_id":"claim-1","generation":1,"owner_id":"test","gpu_index":0,"resource_kind":"vram","amount_bytes":1,"status":"granted"}}`)},
		{output: []byte(`{"gpus":[{"index":1,"uuid":"GPU-1"},{"index":0,"uuid":"GPU-other"}]}`)},
		{output: []byte(`{"claim_id":"claim-1"}`)},
	}}
	_, err := New(WithRunner(runner)).AcquireGPUCapacity(context.Background(), GPUCapacityRequest{OwnerID: "test", GPUUUID: "GPU-1", VRAMBytes: 1})
	if err == nil || !strings.Contains(err.Error(), "identity changed") || len(runner.calls) != 4 || runner.calls[3].args[1] != "release" {
		t.Fatalf("reordered GPU admitted or claim leaked: err=%v calls=%v", err, runner.calls)
	}
}
