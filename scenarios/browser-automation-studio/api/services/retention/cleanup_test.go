package retention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	platform "github.com/vrooli/platform-go"
)

func TestParseCleanupMaxBytesAcceptsHumanAndRawValues(t *testing.T) {
	if got := ParseCleanupMaxBytes("1GiB"); got != 1<<30 {
		t.Fatalf("1GiB parsed as %d", got)
	}
	if got := ParseCleanupMaxBytes("1073741824"); got != 1<<30 {
		t.Fatalf("raw bytes parsed as %d", got)
	}
	if got := ParseCleanupMaxBytes("not-a-size"); got != 0 {
		t.Fatalf("invalid size parsed as %d", got)
	}
}

func TestOwnerCleanupUsesConservativeAgeForMissingOrZeroAge(t *testing.T) {
	for _, raw := range []string{"", "0", "-1", "not-a-number"} {
		if seconds := ParseCleanupAge(raw); seconds != int64((7*24*time.Hour)/time.Second) {
			t.Fatalf("raw age %q: seconds = %d, want conservative default", raw, seconds)
		}
	}
}

func TestCleanupSweepSelectsExpiredCaptureAndHonorsByteCap(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	old := filepath.Join(capturesRoot, "old-bundle")
	newer := filepath.Join(capturesRoot, "newer-bundle")
	for _, path := range []string{old, newer} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "capture.png"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	oldAt := time.Now().Add(-48 * time.Hour)
	newAt := time.Now().Add(-time.Hour)
	for path, at := range map[string]time.Time{old: oldAt, newer: newAt} {
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}

	service := NewService(&evidenceRepository{states: map[uuid.UUID]string{}}, OSFileSystem{}, recordingsRoot, nil).WithCaptureRoot(capturesRoot)
	_, items, _, _, _, err := service.CleanupSweep(context.Background(), false, nil, nil, nil, nil, 24*60*60, 0, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "capture:old-bundle" || items[0].Bytes != 3 {
		t.Fatalf("items = %#v, want only the expired old bundle", items)
	}
}

func TestCleanupSweepBoundsCaptureSizingAndProtectsNewestWithinByteCap(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	for i, item := range []struct {
		name  string
		bytes string
	}{{"old", "xxx"}, {"middle", "yyyy"}, {"new", "zzzzz"}} {
		path := filepath.Join(capturesRoot, item.name)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "capture.bin"), []byte(item.bytes), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-time.Duration(3-i) * time.Hour)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(&evidenceRepository{states: map[uuid.UUID]string{}}, OSFileSystem{}, recordingsRoot, nil).WithCaptureRoot(capturesRoot)
	_, items, _, _, _, err := service.CleanupSweep(context.Background(), false, nil, nil, nil, nil, 0, 1, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "capture:old" || items[0].Bytes != 3 {
		t.Fatalf("items = %#v, want oldest item within byte cap and newest protected", items)
	}
}

func TestCleanupSweepBoundsUnrequestedCaptureWork(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	for i := 0; i < ownerCleanupBatchCap+1; i++ {
		path := filepath.Join(capturesRoot, fmt.Sprintf("capture-%03d", i))
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
	service := NewService(&evidenceRepository{states: map[uuid.UUID]string{}}, OSFileSystem{}, recordingsRoot, nil).WithCaptureRoot(capturesRoot)
	_, items, _, _, _, err := service.CleanupSweep(context.Background(), false, nil, nil, nil, nil, 0, 0, 0, false)
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

func TestCleanupSweepProtectsIndexedAndRecentRecordingDirectories(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	orphanID, protectedID, recentID := uuid.New(), uuid.New(), uuid.New()
	oldAt := time.Now().Add(-48 * time.Hour)
	recentAt := time.Now().Add(-time.Hour)
	for _, item := range []struct {
		id string
		at time.Time
	}{{orphanID.String(), oldAt}, {protectedID.String(), oldAt}, {recentID.String(), recentAt}} {
		path := filepath.Join(recordingsRoot, item.id)
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
	repo := &evidenceRepository{states: map[uuid.UUID]string{protectedID: "completed"}}
	service := NewService(repo, OSFileSystem{}, recordingsRoot, nil).WithCaptureRoot(capturesRoot)
	_, items, _, _, _, err := service.CleanupSweep(context.Background(), false, nil, nil, nil, nil, 24*60*60, 0, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "recording:"+orphanID.String() {
		t.Fatalf("items = %#v, want only expired unindexed recording", items)
	}
}

func TestOSFileSystemRejectsCaptureOutsideConfiguredRoot(t *testing.T) {
	if err := (OSFileSystem{}).DeleteContained(context.Background(), filepath.Join(t.TempDir(), "captures"), filepath.Join(t.TempDir(), "outside")); err == nil {
		t.Fatal("contained deletion accepted a path outside the configured root")
	}
}

func TestApplyCleanupHonorsStorageRecoveryLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "state", "storage-manager", "recovery.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	heldFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	heldRelease, err := platform.LockFile(heldFile, true)
	if err != nil {
		_ = heldFile.Close()
		t.Fatal(err)
	}
	service := NewService(&evidenceRepository{states: map[uuid.UUID]string{}}, OSFileSystem{}, t.TempDir(), nil).WithRecoveryLockPath(lockPath)
	_, err = service.ApplyCleanup(context.Background(), CleanupPreview{}, "first-attempt", false, false)
	if !errors.Is(err, ErrRecoveryLockHeld) {
		t.Fatalf("apply while storage recovery holds lock returned %v", err)
	}
	heldRelease()
	if err := heldFile.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyCleanup(context.Background(), CleanupPreview{}, "second-attempt", false, false); err != nil {
		t.Fatalf("apply after recovery lock release: %v", err)
	}
}

var _ database.Repository = (*evidenceRepository)(nil)
