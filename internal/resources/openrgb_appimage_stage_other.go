//go:build !linux && !darwin && !windows

package resources

import (
	"context"
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func verifyOpenRGBAppDirSeal(context.Context, string) error {
	return fmt.Errorf("OpenRGB AppDir sealing: %w", platform.ErrUnsupported)
}
