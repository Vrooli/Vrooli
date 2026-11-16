package guardmodes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordSuppressedActionRequiresActiveModeAndPersistsMinimalEvidence(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnly(RiskDependencyAdd, now.Add(time.Hour), "reviewed repeated block", "local operator"); err != nil {
		t.Fatal(err)
	}
	mode, active, err := store.RecordSuppressedAction(RiskDependencyAdd, "event-digest", Action("deny"), now)
	if err != nil || !active || mode.Check != RiskDependencyAdd {
		t.Fatalf("record suppression mode=%+v active=%t err=%v", mode, active, err)
	}
	entries, err := store.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Action != "observed" || entries[1].Check != RiskDependencyAdd || entries[1].EventID != "event-digest" || entries[1].OriginalAction != "deny" || !entries[1].ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("audit entries = %+v", entries)
	}
	data, err := os.ReadFile(filepath.Join(store.Dir, guardModeAuditFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "shell") || strings.Contains(string(data), "private payload") {
		t.Fatalf("audit includes event content: %s", data)
	}
	if _, active, err := store.RecordSuppressedAction(RiskDependencyAdd, "allow-event", ActionAllow, now); err != nil || active {
		t.Fatalf("allow event unexpectedly recorded suppression: active=%t err=%v", active, err)
	}
	entries, err = store.Audit()
	if err != nil || len(entries) != 2 {
		t.Fatalf("allow event should not add a suppressed-action audit: entries=%d err=%v", len(entries), err)
	}
}

func TestSetLogOnlyFromAnswerRequiresAndAuditsStableAnswerID(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnlyFromAnswer(RiskDependencyAdd, now.Add(time.Hour), "approved structural answer", "steward", ""); err == nil {
		t.Fatal("missing answer ID accepted")
	}
	if err := store.SetLogOnlyFromAnswer(RiskDependencyAdd, now.Add(time.Hour), "approved structural answer", "steward", "answer-42"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].AuthorizationID != "answer-42" {
		t.Fatalf("authorization evidence = %+v, want stable answer ID", entries)
	}
	active, err := store.Active(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].AuthorizationID != "answer-42" {
		t.Fatalf("active mode = %+v, want answer provenance retained", active)
	}
}

func TestRestoreAllRetriesLockContentionAndClearsActiveMode(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnly(RiskDependencyAdd, now.Add(time.Hour), "test", "operator"); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireFileLock(filepath.Join(store.Dir, guardModeLockFile))
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		checks []CheckID
		err    error
	}
	resultCh := make(chan result, 1)
	go func() {
		checks, restoreErr := store.RestoreAll(HardStopOutsideMessage, "hard-stop-event", now)
		resultCh <- result{checks: checks, err: restoreErr}
	}()
	// Let RestoreAll observe the held lock before releasing it.
	time.Sleep(15 * time.Millisecond)
	unlock()
	select {
	case got := <-resultCh:
		if got.err != nil || len(got.checks) != 1 || got.checks[0] != RiskDependencyAdd {
			t.Fatalf("restore result checks=%v err=%v", got.checks, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("RestoreAll did not resume after the contending lock was released")
	}
	active, err := store.Active(now)
	if err != nil || len(active) != 0 {
		t.Fatalf("mode remained active after restoration: active=%+v err=%v", active, err)
	}
}

func TestAuditRetentionKeepsRecentEntriesAndActiveModeEnable(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	mode := GuardMode{Check: RiskDependencyAdd, Mode: "log_only", EnabledAt: now, ExpiresAt: now.Add(time.Hour), Actor: "operator", Reason: "test"}
	state := guardModeState{SchemaVersion: 1, Active: map[CheckID]GuardMode{RiskDependencyAdd: mode}}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.writeLocked(state); err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	enabled, err := json.Marshal(GuardModeAudit{At: now, Action: "enabled", Check: RiskDependencyAdd, ExpiresAt: mode.ExpiresAt, Actor: mode.Actor, Reason: mode.Reason})
	if err != nil {
		t.Fatal(err)
	}
	data.Write(enabled)
	data.WriteByte('\n')
	for i := 0; i < maxGuardModeAudits+1; i++ {
		entry, err := json.Marshal(GuardModeAudit{At: now, Action: "observed", Check: RiskDependencyAdd, EventID: "event"})
		if err != nil {
			t.Fatal(err)
		}
		data.Write(entry)
		data.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, guardModeAuditFile), []byte(data.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.compactAuditLocked(); err != nil {
		t.Fatal(err)
	}
	entries, err := store.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > maxGuardModeAudits+1 {
		t.Fatalf("audit retained %d entries, bound is %d plus active mode record", len(entries), maxGuardModeAudits)
	}
	if entries[0].Action != "enabled" || entries[0].Check != RiskDependencyAdd {
		t.Fatalf("active mode's enable record was not retained: first=%+v", entries[0])
	}
	active, err := store.Active(now)
	if err != nil || len(active) != 1 {
		t.Fatalf("compacted audit broke active mode recovery: active=%+v err=%v", active, err)
	}
}
