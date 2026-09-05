// Driver-level tests for the orphan-reconciliation surface
// (ListSandboxDirs + CleanupOrphan) added in 2026-04-28.
//
// These tests run against a real filesystem (t.TempDir) but do NOT
// create real overlay mounts — that's covered by the existing per-
// driver mount tests. Here we verify the FS-walk + rm -rf shape and
// the idempotency guarantees the reconciler relies on.

package driver

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/uuid"
)

// listSandboxDirsInBase is the shared helper; test it once here. The
// per-driver ListSandboxDirs methods all delegate to this.
func TestListSandboxDirsInBase(t *testing.T) {
	t.Run("missing baseDir returns empty without error", func(t *testing.T) {
		got, err := listSandboxDirsInBase("/nonexistent/path/that/should/not/exist")
		if err != nil {
			t.Fatalf("unexpected error for missing baseDir: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %v", got)
		}
	})

	t.Run("empty baseDir returns empty", func(t *testing.T) {
		base := t.TempDir()
		got, err := listSandboxDirsInBase(base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty slice for empty dir, got %v", got)
		}
	})

	t.Run("UUID-named dirs returned, others ignored", func(t *testing.T) {
		base := t.TempDir()
		want := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		for _, id := range want {
			if err := os.MkdirAll(filepath.Join(base, id.String()), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}
		// Non-UUID artifacts that must NOT be returned.
		if err := os.MkdirAll(filepath.Join(base, "driver-preference"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "stray.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(base, "not-a-uuid"), 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := listSandboxDirsInBase(base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(want) {
			t.Fatalf("expected %d UUID dirs, got %d (%v)", len(want), len(got), got)
		}
		// Compare as sorted strings for stability.
		sortIDs := func(ids []uuid.UUID) []string {
			out := make([]string, len(ids))
			for i, id := range ids {
				out[i] = id.String()
			}
			sort.Strings(out)
			return out
		}
		gotS, wantS := sortIDs(got), sortIDs(want)
		for i := range gotS {
			if gotS[i] != wantS[i] {
				t.Errorf("index %d: got %s, want %s", i, gotS[i], wantS[i])
			}
		}
	})

	t.Run("blank baseDir returns empty without error", func(t *testing.T) {
		got, err := listSandboxDirsInBase("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

// CopyDriver is the only driver we can exercise on every CI runner
// (no fuse, no overlayfs needed). The orphan-cleanup wire shape is the
// same across drivers: list FS dirs, rm -rf the ones we're told are
// orphans. The fuse/overlay variants additionally try to unmount, but
// we intentionally don't drive a real mount in unit tests.
func TestCopyDriver_OrphanReconciliation(t *testing.T) {
	base := t.TempDir()
	cfg := Config{BaseDir: base}
	d := NewCopyDriver(cfg, testDeps())
	ctx := context.Background()

	a := uuid.New()
	b := uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := os.MkdirAll(filepath.Join(base, id.String(), "workspace"), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	t.Run("ListSandboxDirs surfaces both", func(t *testing.T) {
		ids, err := d.ListSandboxDirs(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 2 {
			t.Fatalf("expected 2 dirs, got %d", len(ids))
		}
	})

	t.Run("CleanupOrphan removes one", func(t *testing.T) {
		if err := d.CleanupOrphan(ctx, a); err != nil {
			t.Fatalf("cleanup: %v", err)
		}
		if _, err := os.Stat(filepath.Join(base, a.String())); !os.IsNotExist(err) {
			t.Errorf("expected dir to be removed, stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Join(base, b.String())); err != nil {
			t.Errorf("expected sibling dir to survive, got err=%v", err)
		}
	})

	t.Run("CleanupOrphan is idempotent on missing dir", func(t *testing.T) {
		if err := d.CleanupOrphan(ctx, a); err != nil {
			t.Errorf("expected idempotent no-op for missing dir, got %v", err)
		}
	})

	t.Run("ListSandboxDirs reflects post-cleanup state", func(t *testing.T) {
		ids, err := d.ListSandboxDirs(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 || ids[0] != b {
			t.Errorf("expected only %s left, got %v", b, ids)
		}
	})
}

// All OverlayDriver flavors share the same FS-walk +
// rm -rf path; the only delta is the unmount step (fusermount vs umount),
// which is itself a best-effort guarded by isMountPoint(). Since we
// don't create real mounts in unit tests, the unmount call short-
// circuits and we end up exercising the same rm -rf path as CopyDriver.
//
// We still run the smoke test on each so a typo in the per-driver
// implementation (wrong BaseDir wiring, missing dir creation, etc.)
// is caught.
func TestFuseOverlayfsDriver_OrphanReconciliation_Smoke(t *testing.T) {
	base := t.TempDir()
	cfg := Config{BaseDir: base}
	d := NewFuseOverlayfsDriver(cfg, testDeps())
	id := uuid.New()
	if err := os.MkdirAll(filepath.Join(base, id.String()), 0o755); err != nil {
		t.Fatal(err)
	}

	ids, err := d.ListSandboxDirs(context.Background())
	if err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("ListSandboxDirs: got %v, err=%v", ids, err)
	}
	if err := d.CleanupOrphan(context.Background(), id); err != nil {
		t.Fatalf("CleanupOrphan: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, id.String())); !os.IsNotExist(err) {
		t.Errorf("dir should be gone, stat err=%v", err)
	}
}

func TestOverlayfsDriver_OrphanReconciliation_Smoke(t *testing.T) {
	base := t.TempDir()
	cfg := Config{BaseDir: base}
	d := NewOverlayfsDriver(cfg, testDeps())
	id := uuid.New()
	if err := os.MkdirAll(filepath.Join(base, id.String()), 0o755); err != nil {
		t.Fatal(err)
	}

	ids, err := d.ListSandboxDirs(context.Background())
	if err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("ListSandboxDirs: got %v, err=%v", ids, err)
	}
	if err := d.CleanupOrphan(context.Background(), id); err != nil {
		t.Fatalf("CleanupOrphan: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, id.String())); !os.IsNotExist(err) {
		t.Errorf("dir should be gone, stat err=%v", err)
	}
}
