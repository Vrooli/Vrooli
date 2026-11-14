package capacity

import (
	"context"
	"strings"
	"testing"
)

func TestCLIBrokerClaimParsesEnvelope(t *testing.T) {
	var gotArgs []string
	b := &CLIBroker{Exec: func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = args
		return []byte(`{"verdict":{"kind":"grant","warnings":["near limit"]},"claim":{"claim_id":"clm-abc"}}`), nil
	}}

	lease, err := b.Claim(context.Background(), "ollama:agent-manager-extract", OllamaExtractVRAMEstimateBytes)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if lease.ClaimID != "clm-abc" {
		t.Errorf("ClaimID = %q, want clm-abc", lease.ClaimID)
	}
	if len(lease.Warnings) != 1 || lease.Warnings[0] != "near limit" {
		t.Errorf("Warnings = %v", lease.Warnings)
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"capacity claim", "--owner-kind op", "--owner-id ollama:agent-manager-extract", "--priority service", "--resource-kind vram"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
}

func TestCLIBrokerReleaseNoopOnEmptyID(t *testing.T) {
	called := false
	b := &CLIBroker{Exec: func(_ context.Context, _ ...string) ([]byte, error) {
		called = true
		return nil, nil
	}}
	b.Release(context.Background(), "")
	if called {
		t.Error("Release(\"\") should not shell out")
	}
	b.Release(context.Background(), "clm-1")
	if !called {
		t.Error("Release(non-empty) should shell out")
	}
}
