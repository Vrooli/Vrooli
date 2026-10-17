// Package bubblewrapuserns owns host AppArmor support for native bubblewrap
// sandboxes. Workspace Sandbox's service-launch profile is a separate boundary.
package bubblewrapuserns

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const profilePath = "/etc/apparmor.d/vrooli-bubblewrap"

const profileContent = `# Managed by Vrooli -- do not edit manually
abi <abi/4.0>,
include <tunables/global>

# Opt in only the distribution bubblewrap executable. Do not disable the
# host's restricted-unprivileged-userns setting or broaden the service profile.
profile vrooli-bubblewrap /{,usr/}bin/bwrap flags=(unconfined) {
  userns,
  include if exists <local/vrooli-bubblewrap>
}
`

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status
	}
	_, _, invokingUserKnown := hostreqkit.InvokingUserIDs()
	if requirement.Manual || (hostreqkit.RunningAsRootFn() && !invokingUserKnown) {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "validate bubblewrap as the invoking unprivileged account; root success does not prove userns readiness")
		return status
	}
	if err := probe(); err == nil {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "bubblewrap user, mount, PID and network namespaces work without elevated launch")
	} else {
		status.Notes = append(status.Notes, err.Error())
	}
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.SupportClass != hostreqkit.SupportSupported || status.Applied {
		return status, nil
	}
	fail := func(err error) (hostreqkit.ItemStatus, error) {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	// Recheck the target before granting host permissions; never attach a
	// profile to an arbitrary PATH executable or repair non-AppArmor failures.
	if _, err := systemBubblewrap(); err != nil {
		return fail(err)
	}
	restriction, err := hostreqkit.ReadFileFn("/proc/sys/kernel/apparmor_restrict_unprivileged_userns")
	if err != nil || strings.TrimSpace(string(restriction)) != "1" {
		return fail(fmt.Errorf("bubblewrap failed without the restricted AppArmor userns policy; no host policy changed"))
	}
	parser, ok := hostreqkit.ResolveCommand([]string{"apparmor_parser", "/usr/sbin/apparmor_parser", "/sbin/apparmor_parser"})
	if !ok {
		return fail(fmt.Errorf("apparmor_parser is required to load %s", profilePath))
	}
	if content, err := hostreqkit.ReadFileFn(profilePath); err == nil && string(content) != profileContent {
		return fail(fmt.Errorf("refusing to replace different existing policy at %s; inspect it before repair", profilePath))
	} else if err != nil && !os.IsNotExist(err) {
		return fail(fmt.Errorf("read existing policy: %w", err))
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "would install/load "+profilePath+"; global AppArmor restrictions remain enabled")
		return status, nil
	}
	if err := hostreqkit.InstallManagedContent(profilePath, profileContent, opts.SudoMode, opts); err != nil {
		return fail(err)
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, parser, []string{"-r", profilePath}, opts); err != nil {
		return fail(fmt.Errorf("profile written but not loaded: %w", err))
	}
	if err := probe(); err != nil {
		return fail(fmt.Errorf("profile loaded but unprivileged namespace validation failed: %w", err))
	}
	// Keep the diagnostic history, but do not present a repaired inspection
	// failure as a current failure alongside the successful validation.
	for i, note := range status.Notes {
		if strings.HasPrefix(note, "bubblewrap namespace probe:") {
			status.Notes[i] = "resolved pre-repair failure: " + note
		}
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "bubblewrap AppArmor profile installed, loaded and validated; global restrictions unchanged")
	return status, nil
}

func systemBubblewrap() (string, error) {
	path, err := hostreqkit.LookPathFn("bwrap")
	if err != nil {
		return "", fmt.Errorf("distribution bubblewrap is required: %w", err)
	}
	if path != "/usr/bin/bwrap" && path != "/bin/bwrap" {
		return "", fmt.Errorf("unsupported bubblewrap path %q; profile attaches only to /usr/bin/bwrap or /bin/bwrap", path)
	}
	return path, nil
}

func probe() error {
	path, err := systemBubblewrap()
	if err != nil {
		return err
	}
	if hostreqkit.RunningAsRootFn() {
		if _, _, ok := hostreqkit.InvokingUserIDs(); !ok {
			return fmt.Errorf("unprivileged invoking-user identity is required for namespace validation")
		}
	}
	command, args := hostreqkit.InvokingUserCommand(path, "--unshare-user", "--unshare-pid", "--unshare-net", "--die-with-parent", "--ro-bind", "/", "/", "--proc", "/proc", "--dev", "/dev", "--", "/bin/true")
	output, err := hostreqkit.CombinedOutputFn(command, args...)
	if err != nil {
		if len(output) > 2048 {
			output = output[:2048]
		}
		return fmt.Errorf("bubblewrap namespace probe: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
