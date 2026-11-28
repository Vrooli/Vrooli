package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
)

// Native code-store locks inherit machine publication authority. In particular,
// the per-user artifact lock owner must never chmod/chown their sealed parents
// or lock inodes while a privileged publisher is acting for an ordinary user.
func acquireOpenRGBStageLock(ctx context.Context, final string) (func(), error) {
	path := final + ".lock"
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return artifactlock.AcquireWithPathContext(ctx, final, func(string) (string, error) { return path, nil })
	}
	if err := requireOpenRGBNativeStagePublisher(); err != nil {
		return nil, err
	}
	root, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(rel) {
		return nil, errors.New("native code-store lock escapes its machine authority")
	}
	if err := platform.VerifyImmutableNativePath(ctx, filepath.Dir(path), true); err != nil {
		return nil, err
	}
	if err := prepareOpenRGBStageLock(ctx, path); err != nil {
		return nil, err
	}
	if err := platform.VerifyImmutableNativePath(ctx, path, false); err != nil {
		return nil, err
	}
	// Both flock and LockFileEx can lock a read-only admitted handle. No
	// filesystem owner, permission or path changes occur in this lock boundary.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err := platform.VerifyLockFile(file, path); err != nil {
		file.Close()
		return nil, err
	}
	release, err := platform.LockFileContext(ctx, file)
	if err != nil {
		file.Close()
		return nil, err
	}
	if err := errors.Join(platform.VerifyLockFile(file, path), platform.VerifyImmutableNativePath(ctx, path, false)); err != nil {
		release()
		file.Close()
		return nil, err
	}
	var once sync.Once
	return func() { once.Do(func() { release(); file.Close() }) }, nil
}
