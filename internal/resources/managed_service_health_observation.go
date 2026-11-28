package resources

import (
	"context"
	"os"
)

// Operational health reads native process ownership and optional health probes,
// never an entire staged runtime tree. Installed here means present on disk;
// ArtifactVerification prevents it from becoming a current integrity claim.
func (d managedServiceDriver) observeManagedServiceHealth(ctx context.Context, controller *Controller, status Status, manifest ResourceManifest, state ManagedServiceState, running, fast bool) (Status, error) {
	status.ArtifactVerification = "not_evaluated"
	if running {
		status.Installed, status.Running = true, true
		status.RunningArtifactVersion = state.ArtifactVersion
		status.ArtifactUpdateAvailable = state.ArtifactVersion != status.DeclaredArtifactVersion
		return d.withRunningHealth(ctx, controller, status, manifest, fast)
	}
	path, err := managedServiceArtifactPath(controller, manifest)
	if err == nil {
		var info os.FileInfo
		info, err = os.Stat(path)
		if err == nil {
			status.Installed = info.Mode().IsRegular() || info.IsDir()
		}
	}
	if err != nil && !os.IsNotExist(err) {
		status.StatusCode, status.ProbeError = StatusCodeUnavailable, err.Error()
	}
	healthy := false
	status.Healthy, status.Health = &healthy, "stopped"
	return status, nil
}
