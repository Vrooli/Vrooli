package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/config"
)

func TestNativeOpenRGBLockNamespaceDoesNotFollowActorHOME(t *testing.T) {
	root := t.TempDir()
	key := "resource-lifecycle:openrgb"
	var paths []string
	p := config.PrivatePathPolicy{OwnerPrincipal: "uid:501"}
	policy := func(context.Context, string) (config.PrivatePathPolicy, error) { return p, nil }
	prepare := func(string, config.PrivatePathPolicy) error { return nil }
	acquire := func(_ context.Context, path string, _ config.PrivatePathPolicy) (func(), error) {
		paths = append(paths, path)
		return func() {}, nil
	}
	for _, home := range []string{filepath.Join(t.TempDir(), "operator"), filepath.Join(t.TempDir(), "machine-service")} {
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		release, err := acquireOpenRGBNativeOwnerLockWith(context.Background(), root, key, policy, prepare, acquire)
		if err != nil {
			t.Fatal(err)
		}
		release()
	}
	if len(paths) != 2 || paths[0] != paths[1] || filepath.Dir(paths[0]) != filepath.Join(root, "state", "locks") {
		t.Fatalf("actors do not share canonical lock: %v", paths)
	}
	other, err := openRGBNativeOwnerLockPath(root, "resource-initial-exec:distinct")
	if err != nil {
		t.Fatal(err)
	}
	if other == paths[0] {
		t.Fatal("different lock identities collided")
	}
}
func TestNativeOpenRGBMissingLockAuthorityHasNoEffects(t *testing.T) {
	prepared, acquired := 0, 0
	policy := func(context.Context, string) (config.PrivatePathPolicy, error) {
		return config.PrivatePathPolicy{}, os.ErrNotExist
	}
	prepare := func(string, config.PrivatePathPolicy) error { prepared++; return nil }
	acquire := func(context.Context, string, config.PrivatePathPolicy) (func(), error) {
		acquired++
		return func() {}, nil
	}
	if _, err := acquireOpenRGBNativeOwnerLockWith(context.Background(), t.TempDir(), "resource-lifecycle:openrgb", policy, prepare, acquire); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if prepared != 0 || acquired != 0 {
		t.Fatal("missing enrollment created lock namespace")
	}
	policy = func(context.Context, string) (config.PrivatePathPolicy, error) {
		return config.PrivatePathPolicy{OwnerPrincipal: "uid:501", RuntimePrincipal: "uid:502"}, nil
	}
	if _, err := acquireOpenRGBNativeOwnerLockWith(context.Background(), t.TempDir(), "resource-lifecycle:openrgb", policy, prepare, acquire); err == nil {
		t.Fatal("backend granted management lock authority")
	}
	if prepared != 0 || acquired != 0 {
		t.Fatal("invalid lock principal caused effects")
	}
}
func TestNativeOpenRGBActorsContendOnTheSamePrivateLock(t *testing.T) {
	root := t.TempDir()
	info, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := config.PrivatePathOwner(root, info)
	if err != nil {
		t.Fatal(err)
	}
	p := config.PrivatePathPolicy{OwnerPrincipal: owner}
	policy := func(context.Context, string) (config.PrivatePathPolicy, error) { return p, nil }
	// Linux executes the real file-lock protocol under its current owner; it
	// cannot attest a native explicit-account ACL. Native targets retain p.
	effective := func(got config.PrivatePathPolicy) config.PrivatePathPolicy {
		if got != p {
			t.Fatal("lock policy changed")
		}
		if runtime.GOOS == "linux" {
			return config.PrivatePathPolicy{}
		}
		return got
	}
	prepare := func(path string, got config.PrivatePathPolicy) error {
		return config.ConfigurePrivateDirectoryAccess(path, effective(got))
	}
	lock := func(ctx context.Context, path string, got config.PrivatePathPolicy) (func(), error) {
		return config.AcquirePrivateFileLock(ctx, path, effective(got))
	}
	acquire := func(ctx context.Context) (func(), error) {
		return acquireOpenRGBNativeOwnerLockWith(ctx, root, "resource-lifecycle:openrgb", policy, prepare, lock)
	}
	release, err := acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if next, err := acquire(ctx); err == nil {
		next()
		release()
		t.Fatal("concurrent native actor bypassed held resource lock")
	}
	release()
	next, err := acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next()
}
