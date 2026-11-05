package recovery

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agentcatalog/harness"
)

func TestCodingAgentBrokerDeduplicatesPersistsAndFallsBack(t *testing.T) {
	path := t.TempDir() + "/coding-agent-records.json"
	base := time.Date(2026, 9, 10, 14, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	done := make(chan struct{}, 1)
	runner := func(_ context.Context, name string, _ CodingAgentRequest) error {
		calls.Add(1)
		if name == "opencode" {
			return errors.New("runner unavailable")
		}
		done <- struct{}{}
		return nil
	}
	b, err := NewCodingAgentBroker(path, runner)
	if err != nil {
		t.Fatal(err)
	}
	b.now = func() time.Time { return base }
	b.runners = []string{"opencode", "codex"}
	first, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "unhealthy"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "same incident"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate admission IDs differ: %s != %s", first.ID, second.ID)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fallback runner did not complete")
	}
	// The runner signals just before returning; allow the broker worker to
	// persist the terminal outcome before exercising reload semantics.
	time.Sleep(20 * time.Millisecond)
	if got := calls.Load(); got != 2 {
		t.Fatalf("runner calls = %d, want opencode failure plus codex fallback", got)
	}

	reloaded, err := NewCodingAgentBroker(path, runner)
	if err != nil {
		t.Fatal(err)
	}
	records := reloaded.Records()
	if len(records) != 1 || records[0].Outcome != "succeeded" || records[0].Runner != "codex" {
		t.Fatalf("persisted records = %+v, want one successful codex record", records)
	}

	reloaded.now = func() time.Time { return base.Add(CodingAgentRecoveryWindow + time.Second) }
	third, err := reloaded.Submit(CodingAgentRequest{Scenario: "agent-manager", Requester: "test", Reason: "next window"})
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == first.ID {
		t.Fatal("new rolling window reused the old admission")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("next-window runner did not complete")
	}
	time.Sleep(20 * time.Millisecond)
}

// Which harnesses may run an unattended investigation is a decision, so a new
// harness has to be placed on one side of it. Antigravity is left out: its live
// enforcement canary and live detection check are still pending.
func TestDefaultRunnersAccountForEveryDeclaredHarness(t *testing.T) {
	excluded := map[string]bool{"agy": true}
	included := map[string]bool{}
	for _, runner := range DefaultCodingAgentRunners {
		included[runner] = true
	}
	all := harness.Builtin().All()
	for _, descriptor := range all {
		if included[descriptor.ID] == excluded[descriptor.ID] {
			t.Errorf("harness %q must be in exactly one of DefaultCodingAgentRunners and the exclusions", descriptor.ID)
		}
	}
	if len(included)+len(excluded) != len(all) {
		t.Errorf("%d default and %d excluded runners, but %d declared harnesses", len(included), len(excluded), len(all))
	}
}
