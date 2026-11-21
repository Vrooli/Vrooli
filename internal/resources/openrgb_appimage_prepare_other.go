//go:build !linux && !darwin && !windows

package resources

import (
	"context"
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func openRGBStagePreparationOwner() (openRGBStagePreparation, error) {
	return openRGBStagePreparation{}, fmt.Errorf("OpenRGB stage preparation: %w", platform.ErrUnsupported)
}

func ensureOpenRGBStageParent(context.Context, string) error {
	return fmt.Errorf("OpenRGB stage preparation: %w", platform.ErrUnsupported)
}

func verifyOpenRGBPendingOwner(string) error {
	return fmt.Errorf("OpenRGB stage preparation: %w", platform.ErrUnsupported)
}
