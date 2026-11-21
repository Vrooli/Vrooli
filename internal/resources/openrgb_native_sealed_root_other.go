//go:build !darwin && !windows

package resources

import (
	"fmt"
	platform "github.com/vrooli/platform-go"
)

func nativeOpenRGBSealedRoot() (string, error) {
	return "", fmt.Errorf("native sealed machine artifacts: %w", platform.ErrUnsupported)
}
