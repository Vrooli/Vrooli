package transcriptindex_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
)

// A live rollout must stay reconcilable: a replacement that met a source still
// moving under it is retried, and a head that cannot be read is never taken
// for a rewrite.

// The state the 40a7e56e research rollout was left in (knw-1795243493817155033):
// locked awaiting reconciliation because it grew while it was staged. It is
// released and tailed to its end, and its run keeps updating.
func TestLiveRolloutLockedWhileGrowingIsReleasedAndTailed(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := webConsoleRollout(r, orphanSession, "sessions")
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	locked := r.row(path)
	locked.Status, locked.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"replacement source changed before snapshot"
	r.save(locked)
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)

	r.now = r.now.Add(time.Minute)
	r.tick()
	row := r.row(path)
	if row.Status != transcriptindex.StatusIndexed || row.Error != "" || row.Offset != row.Size || row.RunID != locked.RunID {
		t.Fatalf("live rollout stayed locked or lost its run: %+v", row)
	}
	if run := r.run(row.RunID); run.Status != domain.RunStatusRunning || r.tokens(row.RunID) != 1010+2020 {
		t.Fatalf("run stopped updating: status=%s tokens=%d", run.Status, r.tokens(row.RunID))
	}
}

// A refusal that needs an operator stays locked: only "the source moved"
// reasons are released.
func TestNonTransientReplacementRefusalStaysLocked(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := webConsoleRollout(r, orphanSession, "sessions")
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	locked := r.row(path)
	locked.Status, locked.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"replacement session identity changed"
	r.save(locked)
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)
	r.settle(time.Minute)
	if row := r.row(path); row.Status != transcriptindex.StatusUnreadable || row.Error != locked.Error || row.Offset != locked.Offset {
		t.Fatalf("a refusal needing reconciliation was released: %+v", row)
	}
}

// An unreadable head (permissions, I/O) defers the tick; it does not stage a
// replacement or lock the checkpoint.
func TestUnreadableHeadIsRetriedNotTakenForARewrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads files regardless of mode")
	}
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := webConsoleRollout(r, orphanSession, "sessions")
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	before := r.row(path)
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	summary := r.settle(time.Minute)
	if failures := strings.Join(summary.Failures, "; "); !strings.Contains(failures, "read transcript head") {
		t.Fatalf("unreadable head must be reported as a retry, got %q", failures)
	}
	if row := r.row(path); row.Status != before.Status || row.Error != "" || row.Offset != before.Offset {
		t.Fatalf("unreadable head changed the checkpoint: %+v", row)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	r.tick()
	if row := r.row(path); row.Offset != row.Size || r.tokens(row.RunID) != 1010+2020 {
		t.Fatalf("readable head did not resume tailing: %+v", row)
	}
}

// A capture locked because its source "changed" when the source in fact still
// matches its checkpoint (run 951a1963 since 2026-10-30) retries from its
// cursor; a source that really was rewritten stays locked.
func TestFalseSourceMismatchLockRetriesFromItsCursor(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	path := webConsoleRollout(r, orphanSession, "sessions")
	write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	locked := r.row(path)
	locked.Status, locked.Error = transcriptindex.StatusUnreadable, "event recording outcome uncertain; cursor preserved: transcript source changed; manual reconciliation required"
	r.save(locked)
	appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)

	r.tick()
	if row := r.row(path); row.Status != transcriptindex.StatusIndexed || row.Error != "" || row.Offset != row.Size || r.tokens(row.RunID) != 1010+2020 {
		t.Fatalf("falsely locked capture did not resume: %+v tokens=%d", row, r.tokens(row.RunID))
	}

	rewritten := r.row(path)
	rewritten.Status, rewritten.Error = transcriptindex.StatusUnreadable, locked.Error
	r.save(rewritten)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), "Codex", "Xedoc", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	r.settle(time.Minute)
	if row := r.row(path); row.Status != transcriptindex.StatusUnreadable || row.Offset != rewritten.Offset {
		t.Fatalf("a really rewritten source was retried: %+v", row)
	}
}

// A checkpoint an oversized parser-version reimport locked (45 rows on this
// host before 2026-11-21) is released while its source still extends it, and
// tailed; one whose source was rewritten stays locked.
func TestOversizedReimportLockIsReleasedOnlyWhileTheSourceExtendsIt(t *testing.T) {
	for _, rewrite := range []bool{false, true} {
		r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
		rollout := newCodexRollout(lineageThread)
		path := webConsoleRollout(r, orphanSession, "sessions")
		write(t, path, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
		r.tick()
		locked := r.row(path)
		locked.Status, locked.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"source exceeds replacement staging limit"
		r.save(locked)
		if rewrite {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data[strings.Index(string(data), "2026")] = '3' // the first record's timestamp, inside the head
			if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		} else {
			appendLines(t, path, rollout.turn(2000, 0, 20, "two")...)
		}
		r.settle(time.Minute)
		row := r.row(path)
		switch {
		case rewrite && (row.Status != transcriptindex.StatusUnreadable || row.Error != locked.Error):
			t.Fatalf("a rewritten oversized source was released: %+v", row)
		case !rewrite && (row.Status != transcriptindex.StatusIndexed || row.Offset != row.Size || r.tokens(row.RunID) != 1010+2020):
			t.Fatalf("an extended source stayed locked: %+v tokens=%d", row, r.tokens(row.RunID))
		}
	}
}
