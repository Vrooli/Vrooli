// Package autohealwatchdog owns project-setup installation of autoheal's

// native user scheduler. The scenario may observe this state, but it does not
// own a second host-repair implementation.
package autohealwatchdog

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const (
	handlerDedicated = "dedicated"
)

const (
	serviceName   = "vrooli-autoheal.service"
	defaultPolicy = handlerDedicated
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
		// Build from inside the loop's own module directory.
		//
		// `go -C <scenario> build ./cli/loop` does not work: the scenario
		// directory has no go.mod, so -C resolves to the repo-root module,
		// which does not contain that package, and the build fails with
		// "main module does not contain package .../cli/loop". The loop is a
		// separate module (vrooli-autoheal-loop) and must be built as one.
		//
		// This mattered: the failure left the watchdog binary frozen at
		// whatever build happened to be on disk while its source moved on, so
		// fixes to the loop never shipped.
		loopDir := filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "cli", "loop")
		return hostreqkit.RunAsInvokingUser("go", []string{"-C", loopDir, "build", "-o", output, "."}, opts)
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
	// A stale binary must keep the safeguard un-applied, not merely annotated.
	// Apply() returns early when status.Applied is true, so reporting
	// "already present" over superseded code is how a fix to the watchdog
	// stops shipping while every signal still reads green.
	loopStale, loopStaleReason := loopBinaryStale(root, loopPath(root, host.OS))
	if loopStale {
		status.Notes = append(status.Notes, "autoheal loop binary needs building ("+loopStaleReason+"); setup will build it")
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
	if bootPolicy(requirement.Config) == handlerDedicated && hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && !lingeringEnabledFn(hostreqkit.InvokingUser()) {
		status.Notes = append(status.Notes, "user service is enabled but boot protection is incomplete: Linux lingering is disabled", recoveryNote())
		return status
	}
	if loopStale {
		// Scheduler is healthy but supervising superseded code: not applied.
		status.Notes = append(status.Notes, "autoheal scheduler is active but its binary is stale; setup will rebuild and restart it")
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
	// Rebuild when the binary is missing OR older than its sources. Presence
	// alone is not enough: the watchdog is a long-lived process that nothing
	// else rebuilds, so a stale binary means every fix to the loop -- including
	// its dependency-drift recovery floor -- silently never ships.
	rebuiltLoop := false
	if stale, reason := loopBinaryStale(root, loop); stale {
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes, "dry-run: would build "+loop+" ("+reason+")")
			return status, nil
		}
		if err := buildLoopFn(root, loop, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal loop build failed: "+err.Error())
			return status, nil
		}
		status.Notes = append(status.Notes, "rebuilt autoheal loop binary ("+reason+")")
		rebuiltLoop = true
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
	// `enable --now` starts a stopped unit but leaves a running one on the old
	// executable, so a rebuild only reaches the supervised process after an
	// explicit restart.
	if rebuiltLoop {
		if err := restartScheduler(host.OS, name, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal scheduler restart after rebuild failed: "+err.Error(), recoveryNote())
			return status, nil
		}
		status.Notes = append(status.Notes, "restarted autoheal scheduler onto the rebuilt binary")
	}

	if hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && bootPolicy(status.Config) == handlerDedicated {
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
	return isMacOS(host.OS) || hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformWindows || (hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && host.SupportsSystemd)
}

func isMacOS(osName string) bool {
	return osName == string(hostreqspec.PlatformDarwin) || osName == "macos"
}

func guiLaunchdAvailable() bool {
	uid := "0"
	if current, err := user.Current(); err == nil && current.Uid != "" {
		uid = current.Uid
	}
	_, err := commandOutputFn("launchctl", "print", "gui/"+uid)
	return err == nil
}

func bootPolicy(config map[string]any) string {
	if value, ok := config["boot_policy"].(string); ok && (value == "shared" || value == handlerDedicated) {
		return value
	}
	return defaultPolicy
}

func recoveryNote() string {
	return "recovery: rerun `vrooli setup --sudo-mode=ask` (or `vrooli setup` when the grant is already in place)"
}

func loopPath(root, osName string) string {
	name := "vrooli-autoheal-loop"
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformWindows {
		name += ".exe"
	}
	return filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal", "cli", name)
}

func nativeDefinition(osName, root, home string) (string, string, string, error) {
	name := serviceName
	path := filepath.Join(home, ".config", "systemd", "user", name)
	if isMacOS(osName) {
		name = "com.vrooli.autoheal"
		path = filepath.Join(home, "Library", "LaunchAgents", name+".plist")
	}
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformWindows {
		name = "VrooliAutoheal"
		path = filepath.Join(os.TempDir(), "vrooli-autoheal.xml")
	}
	renderTarget := osName
	if isMacOS(osName) {
		renderTarget = "macos"
	}
	content, err := platformgo.RenderWatchdogDefinition(renderTarget, platformgo.WatchdogDefinitionOptions{Root: root, Home: home, LoopBinary: loopPath(root, osName), VrooliBinary: filepath.Join(home, repocontractmeta.ProjectConfigDir, "bin", "vrooli")})
	return content, path, name, err
}

func asUserArgs(name string, args ...string) (string, []string) {
	return hostreqkit.InvokingUserCommand(name, args...)
}

func schedulerState(osName, name string) (bool, bool, error) {
	if hostreqspec.PlatformFromGOOS(osName) != hostreqspec.PlatformLinux {
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
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformLinux {
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

// loopSourceDirs lists the directories whose contents the loop binary is built
// from, relative to the repo root. langrecover is included because the loop
// depends on it for dependency-drift recovery: a change there must reach the
// watchdog, or the recovery floor silently runs old logic.
func loopSourceDirs(root string) []string {
	scenario := filepath.Join(root, repocontractmeta.ScenarioDir, "vrooli-autoheal")
	return []string{
		filepath.Join(scenario, "cli", "loop"),
		filepath.Join(scenario, "langrecover"),
	}
}

// loopBinaryStale reports whether the loop binary needs rebuilding, and why.
//
// A missing binary is unambiguous. Otherwise the binary is compared against
// the newest source file in its module and its in-repo dependencies; if the
// sources are newer, the running watchdog would be executing superseded code.
// On any error reading the sources the answer is "not stale", so a probe
// failure never triggers an unnecessary build.
func loopBinaryStale(root, loop string) (bool, string) {
	info, err := os.Stat(loop)
	if err != nil {
		return true, "binary missing"
	}
	builtAt := info.ModTime()

	newest, newestPath, err := newestSourceModTime(loopSourceDirs(root))
	if err != nil || newestPath == "" {
		return false, ""
	}
	if newest.After(builtAt) {
		return true, "binary older than " + filepath.Base(newestPath)
	}
	return false, ""
}

// newestSourceModTime returns the most recent modification time among Go
// source and module files under the supplied directories.
func newestSourceModTime(dirs []string) (time.Time, string, error) {
	var (
		newest     time.Time
		newestPath string
	)
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			name := entry.Name()
			// Test files do not affect the built binary.
			if strings.HasSuffix(name, "_test.go") {
				return nil
			}
			if !strings.HasSuffix(name, ".go") && name != "go.mod" && name != "go.sum" {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.ModTime().After(newest) {
				newest, newestPath = info.ModTime(), path
			}
			return nil
		})
		if err != nil {
			return time.Time{}, "", err
		}
	}
	return newest, newestPath, nil
}

// restartScheduler restarts the native scheduler unit so a freshly built
// binary actually replaces the running process.
func restartScheduler(osName, name string, opts hostreqkit.EnsureOptions) error {
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformLinux {
		cmd, args := asUserArgs("systemctl", "--user", "restart", name)
		return hostreqkit.RunCommandFn(cmd, args, opts)
	}
	if isMacOS(osName) {
		uid := "0"
		if current, err := user.Current(); err == nil && current.Uid != "" {
			uid = current.Uid
		}
		// kickstart -k kills the running job and restarts it in one step.
		cmd, args := asUserArgs("launchctl", "kickstart", "-k", "gui/"+uid+"/"+strings.TrimSuffix(name, ".plist"))
		return hostreqkit.RunCommandFn(cmd, args, opts)
	}
	if err := hostreqkit.RunCommandFn("schtasks", []string{"/End", "/TN", name}, opts); err != nil {
		return err
	}
	return hostreqkit.RunCommandFn("schtasks", []string{"/Run", "/TN", name}, opts)
}
