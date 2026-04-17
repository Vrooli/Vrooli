package remotesessionprotection

import (
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	sysctlPath    = "/etc/sysctl.d/99-vrooli-remote-session-protection.conf"
	systemdDir    = "/etc/systemd/system/user@.service.d"
	systemdPath   = "/etc/systemd/system/user@.service.d/90-vrooli-remote-session-protection.conf"
	logindDir     = "/etc/systemd/logind.conf.d"
	logindPath    = "/etc/systemd/logind.conf.d/90-vrooli-remote-session-protection.conf"
	sysctlContent = "vm.swappiness = 10\nvm.oom_kill_allocating_task = 0\n"
	unitContent   = "[Service]\nOOMScoreAdjust=-900\n"
	logindContent = "[Login]\nKillUserProcesses=no\n"
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string            { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind  { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "remote session protection is only supported on Linux hosts")
		return status
	}
	if !host.SupportsSysctl && !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not expose sysctl or systemd hooks needed for safeguard application")
		return status
	}

	pending := make([]string, 0, 3)
	if host.SupportsSysctl {
		if !hostreqkit.FileContentMatches(sysctlPath, sysctlContent) {
			pending = append(pending, sysctlPath)
		}
	}
	if host.SupportsSystemd {
		if !hostreqkit.FileContentMatches(systemdPath, unitContent) {
			pending = append(pending, systemdPath)
		}
		if !hostreqkit.FileContentMatches(logindPath, logindContent) {
			pending = append(pending, logindPath)
		}
	}
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "remote session protection files are present and match the managed Vrooli configuration")
		return status
	}

	if host.SupportsSysctl {
		status.Notes = append(status.Notes, "will manage Linux memory-pressure defaults via "+sysctlPath)
	}
	if host.SupportsSystemd {
		status.Notes = append(status.Notes, "will protect user sessions with managed systemd overrides")
	}
	status.Notes = append(status.Notes, "pending managed files: "+strings.Join(pending, ", "))
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would apply remote session protection")
		return status, nil
	}

	if host.SupportsSysctl {
		if err := hostreqkit.EnsureManagedDir(filepath.Dir(sysctlPath), opts.SudoMode, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := hostreqkit.InstallManagedContent(sysctlPath, sysctlContent, opts.SudoMode, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sysctl", []string{"-p", sysctlPath}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	}
	if host.SupportsSystemd {
		for _, dir := range []string{systemdDir, logindDir} {
			if err := hostreqkit.EnsureManagedDir(dir, opts.SudoMode, opts); err != nil {
				status.ExecutionState = hostreqkit.ExecutionFailed
				status.Notes = append(status.Notes, err.Error())
				return status, nil
			}
		}
		for _, file := range []struct {
			path    string
			content string
		}{
			{path: systemdPath, content: unitContent},
			{path: logindPath, content: logindContent},
		} {
			if err := hostreqkit.InstallManagedContent(file.path, file.content, opts.SudoMode, opts); err != nil {
				status.ExecutionState = hostreqkit.ExecutionFailed
				status.Notes = append(status.Notes, err.Error())
				return status, nil
			}
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "applied managed sysctl and systemd safeguards for remote Linux sessions")
	status.Notes = append(status.Notes, "existing login sessions may need to reconnect before all systemd protections take effect")
	return status, nil
}
