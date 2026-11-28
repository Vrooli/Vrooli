//go:build darwin

package resources

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

func verifyOpenRGBReleaseRuntimePrerequisites(ctx context.Context, bundle string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	version, err := unix.Sysctl("kern.osproductversion")
	if err != nil {
		return fmt.Errorf("native macOS product version unavailable: %w", err)
	}
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return err
	}
	if !openRGBDarwinVersionAtLeast(version, p.MinimumDarwinVersion) {
		return errors.New("the exact OpenRGB1.1 native bundle requires macOS15.0+ system frameworks; older or unknown hosts are unqualified")
	}

	return p.verifyDarwinLoaderCache(ctx, bundle, openRGBDarwinSharedCacheContains)
}
