package runtime

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type toolHandler struct {
	manifest hostreqkit.ToolManifest
}

func newGenericToolHandler(m hostreqkit.ToolManifest) hostreqkit.Handler {
	return toolHandler{manifest: m}
}

func (h toolHandler) Name() string           { return h.manifest.Name }
func (h toolHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h toolHandler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.PackageName = h.manifest.PackageNameForHost(host)
	command, installed := hostreqkit.ResolveCommand(h.manifest.Commands)
	status.Command = command
	status.Installed = installed
	status.SupportClass = hostreqkit.SupportSupported
	status.InstallSupported = strings.TrimSpace(status.PackageName) != "" && !requirement.Manual
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.InstallSupported = false
	}
	if installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	}
	if !installed {
		if status.SupportClass == hostreqkit.SupportSupported && strings.TrimSpace(status.PackageName) == "" {
			status.SupportClass = hostreqkit.SupportUnsupported
			status.ExecutionState = hostreqkit.ExecutionUnsupported
		}
		if h.manifest.InstallHint != "" {
			status.Notes = append(status.Notes, h.manifest.InstallHint)
		}
		return status
	}

	version := hostreqkit.ReadVersion(command, h.manifest.VersionArgs)
	if version != "" {
		status.Version = version
	}
	if passed, detail := hostreqkit.RunVerificationCheck(h.manifest.VerificationCheck); !passed {
		status.Notes = append(status.Notes, detail)
	}
	return status
}

func (h toolHandler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return status, nil
	}
	if !status.InstallSupported || strings.TrimSpace(status.PackageName) == "" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	}
	command, args, err := hostreqkit.InstallCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.Notes = append(status.Notes, err.Error())
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would run %s %s", command, strings.Join(args, " ")))
		return status, nil
	}
	if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	commandName, installed := hostreqkit.ResolveCommand(h.manifest.Commands)
	status.Command = commandName
	status.Installed = installed
	if installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		status.Version = hostreqkit.ReadVersion(commandName, h.manifest.VersionArgs)
		if passed, detail := hostreqkit.RunVerificationCheck(h.manifest.VerificationCheck); !passed {
			status.Notes = append(status.Notes, detail)
		}
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install command completed but the tool is still not available on PATH")
	return status, nil
}
