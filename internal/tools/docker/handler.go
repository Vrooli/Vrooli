package docker

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/dockerhost"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type handler struct {
	manifest hostreqkit.ToolManifest
}

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.PackageName = h.manifest.PackageNameForHost(host)
	status.Command, _ = hostreqkit.ResolveCommand(h.manifest.Commands)
	status.SupportClass = hostreqkit.SupportSupported
	status.InstallSupported = strings.TrimSpace(status.PackageName) != "" && !requirement.Manual
	if h.manifest.InstallHint != "" {
		status.Notes = append(status.Notes, h.manifest.InstallHint)
	}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.InstallSupported = false
		return status
	}
	if status.Command == "" {
		if status.SupportClass == hostreqkit.SupportSupported && strings.TrimSpace(status.PackageName) == "" {
			status.SupportClass = hostreqkit.SupportUnsupported
			status.ExecutionState = hostreqkit.ExecutionUnsupported
		}
		return status
	}

	status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	health := dockerhost.InspectHealth()
	if health.InfoOK {
		status.Installed = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "Docker daemon is reachable")
		return status
	}
	status.Installed = false
	status.ExecutionState = hostreqkit.ExecutionPending
	status.Notes = append(status.Notes, dockerHealthNotes(host, health)...)
	if host.OS == "linux" && host.SupportsSystemd && !health.PermissionDenied {
		status.BlockingReason = hostreqkit.BlockingNeedsSudo
	}
	if health.PermissionDenied {
		status.BlockingReason = hostreqkit.BlockingManual
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Command == "" {
		installed, updated := h.installClient(host, status, opts)
		status = updated
		if !installed {
			return status, nil
		}
	}

	health := dockerhost.InspectHealth()
	if health.InfoOK {
		return markDockerHealthy(status, hostreqkit.ExecutionAlreadyPresent), nil
	}
	if health.PermissionDenied {
		status.Installed = false
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		status.Notes = append(status.Notes, dockerHealthNotes(host, health)...)
		return status, nil
	}
	if host.OS != "linux" || !host.SupportsSystemd {
		status.Installed = false
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.BlockingReason = hostreqkit.BlockingManual
		status.Notes = append(status.Notes, dockerHealthNotes(host, health)...)
		return status, nil
	}
	if opts.DryRun {
		status.Installed = false
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.BlockingReason = hostreqkit.BlockingNeedsSudo
		status.Notes = append(status.Notes,
			"dry-run: would sanitize /etc/docker/daemon.json, reset failed Docker service state, start Docker, and verify docker info",
		)
		return status, nil
	}
	result, err := dockerhost.SanitizeDaemonConfig(dockerhost.DaemonConfigPath, dockerhost.ConfigOptions{}, opts)
	if err != nil {
		status.Installed = false
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		if hostreqkit.IsSudoSkipped(err) {
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
		}
		return status, nil
	}
	if result.Changed {
		status.Notes = append(status.Notes, "repaired Docker daemon config")
		if len(result.RemovedInvalidKeys) > 0 {
			status.Notes = append(status.Notes, "removed invalid Docker daemon config keys: "+strings.Join(result.RemovedInvalidKeys, ", "))
		}
	}
	if err := dockerhost.StartDockerService(opts); err != nil {
		status.Installed = false
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		if hostreqkit.IsSudoSkipped(err) {
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
		}
		return status, nil
	}
	health = dockerhost.InspectHealth()
	if health.InfoOK {
		return markDockerHealthy(status, hostreqkit.ExecutionApplied), nil
	}
	status.Installed = false
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, dockerHealthNotes(host, health)...)
	return status, nil
}

func markDockerHealthy(status hostreqkit.ItemStatus, state hostreqkit.ExecutionState) hostreqkit.ItemStatus {
	status.Installed = true
	status.ExecutionState = state
	status.BlockingReason = hostreqkit.BlockingNone
	status.Notes = append(filterStaleDockerFailureNotes(status.Notes), "Docker daemon is reachable")
	return status
}

func filterStaleDockerFailureNotes(notes []string) []string {
	filtered := make([]string, 0, len(notes))
	for _, note := range notes {
		switch {
		case strings.HasPrefix(note, "Docker CLI is installed but daemon verification failed"):
			continue
		case strings.HasPrefix(note, "Re-run as `sudo vrooli setup`"):
			continue
		case note == "Docker systemd service is failed":
			continue
		case strings.HasPrefix(note, "Docker daemon config validation failed"):
			continue
		case note == "Docker daemon is reachable":
			continue
		default:
			filtered = append(filtered, note)
		}
	}
	return filtered
}

func (h handler) installClient(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (bool, hostreqkit.ItemStatus) {
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return false, status
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return false, status
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return false, status
	}
	if !status.InstallSupported || strings.TrimSpace(status.PackageName) == "" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return false, status
	}
	command, args, err := hostreqkit.InstallCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.Notes = append(status.Notes, err.Error())
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return false, status
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would run %s %s", command, strings.Join(args, " ")))
		return false, status
	}
	if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return false, status
	}
	status.Command, status.Installed = hostreqkit.ResolveCommand(h.manifest.Commands)
	if status.Installed {
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		return true, status
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install command completed but Docker is still not available on PATH")
	return false, status
}

func dockerHealthNotes(host hostreqkit.Host, health dockerhost.Health) []string {
	notes := []string{}
	if !health.ClientInstalled {
		return append(notes, "Docker CLI is not installed")
	}
	if health.Detail != "" {
		notes = append(notes, "Docker CLI is installed but daemon verification failed: "+dockerhost.DiagnosticLine(health.Detail))
	} else {
		notes = append(notes, "Docker CLI is installed but daemon verification failed")
	}
	if health.PermissionDenied {
		notes = append(notes, "Current user cannot access the Docker socket; add the user to the docker group, start a new login session, or run the command with sudo")
		return notes
	}
	switch {
	case host.OS == "linux" && host.SupportsSystemd:
		notes = append(notes, "Re-run as `sudo vrooli setup` or pass --sudo-mode=ask to repair and start Docker")
	case host.OS == "darwin" || host.OS == "windows":
		notes = append(notes, "Start Docker Desktop, then re-run `vrooli setup`")
	default:
		notes = append(notes, "Start the Docker daemon, then re-run `vrooli setup`")
	}
	if !health.ConfigValid && health.ValidationDetail != "" {
		notes = append(notes, "Docker daemon config validation failed: "+health.ValidationDetail)
	}
	if health.ServiceFailed {
		notes = append(notes, "Docker systemd service is failed")
	}
	return notes
}
