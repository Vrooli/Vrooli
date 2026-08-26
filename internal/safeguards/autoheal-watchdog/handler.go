// Package autohealwatchdog owns project-setup installation of autoheal's
// native user scheduler. The scenario may observe this state, but it does not
// own a second host-repair implementation.
package autohealwatchdog

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	serviceName   = "vrooli-autoheal.service"
	defaultPolicy = "dedicated"
)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

var (
	resolveRootFn      = repocontract.ResolveRepoRoot
	userHomeFn         = hostreqkit.InvokingUserHomeDir
	commandOutputFn    = func(name string, args ...string) ([]byte, error) { return hostreqkit.CombinedOutputFn(name, args...) }
	lingeringEnabledFn = lingeringEnabled
	buildLoopFn        = func(root, output string, opts hostreqkit.EnsureOptions) error {
		return hostreqkit.RunAsInvokingUser("go", []string{"-C", filepath.Join(root, "scenarios", "vrooli-autoheal"), "build", "-o", output, "./cli/loop"}, opts)
	}
)

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if !supported(host) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "autoheal watchdog has no usable native scheduler on "+host.OS)
		return status
	}
	if isMacOS(host.OS) && !guiLaunchdAvailable() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "the invoking user's GUI launchd domain is unavailable; this user LaunchAgent is not applicable to the current SSH/headless session")
		return status
	}
	home, err := userHomeFn()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status
	}
	root, err := resolveRootFn()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine Vrooli root: "+err.Error())
		return status
	}
	definition, path, name, err := nativeDefinition(host.OS, root, home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status
	}
	if _, err := os.Stat(loopPath(root, host.OS)); err != nil {
		status.Notes = append(status.Notes, "autoheal loop binary missing; setup will build it")
	}
	if !hostreqkit.FileContentMatches(path, definition) {
		status.Notes = append(status.Notes, "autoheal native scheduler definition is missing or stale")
	}
	enabled, active, probeErr := schedulerState(host.OS, name)
	if probeErr != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "native scheduler bus unavailable: "+probeErr.Error(), recoveryNote())
		return status
	}
	if !enabled || !active {
		status.Notes = append(status.Notes, fmt.Sprintf("autoheal scheduler pending: enabled=%t active=%t", enabled, active))
		return status
	}
	if bootPolicy(requirement.Config) == "dedicated" && host.OS == "linux" && !lingeringEnabledFn(hostreqkit.InvokingUser()) {
		status.Notes = append(status.Notes, "user service is enabled but boot protection is incomplete: Linux lingering is disabled", recoveryNote())
		return status
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "autoheal scheduler enabled and active; boot protection verified for "+bootPolicy(requirement.Config)+" policy")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.SupportClass != hostreqkit.SupportSupported || status.Applied {
		return status, nil
	}
	home, err := userHomeFn()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		return status, nil
	}
	root, err := resolveRootFn()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	loop := loopPath(root, host.OS)
	if _, err := os.Stat(loop); err != nil {
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes, "dry-run: would build "+loop)
			return status, nil
		}
		if err := buildLoopFn(root, loop, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal loop build failed: "+err.Error())
			return status, nil
		}
	}
	definition, path, name, err := nativeDefinition(host.OS, root, home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install and enable "+path)
		return status, nil
	}
	if err := installAsInvokingUser(path, definition, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if err := enableScheduler(host.OS, name, path, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "native scheduler enable failed: "+err.Error(), recoveryNote())
		return status, nil
	}
	if host.OS == "linux" && bootPolicy(status.Config) == "dedicated" {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "loginctl", []string{"enable-linger", hostreqkit.InvokingUser()}, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "Linux lingering enable failed: "+err.Error(), recoveryNote())
			return status, nil
		}
	}
	verified := h.Inspect(host, hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required, Config: status.Config})
	if !verified.Applied {
		verified.ExecutionState = hostreqkit.ExecutionFailed
		verified.Notes = append(verified.Notes, "scheduler mutation completed but verification did not prove protection", recoveryNote())
	}
	return verified, nil
}

func supported(host hostreqkit.Host) bool {
	return isMacOS(host.OS) || host.OS == "windows" || (host.OS == "linux" && host.SupportsSystemd)
}

func isMacOS(osName string) bool { return osName == "darwin" || osName == "macos" }

func guiLaunchdAvailable() bool {
	uid := "0"
	if current, err := user.Current(); err == nil && current.Uid != "" {
		uid = current.Uid
	}
	_, err := commandOutputFn("launchctl", "print", "gui/"+uid)
	return err == nil
}

func bootPolicy(config map[string]any) string {
	if value, ok := config["boot_policy"].(string); ok && (value == "shared" || value == "dedicated") {
		return value
	}
	return defaultPolicy
}

func recoveryNote() string {
	return "recovery: rerun `vrooli setup --sudo-mode=ask` (or `vrooli setup` when the grant is already in place)"
}

func loopPath(root, osName string) string {
	name := "vrooli-autoheal-loop"
	if osName == "windows" {
		name += ".exe"
	}
	return filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", name)
}

func nativeDefinition(osName, root, home string) (string, string, string, error) {
	name := serviceName
	path := filepath.Join(home, ".config", "systemd", "user", name)
	if isMacOS(osName) {
		name = "com.vrooli.autoheal"
		path = filepath.Join(home, "Library", "LaunchAgents", name+".plist")
	}
	if osName == "windows" {
		name = "VrooliAutoheal"
		path = filepath.Join(os.TempDir(), "vrooli-autoheal.xml")
	}
	renderTarget := osName
	if isMacOS(osName) {
		renderTarget = "macos"
	}
	content, err := platformgo.RenderWatchdogDefinition(renderTarget, platformgo.WatchdogDefinitionOptions{Root: root, Home: home, LoopBinary: loopPath(root, osName), VrooliBinary: filepath.Join(home, ".vrooli", "bin", "vrooli")})
	return content, path, name, err
}

func asUserArgs(name string, args ...string) (string, []string) {
	return hostreqkit.InvokingUserCommand(name, args...)
}

func schedulerState(osName, name string) (bool, bool, error) {
	if osName != "linux" {
		return true, true, nil
	}
	cmd, args := asUserArgs("systemctl", "--user", "is-enabled", name)
	enabledOut, enabledErr := commandOutputFn(cmd, args...)
	cmd, args = asUserArgs("systemctl", "--user", "is-active", name)
	activeOut, activeErr := commandOutputFn(cmd, args...)
	if enabledErr != nil || activeErr != nil {
		return false, false, fmt.Errorf("systemctl user bus: %s", strings.TrimSpace(string(append(enabledOut, activeOut...))))
	}
	return strings.TrimSpace(string(enabledOut)) == "enabled", strings.TrimSpace(string(activeOut)) == "active", nil
}

func enableScheduler(osName, name, path string, opts hostreqkit.EnsureOptions) error {
	if osName == "linux" {
		for _, args := range [][]string{{"--user", "daemon-reload"}, {"--user", "enable", "--now", name}} {
			cmd, wrapped := asUserArgs("systemctl", args...)
			if err := hostreqkit.RunCommandFn(cmd, wrapped, opts); err != nil {
				return err
			}
		}
		return nil
	}
	if isMacOS(osName) {
		uid := "0"
		if current, err := user.Current(); err == nil && current.Uid != "" {
			uid = current.Uid
		}
		cmd, args := asUserArgs("launchctl", "bootstrap", "gui/"+uid, path)
		return hostreqkit.RunCommandFn(cmd, args, opts)
	}
	return hostreqkit.RunCommandFn("schtasks", []string{"/Create", "/TN", name, "/XML", path, "/F"}, opts)
}

func installAsInvokingUser(path, content string, opts hostreqkit.EnsureOptions) error {
	return hostreqkit.InstallUserFile(path, content, opts)
}

func lingeringEnabled(user string) bool {
	if user == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join("/var/lib/systemd/linger", user)); err == nil {
		return true
	}
	out, err := commandOutputFn("loginctl", "show-user", user, "--property=Linger")
	return err == nil && strings.TrimSpace(string(out)) == "Linger=yes"
}
