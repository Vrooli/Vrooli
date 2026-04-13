package runtime

import (
	"fmt"
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

	scriptLines := []string{"set -e"}
	if host.SupportsSysctl {
		scriptLines = append(scriptLines,
			fmt.Sprintf("mkdir -p %s", shellQuotePath(parentDir(remoteSessionSysctlPath))),
			writeFileShellSnippet(remoteSessionSysctlPath, remoteSessionSysctlContent),
			fmt.Sprintf("sysctl -p %s >/dev/null", shellQuotePath(remoteSessionSysctlPath)),
		)
	}
	if host.SupportsSystemd {
		scriptLines = append(scriptLines,
			fmt.Sprintf("mkdir -p %s %s", shellQuotePath(remoteSessionSystemdDir), shellQuotePath(remoteSessionLogindDir)),
			writeFileShellSnippet(remoteSessionSystemdPath, remoteSessionUnitContent),
			writeFileShellSnippet(remoteSessionLogindPath, remoteSessionLogindContent),
			"systemctl daemon-reload >/dev/null",
		)
	}
	script := strings.Join(scriptLines, "\n")

	if opts.DryRun {
		status.ExecutionState = ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would apply remote session protection")
		return status, nil
	}

	if err := runShellScript(script, opts.SudoMode, opts); err != nil {
		status.ExecutionState = ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
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

func writeFileShellSnippet(path, content string) string {
	return fmt.Sprintf("cat <<'EOF' > %s\n%sEOF", shellQuotePath(path), content)
}

func shellQuotePath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

func parentDir(path string) string {
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return "."
	}
	return path[:index]
}
