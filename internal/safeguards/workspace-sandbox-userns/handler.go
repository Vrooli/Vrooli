package workspacesandboxuserns

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/apparmorprofiles"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	profileName = apparmorprofiles.WorkspaceSandboxName
)

var workspaceProfile, _ = apparmorprofiles.Lookup(string(apparmorprofiles.WorkspaceSandboxUserns))

const profileID = string(apparmorprofiles.WorkspaceSandboxUserns)

var profilePath, profileContent = workspaceProfile.Path, workspaceProfile.Content

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
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "workspace-sandbox user namespace launch safeguard is Linux-only")
		return status
	}

	status.Notes = append(status.Notes, sysctlNotes()...)
	if !hostreqkit.RunningAsRootFn() && commandSucceeds("unshare", "-U", "-m", "-r", "true") {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "plain `unshare -U -m -r true` succeeds; managed AppArmor profile is not needed on this host")
		return status
	}
	if appArmorProfileSucceeds() {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "managed AppArmor profile allows workspace-sandbox userns launch")
		return status
	}

	if !hostreqkit.FileContentMatches(profilePath, profileContent) {
		status.Notes = append(status.Notes, "managed AppArmor profile missing or stale at "+profilePath)
	} else {
		status.Notes = append(status.Notes, "managed AppArmor profile exists but is not loaded or did not pass validation")
	}
	status.Notes = append(status.Notes, "setup will install/load the profile and validate with `aa-exec -p "+profileName+" -- unshare -U -m -r true`")
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
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install and load "+profilePath)
		return status, nil
	}

	if _, ok := hostreqkit.ResolveCommand([]string{"aa-exec"}); !ok {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "aa-exec is required to enter the managed AppArmor profile")
		return status, nil
	}
	parser, ok := appArmorParserCommand()
	if !ok {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "apparmor_parser is required to load the managed AppArmor profile")
		return status, nil
	}

	if opts.SetupPrivilegeSession != nil {
		if err := opts.SetupPrivilegeSession.InstallAppArmorProfile(profileID); err != nil {
			return hostreqkit.FailureStatus(status, err), nil
		}
	} else {
		if err := hostreqkit.EnsureManagedDir(apparmorprofiles.ProfileDirectory, opts); err != nil {
			return hostreqkit.FailureStatus(status, err), nil
		}
		if err := hostreqkit.InstallManagedContent(profilePath, profileContent, opts); err != nil {
			return hostreqkit.FailureStatus(status, err), nil
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, parser, []string{"-r", profilePath}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "profile written but apparmor_parser failed: "+err.Error())
			return status, nil
		}
	}
	if !appArmorProfileSucceeds() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "profile loaded but validation command failed")
		return status, nil
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "workspace-sandbox AppArmor userns profile installed and validated")
	return status, nil
}

func appArmorParserCommand() (string, bool) {
	return hostreqkit.ResolveCommand(apparmorprofiles.ParserCandidates())
}

func appArmorProfileSucceeds() bool {
	return commandSucceeds("aa-exec", "-p", profileName, "--", "unshare", "-U", "-m", "-r", "true")
}

func commandSucceeds(name string, args ...string) bool {
	if _, ok := hostreqkit.ResolveCommand([]string{name}); !ok {
		return false
	}
	_, err := hostreqkit.CombinedOutputFn(name, args...)
	return err == nil
}

func sysctlNotes() []string {
	var notes []string
	for _, item := range []struct {
		path  string
		label string
	}{
		{"/proc/sys/kernel/unprivileged_userns_clone", "kernel.unprivileged_userns_clone"},
		{"/proc/sys/user/max_user_namespaces", "user.max_user_namespaces"},
		{"/proc/sys/kernel/apparmor_restrict_unprivileged_userns", "kernel.apparmor_restrict_unprivileged_userns"},
	} {
		value := readProcInt(item.path)
		if value == "" {
			continue
		}
		notes = append(notes, fmt.Sprintf("%s=%s", item.label, value))
	}
	return notes
}

func readProcInt(path string) string {
	data, err := hostreqkit.ReadFileFn(path)
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(data))
	if _, err := strconv.Atoi(value); err != nil {
		return ""
	}
	return value
}
