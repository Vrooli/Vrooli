//go:build darwin && !cgo

package resources

import (
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func openRGBDarwinSharedCacheContains(string) (bool, error) {
	return false, fmt.Errorf("native dyld cache prerequisite inspection requires Darwin CGO SDK qualification: %w", platform.ErrUnsupported)
}
