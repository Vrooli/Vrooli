package transcriptindex_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"agent-manager/internal/transcriptindex"
)

// The rule that settles an imported run whose source is gone.

const (
	observedEnd = "2026-10-28T10:01:00Z" // the last tokenCount line of activeImport's fixture
	pastGrace   = 25 * time.Hour         // beyond the default 24h source-gone grace
)

// activeImport indexes one Codex rollout and returns its path and active row.
func activeImport(t *testing.T, r *rig) (string, transcriptindex.FileRow) {
	t.Helper()
	path := codexPath(r, "rollout-2026-10-28T10-00-00-"+parentID+".jsonl")
	write(t, path, sessionMeta(parentID), turnContext(), tokenCount(1000, 0, 100, 1000, 0, 100, 1))
	r.tick()
	row := r.row(path)
	if !row.Active || r.run(row.RunID).Status != domain.RunStatusRunning {
		t.Fatalf("fixture must begin active and running: %+v", row)
	}
	return path, row
}

func (r *rig) save(row transcriptindex.FileRow) {
	r.t.Helper()
	if err := r.repos.TranscriptIndex.SaveTranscriptFile(context.Background(), row, nil); err != nil {
		r.t.Fatal(err)
	}
}

// settle advances the rig clock and runs a tick that may report failures (a
// locked or unreadable source is reported every tick by design).
func (r *rig) settle(advance time.Duration) transcriptindex.Summary {
	r.t.Helper()
	r.now = r.now.Add(advance)
	summary, err := r.index.RunOnce(context.Background())
	if err != nil {
		r.t.Fatal(err)
	}
	return summary
}

func TestVanishedSourceSettlesRunToUnknownAfterGrace(t *testing.T) {
	for _, tc := range []struct {
		name string
		lock func(*rig, transcriptindex.FileRow) // the row's state when its source vanishes
	}{
		{"indexed", func(*rig, transcriptindex.FileRow) {}},
		{"replacement awaiting reconciliation", func(r *rig, row transcriptindex.FileRow) {
			row.Status, row.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"replacement source changed before snapshot"
			r.save(row)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
			path, row := activeImport(t, r)
			tc.lock(r, row)
			locked := r.row(path)
			tokens := r.tokens(row.RunID)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}

			if summary := r.settle(time.Hour); summary.Unknown != 0 || r.run(row.RunID).Status != domain.RunStatusRunning {
				t.Fatalf("source gone within the grace settled: summary=%+v status=%s", summary, r.run(row.RunID).Status)
			}

			summary := r.settle(pastGrace)
			run := r.run(row.RunID)
			want, _ := time.Parse(time.RFC3339, observedEnd)
			if summary.Unknown != 1 || summary.Completed != 0 || run.Status != domain.RunStatusUnknown || run.EndedAt == nil || !run.EndedAt.Equal(want) {
				t.Fatalf("vanished source: summary=%+v status=%s ended=%v", summary, run.Status, run.EndedAt)
			}
			after := r.row(path)
			if after.Active || after.Status != locked.Status || after.Error != locked.Error || after.Offset != locked.Offset ||
				after.UsageReports != locked.UsageReports || after.RunID != locked.RunID || r.tokens(row.RunID) != tokens {
				t.Fatalf("settling lost the row's evidence: before=%+v after=%+v", locked, after)
			}

			if summary := r.settle(time.Hour); summary.Unknown != 0 || r.run(row.RunID).Status != domain.RunStatusUnknown {
				t.Fatalf("settled run was settled again or changed: summary=%+v status=%s", summary, r.run(row.RunID).Status)
			}
		})
	}
}

func TestSourceGoneRuleLeavesUnprovenRunsRunning(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*testing.T, *rig, string, transcriptindex.FileRow)
	}{
		{"locked source still present", func(_ *testing.T, r *rig, _ string, row transcriptindex.FileRow) {
			row.Status, row.Error = transcriptindex.StatusUnreadable, transcriptindex.ReplacementReconciliationRequired+"replacement session identity changed"
			r.save(row)
		}},
		{"uncertain recording with changed source", func(t *testing.T, r *rig, path string, row transcriptindex.FileRow) {
			row.Status, row.Error = transcriptindex.StatusUnreadable, "event recording outcome uncertain; cursor preserved: transcript source changed; manual reconciliation required"
			r.save(row)
			write(t, path, sessionMeta(childID), turnContext(), tokenCount(3000, 0, 300, 2000, 0, 200, 3)) // rewritten, not appended
		}},
		{"uncertain recording with vanished source", func(t *testing.T, r *rig, path string, row transcriptindex.FileRow) {
			row.Status, row.Error = transcriptindex.StatusUnreadable, "event recording outcome uncertain; cursor preserved: transcript source changed; manual reconciliation required"
			r.save(row)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		}},
		{"source unreadable by permission", func(t *testing.T, _ *rig, path string, _ transcriptindex.FileRow) {
			if os.Geteuid() == 0 {
				t.Skip("root bypasses directory permissions")
			}
			dir := filepath.Dir(path)
			if err := os.Chmod(dir, 0); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			if _, err := os.Lstat(path); err == nil || os.IsNotExist(err) {
				t.Fatalf("fixture must make the source unverifiable, got %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
			path, row := activeImport(t, r)
			tc.prepare(t, r, path, row)
			summary := r.settle(pastGrace)
			if run := r.run(row.RunID); summary.Unknown != 0 || run.Status != domain.RunStatusRunning {
				t.Fatalf("unproven source settled the run: summary=%+v status=%s", summary, run.Status)
			}
			if after := r.row(path); !after.Active {
				t.Fatalf("unproven source deactivated the row: %+v", after)
			}
		})
	}
}

// orphanImport adopts a Claude project file through a store whose first
// checkpoint fails, as a restart or contention between run creation and the
// checkpoint did: the file's run exists and no transcript row names it.
func orphanImport(t *testing.T, r *rig) string {
	t.Helper()
	path := filepath.Join(r.home, ".claude", "projects", "-proj", "6f1c2a1e-1d55-4a7e-9a34-1c0f6d3c2b10.jsonl")
	write(t, path, claudeAssistant("6f1c2a1e-1d55-4a7e-9a34-1c0f6d3c2b10", "msg_1", 100, 10, false))
	store := &failingIntentStore{&failingCheckpointStore{Store: r.repos.TranscriptIndex}}
	index := orchestration.NewTranscriptIndex(r.svc, store, transcriptindex.Config{Roots: r.roots(), RecallMaxBytes: r.recall, Now: func() time.Time { return r.now }})
	if summary, err := index.RunOnce(context.Background()); err != nil || len(summary.Failures) == 0 {
		t.Fatalf("expected the file's checkpoint to fail: %+v %v", summary, err)
	}
	if _, ok := r.files()[path]; ok {
		t.Fatal("fixture must leave the file without a transcript row")
	}
	if len(r.runsAt(path)) != 1 {
		t.Fatalf("fixture must leave exactly one unnamed run for the file: %d", len(r.runsAt(path)))
	}
	return path
}

func (r *rig) runsAt(path string) []*domain.Run {
	r.t.Helper()
	runs, err := r.repos.Runs.List(context.Background(), repository.RunListFilter{TagPrefix: "agent-manager-imported"})
	if err != nil {
		r.t.Fatal(err)
	}
	var out []*domain.Run
	for _, run := range runs {
		if run.TranscriptPath == path {
			out = append(out, run)
		}
	}
	return out
}

func TestUnlinkedImportSettlesToUnknownOnceItsSourceIsGone(t *testing.T) {
	r := newRig(t, transcriptindex.DefaultRecallMaxBytes)
	_, first := activeImport(t, r)
	orphaned := orphanImport(t, r)
	orphan := r.runsAt(orphaned)[0]
	if err := os.Remove(orphaned); err != nil {
		t.Fatal(err)
	}

	if summary := r.settle(time.Hour); summary.Unknown != 0 || r.run(orphan.ID.String()).Status != domain.RunStatusRunning {
		t.Fatalf("unlinked run settled within the grace: %+v %s", summary, r.run(orphan.ID.String()).Status)
	}
	summary := r.settle(pastGrace)
	if run := r.run(orphan.ID.String()); summary.Unknown != 1 || run.Status != domain.RunStatusUnknown {
		t.Fatalf("unlinked run with a vanished source: summary=%+v status=%s", summary, run.Status)
	}
	if run := r.run(first.RunID); run.Status != domain.RunStatusComplete {
		t.Fatalf("the linked run must complete normally, got %s", run.Status)
	}
	if failures := strings.Join(summary.Failures, "; "); failures != "" {
		t.Fatalf("settling reported failures: %s", failures)
	}
}
