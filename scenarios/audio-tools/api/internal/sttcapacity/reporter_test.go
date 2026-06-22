package sttcapacity

import (
	"context"
	"strings"
	"testing"
)

func TestActiveResolvesClaimAndReports(t *testing.T) {
	var calls [][]string
	r := &CLIReporter{vrooliBin: "vrooli", exec: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		if len(args) >= 2 && args[0] == "capacity" && args[1] == "list" {
			return []byte(`{"claims":[{"claim_id":"clm-w","owner_id":"whisper"}]}`), nil
		}
		return []byte(`{}`), nil
	}}

	claimID := r.Active(context.Background(), "whisper")
	if claimID != "clm-w" {
		t.Fatalf("claimID = %q, want clm-w", claimID)
	}
	if len(calls) != 2 {
		t.Fatalf("expected list + activity calls, got %d: %v", len(calls), calls)
	}
	if got := strings.Join(calls[0], " "); !strings.Contains(got, "capacity list --owner whisper --active") {
		t.Errorf("list args = %q", got)
	}
	if got := strings.Join(calls[1], " "); !strings.Contains(got, "capacity activity --claim-id clm-w --state active") {
		t.Errorf("activity args = %q", got)
	}
}

func TestActiveRejectsUnknownResource(t *testing.T) {
	called := false
	r := &CLIReporter{vrooliBin: "vrooli", exec: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		called = true
		return nil, nil
	}}
	if id := r.Active(context.Background(), "reranker"); id != "" {
		t.Errorf("Active(reranker) = %q, want empty (not whitelisted)", id)
	}
	if called {
		t.Error("should not shell out for a non-whitelisted resource")
	}
}

func TestActiveNoClaimReturnsEmpty(t *testing.T) {
	r := &CLIReporter{vrooliBin: "vrooli", exec: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(`{"claims":[]}`), nil
	}}
	if id := r.Active(context.Background(), "whisper"); id != "" {
		t.Errorf("Active with no claim = %q, want empty", id)
	}
}

func TestMissingBinaryNoOps(t *testing.T) {
	r := &CLIReporter{} // no vrooliBin
	if id := r.Active(context.Background(), "whisper"); id != "" {
		t.Errorf("Active without binary = %q, want empty", id)
	}
	r.Idle(context.Background(), "clm-w") // must not panic
}

func TestIdleReportsState(t *testing.T) {
	var got []string
	r := &CLIReporter{vrooliBin: "vrooli", exec: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		got = args
		return nil, nil
	}}
	r.Idle(context.Background(), "clm-w")
	if j := strings.Join(got, " "); !strings.Contains(j, "capacity activity --claim-id clm-w --state idle") {
		t.Errorf("idle args = %q", j)
	}
}
