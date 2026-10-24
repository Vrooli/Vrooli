package recovery

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
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
