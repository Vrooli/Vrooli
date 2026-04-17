package kernelconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const configPath = "/etc/sysctl.d/99-vrooli.conf"

var managedParameters = []struct {
	Name     string
	MinValue int
}{
	{"fs.inotify.max_user_watches", 1048576},
	{"fs.inotify.max_user_instances", 2048},
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

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
		status.Notes = append(status.Notes, "kernel parameter management is only supported on Linux")
		return status
	}

	if !host.SupportsSysctl {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not support sysctl")
		return status
	}

	pending := make([]string, 0, len(managedParameters))
	for _, p := range managedParameters {
		current := readSysctlValue(p.Name)
		if current < p.MinValue {
			pending = append(pending, fmt.Sprintf("%s=%d (current: %d, minimum: %d)", p.Name, p.MinValue, current, p.MinValue))
		}
	}

	if !hostreqkit.FileContentMatches(configPath, buildConfigContent()) {
		if len(pending) == 0 {
			pending = append(pending, configPath+" needs update")
		}
	}

	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "all kernel parameters meet minimum values")
		return status
	}

	status.Notes = append(status.Notes, "kernel parameters below minimum values")
	status.Notes = append(status.Notes, "pending: "+strings.Join(pending, ", "))
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
		status.Notes = append(status.Notes, "dry-run: would configure kernel parameters in "+configPath)
		return status, nil
	}

	if err := hostreqkit.EnsureManagedDir("/etc/sysctl.d", opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.InstallManagedContent(configPath, buildConfigContent(), opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "sysctl", []string{"-p", configPath}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "config written but sysctl -p failed: "+err.Error())
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "kernel parameters configured and applied")
	return status, nil
}

func readSysctlValue(param string) int {
	procPath := "/proc/sys/" + strings.ReplaceAll(param, ".", "/")
	data, err := hostreqkit.ReadFileFn(procPath)
	if err != nil {
		return 0
	}
	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return val
}

func buildConfigContent() string {
	var b strings.Builder
	b.WriteString("# Managed by Vrooli -- do not edit manually\n")
	for _, p := range managedParameters {
		fmt.Fprintf(&b, "%s = %d\n", p.Name, p.MinValue)
	}
	return b.String()
}
