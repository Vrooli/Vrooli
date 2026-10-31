package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"web-console/internal/sessionstore"
)

type homeFixture struct {
	t    *testing.T
	root string
}

func (f homeFixture) write(rel string, size int) string {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		f.t.Fatal(err)
	}
	return path
}

func (f homeFixture) dir(rel string) string {
	f.t.Helper()
	path := filepath.Join(f.root, rel)
	if err := os.MkdirAll(path, 0o755); err != nil {
		f.t.Fatal(err)
	}
	return path
}

// age backdates every entry beneath rel without following symlinks.
func (f homeFixture) age(rel string, by time.Duration) {
	f.t.Helper()
	when := time.Now().Add(-by)
	_ = filepath.WalkDir(filepath.Join(f.root, rel), func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Type()&fs.ModeSymlink == 0 {
			_ = os.Chtimes(path, when, when)
		}
		return nil
	})
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Lstat(path)
	return err == nil
}

func newHomeRetention(root string, rows []sessionstore.Metadata, live ...string) *sessionHomeRetention {
	liveSet := map[string]bool{}
	for _, id := range live {
		liveSet[id] = true
	}
	return &sessionHomeRetention{
		root:       root,
		minIdle:    7 * 24 * time.Hour,
		maxActions: 1000,
		maxBytes:   1 << 30,
		isLive:     func(id string) bool { return liveSet[id] },
		rows:       func(context.Context) ([]sessionstore.Metadata, error) { return rows, nil },
	}
}

func TestSessionHomeRetentionStripsRuntimeStateAndKeepsTranscript(t *testing.T) {
	root := t.TempDir()
	shared := t.TempDir()
	f := homeFixture{t: t, root: root}
	sharedPlugin := homeFixture{t: t, root: shared}.write("plugins/keep.bin", 64)

	rollout := f.write("codex/finished/sessions/2026/08/01/rollout-a.jsonl", 1000)
	history := f.write("codex/finished/history.jsonl", 100)
	thread := f.write("codex/finished/thread_history_1.sqlite", 500)
	state := f.write("codex/finished/state_5.sqlite", 50)
	f.write("codex/finished/packages/app-server-daemon/releases/0.161.0/bin/codex", 4000)
	f.write("codex/finished/plugins/cache/p.bin", 300)
	f.write("codex/finished/.tmp/plugins-sync/x", 200)
	f.write("codex/finished/logs_2.sqlite", 700)
	f.write("codex/finished/logs_2.sqlite-wal", 70)
	if err := os.Symlink(filepath.Join(shared, "plugins"), filepath.Join(root, "codex", "finished", "cache")); err != nil {
		t.Fatal(err)
	}
	f.age("codex/finished", 40*24*time.Hour)

	rows := []sessionstore.Metadata{{ID: "finished", AgentType: sessionstore.AgentCodex, Status: sessionstore.StatusLive, ArchivedAt: time.Now().Add(-40 * 24 * time.Hour)}}
	result, err := newHomeRetention(root, rows).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, kept := range []string{rollout, history, thread, state} {
		if !exists(t, kept) {
			t.Fatalf("transcript file removed: %s", kept)
		}
	}
	for _, gone := range []string{"packages", "plugins", ".tmp", "logs_2.sqlite", "logs_2.sqlite-wal"} {
		if exists(t, filepath.Join(root, "codex", "finished", gone)) {
			t.Fatalf("regenerable %s survived", gone)
		}
	}
	if !exists(t, filepath.Join(root, "codex", "finished", "cache")) || !exists(t, sharedPlugin) {
		t.Fatal("a symlink into the shared agent home, or its target, was removed")
	}
	if result.FreedBytes != 4000+300+200+700+70 {
		t.Fatalf("freed = %d", result.FreedBytes)
	}
	if result.TranscriptBytes != 1000+100+500+50 || result.Protected[homeProtectedTranscript] != 1 {
		t.Fatalf("transcript accounting = %d bytes, protected %v", result.TranscriptBytes, result.Protected)
	}
	if result.StrippedHomes != 1 || result.UsedBytesAfter != result.UsedBytesBefore-result.FreedBytes {
		t.Fatalf("result = %+v", result)
	}
}

func TestSessionHomeRetentionLeavesOpenAndRecentSessionsUntouched(t *testing.T) {
	root := t.TempDir()
	f := homeFixture{t: t, root: root}
	for _, id := range []string{"in-memory", "open-row", "recent", "awaiting"} {
		f.write("codex/"+id+"/packages/bin/codex", 4000)
		f.write("codex/"+id+"/sessions/rollout.jsonl", 10)
	}
	f.write("grok/in-memory/downloads/grok-bin", 300)
	f.age("codex/in-memory", 40*24*time.Hour)
	f.age("grok/in-memory", 40*24*time.Hour)
	f.age("codex/open-row", 40*24*time.Hour)
	f.age("codex/awaiting", 40*24*time.Hour)

	rows := []sessionstore.Metadata{
		{ID: "open-row", Status: sessionstore.StatusLive},
		{ID: "awaiting", Status: sessionstore.StatusAwaitingRecovery},
	}
	result, err := newHomeRetention(root, rows, "in-memory").Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"in-memory", "open-row", "recent"} {
		if !exists(t, filepath.Join(root, "codex", id, "packages")) {
			t.Fatalf("%s session lost runtime state", id)
		}
	}
	if !exists(t, filepath.Join(root, "grok", "in-memory", "downloads")) {
		t.Fatal("live grok session lost its downloads")
	}
	// A crashed session awaiting recovery is not open: recovery copies only its
	// rollout tree, so its runtime state is reclaimable once idle.
	if exists(t, filepath.Join(root, "codex", "awaiting", "packages")) || !exists(t, filepath.Join(root, "codex", "awaiting", "sessions", "rollout.jsonl")) {
		t.Fatal("awaiting-recovery session: want runtime state removed and rollout kept")
	}
	want := map[string]int{homeProtectedLiveSession: 2, homeProtectedOpenSessionRow: 1, homeProtectedRecentActivity: 1, homeProtectedTranscript: 1}
	for reason, count := range want {
		if result.Protected[reason] != count {
			t.Fatalf("protected[%s] = %d, want %d (all: %v)", reason, result.Protected[reason], count, result.Protected)
		}
	}
}

func TestSessionHomeRetentionRemovesOnlyUnclaimedEmptyScaffolds(t *testing.T) {
	root := t.TempDir()
	shared := t.TempDir()
	f := homeFixture{t: t, root: root}
	for _, id := range []string{"orphan", "claimed"} {
		f.dir("grok/" + id + "/sessions")
		if err := os.Symlink(shared, filepath.Join(root, "grok", id, "skills")); err != nil {
			t.Fatal(err)
		}
		f.age("grok/"+id, 40*24*time.Hour)
	}
	rows := []sessionstore.Metadata{{ID: "claimed", Status: sessionstore.StatusDismissed}}
	result, err := newHomeRetention(root, rows).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if exists(t, filepath.Join(root, "grok", "orphan")) {
		t.Fatal("unclaimed empty scaffold survived")
	}
	if !exists(t, filepath.Join(root, "grok", "claimed")) || !exists(t, shared) {
		t.Fatal("row-backed scaffold or the shared home was removed")
	}
	if result.RemovedScaffolds != 1 || result.Protected[homeProtectedScaffoldWithRow] != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestSessionHomeRetentionHonorsCycleActionBudget(t *testing.T) {
	root := t.TempDir()
	f := homeFixture{t: t, root: root}
	for _, id := range []string{"a", "b", "c"} {
		f.dir("codex/" + id + "/sessions")
		f.age("codex/"+id, 40*24*time.Hour)
	}
	owner := newHomeRetention(root, nil)
	owner.maxActions = 2
	result, err := owner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedScaffolds != 2 || !result.ActionBudgetSpent {
		t.Fatalf("result = %+v", result)
	}
	if err := sessionHomeReceiptError(result, nil); err == nil || !strings.Contains(err.Error(), "action budget") {
		t.Fatalf("receipt error = %v; want backlog reported", err)
	}
}

func TestSessionHomeReceiptReportsOverBudgetWithProtectedReasons(t *testing.T) {
	result := sessionHomeRetentionResult{
		MaxBytes:       100,
		UsedBytesAfter: 150,
		Protected:      map[string]int{homeProtectedTranscript: 3, homeProtectedLiveSession: 1},
	}
	err := sessionHomeReceiptError(result, nil)
	if err == nil {
		t.Fatal("over-budget cycle recorded as enforced")
	}
	for _, want := range []string{"over budget", "live_session=1", "transcript_retained=3"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("receipt error %q missing %q", err, want)
		}
	}
	result.UsedBytesAfter = 90
	if err := sessionHomeReceiptError(result, nil); err != nil {
		t.Fatalf("within-budget cycle reported error: %v", err)
	}
}

func TestSessionHomeCycleReceiptPersistsProtectedReasons(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	want := sessionHomeRetentionResult{FreedBytes: 42, Protected: map[string]int{homeProtectedLiveSession: 2, homeProtectedTranscript: 5}}
	if err := writeSessionHomeCycleReceipt(want); err != nil {
		t.Fatal(err)
	}
	path, err := coreRetention.EnforcementReceiptPath(sessionHomeRetentionOwner)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), sessionHomeRetentionCycleFile))
	if err != nil {
		t.Fatal(err)
	}
	var got sessionHomeRetentionResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.FreedBytes != 42 || got.Protected[homeProtectedLiveSession] != 2 || got.Protected[homeProtectedTranscript] != 5 {
		t.Fatalf("persisted receipt = %+v", got)
	}
}
