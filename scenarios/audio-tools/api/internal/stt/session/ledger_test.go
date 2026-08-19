// [REQ:ATD-P0-001] Server-owned replay ledger preserves every captured range.
package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestAcceleratedServerRetentionBaseline(t *testing.T) {
	const (
		sampleRate          = 16_000
		bytesPerSample      = 2
		frameSeconds        = 60
		simulatedSeconds    = 60 * 60
		frameSamples        = sampleRate * frameSeconds
		frameBytes          = frameSamples * bytesPerSample
		maxRetainedBytes    = 64 * 1024 * 1024
		predictedCeilingSec = float64(maxRetainedBytes) / float64(sampleRate*bytesPerSample)
	)

	ledger := newLedger(t, maxRetainedBytes)
	framesReceived := 0
	failureSeconds := 0
	for sequence := 0; sequence < simulatedSeconds/frameSeconds; sequence++ {
		_, err := ledger.Receive(Chunk{
			Sequence:    uint64(sequence),
			StartSample: int64(sequence * frameSamples),
			EndSample:   int64((sequence + 1) * frameSamples),
			Audio:       make([]byte, frameBytes),
		})
		if errors.Is(err, ErrResourceExhausted) {
			failureSeconds = (sequence + 1) * frameSeconds
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		framesReceived++
	}
	if failureSeconds == 0 {
		t.Fatal("unchanged server ledger did not reach its retention ceiling")
	}
	if delta := float64(failureSeconds) - predictedCeilingSec; delta < -60 || delta > 60 {
		t.Fatalf("server retention ceiling = %ds, predicted %.3fs", failureSeconds, predictedCeilingSec)
	}
	if got := ledger.Snapshot().TerminalReason; got != TerminalResourceExhausted {
		t.Fatalf("terminal reason = %q, want %q", got, TerminalResourceExhausted)
	}
}

func TestAcceleratedServerCoverageAcknowledgementKeepsRetentionBounded(t *testing.T) {
	const (
		sampleRate       = 16_000
		frameSeconds     = 60
		simulatedSeconds = 60 * 60
		frameSamples     = sampleRate * frameSeconds
		frameBytes       = frameSamples * 2
		maxRetainedBytes = 64 * 1024 * 1024
	)
	ledger := newLedger(t, maxRetainedBytes)
	for sequence := 0; sequence < simulatedSeconds/frameSeconds; sequence++ {
		_, err := ledger.Receive(Chunk{
			Sequence: uint64(sequence), StartSample: int64(sequence * frameSamples),
			EndSample: int64((sequence + 1) * frameSamples), Audio: make([]byte, frameBytes),
		})
		if err != nil {
			t.Fatalf("coverage-driven receive %d: %v", sequence, err)
		}
		if err := ledger.AcknowledgeProcessed(uint64(sequence)); err != nil {
			t.Fatalf("coverage-driven acknowledgement %d: %v", sequence, err)
		}
		state := ledger.Snapshot()
		if len(state.Replay) != 0 || state.ReceivedSequence != int64(sequence) || state.ProcessedSequence != int64(sequence) {
			t.Fatalf("retention grew after coverage acknowledgement %d: %#v", sequence, state)
		}
	}
	state := ledger.Snapshot()
	if state.TerminalReason != TerminalNone {
		t.Fatalf("coverage lane reached terminal state: %q", state.TerminalReason)
	}
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

func TestDiskRegistryRemovesTerminalSessionAndReplaySpool(t *testing.T) {
	dir := t.TempDir()
	registry, err := NewDiskRegistry(dir, 8)
	if err != nil {
		t.Fatal(err)
	}
	ledger, _, err := registry.Open("session-a", "token-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Receive(Chunk{Sequence: 0, StartSample: 0, EndSample: 2, Audio: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	ledger.Fail(TerminalCompleted)
	if err := registry.Persist(ledger); err != nil {
		t.Fatal(err)
	}
	if err := registry.Remove("session-a"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".json", ".jsonl"} {
		if _, err := os.Stat(filepath.Join(dir, "session-a"+suffix)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("terminal spool %s still exists, err=%v", suffix, err)
		}
	}
	if _, resumed, err := registry.Open("session-a", "token-a"); err != nil || resumed {
		t.Fatalf("removed session reopened as resumed=%t err=%v", resumed, err)
	}
}

func TestRegistryExpiresAbandonedSessionAfterRecoveryWindow(t *testing.T) {
	registry := newRegistry(8, 10*time.Millisecond)
	if _, resumed, err := registry.Open("session-a", "token-a"); err != nil || resumed {
		t.Fatalf("initial open resumed=%t err=%v", resumed, err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, resumed, err := registry.Open("session-a", "token-a"); err != nil || resumed {
		t.Fatalf("expired session reopened as resumed=%t err=%v", resumed, err)
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
	newLease := t.TempDir()
	if err := roots.ClearTestRoots("lease-a"); err != nil {
		t.Fatal(err)
	}
	if err := roots.InstallTestRoots(storage.Paths{DataDir: newLease}, "lease-b", 0); err != nil {
		t.Fatal(err)
	}
	if err := registry.RemoveContext(ctx, "session-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(leased, "stt-session-spool", "session-a.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original leased ledger should be removed after lease change, err=%v", err)
	}
}
