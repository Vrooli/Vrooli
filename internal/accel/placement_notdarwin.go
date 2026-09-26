//go:build !darwin

package accel

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// observeMetalProcess reports that Metal is unreachable off macOS. It returns a
// typed unsupported result with a named reason rather than failing to compile,
// so the same driver code path builds for every platform.
func observeMetalProcess(snapshot hostinventory.Snapshot, _ HostProcess) (Backend, AccessState, string, error) {
	return "", AccessUnknown, fmt.Sprintf("metal is only reachable on darwin; this host is %s", snapshot.OS), nil
}
