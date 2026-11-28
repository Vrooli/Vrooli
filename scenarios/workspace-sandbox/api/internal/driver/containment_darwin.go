//go:build darwin

package driver

import (
	"context"
	"fmt"

	"workspace-sandbox/internal/process"
)

// GetContainmentInfo reports the macOS process-containment backend (Seatbelt,
// via the system sandbox-exec binary). The backend is honestly partial: as
// configured by exec.SeatbeltProfile it denies file writes outside the
// sandbox writable set (filesystem-write-containment) and denies network when
// the profile disallows it (network-deny), but it does NOT rewrite paths
// (no path-illusion) and does NOT unshare the pid namespace (no pid-namespace).
// Available flips false when sandbox-exec is not on PATH, in which case the
// report degrades to the direct-path backend so the absent guarantees are not
// advertised.
func GetContainmentInfo(_ context.Context, starter process.Starter) (*ContainmentInfo, error) {
	path, err := starter.LookPath("sandbox-exec")
	info := seatbeltContainmentInfo(err == nil, path)
	if err != nil {
		info.Error = fmt.Sprintf("sandbox-exec not found: %v", err)
	}
	return info, nil
}
