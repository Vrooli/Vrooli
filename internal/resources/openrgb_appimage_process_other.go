//go:build !linux && !darwin && !windows

package resources

import (
	"context"
	"fmt"
	"os"

	platform "github.com/vrooli/platform-go"
)

func (o openRGBServingObservation) observeAppImageStage(context.Context, openRGBAppImageStage, os.FileInfo, string) error {
	return fmt.Errorf("OpenRGB staged process observation: %w", platform.ErrUnsupported)
}
