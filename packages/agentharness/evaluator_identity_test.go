package agentharness

import (
	"testing"
	"time"
)

func TestTemporaryModeMatchesOneProviderEvaluatorBeforeCombine(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
	first := testSnapshot(now, "provider-a", MaturityEnforcing, HealthHealthy, EvidenceClean)
	first.Rules = []PolicyRule{{EvaluatorID: "provider-a:dependency-addition", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "first evaluator denied"}}
	second := testSnapshot(now, "provider-b", MaturityEnforcing, HealthHealthy, EvidenceClean)
	second.Rules = []PolicyRule{{EvaluatorID: "provider-b:dependency-addition", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "second evaluator denied"}}
	publishTestBundle(t, store, first, second)
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	if err := modes.SetLogOnly("provider-a:dependency-addition", now.Add(time.Hour), "reviewed exact evaluator", "operator"); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{Profile: ProfileEnforcing, Store: store, GuardModes: modes, Now: func() time.Time { return now }}
	got, err := runtime.Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != ActionDeny || got.ProviderID != "provider-b" || got.EvaluatorID != "provider-b:dependency-addition" {
		t.Fatalf("unconfigured provider evaluator did not retain enforcement: %+v", got)
	}
	entries, err := modes.Audit()
	if err != nil || len(entries) != 2 || entries[1].Check != "provider-a:dependency-addition" {
		t.Fatalf("mode audit = %+v, err=%v", entries, err)
	}
}

func TestMissingLegacyAndForeignEvaluatorIDsAreNotSelectable(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"", string(RiskDependencyAdd), "provider-b:rule", "provider-a:bad rule", "provider-a:two:parts"} {
		t.Run(id, func(t *testing.T) {
			store := NewBundleStore(t.TempDir())
			snapshot := testSnapshot(now, "provider-a", MaturityEnforcing, HealthHealthy, EvidenceClean)
			snapshot.Rules = []PolicyRule{{EvaluatorID: id, Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "denied"}}
			publishTestBundle(t, store, snapshot)
			modes := NewGuardModeStore(store.Dir)
			modes.Now = func() time.Time { return now }
			if err := modes.SetLogOnly("provider-a:dependency-addition", now.Add(time.Hour), "reviewed", "operator"); err != nil {
				t.Fatal(err)
			}
			runtime := Runtime{Profile: ProfileEnforcing, Store: store, GuardModes: modes, Now: func() time.Time { return now }}
			got, err := runtime.Evaluate(testEvent())
			if err != nil || got.Action != ActionDeny || got.EvaluatorID != "" {
				t.Fatalf("legacy/foreign rule became selectable: decision=%+v err=%v", got, err)
			}
		})
	}
}

func TestSnapshotRejectsDuplicateEvaluatorIDs(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snapshot := testSnapshot(now, "provider-a", MaturityEnforcing, HealthHealthy, EvidenceClean)
	snapshot.Rules = []PolicyRule{
		{EvaluatorID: "provider-a:shared", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "first"},
		{EvaluatorID: "provider-a:shared", Risk: RiskDependencyUpgrade, Action: ActionDeny, Reason: "second"},
	}
	if err := ValidateSnapshot(snapshot); err == nil {
		t.Fatal("snapshot with duplicate evaluator IDs was accepted")
	}
}
