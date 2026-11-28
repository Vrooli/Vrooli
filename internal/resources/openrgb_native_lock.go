package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

type openRGBNativeBootstrapLockKey struct{}

func nativeOpenRGBUsesPrivateLocks(resource string) bool {
	return resource == openRGBResourceName && (runtime.GOOS == openRGBPlatformDarwin || runtime.GOOS == openRGBPlatformWindows)
}

func openRGBNativeOwnerLockPath(root, key string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || key == "" || len(key) > 2048 || strings.ContainsAny(key, "\x00\r\n") {
		return "", errors.New("native OpenRGB lock requires its canonical resource root and bounded identity")
	}
	return filepath.Join(root, "state", "locks", fmt.Sprintf("%x.lock", sha256.Sum256([]byte(key)))), nil
}
func openRGBNativeOwnerLockPolicy(ctx context.Context, path string) (config.PrivatePathPolicy, error) {
	owner, err := readOpenRGBNativeOwnerContext(ctx)
	if err == nil {
		return owner.policyFor(path)
	}
	if !os.IsNotExist(err) {
		return config.PrivatePathPolicy{}, err
	}
	// Only explicit privileged preparation can bootstrap this owner namespace.
	// A missing descriptor during ordinary start/update/status creates nothing.
	p, ok := ctx.Value(openRGBNativeBootstrapLockKey{}).(config.PrivatePathPolicy)
	if !ok || p.OwnerPrincipal == "" || p.RuntimePrincipal != "" {
		return config.PrivatePathPolicy{}, err
	}
	return p, nil
}
func acquireOpenRGBNativeOwnerLock(ctx context.Context, key string) (func(), error) {
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return nil, err
	}
	return acquireOpenRGBNativeOwnerLockWith(ctx, root, key, openRGBNativeOwnerLockPolicy, config.ConfigurePrivateDirectoryAccess, config.AcquirePrivateFileLock)
}
func acquireOpenRGBNativeOwnerLockWith(ctx context.Context, root, key string, policy func(context.Context, string) (config.PrivatePathPolicy, error), prepare func(string, config.PrivatePathPolicy) error, acquire func(context.Context, string, config.PrivatePathPolicy) (func(), error)) (func(), error) {
	if ctx == nil {
		return nil, errors.New("native resource lock requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := openRGBNativeOwnerLockPath(root, key)
	if err != nil {
		return nil, err
	}
	p, err := policy(ctx, path)
	if err != nil {
		return nil, err
	}
	if p.OwnerPrincipal == "" || p.RuntimePrincipal != "" {
		return nil, errors.New("native OpenRGB owner locks must exclude backend principals")
	}
	for _, dir := range []string{filepath.Join(root, "state"), filepath.Join(root, "state", "locks")} {
		if err = prepare(dir, p); err != nil {
			return nil, err
		}
	}
	return acquire(ctx, path, p)
}

func acquireResourceArtifactOwnerLock(ctx context.Context, resource string) (func(), error) {
	if nativeOpenRGBUsesPrivateLocks(resource) {
		return acquireOpenRGBNativeOwnerLock(ctx, "resource:"+resource)
	}
	return artifactlock.AcquireContext(ctx, "resource:"+resource)
}

func (c *Controller) openRGBNativePreparationLockContext(ctx context.Context) (context.Context, error) {
	if _, err := readOpenRGBNativeOwnerContext(ctx); err == nil {
		return ctx, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	info, err := os.Lstat(c.Home)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("native OpenRGB preparation home is an alias or non-directory")
	}
	principal, err := config.PrivatePathOwner(c.Home, info)
	if err != nil {
		return nil, err
	}
	invoking, err := openRGBNativeOperatorPrincipal()
	if err != nil {
		return nil, err
	}
	if principal != invoking {
		return nil, errors.New("native OpenRGB bootstrap home differs from invoking management principal")
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return nil, err
	}
	if err = prepareOpenRGBNativeRootParents(root); err != nil {
		return nil, err
	}
	p := config.PrivatePathPolicy{OwnerPrincipal: principal, NoInherit: true}
	if err = config.ConfigurePrivateDirectoryAccess(root, p); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, openRGBNativeBootstrapLockKey{}, p), nil
}

func acquireOpenRGBDevicePolicyPublicationLock(ctx context.Context, path string) (func(), error) {
	if nativeOpenRGBUsesPrivateLocks(openRGBResourceName) {
		return acquireOpenRGBNativeOwnerLock(ctx, "resource-device-policy")
	}
	return artifactlock.AcquireWithPathContext(ctx, path, func(string) (string, error) { return path + ".lock", nil })
}

