package filerouting

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time                            { return c.now }
func (c *fakeClock) NewTimer(d time.Duration) schedule.Timer   { return schedule.System().NewTimer(d) }
func (c *fakeClock) NewTicker(d time.Duration) schedule.Ticker { return schedule.System().NewTicker(d) }
func (c *fakeClock) Sleep(d time.Duration)                     { c.now = c.now.Add(d) }

func testPaths(root string) storage.Paths {
	return storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")}
}

func TestInstallLeasedTestRootsSeedsConfigAndClearRemovesTree(t *testing.T) {
	primary := testPaths(t.TempDir())
	if err := os.MkdirAll(primary.ConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(primary.ConfigDir, "settings.json"), []byte(`{"safe":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	routes := New(primary)
	roots, err := routes.InstallLeasedTestRoots("lease-a", time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(roots.ConfigDir, "settings.json")); err != nil || string(data) != `{"safe":true}` {
		t.Fatalf("seeded config = %q, %v", data, err)
	}
	if entries, err := os.ReadDir(roots.DataDir); err != nil || len(entries) != 0 {
		t.Fatalf("test data root should be empty, entries=%v err=%v", entries, err)
	}
	base := filepath.Dir(roots.ConfigDir)
	if err := routes.ClearTestRoots("lease-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("leased root remains after clear: %v", err)
	}
}

func TestRoutedRootsPickAndWriteEvidence(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	routes := New(testPaths("/primary"))
	routes.SetClock(clock)
	if err := routes.InstallTestRoots(testPaths("/leased"), "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}

	plain, err := routes.Pick(context.Background(), storage.ClassConfig)
	if err != nil || plain != "/primary/config" {
		t.Fatalf("plain Pick = %q, %v", plain, err)
	}
	testMode := database.WithTestMode(context.Background())
	routed, err := routes.Pick(testMode, storage.ClassConfig)
	if err != nil || routed != "/leased/config" {
		t.Fatalf("test-mode Pick = %q, %v", routed, err)
	}
	routes.RecordWrite(testMode)
	if got := routes.LeaseStats().TestRootWrites; got != 1 {
		t.Fatalf("test root writes = %d, want 1", got)
	}

	clock.now = clock.now.Add(2 * time.Minute)
	expired, err := routes.Pick(testMode, storage.ClassConfig)
	if err != nil || expired != "/primary/config" {
		t.Fatalf("expired Pick = %q, %v", expired, err)
	}
	routes.RecordWrite(testMode)
	if got := routes.LeaseStats().PrimaryWritesDuringTestMode; got != 1 {
		t.Fatalf("primary writes during test mode = %d, want 1", got)
	}
}

func TestPickRequiredRefusesMissingAndExpiredTestRoots(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	routes := New(testPaths("/primary"))
	routes.SetClock(clock)
	ctx := database.WithTestMode(context.Background())
	if _, err := routes.PickRequired(ctx, storage.ClassData); err == nil {
		t.Fatal("missing lease fell back to production")
	}
	if err := routes.InstallTestRoots(testPaths("/isolated"), "lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	root, err := routes.PickRequired(ctx, storage.ClassData)
	if err != nil || root != "/isolated/data" {
		t.Fatalf("%q %v", root, err)
	}
	clock.now = clock.now.Add(2 * time.Minute)
	if _, err := routes.PickRequired(ctx, storage.ClassData); err == nil {
		t.Fatal("expired lease fell back to production")
	}
	root, err = routes.PickRequired(context.Background(), storage.ClassData)
	if err != nil || root != "/primary/data" {
		t.Fatalf("production route: %q %v", root, err)
	}
}

func TestRoutedRootsClearHonorsLease(t *testing.T) {
	routes := New(testPaths("/primary"))
	if err := routes.InstallTestRoots(testPaths("/leased"), "lease-a", 0); err != nil {
		t.Fatal(err)
	}
	if err := routes.ClearTestRoots("lease-b"); err == nil {
		t.Fatal("expected lease mismatch")
	}
	if err := routes.ClearTestRoots("lease-a"); err != nil {
		t.Fatal(err)
	}
	path, err := routes.Pick(database.WithTestMode(context.Background()), storage.ClassData)
	if err != nil || path != "/primary/data" {
		t.Fatalf("Pick after clear = %q, %v", path, err)
	}
}
