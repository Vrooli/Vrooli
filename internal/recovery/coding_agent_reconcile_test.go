package recovery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodingAgentReconciliationPreservesUnknownExecutionAndSuppression(t *testing.T) {
	now := time.Now().UTC()
	records := []CodingAgentRecord{
		{ID: "legacy", Scenario: "legacy-app", Outcome: "running", Error: "retained original diagnostic", CreatedAt: now.Add(-1000 * time.Hour)},
		{ID: "expired", Scenario: "expired-app", Outcome: "scheduled", CreatedAt: now.Add(-2 * time.Hour), DeadlineAt: now.Add(-time.Hour)},
		{ID: "current", Scenario: "current-app", Outcome: "running", CreatedAt: now, DeadlineAt: now.Add(time.Hour)},
		{ID: "terminal", Scenario: "done-app", Outcome: "succeeded", CreatedAt: now.Add(-time.Hour), FinishedAt: now},
	}
	path := filepath.Join(t.TempDir(), "coding-agent-records.json")
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	b, err := NewCodingAgentBroker(path, func(context.Context, string, CodingAgentRequest) error { calls++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	b.now = func() time.Time { return now }
	for _, wantID := range []string{"legacy", "expired"} {
		report, err := b.ReconcileUnverified(context.Background(), 1)
		if err != nil || len(report.Changed) != 1 || report.Changed[0].ID != wantID || report.ExitProven || report.ExecutionEffects {
			t.Fatalf("reconciliation invented effects: %+v %v", report, err)
		}
		r := report.Changed[0]
		if r.Outcome != "unconfirmed" || r.ReconciledAt == nil || !r.ReconciledAt.Equal(now) || !r.FinishedAt.IsZero() {
			t.Fatalf("missing native proof became completion: %+v", r)
		}
		got, err := b.Submit(CodingAgentRequest{Scenario: r.Scenario, Requester: "fixture"})
		if err != nil || got.ID != r.ID || got.Outcome != "unconfirmed" {
			t.Fatalf("reconciliation released unresolved budget: %+v %v", got, err)
		}
	}
	if calls != 0 {
		t.Fatal("metadata reconciliation spawned an agent")
	}
	report, err := b.ReconcileUnverified(context.Background(), 32)
	if err != nil || len(report.Changed) != 0 || report.Remaining != 0 {
		t.Fatalf("replay changed the ledger: %+v %v", report, err)
	}
	loaded, err := NewCodingAgentBroker(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	current, err := loaded.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if current[2].Outcome != "running" || current[3].Outcome != "succeeded" {
		t.Fatal("current or terminal execution was rewritten")
	}
	if current[0].ReconciledAt == nil {
		t.Fatal("reload lost uncertainty audit")
	}
	if current[0].Error != "retained original diagnostic" || current[0].ReconciliationReason == "" {
		t.Fatal("reconciliation overwrote causal evidence")
	}
}
