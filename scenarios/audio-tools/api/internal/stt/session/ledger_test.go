// [REQ:ATD-P0-001] Server-owned replay ledger preserves every captured range.
package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func newLedger(t *testing.T, maxBytes int) *Ledger {
	t.Helper()
	ledger, err := New(Config{SessionID: "session-a", ResumeToken: "resume-a", MaxBytes: maxBytes})
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func TestLedgerReceiveRejectsGapsAndDeduplicatesChunks(t *testing.T) {
	ledger := newLedger(t, 32)

	first := Chunk{Sequence: 0, StartSample: 0, EndSample: 4, Audio: []byte{1, 2, 3, 4}}
	if result, err := ledger.Receive(first); err != nil || result != ReceivedNew {
		t.Fatalf("Receive(first) = %v, %v; want ReceivedNew, nil", result, err)
	}
	if result, err := ledger.Receive(first); err != nil || result != ReceivedDuplicate {
		t.Fatalf("Receive(duplicate) = %v, %v; want ReceivedDuplicate, nil", result, err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 2, StartSample: 8, EndSample: 12, Audio: []byte{5}}); !errors.Is(err, ErrGap) {
		t.Fatalf("gap error = %v; want ErrGap", err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 1, StartSample: 4, EndSample: 8, Audio: []byte{5, 6, 7, 8}}); err != nil {
		t.Fatal(err)
	}
}

func TestLedgerResumeReturnsOnlyUnprocessedReplay(t *testing.T) {
	ledger := newLedger(t, 32)
	for _, chunk := range []Chunk{
		{Sequence: 0, StartSample: 0, EndSample: 4, Audio: []byte{1, 2, 3, 4}},
		{Sequence: 1, StartSample: 4, EndSample: 8, Audio: []byte{5, 6, 7, 8}},
	} {
		if _, err := ledger.Receive(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := ledger.AcknowledgeProcessed(0); err != nil {
		t.Fatal(err)
	}
	beforeResume, err := ledger.Resume("resume-a")
	if err != nil {
		t.Fatal(err)
	}
	if beforeResume.ReceivedSequence != 1 || beforeResume.ProcessedSequence != 0 || len(beforeResume.Replay) != 1 || beforeResume.Replay[0].Sequence != 1 {
		t.Fatalf("unexpected resume snapshot: %#v", beforeResume)
	}
}

func TestLedgerCommitIsIdempotentAndCompleteClearsReplay(t *testing.T) {
	ledger := newLedger(t, 32)
	for _, chunk := range []Chunk{
		{Sequence: 0, StartSample: 0, EndSample: 4, Audio: []byte{1, 2, 3, 4}},
		{Sequence: 1, StartSample: 4, EndSample: 8, Audio: []byte{5, 6, 7, 8}},
	} {
		if _, err := ledger.Receive(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if newCommit, err := ledger.Commit(Segment{ID: "segment-a", Text: "hello", StartSample: 0, EndSample: 4}); err != nil || !newCommit {
		t.Fatalf("Commit = %t, %v; want true, nil", newCommit, err)
	}
	if newCommit, err := ledger.Commit(Segment{ID: "segment-a", Text: "hello", StartSample: 0, EndSample: 4}); err != nil || newCommit {
		t.Fatalf("duplicate Commit = %t, %v; want false, nil", newCommit, err)
	}
	if err := ledger.AcknowledgeProcessed(1); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Complete(); err != nil {
		t.Fatal(err)
	}
	state := ledger.Snapshot()
	if state.TerminalReason != TerminalCompleted || len(state.Replay) != 0 || state.ProcessedSequence != 1 {
		t.Fatalf("unexpected completed state: %#v", state)
	}
}

func TestLedgerNeverSilentlyCompletesIncompleteCoverage(t *testing.T) {
	ledger, err := New(Config{SessionID: "session-a", ResumeToken: "resume-a", MaxBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Complete(); !errors.Is(err, ErrIncompleteCoverage) {
		t.Fatalf("Complete error = %v; want ErrIncompleteCoverage", err)
	}
	if got := ledger.Snapshot().TerminalReason; got != TerminalIncompleteCoverage {
		t.Fatalf("terminal reason = %q; want %q", got, TerminalIncompleteCoverage)
	}
}

func TestLedgerRejectsConflictingDuplicateAndResourceExhaustion(t *testing.T) {
	ledger, err := New(Config{SessionID: "session-a", ResumeToken: "resume-a", MaxBytes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{2, 1}}); !errors.Is(err, ErrChunkConflict) {
		t.Fatalf("conflicting duplicate error = %v; want ErrChunkConflict", err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 1, StartSample: 2, EndSample: 4, Audio: []byte{3}}); !errors.Is(err, ErrResourceExhausted) {
		t.Fatalf("resource exhausted error = %v; want ErrResourceExhausted", err)
	}
	if got := ledger.Snapshot().TerminalReason; got != TerminalResourceExhausted {
		t.Fatalf("terminal reason = %q; want %q", got, TerminalResourceExhausted)
	}
}

func TestRegistryReopensOnlyWithTheOriginalResumeToken(t *testing.T) {
	registry := NewRegistry(8)
	first, resumed, err := registry.Open("session-a", "token-a")
	if err != nil || resumed {
		t.Fatalf("first open = %v, %t, %v", first, resumed, err)
	}
	second, resumed, err := registry.Open("session-a", "token-a")
	if err != nil || !resumed || second != first {
		t.Fatalf("resume = %v, %t, %v", second, resumed, err)
	}
	if _, _, err := registry.Open("session-a", "wrong-token"); !errors.Is(err, ErrInvalidResumeToken) {
		t.Fatalf("wrong resume token error = %v; want ErrInvalidResumeToken", err)
	}
}

func TestLedgerFailureRetainsReplayCoverage(t *testing.T) {
	ledger, err := New(Config{SessionID: "session-a", ResumeToken: "token-a", MaxBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	ledger.Fail("backend_restart")
	state := ledger.Snapshot()
	if state.TerminalReason != "backend_restart" || len(state.Replay) != 1 {
		t.Fatalf("failure must retain replay data: %#v", state)
	}
}

func TestDiskRegistryRestoresReplayAfterProcessRestart(t *testing.T) {
	dir := t.TempDir()
	first, err := NewDiskRegistry(dir, 8)
	if err != nil {
		t.Fatal(err)
	}
	ledger, _, err := first.Open("session-a", "token-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := first.Persist(ledger); err != nil {
		t.Fatal(err)
	}
	second, err := NewDiskRegistry(dir, 8)
	if err != nil {
		t.Fatal(err)
	}
	recovered, resumed, err := second.Open("session-a", "token-a")
	if err != nil || !resumed {
		t.Fatalf("reopen = %v, %t, %v", recovered, resumed, err)
	}
	if state := recovered.Snapshot(); len(state.Replay) != 1 || state.Replay[0].Sequence != 0 {
		t.Fatalf("missing restart replay coverage: %#v", state)
	}
}

func TestRoutedDiskRegistryWritesIntoTheLeasedDataRoot(t *testing.T) {
	primary := t.TempDir()
	leased := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: leased}, "lease-a", 0); err != nil {
		t.Fatal(err)
	}
	registry, err := NewRoutedDiskRegistry(roots, 8)
	if err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	ledger, _, err := registry.OpenContext(ctx, "session-a", "token-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.PersistContext(ctx, ledger); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(leased, "stt-session-spool", "session-a.json")); err != nil {
		t.Fatalf("leased ledger missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(primary, "stt-session-spool", "session-a.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("primary ledger should not exist, err=%v", err)
	}
	if roots.LeaseStats().TestRootWrites == 0 {
		t.Fatal("expected leased file-write evidence")
	}
}
