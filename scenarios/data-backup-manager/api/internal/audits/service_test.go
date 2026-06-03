package audits_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/audits"
	auditsmocks "data-backup-manager/internal/audits/mocks"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/sources"
	sourcesmocks "data-backup-manager/internal/sources/mocks"
	"data-backup-manager/internal/testutil/mocks"
)

const (
	auditTargetID = "tgt-fs"
	auditDestID   = "dst-1"
	auditSnapID   = "snap-1"
)

// writeTree writes a map of relative path -> content under root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

// auditHarness wires the audit service against fakes plus real scratch dirs.
type auditHarness struct {
	svc       audits.Service
	repo      *auditsmocks.InMemoryRepository
	engine    *mocks.FakeKopiaEngine
	capturer  *sourcesmocks.FakeCapturer
	scratch   string
	snapFiles map[string]string
	liveFiles map[string]string
	snapTime  string // RFC3339; empty leaves snapshot time unknown
	liveMtime time.Time
}

func newAuditHarness(t *testing.T) *auditHarness {
	t.Helper()
	h := &auditHarness{
		repo:    auditsmocks.NewInMemoryRepository(),
		scratch: t.TempDir(),
	}
	h.capturer = &sourcesmocks.FakeCapturer{
		SourceKind: sources.KindFilesystem,
		CaptureFn: func(_ context.Context, spec sources.CaptureSpec) (sources.Artifact, error) {
			art := filepath.Join(spec.StageDir, "artifact")
			writeTree(t, art, h.liveFiles)
			if !h.liveMtime.IsZero() {
				_ = filepath.Walk(art, func(p string, _ os.FileInfo, _ error) error {
					_ = os.Chtimes(p, h.liveMtime, h.liveMtime)
					return nil
				})
			}
			return sources.Artifact{Path: art}, nil
		},
	}
	h.engine = &mocks.FakeKopiaEngine{
		SnapshotRestoreFn: func(_ context.Context, _, _, target string) error {
			writeTree(t, target, h.snapFiles)
			return nil
		},
		SnapshotListFn: func(_ context.Context, _, _ string) ([]engine.Snapshot, error) {
			return []engine.Snapshot{{ID: auditSnapID, StartTime: h.snapTime}}, nil
		},
	}
	h.svc = audits.NewService(audits.Deps{
		Repo: h.repo,
		Targets: &auditsmocks.FakeTargetLookup{Targets: map[string]audits.TargetForAudit{
			auditTargetID: {ID: auditTargetID, Kind: sources.KindFilesystem, Locator: "/live/path"},
		}},
		Destinations: &auditsmocks.FakeDestinationLookup{Destinations: map[string]audits.DestinationForAudit{
			auditDestID: {ID: auditDestID, Name: "nightly"},
		}},
		Engine:      h.engine,
		Sources:     sources.NewRegistry(h.capturer),
		Clock:       mocks.NewFakeClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)),
		ScratchRoot: h.scratch,
		Executor:    auditsmocks.NewSyncExecutor(),
	})
	return h
}

func (h *auditHarness) run(t *testing.T, content, sqliteChecks bool) audits.Audit {
	t.Helper()
	rec, err := h.svc.RunSnapshotAudit(context.Background(), auditTargetID, auditDestID, auditSnapID, content, sqliteChecks)
	if err != nil {
		t.Fatalf("RunSnapshotAudit: %v", err)
	}
	// SyncExecutor runs inline, so GetAudit already reflects the terminal state.
	got, err := h.svc.GetAudit(context.Background(), rec.ID)
	if err != nil {
		t.Fatalf("GetAudit: %v", err)
	}
	return got
}

func TestRunAudit_ExactMatch(t *testing.T) {
	h := newAuditHarness(t)
	tree := map[string]string{"a.txt": "alpha", "sub/b.txt": "bravo"}
	h.snapFiles = tree
	h.liveFiles = tree

	got := h.run(t, true, true)
	if got.Status != audits.AuditCompleted {
		t.Fatalf("status = %q, want completed (err=%q)", got.Status, got.Error)
	}
	if !got.Restorable {
		t.Errorf("expected restorable=true")
	}
	if got.Comparison == nil || !got.Comparison.Matches {
		t.Errorf("expected comparison.matches=true, got %+v", got.Comparison)
	}
	if got.Live == nil || got.Snapshot == nil {
		t.Fatalf("expected both inventories populated")
	}
	if got.Live.Files != 2 || got.Snapshot.Files != 2 {
		t.Errorf("file counts: live=%d snapshot=%d, want 2/2", got.Live.Files, got.Snapshot.Files)
	}
}

func TestRunAudit_ContentMismatch(t *testing.T) {
	h := newAuditHarness(t)
	h.snapFiles = map[string]string{"a.txt": "alpha"}
	h.liveFiles = map[string]string{"a.txt": "alpha", "extra.txt": "new file"}

	got := h.run(t, true, true)
	if got.Status != audits.AuditCompleted {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if got.Comparison == nil || got.Comparison.Matches {
		t.Fatalf("expected matches=false")
	}
	if len(got.Comparison.Mismatches) == 0 {
		t.Errorf("expected mismatch details")
	}
}

func TestRunAudit_LiveNewerSetsDrift(t *testing.T) {
	h := newAuditHarness(t)
	h.snapFiles = map[string]string{"a.txt": "alpha"}
	h.liveFiles = map[string]string{"a.txt": "alpha", "extra.txt": "drifted"}
	h.snapTime = "2026-01-01T00:00:00Z"
	h.liveMtime = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) // after snapshot

	got := h.run(t, true, true)
	if got.Comparison == nil || got.Comparison.Matches {
		t.Fatalf("expected mismatch")
	}
	if !got.Comparison.LiveNewerThanSnapshot {
		t.Errorf("expected drift flag set (live mtime after snapshot time)")
	}
	if got.SnapshotTime.IsZero() {
		t.Errorf("expected snapshot_time resolved from engine snapshot list")
	}
}

func TestRunAudit_RestoreFailurePersistsFailed(t *testing.T) {
	h := newAuditHarness(t)
	h.engine.SnapshotRestoreFn = func(_ context.Context, _, _, _ string) error {
		return errors.New("repo offline")
	}
	h.liveFiles = map[string]string{"a.txt": "alpha"}

	got := h.run(t, true, true)
	if got.Status != audits.AuditFailed {
		t.Fatalf("status = %q, want failed", got.Status)
	}
	if got.Restorable {
		t.Errorf("expected restorable=false on restore failure")
	}
	if !strings.Contains(got.Error, "snapshot restore") {
		t.Errorf("error = %q, want snapshot restore failure", got.Error)
	}
}

func TestRunAudit_ScratchCleanedUp(t *testing.T) {
	h := newAuditHarness(t)
	h.snapFiles = map[string]string{"a.txt": "alpha"}
	h.liveFiles = map[string]string{"a.txt": "alpha"}

	h.run(t, true, true)

	entries, err := os.ReadDir(h.scratch)
	if err != nil {
		t.Fatalf("read scratch: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "dbm-audit-") {
			t.Errorf("scratch dir not cleaned up: %s", e.Name())
		}
	}
}

func TestRunAudit_UnknownTargetFailsSynchronously(t *testing.T) {
	h := newAuditHarness(t)
	rec, err := h.svc.RunSnapshotAudit(context.Background(), "does-not-exist", auditDestID, auditSnapID, true, true)
	if err != nil {
		t.Fatalf("expected a recorded failed audit, not a transport error: %v", err)
	}
	if rec.Status != audits.AuditFailed {
		t.Errorf("status = %q, want failed", rec.Status)
	}
	if !strings.Contains(rec.Error, "resolve target") {
		t.Errorf("error = %q, want resolve target failure", rec.Error)
	}
}

func TestRunAudit_MissingFieldsRejected(t *testing.T) {
	h := newAuditHarness(t)
	cases := []struct{ target, dest, snap string }{
		{"", auditDestID, auditSnapID},
		{auditTargetID, "", auditSnapID},
		{auditTargetID, auditDestID, ""},
	}
	for _, c := range cases {
		if _, err := h.svc.RunSnapshotAudit(context.Background(), c.target, c.dest, c.snap, true, true); err == nil {
			t.Errorf("expected validation error for %+v", c)
		}
	}
}

func TestReconcile_FailsOrphanedAudits(t *testing.T) {
	h := newAuditHarness(t)
	// Seed a non-terminal audit directly into the repo (simulating a crash).
	orphan, _ := h.repo.CreateAudit(context.Background(), audits.Audit{
		TargetID: auditTargetID, DestinationID: auditDestID, SnapshotID: auditSnapID,
		Status: audits.AuditRunning,
	})
	if err := h.svc.Reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, _ := h.svc.GetAudit(context.Background(), orphan.ID)
	if got.Status != audits.AuditFailed {
		t.Errorf("orphan status = %q, want failed", got.Status)
	}
	if !strings.Contains(got.Error, "reconciled") {
		t.Errorf("expected reconciliation reason, got %q", got.Error)
	}
}
