// Package netconsole installs the kernel's netconsole module so kernel-level
// oops/panic output streams over UDP to a remote log collector. This is the
// only way to capture pre-crash console messages when the local journald is
// killed mid-write — the symptom of the unexplained hard resets that
// motivated this safeguard.
//
// Target host configuration is supplied through the declared `target`
// parameter because there is no sensible default — each operator's box has a
// different log collector. When unset, the safeguard reports NotApplicable
// rather than loading the module with no destination.
package netconsole

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/modules"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// ModuleName matches the kernel module's name; we pass it through to the
// modules helper for both the at-boot config files and the live modprobe call.
const ModuleName = "netconsole"

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the constructor referenced by customSafeguardHandlers in
// internal/runtime/registry.go. The manifest is plumbed through Name()/Kind()
// reporting so the runtime registry's lookup paths see consistent values.
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
		status.Notes = append(status.Notes, "netconsole is a Linux-only kernel module")
		return status
	}

	target, configured := requirement.ConfigString("target")
	target = strings.TrimSpace(target)
	if !configured || target == "" {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, fmt.Sprintf(
			"netconsole skipped: target is not set. Set it to the kernel netconsole target string "+
				"(format: src-port@src-ip/dev,tgt-port@tgt-ip/tgt-mac) to enable.",
		))
		return status
	}

	wantOptions := map[string]string{ModuleName: target}
	loadMatch := hostreqkit.FileContentMatches(modules.LoadFilePath(ModuleName), expectedLoadContent())
	optMatch := hostreqkit.FileContentMatches(modules.OptionsFilePath(ModuleName), expectedOptionsContent(target))
	loaded := modules.IsLoaded(ModuleName)

	if loadMatch && optMatch && loaded {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("netconsole loaded with target=%s", target))
		_ = wantOptions
		return status
	}

	missing := []string{}
	if !loadMatch {
		missing = append(missing, modules.LoadFilePath(ModuleName))
	}
	if !optMatch {
		missing = append(missing, modules.OptionsFilePath(ModuleName))
	}
	if !loaded {
		missing = append(missing, "/sys/module/"+ModuleName)
	}
	status.Notes = append(status.Notes, "netconsole pending: "+strings.Join(missing, ", "))
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
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	target, configured := configString(status.Config, "target")
	target = strings.TrimSpace(target)
	if !configured || target == "" {
		// Belt-and-suspenders: Inspect should have caught this, but Apply may
		// be called with a stale status whose env was set during Inspect and
		// unset before Apply.
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "apply skipped: target parameter is unset")
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf(
			"dry-run: would write %s, %s, then run modprobe netconsole netconsole=%s",
			modules.LoadFilePath(ModuleName), modules.OptionsFilePath(ModuleName), target))
		return status, nil
	}

	options := map[string]string{ModuleName: target}

	if _, err := modules.EnsureLoadAtBoot(ModuleName, options, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("ensure at-boot load failed: %s", err))
		return status, nil
	}

	// Live activation. Re-running modprobe with options against an already-
	// loaded netconsole module fails with "module already loaded" — this is
	// acceptable; we only Modprobe when not yet loaded.
	if !modules.IsLoaded(ModuleName) {
		if err := modules.Modprobe(ModuleName, options, opts.SudoMode, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("modprobe netconsole failed: %s", err))
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, fmt.Sprintf("netconsole streaming to %s", target))
	return status, nil
}

func expectedLoadContent() string {
	return modules.ManagedHeader + "\n" + ModuleName + "\n"
}

func expectedOptionsContent(target string) string {
	return fmt.Sprintf("%s\noptions %s %s=%s\n", modules.ManagedHeader, ModuleName, ModuleName, target)
}

func configString(config map[string]any, key string) (string, bool) {
	value, ok := config[key].(string)
	return value, ok
}
