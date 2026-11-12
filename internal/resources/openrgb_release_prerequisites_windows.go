//go:build windows

package resources

import (
	"context"

	msvcruntime "github.com/vrooli/vrooli/internal/tools/msvc-runtime"
)

func verifyOpenRGBReleaseRuntimePrerequisites(ctx context.Context, bundle string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := verifyOpenRGBNativeHostArchitecture(); err != nil {
		return err
	}
	_, err := msvcruntime.Check()
	if err != nil {
		return err
	}
	return msvcruntime.CheckBundleImports(bundle)
}
