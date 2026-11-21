//go:build !windows

package msvcruntime

import (
	"fmt"

	platform "github.com/vrooli/platform-go"
)

func Check() (string, error) {
	return "", fmt.Errorf("Microsoft C++ x64 runtime: %w", platform.ErrUnsupported)
}
