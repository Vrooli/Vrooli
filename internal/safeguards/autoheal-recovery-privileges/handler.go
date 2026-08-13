// Package autohealrecoveryprivileges provisions autoheal's fixed service
// recovery argv at setup time. Runtime autoheal never asks for a password.
package autohealrecoveryprivileges

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	sudoersPath   = "/etc/sudoers.d/vrooli-autoheal"
	systemctlPath = "/usr/bin/systemctl"
)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if host.OS != "linux" {
		return unsupported(status, "autoheal recovery privileges use Linux sudoers; use the native service mechanism on this platform")
	}
	user := hostreqkit.InvokingUser()
	if user == "" {
		return unsupported(status, "could not resolve the invoking user for a scoped grant")
	}
	if hostreqkit.FileContentMatches(sudoersPath, buildSudoersContent(user)) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status
	}
	if hostreqkit.RunningAsRootFn() {
		status.Required = true
	} else {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	}
	status.Notes = append(status.Notes, "autoheal recovery grant missing at "+sudoersPath+"; run sudo vrooli setup")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.SupportClass == hostreqkit.SupportNotApplicable {
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}
	if status.SupportClass == hostreqkit.SupportUnsupported || status.ExecutionState == hostreqkit.ExecutionManualActionRequired {
		if status.SupportClass == hostreqkit.SupportUnsupported {
			status.ExecutionState = hostreqkit.ExecutionUnsupported
		}
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	user := hostreqkit.InvokingUser()
	if user == "" {
		return status, fmt.Errorf("autoheal recovery privileges: invoking user is empty")
	}
	content := buildSudoersContent(user)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would validate and install "+sudoersPath)
		return status, nil
	}
	tmp, err := hostreqkit.WriteTempFileFn(content)
	if err != nil {
		return status, fmt.Errorf("prepare sudoers grant: %w", err)
	}
	defer os.Remove(tmp)
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "visudo", []string{"-c", "-f", tmp}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "sudoers validation failed; real drop-in was not touched: "+err.Error())
		return status, nil
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "install", []string{"-m", "0440", tmp, sudoersPath}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, nil
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	return status, nil
}

func unsupported(status hostreqkit.ItemStatus, note string) hostreqkit.ItemStatus {
	status.SupportClass = hostreqkit.SupportUnsupported
	status.ExecutionState = hostreqkit.ExecutionUnsupported
	status.Notes = append(status.Notes, note)
	return status
}

func buildSudoersContent(user string) string {
	user = strings.TrimSpace(user)
	units := []string{
		"docker", "systemd-resolved", "cloudflared", "NetworkManager",
		"systemd-networkd", "systemd-timesyncd", "gnome-remote-desktop",
		"gnome-remote-desktop.service", "xrdp", "gdm", "gdm3", "lightdm", "sddm",
	}
	commands := make([]string, 0, len(units)*2)
	for _, unit := range units {
		commands = append(commands,
			fmt.Sprintf("%s start %s", systemctlPath, unit),
			fmt.Sprintf("%s restart %s", systemctlPath, unit),
		)
	}
	return fmt.Sprintf("# Managed by Vrooli -- do not edit manually\n%s ALL=(root) NOPASSWD: %s\n", user, strings.Join(commands, ", "))
}
