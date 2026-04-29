package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// TestDriverContract is the parameterized lifecycle test that every driver
// must pass: Mount → write → GetChangedFiles (Added) → RemoveFromUpper →
// GetChangedFiles (empty) → Unmount → CleanupOrphan → second
// CleanupOrphan is a no-op.
//
// Drivers that implement MountVerifier additionally have their
// VerifyMountIntegrity exercised: pre-Unmount returns nil, post-Unmount
// returns an error.
//
// Drivers whose IsAvailable returns false on the test host are skipped.
// CI runs this with and without `unshare -Umr` to cover both code paths.
func TestDriverContract(t *testing.T) {
	tmpDir := t.TempDir()

	// Source project: minimal lower layer.
	scope := filepath.Join(tmpDir, "scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scope, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("seed scope: %v", err)
	}

	cases := []struct {
		name string
		ctor func(Config) Driver
	}{
		{"copy", func(cfg Config) Driver { return NewCopyDriver(cfg) }},
		{"fuse-overlayfs", func(cfg Config) Driver { return NewFuseOverlayfsDriver(cfg) }},
		{"overlayfs", func(cfg Config) Driver { return NewOverlayfsDriver(cfg) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseDir := filepath.Join(tmpDir, "drv-"+tc.name)
			cfg := Config{BaseDir: baseDir}
			drv := tc.ctor(cfg)

			ctx := context.Background()
			available, err := drv.IsAvailable(ctx)
			if err != nil || !available {
				t.Skipf("driver %q not available on this host (err=%v): skipping", tc.name, err)
			}

			id := uuid.New()
			sb := &types.Sandbox{
				ID:          id,
				ScopePath:   scope,
				ProjectRoot: scope,
			}

			paths, err := drv.Mount(ctx, sb)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			sb.LowerDir = paths.LowerDir
			sb.UpperDir = paths.UpperDir
			sb.WorkDir = paths.WorkDir
			sb.MergedDir = paths.MergedDir
			sb.HomeMergedDir = paths.HomeMergedDir

			// MountVerifier (when supported) should be happy pre-Unmount.
			if v, ok := drv.(MountVerifier); ok {
				if err := v.VerifyMountIntegrity(ctx, sb); err != nil {
					t.Errorf("VerifyMountIntegrity pre-Unmount: %v", err)
				}
			}

			// Write a file in the merged view; expect it as Added.
			added := filepath.Join(sb.MergedDir, "added.txt")
			if err := os.WriteFile(added, []byte("new"), 0o644); err != nil {
				t.Fatalf("write added: %v", err)
			}

			changes, err := drv.GetChangedFiles(ctx, sb)
			if err != nil {
				t.Fatalf("GetChangedFiles: %v", err)
			}
			sawAdded := false
			for _, c := range changes {
				if c.FilePath == "added.txt" && c.ChangeType == types.ChangeTypeAdded {
					sawAdded = true
				}
			}
			if !sawAdded {
				t.Errorf("expected added.txt as Added, got %v", changes)
			}

			// RemoveFromUpper drops the change; subsequent GetChangedFiles
			// should not list it.
			if err := drv.RemoveFromUpper(ctx, sb, "added.txt"); err != nil {
				t.Fatalf("RemoveFromUpper: %v", err)
			}
			changes, err = drv.GetChangedFiles(ctx, sb)
			if err != nil {
				t.Fatalf("GetChangedFiles after remove: %v", err)
			}
			for _, c := range changes {
				if c.FilePath == "added.txt" {
					t.Errorf("added.txt still present after RemoveFromUpper: %v", c)
				}
			}

			if err := drv.Unmount(ctx, sb); err != nil {
				t.Errorf("Unmount: %v", err)
			}

			// CleanupOrphan must be idempotent.
			if err := drv.CleanupOrphan(ctx, id); err != nil {
				t.Errorf("CleanupOrphan first call: %v", err)
			}
			if err := drv.CleanupOrphan(ctx, id); err != nil {
				t.Errorf("CleanupOrphan second call (should be no-op): %v", err)
			}
		})
	}
}
