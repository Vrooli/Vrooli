//go:build !linux

package livedesktop

import (
	"fmt"
	"log/slog"
)

// NewPlatformBackend returns a typed capability result instead of compiling a
// Linux display implementation into other operating systems.
func NewPlatformBackend(*slog.Logger) (PlatformBackend, error) {
	return nil, fmt.Errorf("desktop emulator local backend is unsupported on this operating system")
}
