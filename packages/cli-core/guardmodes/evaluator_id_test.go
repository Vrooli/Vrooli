package guardmodes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyRiskClassesCannotBeSelectedForNewModes(t *testing.T) {
	store := NewStore(t.TempDir())
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { return now }
	for _, legacy := range []CheckID{RiskDependencyAdd, RiskLifecycle, RiskUnknown} {
		if err := store.SetLogOnly(legacy, now.Add(time.Hour), "reason", "actor"); err == nil {
			t.Errorf("legacy risk %q accepted for a new mode", legacy)
		}
	}
	for _, malformed := range []CheckID{"", ":rule", "provider:", "provider:two:parts", "provider:bad rule"} {
		if err := store.SetLogOnly(malformed, now.Add(time.Hour), "reason", "actor"); err == nil {
			t.Errorf("malformed evaluator ID %q accepted", malformed)
		}
	}
	if err := store.SetLogOnly("scenario-dependency-analyzer:dependency-addition", now.Add(time.Hour), "reason", "actor"); err != nil {
		t.Fatalf("exact evaluator ID rejected: %v", err)
	}
}

func TestLegacyActiveModeRemainsVisibleAndRestorableButCannotSuppress(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	legacy := GuardMode{Check: RiskDependencyAdd, Mode: "log_only", EnabledAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), Actor: "operator", Reason: "historical mode"}
	state := guardModeState{SchemaVersion: 1, Active: map[CheckID]GuardMode{RiskDependencyAdd: legacy}}
	if err := store.writeLocked(state); err != nil {
		t.Fatal(err)
	}
	if err := store.appendAuditLocked(GuardModeAudit{At: legacy.EnabledAt, Action: "enabled", Check: RiskDependencyAdd, ExpiresAt: legacy.ExpiresAt, Actor: legacy.Actor, Reason: legacy.Reason}); err != nil {
		t.Fatal(err)
	}
	active, err := store.Active(now)
	if err != nil || len(active) != 1 || active[0].Check != RiskDependencyAdd {
		t.Fatalf("legacy mode was not surfaced: active=%+v err=%v", active, err)
	}
	if _, suppressed, err := store.RecordSuppressedAction(RiskDependencyAdd, "event", Action("deny"), now); err == nil || suppressed {
		t.Fatalf("legacy mode was allowed to suppress: active=%t err=%v", suppressed, err)
	}
	if err := store.Disable(RiskDependencyAdd, "operator", "cleanup"); err == nil {
		t.Fatal("legacy mode became selectable for direct disable")
	}
	if _, err := os.Stat(filepath.Join(store.Dir, guardModeStateFile)); err != nil {
		t.Fatal(err)
	}
}
