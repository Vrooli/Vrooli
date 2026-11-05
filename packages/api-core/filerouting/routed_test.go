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

func createRootTree(t *testing.T, root string) storage.Paths {
	t.Helper()
	paths := testPaths(root)
	for _, class := range []storage.Class{storage.ClassConfig, storage.ClassData, storage.ClassCache, storage.ClassLogs, storage.ClassState} {
		path, err := paths.ForClass(class)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return paths
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

func TestClearRetainedTestRootsRequiresExactDetachedOwnerState(t *testing.T) {
	primary := createRootTree(t, t.TempDir())
	if err := os.WriteFile(filepath.Join(primary.DataDir, "sentinel"), []byte("primary"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldOwner := New(primary)
	leased, err := oldOwner.InstallLeasedTestRoots("expired-lease", time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	state, err := oldOwner.TestRootsRestartState("expired-lease")
	if err != nil {
		t.Fatal(err)
	}
	retained := filepath.Dir(leased.ConfigDir)
	newOwner := New(primary)
	if err := newOwner.ClearRetainedTestRoots(state, newOwner.PrimaryFingerprint()); err != nil {
		t.Fatalf("clear exact detached owner tree: %v", err)
	}
	if _, err := os.Lstat(retained); !os.IsNotExist(err) {
		t.Fatalf("retained tree remains: %v", err)
	}
	if content, err := os.ReadFile(filepath.Join(primary.DataDir, "sentinel")); err != nil || string(content) != "primary" {
		t.Fatalf("primary tree changed: content=%q err=%v", content, err)
	}
	if err := newOwner.ClearRetainedTestRoots(state, newOwner.PrimaryFingerprint()); err != nil {
		t.Fatalf("detached cleanup retry should be idempotent: %v", err)
	}
	if err := oldOwner.ClearRetainedTestRoots(state, oldOwner.PrimaryFingerprint()); err == nil {
		t.Fatal("active owner allowed retained cleanup")
	}
	wrong := state
	wrong.PrimaryFingerprint = "different"
	if err := newOwner.ClearRetainedTestRoots(wrong, newOwner.PrimaryFingerprint()); err == nil {
		t.Fatal("mismatched primary fingerprint allowed cleanup")
	}
}

func TestClearRetainedTestRootsRefusesAnotherActiveLease(t *testing.T) {
	primary := createRootTree(t, t.TempDir())
	oldOwner := New(primary)
	_, err := oldOwner.InstallLeasedTestRoots("expired-lease", time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	state, err := oldOwner.TestRootsRestartState("expired-lease")
	if err != nil {
		t.Fatal(err)
	}
	retained := state.LeaseRoot
	newOwner := New(primary)
	if _, err := newOwner.InstallLeasedTestRoots("other-active-lease", time.Minute, true); err != nil {
		t.Fatal(err)
	}
	if err := newOwner.ClearRetainedTestRoots(state, newOwner.PrimaryFingerprint()); err == nil {
		t.Fatal("cleanup replaced another active lease")
	}
	if _, err := os.Stat(retained); err != nil {
		t.Fatalf("refused cleanup removed retained roots: %v", err)
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

func TestResumeTestRootsReplacesExpiredBindingAndInvalidatesOldPin(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	primary := createRootTree(t, t.TempDir())
	routes := New(primary)
	routes.SetClock(clock)
	oldRoots, err := routes.InstallLeasedTestRoots("expired", time.Minute, true)
	if err != nil {
		t.Fatalf("install old roots: %v", err)
	}
	oldRoot := filepath.Dir(oldRoots.ConfigDir)
	pinned, err := routes.PinTestRoots(context.Background(), "expired")
	if err != nil {
		t.Fatalf("pin old roots: %v", err)
	}
	routes.RecordWrite(pinned)
	if got := routes.LeaseStats().TestRootWrites; got != 1 {
		t.Fatalf("initial test writes = %d, want 1", got)
	}
	clock.now = clock.now.Add(time.Minute + time.Second)

	newRoot := t.TempDir()
	newRoots := createRootTree(t, newRoot)
	state := TestRootsRestartState{
		LeaseID: "resumed", ExpiresAt: clock.Now().Add(time.Minute), LeaseRoot: newRoot,
		Roots: newRoots, PrimaryFingerprint: routes.PrimaryFingerprint(),
	}
	if err := routes.ResumeTestRoots(state, routes.PrimaryFingerprint()); err != nil {
		t.Fatalf("resume over expired roots: %v", err)
	}
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expired owner-created root remains: %v", err)
	}
	if got := routes.LeaseStats(); got.TestRootWrites != 0 || got.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("resumed roots did not reset stats: %#v", got)
	}
	if got, err := routes.PickRequired(database.WithTestMode(context.Background()), storage.ClassData); err != nil || got != newRoots.DataDir {
		t.Fatalf("resumed route = %q, %v; want %q", got, err, newRoots.DataDir)
	}
	if _, err := routes.PickRequired(pinned, storage.ClassData); err == nil {
		t.Fatal("old pin selected resumed roots")
	}
	if _, err := routes.Pick(pinned, storage.ClassData); err == nil {
		t.Fatal("old pin fell through to a route")
	}
	routes.RecordWrite(pinned)
	if got := routes.LeaseStats(); got.TestRootWrites != 0 || got.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("stale pinned write changed resumed stats: %#v", got)
	}
}

func TestResumeTestRootsRefusesLiveBindingWithSameLeaseID(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	primary := createRootTree(t, t.TempDir())
	routes := New(primary)
	routes.SetClock(clock)
	activeRoots, err := routes.InstallLeasedTestRoots("lease-a", time.Minute, true)
	if err != nil {
		t.Fatalf("install active roots: %v", err)
	}
	newRoot := t.TempDir()
	state := TestRootsRestartState{
		LeaseID: "lease-a", ExpiresAt: clock.Now().Add(time.Minute), LeaseRoot: newRoot,
		Roots: createRootTree(t, newRoot), PrimaryFingerprint: routes.PrimaryFingerprint(),
	}
	if err := routes.ResumeTestRoots(state, routes.PrimaryFingerprint()); err == nil {
		t.Fatal("resume replaced a live binding with the same lease ID")
	}
	if got, err := routes.PickRequired(database.WithTestMode(context.Background()), storage.ClassData); err != nil || got != activeRoots.DataDir {
		t.Fatalf("live binding changed: got %q, %v; want %q", got, err, activeRoots.DataDir)
	}
}

func TestPinnedTestRootsRejectSameIDReplacement(t *testing.T) {
	routes := New(testPaths(t.TempDir()))
	first := testPaths(t.TempDir())
	second := testPaths(t.TempDir())
	if err := routes.InstallTestRoots(first, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx, err := routes.PinTestRoots(context.Background(), "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := routes.InstallTestRoots(second, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := routes.PinTestRoots(ctx, "lease-a"); err == nil {
		t.Fatal("stale context rebound to a same-ID replacement")
	}
	if _, err := routes.PinTestRoots(ctx, "lease-b"); err == nil {
		t.Fatal("lease-A context rebound to lease B")
	}
	if _, err := routes.PickRequired(ctx, storage.ClassData); err == nil {
		t.Fatal("pinned PickRequired selected a same-ID replacement")
	}
	if _, err := routes.Pick(ctx, storage.ClassData); err == nil {
		t.Fatal("pinned Pick fell through or selected a replacement")
	}
	root, err := routes.PickRequired(database.WithTestMode(context.Background()), storage.ClassData)
	if err != nil || root != second.DataDir {
		t.Fatalf("unbound routing changed: got %q, %v; want %q", root, err, second.DataDir)
	}
}

func TestPinnedTestRootsFromAnotherOwnerFailsWithoutChangingEitherOwner(t *testing.T) {
	first := New(testPaths(t.TempDir()))
	second := New(testPaths(t.TempDir()))
	firstRoots := testPaths(t.TempDir())
	secondRoots := testPaths(t.TempDir())
	if err := first.InstallTestRoots(firstRoots, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := second.InstallTestRoots(secondRoots, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	pinned, err := first.PinTestRoots(context.Background(), "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.PickRequired(pinned, storage.ClassData); err == nil {
		t.Fatal("foreign owner accepted another root owner's pin")
	}
	if got, err := first.PickRequired(pinned, storage.ClassData); err != nil || got != firstRoots.DataDir {
		t.Fatalf("foreign owner attempt changed pin owner: got %q, %v", got, err)
	}
	if got, err := second.PickRequired(database.WithTestMode(context.Background()), storage.ClassData); err != nil || got != secondRoots.DataDir {
		t.Fatalf("foreign owner attempt changed target: got %q, %v", got, err)
	}
}

func TestPinnedTestRootsExpiryAndLeaseMismatchFailClosed(t *testing.T) {
	clock := &fakeClock{now: time.Unix(100, 0)}
	primary := testPaths(t.TempDir())
	routes := New(primary)
	routes.SetClock(clock)
	leased := testPaths(t.TempDir())
	if err := routes.InstallTestRoots(leased, "lease-a", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := routes.PinTestRoots(context.Background(), "lease-b"); err == nil {
		t.Fatal("pin accepted a different lease identity")
	}
	ctx, err := routes.PinTestRoots(context.Background(), "lease-a")
	if err != nil {
		t.Fatal(err)
	}
	clock.now = clock.now.Add(time.Minute + time.Second)
	if _, err := routes.PickRequired(ctx, storage.ClassData); err == nil {
		t.Fatal("expired pin selected a root")
	}
	if _, err := routes.Pick(ctx, storage.ClassData); err == nil {
		t.Fatal("expired pin fell through to a route")
	}
	routes.RecordWrite(ctx)
	if got := routes.LeaseStats(); got.PrimaryWritesDuringTestMode != 0 || got.TestRootWrites != 0 {
		t.Fatalf("stale pinned write was counted against another lease: %#v", got)
	}
	root, err := routes.Pick(database.WithTestMode(context.Background()), storage.ClassData)
	if err != nil || root != primary.DataDir {
		t.Fatalf("unbound routing behavior changed: got %q, %v", root, err)
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
