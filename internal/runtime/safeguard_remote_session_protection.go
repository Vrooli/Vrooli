package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

const (
	remoteSessionSysctlPath    = "/etc/sysctl.d/99-vrooli-remote-session-protection.conf"
	remoteSessionSystemdDir    = "/etc/systemd/system/user@.service.d"
	remoteSessionSystemdPath   = "/etc/systemd/system/user@.service.d/90-vrooli-remote-session-protection.conf"
	remoteSessionLogindDir     = "/etc/systemd/logind.conf.d"
	remoteSessionLogindPath    = "/etc/systemd/logind.conf.d/90-vrooli-remote-session-protection.conf"
	remoteSessionSysctlContent = "vm.swappiness = 10\nvm.oom_kill_allocating_task = 0\n"
	remoteSessionUnitContent   = "[Service]\nOOMScoreAdjust=-900\n"
	remoteSessionLogindContent = "[Login]\nKillUserProcesses=no\n"
)

type remoteSessionProtectionHandler struct{}

func newRemoteSessionProtectionSafeguard() handler {
	return remoteSessionProtectionHandler{}
}

func (remoteSessionProtectionHandler) Name() string       { return "remote_session_protection" }
func (remoteSessionProtectionHandler) Kind() hostreq.Kind { return hostreq.KindSafeguard }

func (remoteSessionProtectionHandler) Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	status := baseStatus(requirement)
	status.SupportClass = SupportSupported
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "remote session protection is only supported on Linux hosts")
		return status
	}
	if !host.SupportsSysctl && !host.SupportsSystemd {
		status.SupportClass = SupportNotApplicable
		status.ExecutionState = ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not expose sysctl or systemd hooks needed for safeguard application")
		return status
	}

	pending := make([]string, 0, 3)
	if host.SupportsSysctl {
		if !fileContentMatches(remoteSessionSysctlPath, remoteSessionSysctlContent) {
			pending = append(pending, remoteSessionSysctlPath)
		}
	}
	if host.SupportsSystemd {
		if !fileContentMatches(remoteSessionSystemdPath, remoteSessionUnitContent) {
			pending = append(pending, remoteSessionSystemdPath)
		}
		if !fileContentMatches(remoteSessionLogindPath, remoteSessionLogindContent) {
			pending = append(pending, remoteSessionLogindPath)
		}
	}
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "remote session protection files are present and match the managed Vrooli configuration")
		return status
	}

	if host.SupportsSysctl {
		status.Notes = append(status.Notes, "will manage Linux memory-pressure defaults via "+remoteSessionSysctlPath)
	}
	if host.SupportsSystemd {
		status.Notes = append(status.Notes, "will protect user sessions with managed systemd overrides")
	}
	status.Notes = append(status.Notes, "pending managed files: "+strings.Join(pending, ", "))
	return status
}

func (remoteSessionProtectionHandler) Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	switch status.SupportClass {
	case SupportUnsupported:
		status.ExecutionState = ExecutionUnsupported
		return status, nil
	case SupportNotApplicable:
		status.ExecutionState = ExecutionNotApplicable
		return status, nil
	case SupportManualOnly:
		status.ExecutionState = ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would apply remote session protection")
		return status, nil
	}

	if host.SupportsSysctl {
		if err := ensureManagedDir(filepath.Dir(remoteSessionSysctlPath), opts.SudoMode, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := installManagedContent(remoteSessionSysctlPath, remoteSessionSysctlContent, opts.SudoMode, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
		if err := runPrivilegedCommand(opts.SudoMode, "sysctl", []string{"-p", remoteSessionSysctlPath}, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	}
	if host.SupportsSystemd {
		for _, dir := range []string{remoteSessionSystemdDir, remoteSessionLogindDir} {
			if err := ensureManagedDir(dir, opts.SudoMode, opts); err != nil {
				status.ExecutionState = ExecutionFailed
				status.Notes = append(status.Notes, err.Error())
				return status, nil
			}
		}
		for _, file := range []struct {
			path    string
			content string
		}{
			{path: remoteSessionSystemdPath, content: remoteSessionUnitContent},
			{path: remoteSessionLogindPath, content: remoteSessionLogindContent},
		} {
			if err := installManagedContent(file.path, file.content, opts.SudoMode, opts); err != nil {
				status.ExecutionState = ExecutionFailed
				status.Notes = append(status.Notes, err.Error())
				return status, nil
			}
		}
		if err := runPrivilegedCommand(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
			status.ExecutionState = ExecutionFailed
			status.Notes = append(status.Notes, err.Error())
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = ExecutionApplied
	status.Notes = append(status.Notes, "applied managed sysctl and systemd safeguards for remote Linux sessions")
	status.Notes = append(status.Notes, "existing login sessions may need to reconnect before all systemd protections take effect")
	return status, nil
}

func fileContentMatches(path, want string) bool {
	content, err := readFileFn(path)
	if err != nil {
		return false
	}
	return string(content) == want
}

func ensureManagedDir(path, sudoMode string, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	return runPrivilegedCommand(sudoMode, "mkdir", []string{"-p", path}, opts)
}

func installManagedContent(path, content, sudoMode string, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	file, err := os.CreateTemp("", "vrooli-managed-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", path, err)
	}
	return runPrivilegedCommand(sudoMode, "install", []string{"-m", "0644", tempPath, path}, opts)
}
