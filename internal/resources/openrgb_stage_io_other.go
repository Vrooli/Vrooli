//go:build !windows

package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func openRGBMakeStageDirectory(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.MkdirAll(path, 0o700)
}

func openRGBMakeStageScratch(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Mkdir(path, 0o700)
}

func openRGBWriteStageFile(ctx context.Context, path string, perm os.FileMode, write func(io.Writer) error) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	return errors.Join(write(f), f.Close())
}

func openRGBStageParentLease(ctx context.Context, path string) (func() error, error) {
	if err := ensureOpenRGBStageParent(ctx, path); err != nil {
		return nil, err
	}
	return func() error { return nil }, nil
}
func prepareOpenRGBStageLock(ctx context.Context, path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o444)
	if os.IsExist(err) {
		return nil // The lock owner verifies the existing inode and its full seal.
	}
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func requireOpenRGBNativeStagePublisher() error {
	if runtime.GOOS == "darwin" {
		if err := requireCanonicalOpenRGBPublicationRoot(); err != nil {
			return err
		}
	}
	if runtime.GOOS == "darwin" && os.Geteuid() != 0 {
		return fmt.Errorf("native root-sealed publication requires explicit privileged setup: %w", hostreqkit.ErrElevationRequired)
	}
	return nil
}
