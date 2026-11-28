//go:build !linux && !darwin

package driver

import (
	"context"

	"workspace-sandbox/internal/process"
)

// GetContainmentInfo reports that no OS-native containment backend is
// present on this OS. Linux ships bwrap (containment_linux.go) and macOS
// ships Seatbelt (containment_darwin.go); everywhere else execution falls
// back to the direct path, which enforces nothing, so the report is honest
// about the absent guarantees: backend "none", not available, empty
// enforcement list.
func GetContainmentInfo(_ context.Context, _ process.Starter) (*ContainmentInfo, error) {
	return &ContainmentInfo{
		Backend:      "none",
		Available:    false,
		Enforcements: []string{},
	}, nil
}
