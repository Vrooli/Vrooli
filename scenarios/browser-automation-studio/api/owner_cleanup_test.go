package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
)

func TestParseCleanupMaxBytesAcceptsHumanAndRawValues(t *testing.T) {
	if got := parseCleanupMaxBytes("1GiB"); got != 1<<30 {
		t.Fatalf("1GiB parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("1073741824"); got != 1<<30 {
		t.Fatalf("raw bytes parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("not-a-size"); got != 0 {
		t.Fatalf("invalid size parsed as %d", got)
	}
}

func TestOwnerCleanupSweepUsesConservativeAgeForMissingOrZeroAge(t *testing.T) {
	for _, raw := range []string{"", "0", "-1", "not-a-number"} {
		if seconds := parseCleanupAge(raw); seconds != int64((7*24*time.Hour)/time.Second) {
			t.Fatalf("raw age %q: seconds = %d, want conservative default", raw, seconds)
		}
	}
}

func TestCaptureCandidatesSelectsExpiredBundlesAndHonorsByteCap(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old-bundle")
	newer := filepath.Join(root, "newer-bundle")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newer, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "capture.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newer, "capture.png"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().Add(-48 * time.Hour)
	newAt := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, oldAt, oldAt); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, newAt, newAt); err != nil {
		t.Fatal(err)
	}

	service := &ownerCleanupService{capturesRoot: root}
	items, err := service.captureCandidates(context.Background(), 24*60*60, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "capture:old-bundle" || items[0].Bytes != 3 {
		t.Fatalf("items = %#v, want only the expired old bundle", items)
	}
	if err := removeCapture(items[0].Path, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old capture still exists: %v", err)
	}
	if _, err := os.Stat(newer); err != nil {
		t.Fatalf("new capture was unexpectedly removed: %v", err)
	}
}

func TestCaptureCandidatesBoundsRecursiveSizingToOldestBatch(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < ownerCleanupBatchCap+1; i++ {
		path := filepath.Join(root, fmt.Sprintf("capture-%03d", i))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "capture.bin"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-time.Duration(i+1) * time.Hour)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}

	service := &ownerCleanupService{capturesRoot: root}
	items, err := service.captureCandidates(context.Background(), 0, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != ownerCleanupBatchCap {
		t.Fatalf("items = %d, want bounded batch %d", len(items), ownerCleanupBatchCap)
	}
	if items[0].ID != "capture:capture-200" {
		t.Fatalf("oldest selected item = %q, want capture:capture-200", items[0].ID)
	}
}

func TestOrphanRecordingCandidatesProtectIndexedAndRecentDirectories(t *testing.T) {
	root := t.TempDir()
	orphanID := uuid.New()
	protectedID := uuid.New()
	recentID := uuid.New()
	oldAt := time.Now().Add(-48 * time.Hour)
	recentAt := time.Now().Add(-time.Hour)
	for _, item := range []struct {
		id string
		at time.Time
	}{
		{orphanID.String(), oldAt}, {protectedID.String(), oldAt}, {recentID.String(), recentAt},
	} {
		path := filepath.Join(root, item.id)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "recording.bin"), []byte("orphan"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, item.at, item.at); err != nil {
			t.Fatal(err)
		}
	}

	items, err := orphanRecordingCandidates(context.Background(), root, map[string]struct{}{protectedID.String(): {}}, 24*60*60, 0, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "recording:"+orphanID.String() || items[0].Bytes != int64(len("orphan")) {
		t.Fatalf("items = %#v, want only the expired unindexed recording", items)
	}
}

func TestRemoveCaptureRejectsPathsOutsideRoot(t *testing.T) {
	if err := removeCapture(filepath.Join(t.TempDir(), "outside"), filepath.Join(t.TempDir(), "captures")); err == nil {
		t.Fatal("removeCapture accepted a path outside the configured root")
	}
}

func TestOwnerCleanupRecoveryLockHonorsStorageRecoveryHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "storage-manager", "recovery.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	heldFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	heldRelease, err := platform.LockFile(heldFile, true)
	if err != nil {
		_ = heldFile.Close()
		t.Fatal(err)
	}
	defer func() {
		heldRelease()
		_ = heldFile.Close()
	}()

	service := &ownerCleanupService{recoveryLockPath: path}
	if _, err := service.acquireRecoveryLock(); err == nil {
		t.Fatal("owner cleanup acquired a lock held by storage recovery")
	}

	heldRelease()
	if err := heldFile.Close(); err != nil {
		t.Fatal(err)
	}
	release, err := service.acquireRecoveryLock()
	if err != nil {
		t.Fatalf("owner cleanup could not acquire released lock: %v", err)
	}
	release()
}
