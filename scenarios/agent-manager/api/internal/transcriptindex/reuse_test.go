package transcriptindex_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"

	"github.com/klauspost/compress/zstd"
)

const otherThread = "01a1fc20-7d31-7a40-b9c2-0f6e1d2c3b4a"

// rolloutFor is a session's rollout path in the orphaned web-console home.
func rolloutFor(r *rig, thread, stamp string) string {
	return filepath.Join(r.wcRoot, orphanSession, "sessions", "2026", "11", "20", "rollout-2026-11-20T"+stamp+"-"+thread+".jsonl")
}

// reuseInode leaves dead's device and inode holding the compressed bytes of
// live's rollout under live's compressed name, and live's plain rollout gone:
// what Codex compressing live leaves when the filesystem hands it the inode
// dead's deleted rollout had.
func reuseInode(t *testing.T, dead, live string) string {
	t.Helper()
	data, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	compressed := live + ".zst"
	if err := os.Rename(dead, compressed); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compressed, encoder.EncodeAll(data, nil), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(compressed, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(live); err != nil {
		t.Fatal(err)
	}
	return compressed
}

// A compressed rollout that got the inode of another session's deleted
// rollout is indexed as itself: the old checkpoint keeps its run and usage
// under a retired key, the compressed file joins its own session's run, and
// nothing is locked or counted twice. A checkpoint locked by this before
// reuse was recognised is restored.
func TestCompressedRolloutOnAReusedInodeIsIndexedAsItself(t *testing.T) {
	for _, alreadyLocked := range []bool{false, true} {
		r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
		first, second := newCodexRollout(lineageThread), newCodexRollout(otherThread)
		dead := rolloutFor(r, lineageThread, "22-20-52")
		live := rolloutFor(r, otherThread, "22-25-00")
		write(t, dead, append([]string{first.meta()}, first.turn(1000, 0, 10, "one")...)...)
		write(t, live, append([]string{second.meta()}, second.turn(4000, 0, 40, "four")...)...)
		r.tick()
		deadRow, liveRow := r.row(dead), r.row(live)
		if alreadyLocked {
			deadRow.Status, deadRow.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"replacement session identity changed"
			r.save(deadRow)
		}

		compressed := reuseInode(t, dead, live)
		old := r.now.Add(-2 * time.Hour) // compressed once idle
		if err := os.Chtimes(compressed, old, old); err != nil {
			t.Fatal(err)
		}
		r.settle(time.Minute)

		reused := r.row(compressed)
		if reused.Key != deadRow.Key || reused.RunID != liveRow.RunID || reused.SessionID != otherThread || reused.Status != transcriptindex.StatusIndexed || reused.Offset != reused.Size {
			t.Fatalf("locked=%v: compressed rollout was not indexed as its own session: %+v", alreadyLocked, reused)
		}
		var retired *transcriptindex.FileRow
		rows, err := r.repos.TranscriptIndex.ListTranscriptFiles(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for i := range rows {
			if rows[i].Status == transcriptindex.StatusUnreadable {
				t.Fatalf("locked=%v: a checkpoint was locked: %+v", alreadyLocked, rows[i])
			}
			if strings.HasPrefix(rows[i].Key, deadRow.Key+"~") {
				retired = &rows[i]
			}
		}
		if retired == nil || retired.RunID != deadRow.RunID || retired.Path != dead || retired.Status != transcriptindex.StatusIndexed || retired.Offset != deadRow.Offset {
			t.Fatalf("locked=%v: old checkpoint was not retired intact: %+v", alreadyLocked, retired)
		}
		if r.tokens(deadRow.RunID) != 1010 || r.tokens(liveRow.RunID) != 4040 {
			t.Fatalf("locked=%v: usage moved or doubled: dead=%d live=%d", alreadyLocked, r.tokens(deadRow.RunID), r.tokens(liveRow.RunID))
		}
		if run := r.run(liveRow.RunID); run.Status != domain.RunStatusComplete {
			t.Fatalf("locked=%v: live session's run did not finish: %s", alreadyLocked, run.Status)
		}
	}
}

// A rollout moved to archived_sessions that a newer parser reimports keeps
// its run: the run's transcript path follows the move instead of refusing the
// promotion as a changed source.
func TestArchivedRolloutReimportFollowsTheMove(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	rollout := newCodexRollout(lineageThread)
	live := webConsoleRollout(r, orphanSession, "sessions")
	write(t, live, append([]string{rollout.meta()}, rollout.turn(1000, 0, 10, "one")...)...)
	r.tick()
	before := r.row(live)
	before.ParserVersion = 0 // indexed by an older parser
	r.save(before)
	archived := filepath.Join(r.wcRoot, orphanSession, "archived_sessions", rolloutName())
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(live, archived); err != nil {
		t.Fatal(err)
	}

	r.settle(time.Minute)
	row := r.row(archived)
	if row.Status != transcriptindex.StatusIndexed || row.Error != "" || row.RunID != before.RunID || row.ParserVersion == 0 {
		t.Fatalf("archived rollout was not reimported in place: %+v", row)
	}
	if run := r.run(before.RunID); run.TranscriptPath != archived || r.tokens(before.RunID) != 1010 {
		t.Fatalf("run did not follow the move: path=%s tokens=%d", run.TranscriptPath, r.tokens(before.RunID))
	}
}
