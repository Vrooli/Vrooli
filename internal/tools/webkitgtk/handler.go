// Package webkitgtk owns the Linux WebKitGTK development-package check and
// installation path used by Wails desktop builds.
package webkitgtk

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var checkDevelopmentModules = func() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := []string{"--exists", "gtk+-3.0", "webkit2gtk-4.1", "libsoup-3.0"}
	return exec.CommandContext(ctx, "pkg-config", args...).Run() == nil
}

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
	if hostreqspec.PlatformFromGOOS(host.OS) != hostreqspec.PlatformLinux {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status
	}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, h.manifest.InstallHint)
		return status
	}

	status.PackageName = h.manifest.PackageNameForHost(host)
	status.SupportClass = hostreqkit.SupportSupported
	status.InstallSupported = strings.TrimSpace(status.PackageName) != ""
	status.Command, _ = hostreqkit.ResolveCommand(h.manifest.Commands)
	if !status.InstallSupported {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "no Linux package mapping is declared for the WebKitGTK development headers")
		return status
	}
	if checkDevelopmentModules() {
		status.Installed = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		if status.Command != "" {
			status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
		}
		return status
	}
	status.ExecutionState = hostreqkit.ExecutionPending
	status.Notes = append(status.Notes, h.manifest.InstallHint)
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	if status.SupportClass == hostreqkit.SupportNotApplicable {
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}
	if status.SupportClass == hostreqkit.SupportManualOnly {
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}
	if status.SupportClass == hostreqkit.SupportUnsupported {
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}

	command, args, err := hostreqkit.InstallCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, "dry-run: "+command+" "+strings.Join(args, " "))
		return status, nil
	}
	if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if !checkDevelopmentModules() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "package installation finished, but pkg-config cannot resolve gtk+-3.0, webkit2gtk-4.1, and libsoup-3.0")
		return status, nil
	}
	status.Command, _ = hostreqkit.ResolveCommand(h.manifest.Commands)
	status.Installed = true
	status.ExecutionState = hostreqkit.ExecutionInstalled
	if status.Command != "" {
		status.Version = hostreqkit.ReadVersion(status.Command, h.manifest.VersionArgs)
	}
	return status, nil
}
