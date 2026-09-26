//go:build linux

package driver

import (
	"context"

	"workspace-sandbox/internal/process"
)

// GetContainmentInfo reports the Linux process-containment backend
// (bubblewrap). The enforcement list is intrinsic to bwrap as configured
// by exec.BuildBwrapArgs: it unshares the network (network-deny) and PID
// namespace (pid-namespace), and binds the sandbox merged dir over the
// workspace and host $HOME so writes land in the overlay upper layer
// (filesystem-write-containment) at their host-visible paths
// (path-illusion). Available flips false when the bwrap binary is missing.
func GetContainmentInfo(ctx context.Context, starter process.Starter) (*ContainmentInfo, error) {
	info := &ContainmentInfo{
		Backend: "bwrap",
		Enforcements: []string{
			EnforcementFilesystemWriteContainment,
			EnforcementNetworkDeny,
			EnforcementPIDNamespace,
			EnforcementPathIllusion,
		},
	}
	available, version, err := IsBwrapAvailable(ctx, starter)
	if !available {
		info.Available = false
		if err != nil {
			info.Error = err.Error()
		}
		return info, nil
	}
	bwrapPath, _ := starter.LookPath("bwrap")
	info.Available = true
	info.Version = version
	info.Path = bwrapPath
	return info, nil
}
