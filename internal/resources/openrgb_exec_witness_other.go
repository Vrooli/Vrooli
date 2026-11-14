//go:build !linux && !darwin && !windows

package resources

import (
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func confineOpenRGBExecThread() (func(), error) {
	return nil, fmt.Errorf("OpenRGB device-access witness: %w", platform.ErrUnsupported)
}

func openRGBExecContext(platform.ServiceRef, string) (string, error) {
	return "", fmt.Errorf("OpenRGB device-access witness: %w", platform.ErrUnsupported)
}

func openRGBServingContext(ref platform.ServiceRef, incarnation string) (string, error) {
	return openRGBExecContext(ref, incarnation)
}
