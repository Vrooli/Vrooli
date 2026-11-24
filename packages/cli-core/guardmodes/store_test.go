package guardmodes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testEvaluatorID CheckID = "test-provider:dependency-addition"

func TestRecordSuppressedActionRequiresActiveModeAndPersistsMinimalEvidence(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnly(testEvaluatorID, now.Add(time.Hour), "reviewed repeated block", "local operator"); err != nil {
		t.Fatal(err)
	}
	mode, active, err := store.RecordSuppressedAction(testEvaluatorID, "event-digest", Action("deny"), now)
	if err != nil || !active || mode.Check != testEvaluatorID {
		t.Fatalf("record suppression mode=%+v active=%t err=%v", mode, active, err)
	}
	entries, err := store.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Action != "observed" || entries[1].Check != testEvaluatorID || entries[1].EventID != "event-digest" || entries[1].OriginalAction != "deny" || !entries[1].ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("audit entries = %+v", entries)
	}
	data, err := os.ReadFile(filepath.Join(store.Dir, guardModeAuditFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "shell") || strings.Contains(string(data), "private payload") {
		t.Fatalf("audit includes event content: %s", data)
	}
	if _, active, err := store.RecordSuppressedAction(testEvaluatorID, "allow-event", ActionAllow, now); err != nil || active {
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
	if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved structural answer", "steward", ""); err == nil {
		t.Fatal("missing answer ID accepted")
	}
	if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved structural answer", "steward", "answer-42"); err != nil {
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

func TestSetLogOnlyFromAnswerConsumesOnceAcrossRestartAndExpiry(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	store := NewStore(dir)
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(24*time.Hour), "approved", "steward", "answer-once"); err != nil {
		t.Fatal(err)
	}
	restarted := NewStore(dir)
	restarted.Now = func() time.Time { return now }
	if err := restarted.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(24*time.Hour), "approved", "steward", "answer-once"); err == nil || !strings.Contains(err.Error(), "already consumed") {
		t.Fatalf("restart replay error = %v, want consumed answer", err)
	}
	active, err := store.Active(now)
	if err != nil || len(active) != 1 || !active[0].ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("replay changed active interval: %+v, err=%v", active, err)
	}

	now = now.Add(25 * time.Hour)
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(24*time.Hour), "replayed", "steward", "answer-once"); err == nil || !strings.Contains(err.Error(), "already consumed") {
		t.Fatalf("after-expiry replay error = %v, want consumed answer", err)
	}
	active, err = store.Active(now)
	if err != nil || len(active) != 0 {
		t.Fatalf("expired mode was reinstated: %+v, err=%v", active, err)
	}
}

func TestSetLogOnlyFromAnswerConcurrentReplayHasOneCommit(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	const workers = 16
	start := make(chan struct{})
	results := make(chan error, workers)
	for i := 0; i < workers; i++ {
		store := NewStore(dir)
		store.Now = func() time.Time { return now }
		go func() {
			<-start
			results <- store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(24*time.Hour), "approved", "steward", "answer-concurrent")
		}()
	}
	close(start)
	successes, consumed := 0, 0
	for i := 0; i < workers; i++ {
		err := <-results
		if err == nil {
			successes++
		} else if strings.Contains(err.Error(), "already consumed") {
			consumed++
		} else {
			t.Fatalf("concurrent apply returned unexpected error: %v", err)
		}
	}
	if successes != 1 || consumed != workers-1 {
		t.Fatalf("successful applies=%d consumed replays=%d", successes, consumed)
	}
	active, err := NewStore(dir).Active(now)
	if err != nil || len(active) != 1 || !active[0].ExpiresAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("concurrent result active=%+v err=%v", active, err)
	}
}

func TestSetLogOnlyFromAnswerFailureOrdering(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	t.Run("audit write failure leaves answer retryable and mode unchanged", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, guardModeAuditFile), 0o700); err != nil {
			t.Fatal(err)
		}
		store := NewStore(dir)
		store.Now = func() time.Time { return now }
		if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved", "steward", "answer-audit-fail"); err == nil {
			t.Fatal("expected audit append failure")
		}
		if _, err := os.Stat(filepath.Join(dir, guardModeStateFile)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("audit failure wrote state: stat error=%v", err)
		}
		if err := os.Remove(filepath.Join(dir, guardModeAuditFile)); err != nil {
			t.Fatal(err)
		}
		if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved", "steward", "answer-audit-fail"); err != nil {
			t.Fatalf("retry after audit failure did not apply: %v", err)
		}
	})

	t.Run("state write failure burns answer and restart refuses replay", func(t *testing.T) {
		dir := t.TempDir()
		store := NewStore(dir)
		store.Now = func() time.Time { return now }
		store.writeState = func(guardModeState) error { return os.ErrPermission }
		if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved", "steward", "answer-state-fail"); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("state write error = %v, want permission error", err)
		}
		restarted := NewStore(dir)
		restarted.Now = func() time.Time { return now }
		if err := restarted.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(time.Hour), "approved", "steward", "answer-state-fail"); err == nil || !strings.Contains(err.Error(), "already consumed") {
			t.Fatalf("retry after failed state write = %v, want consumed answer", err)
		}
		if active, err := restarted.Active(now); err != nil || len(active) != 0 {
			t.Fatalf("failed commit produced active mode: %+v err=%v", active, err)
		}
	})
}

func TestSetLogOnlyFromAnswerMigratesLegacyAnswerAuditTombstones(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	legacyState := []byte(`{"schema_version":1,"active":{}}`)
	if err := os.WriteFile(filepath.Join(dir, guardModeStateFile), legacyState, 0o600); err != nil {
		t.Fatal(err)
	}
	entry, err := json.Marshal(GuardModeAudit{At: now.Add(-time.Hour), Action: "enabled", Check: testEvaluatorID,
		ExpiresAt: now.Add(time.Hour), Actor: "steward", Reason: "legacy answer", AuthorizationID: "legacy-answer-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, guardModeAuditFile), append(entry, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnlyFromAnswer(testEvaluatorID, now.Add(24*time.Hour), "replayed", "steward", "legacy-answer-1"); err == nil || !strings.Contains(err.Error(), "already consumed") {
		t.Fatalf("legacy G11 answer replay = %v, want permanently consumed", err)
	}
}

func TestAuditRetentionKeepsRecentEntriesAndActiveModeEnable(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	mode := GuardMode{Check: testEvaluatorID, Mode: "log_only", EnabledAt: now, ExpiresAt: now.Add(time.Hour), Actor: "operator", Reason: "test"}
	state := guardModeState{SchemaVersion: 1, Active: map[CheckID]GuardMode{testEvaluatorID: mode}}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.writeLocked(state); err != nil {
		t.Fatal(err)
	}
	var data strings.Builder
	enabled, err := json.Marshal(GuardModeAudit{At: now, Action: "enabled", Check: testEvaluatorID, ExpiresAt: mode.ExpiresAt, Actor: mode.Actor, Reason: mode.Reason})
	if err != nil {
		t.Fatal(err)
	}
	data.Write(enabled)
	data.WriteByte('\n')
	for i := 0; i < maxGuardModeAudits+1; i++ {
		entry, err := json.Marshal(GuardModeAudit{At: now, Action: "observed", Check: testEvaluatorID, EventID: "event"})
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
	if entries[0].Action != "enabled" || entries[0].Check != testEvaluatorID {
		t.Fatalf("active mode's enable record was not retained: first=%+v", entries[0])
	}
	active, err := store.Active(now)
	if err != nil || len(active) != 1 {
		t.Fatalf("compacted audit broke active mode recovery: active=%+v err=%v", active, err)
	}
}

// Hooks read one check's mode on every guard stop, so the read never waits
// for the lock and never sees an expired mode.
func TestModeForReadsStateWithoutTheLock(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if _, active, err := store.ModeFor(testEvaluatorID, now); err != nil || active {
		t.Fatalf("empty store: active=%t err=%v", active, err)
	}
	if err := store.SetLogOnly(testEvaluatorID, now.Add(time.Hour), "test", "operator"); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireFileLock(filepath.Join(store.Dir, guardModeLockFile))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if mode, active, err := store.ModeFor(testEvaluatorID, now); err != nil || !active || mode.Reason != "test" {
		t.Fatalf("held lock must not block the read: mode=%+v active=%t err=%v", mode, active, err)
	}
	if _, active, _ := store.ModeFor("other:check", now); active {
		t.Fatal("another check reported active")
	}
	if _, active, err := store.ModeFor(testEvaluatorID, now.Add(2*time.Hour)); err != nil || active {
		t.Fatalf("expired mode reported active=%t err=%v", active, err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, guardModeStateFile), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ModeFor(testEvaluatorID, now); err == nil {
		t.Fatal("unparseable state must be reported")
	}
}

// The operator's answer can relax a guard for up to 30 days, never longer.
func TestAnswerIntervalIsBoundedByThirtyDays(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	store := NewStore(t.TempDir())
	store.Now = func() time.Time { return now }
	if err := store.SetLogOnlyFromAnswer("agent-guard:secret", now.Add(MaxModeTTL), "answer", "operator", "a-30d"); err != nil {
		t.Fatalf("30-day answer refused: %v", err)
	}
	if err := store.SetLogOnlyFromAnswer("agent-guard:money", now.Add(MaxModeTTL+time.Hour), "answer", "operator", "a-31d"); err == nil {
		t.Fatal("an answer beyond 30 days was accepted")
	}
}
