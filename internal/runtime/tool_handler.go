package runtime

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

type toolHandler struct {
	name           string
	commands       []string
	versionArgs    []string
	defaultPackage string
	packageNames   map[string]string
	installHint    string
}

func newToolHandler(name string, commands, versionArgs []string, defaultPackage string, packageNames map[string]string, installHint string) toolHandler {
	return toolHandler{
		name:           name,
		commands:       append([]string(nil), commands...),
		versionArgs:    append([]string(nil), versionArgs...),
		defaultPackage: defaultPackage,
		packageNames:   packageNames,
		installHint:    installHint,
	}
}

func (h toolHandler) Name() string       { return h.name }
func (h toolHandler) Kind() hostreq.Kind { return hostreq.KindTool }

func (h toolHandler) Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	status := baseStatus(requirement)
	status.PackageName = h.packageNameForHost(host)
	command, installed := resolveCommand(h.commands)
	status.Command = command
	status.Installed = installed
	status.SupportClass = SupportSupported
	status.InstallSupported = strings.TrimSpace(status.PackageName) != "" && !requirement.Manual
	if requirement.Manual {
		status.SupportClass = SupportManualOnly
		status.ExecutionState = ExecutionManualActionRequired
		status.InstallSupported = false
	}
	if installed {
		status.ExecutionState = ExecutionAlreadyPresent
	}
	if !installed {
		if status.SupportClass == SupportSupported && strings.TrimSpace(status.PackageName) == "" {
			status.SupportClass = SupportUnsupported
			status.ExecutionState = ExecutionUnsupported
		}
		if h.installHint != "" {
			status.Notes = append(status.Notes, h.installHint)
		}
		return status
	}

	version := readVersion(command, h.versionArgs)
	if version != "" {
		status.Version = version
	}
	return status
}

func (h toolHandler) Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case SupportManualOnly:
		status.ExecutionState = ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return status, nil
	case SupportUnsupported:
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	case SupportNotApplicable:
		status.ExecutionState = ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return status, nil
	}
	if !status.InstallSupported || strings.TrimSpace(status.PackageName) == "" {
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	}
	command, args, err := installCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.Notes = append(status.Notes, err.Error())
		status.SupportClass = SupportUnsupported
		status.ExecutionState = ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = ExecutionWouldInstall
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would run %s %s", command, strings.Join(args, " ")))
		return status, nil
	}
	if err := runInstallCommand(command, args, opts); err != nil {
		status.ExecutionState = ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	commandName, installed := resolveCommand(h.commands)
	status.Command = commandName
	status.Installed = installed
	if installed {
		status.ExecutionState = ExecutionInstalled
		status.Version = readVersion(commandName, h.versionArgs)
		return status, nil
	}
	status.ExecutionState = ExecutionFailed
	status.Notes = append(status.Notes, "install command completed but the tool is still not available on PATH")
	return status, nil
}

func (h toolHandler) packageNameForHost(host Host) string {
	if value, ok := h.packageNames[host.PackageManager]; ok {
		return value
	}
	return h.defaultPackage
}
