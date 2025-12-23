package driver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

func TestFuseOverlayfsGetChangedFilesSkipsOpaqueAndMapsWhiteouts(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fuse-overlayfs tests require Linux")
	}

	tmpDir := t.TempDir()
	lowerDir := filepath.Join(tmpDir, "lower")
	upperDir := filepath.Join(tmpDir, "upper")

	if err := os.MkdirAll(lowerDir, 0o755); err != nil {
		t.Fatalf("failed to create lower dir: %v", err)
	}
	if err := os.MkdirAll(upperDir, 0o755); err != nil {
		t.Fatalf("failed to create upper dir: %v", err)
	}

	lowerFile := filepath.Join(lowerDir, "dir", "removed.txt")
	if err := os.MkdirAll(filepath.Dir(lowerFile), 0o755); err != nil {
		t.Fatalf("failed to create lower subdir: %v", err)
	}
	if err := os.WriteFile(lowerFile, []byte("original"), 0o644); err != nil {
		t.Fatalf("failed to create lower file: %v", err)
	}

	whiteout := filepath.Join(upperDir, "dir", ".wh.removed.txt")
	if err := os.MkdirAll(filepath.Dir(whiteout), 0o755); err != nil {
		t.Fatalf("failed to create upper subdir: %v", err)
	}
	if err := os.WriteFile(whiteout, []byte("marker"), 0o644); err != nil {
		t.Fatalf("failed to create whiteout marker: %v", err)
	}

	opaque := filepath.Join(upperDir, "tmp", ".wh..opq")
	if err := os.MkdirAll(filepath.Dir(opaque), 0o755); err != nil {
		t.Fatalf("failed to create opaque dir: %v", err)
	}
	if err := os.WriteFile(opaque, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create opaque marker: %v", err)
	}

	recursiveOpaque := filepath.Join(upperDir, "tmp", ".wh..wh..opq")
	if err := os.WriteFile(recursiveOpaque, []byte("opaque"), 0o644); err != nil {
		t.Fatalf("failed to create recursive opaque marker: %v", err)
	}

	added := filepath.Join(upperDir, "added.txt")
	if err := os.WriteFile(added, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to create added file: %v", err)
	}

	drv := NewFuseOverlayfsDriver(DefaultConfig())
	sb := &types.Sandbox{
		ID:       uuid.New(),
		LowerDir: lowerDir,
		UpperDir: upperDir,
	}

	changes, err := drv.GetChangedFiles(context.Background(), sb)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	var sawDeleted bool
	var sawOpaque bool
	var sawAdded bool

	for _, change := range changes {
		if change.FilePath == "dir/removed.txt" && change.ChangeType == types.ChangeTypeDeleted {
			sawDeleted = true
		}
		if change.FilePath == "tmp/.wh..opq" || change.FilePath == ".wh..opq" {
			sawOpaque = true
		}
		if change.FilePath == "tmp/.wh..wh..opq" {
			sawOpaque = true
		}
		if change.FilePath == "added.txt" && change.ChangeType == types.ChangeTypeAdded {
			sawAdded = true
		}
	}

	if !sawDeleted {
		t.Error("expected whiteout marker to map to deleted file change")
	}
	if sawOpaque {
		t.Error("opaque marker should be ignored")
	}
	if !sawAdded {
		t.Error("expected added file change")
	}
}
