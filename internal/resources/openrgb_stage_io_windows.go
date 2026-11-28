//go:build windows

package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"golang.org/x/sys/windows"
)

func openRGBMakeStageDirectory(ctx context.Context, path string) error {
	if err := boundOpenRGBWindowsSealedPath(path); err != nil {
		return err
	}
	release, err := platform.PrepareImmutableNativeDirectory(ctx, path)
	if err != nil {
		return err
	}
	return release()
}

func openRGBMakeStageScratch(ctx context.Context, path string) error {
	return openRGBMakeStageDirectory(ctx, path)
}

func openRGBWriteStageFile(ctx context.Context, path string, _ os.FileMode, write func(io.Writer) error) error {
	if err := boundOpenRGBWindowsSealedPath(path); err != nil {
		return err
	}
	return platform.WriteImmutableNativeFile(ctx, path, write)
}

func openRGBStageParentLease(ctx context.Context, path string) (func() error, error) {
	if err := boundOpenRGBWindowsSealedPath(path); err != nil {
		return nil, err
	}
	root, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return nil, errors.New("native stage parent escapes closed sealed root")
	}
	release, err := platform.PrepareImmutableNativeDirectory(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := configureOpenRGBSealedAncestors(ctx, path, root); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func prepareOpenRGBStageLock(ctx context.Context, path string) error {
	if err := boundOpenRGBWindowsSealedPath(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return platform.VerifyImmutableNativePath(ctx, path, false)
	} else if !os.IsNotExist(err) {
		return err
	}
	err := platform.WriteImmutableNativeFile(ctx, path, func(io.Writer) error { return nil })
	if err != nil {
		if _, statErr := os.Lstat(path); statErr == nil {
			return platform.VerifyImmutableNativePath(ctx, path, false)
		}
	}
	return err
}

func requireOpenRGBNativeStagePublisher() error {
	if err := requireCanonicalOpenRGBPublicationRoot(); err != nil {
		return err
	}
	if !windows.GetCurrentThreadEffectiveToken().IsElevated() {
		return fmt.Errorf("native sealed publication requires explicit elevated setup: %w", hostreqkit.ErrElevationRequired)
	}
	return nil
}

func boundOpenRGBWindowsSealedPath(path string) error {
	if err := requireCanonicalOpenRGBPublicationRoot(); err != nil {
		return err
	}
	root, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) || filepath.Clean(path) != path {
		return errors.New("native artifact write escapes canonical OpenRGB machine code namespace")
	}
	return nil
}
