package services

import (
	"context"

	capacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

// dockerFallback adapts the platform capacity DockerAttributor (cgroup -> docker
// inspect) to the procsampler.DockerFallback seam. It is the FALLBACK branch
// only: the primary attribution path is the cheap bare-host /proc heuristic
// (cwd + binary + PPID walk). On a bare-host deployment this resolves nothing
// (no cgroup container id) and the sample stays "unknown" — which is correct.
type dockerFallback struct {
	attributor capacity.Attributor
}

// NewDockerFallback builds the production docker fallback for this platform.
func NewDockerFallback() procsampler.DockerFallback {
	return dockerFallback{attributor: capacity.NewDockerAttributor()}
}

// Attribute returns the owner for a containerized pid, or "" when the pid is not
// containerized (so the caller leaves it in the "unknown" bucket).
func (d dockerFallback) Attribute(ctx context.Context, pid int) string {
	out := d.attributor.Attribute(ctx, pid)
	if out.OwnerID == "" || out.OwnerID == capacity.OwnerUnknown {
		return ""
	}
	return out.OwnerID
}
