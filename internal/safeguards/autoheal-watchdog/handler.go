// Package autohealwatchdog owns project-setup installation of autoheal's

// native user scheduler. The scenario may observe this state, but it does not
// own a second host-repair implementation.
package autohealwatchdog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const (
	handlerDedicated = "dedicated"
)

const (
	serviceName      = "vrooli-autoheal.service"
	defaultPolicy    = handlerDedicated
	autohealScenario = "vrooli-autoheal"
	loopBinaryName   = "vrooli-autoheal-loop"

	// Loop freshness verdicts. The safeguard never computes freshness itself:
	// it reads the manifest the lifecycle engine stamped when it built the
	// loop component and reports what that manifest says.
	verdictFresh   = "fresh"
	verdictStale   = "stale"
	verdictUnknown = "unknown"
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
	// setupScenarioFn rebuilds the loop the only way anything in this
	// repository builds a scenario component: through the lifecycle engine.
	// The safeguard used to run `go build` itself from an mtime comparison,
	// which made it a second freshness implementation with its own bugs (the
	// 2026-09-01 "binary months older than its source, reported Already
	// present" case). Now the engine owns building and stamping; the safeguard
	// verifies the stamp and restarts the unit.
	setupScenarioFn = setupScenarioThroughLifecycle
	// procRoot is the procfs mount used to identify the running unit's
	// executable. Tests point it at a fixture tree.
	procRoot = "/proc"
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
	artifact, path, name, err := nativeArtifact(host.OS, root, home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status
	}
	definition := artifact.Primary().Content
	recordVerdict(&status, platformgo.ValidateArtifact(artifact, platformgo.ScopeUser))
	runState := unitRunState(host.OS, name)
	if len(runState) > 0 {
		status.Evidence["unit_state"] = runState
		if runState["Result"] != "" && runState["Result"] != "success" {
			status.Notes = append(status.Notes, "autoheal unit last result is "+runState["Result"]+" after "+runState["NRestarts"]+" restarts")
		}
	}
	// A stale binary must keep the safeguard un-applied, not merely annotated.
	// Apply() returns early when status.Applied is true, so reporting
	// "already present" over superseded code is how a fix to the watchdog
	// stops shipping while every signal still reads green.
	loop := loopPath(root, host.OS)
	loopVerdict := recordLoopFreshness(&status, root, loop)
	// The binary on disk can be fresh while the process the unit is running
	// was exec'd from an older build: setup rebuilt the file and nothing
	// restarted the unit. That process is the one protecting boot, so it is
	// the one that has to match.
	processStale := recordProcessIdentity(&status, host.OS, loop, runState)
	// A unit that no longer matches its definition is not applied, however
	// healthy it looks: the 2026-09-02 boot ran a unit rendered two weeks
	// earlier with an argv the CLI had retired, and every signal read green.
	definitionStale := !hostreqkit.FileContentMatches(path, definition)
	if definitionStale {
		status.Notes = append(status.Notes, "autoheal native scheduler definition is missing or stale at "+path+"; setup will re-render and restart it")
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
	if loopVerdict != verdictFresh {
		// Scheduler is healthy but supervising code the engine has not
		// proven current: not applied.
		status.Notes = append(status.Notes, "autoheal scheduler is active but its binary is "+loopVerdict+"; setup will rebuild it through the lifecycle engine and restart the unit")
		return status
	}
	if processStale {
		return status
	}
	if definitionStale {
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
	// Rebuild when the engine's manifest says the binary is stale, and also
	// when there is no manifest to say anything: an unproven binary is not a
	// fresh one. The rebuild is `vrooli scenario setup vrooli-autoheal`, so
	// the loop is built and stamped by the same engine as the API.
	rebuiltLoop := false
	if verdict, reason := loopBinaryVerdict(root, loop); verdict != verdictFresh {
		if opts.DryRun {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			status.Notes = append(status.Notes, "dry-run: would run `vrooli scenario setup "+autohealScenario+"` to rebuild "+loop+" ("+verdict+": "+reason+")")
			return status, nil
		}
		before, _ := fileSHA256(loop)
		if err := setupScenarioFn(root, home, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal loop rebuild through `vrooli scenario setup "+autohealScenario+"` failed: "+err.Error())
			return status, nil
		}
		after, err := fileSHA256(loop)
		if err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal loop binary is missing after scenario setup: "+err.Error())
			return status, nil
		}
		if after != before {
			rebuiltLoop = true
			status.Notes = append(status.Notes, "rebuilt autoheal loop binary through the lifecycle engine ("+verdict+": "+reason+"); sha256 "+shortDigest(before)+" -> "+shortDigest(after))
		} else {
			status.Notes = append(status.Notes, "lifecycle engine re-verified the autoheal loop binary ("+verdict+": "+reason+"); sha256 unchanged "+shortDigest(after))
		}
	}
	artifact, path, name, err := nativeArtifact(host.OS, root, home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	definition := artifact.Primary().Content
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install and enable "+path)
		return status, nil
	}
	definitionChanged := !hostreqkit.FileContentMatches(path, definition)
	// Ask the native manager whether it would load the render before the
	// unit directory is touched: a rejected render must never replace a
	// working unit.
	verdict := platformgo.ValidateArtifact(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	if verdict.Rejected() {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "native validator rejected the rendered autoheal unit; nothing was installed: "+verdict.Output)
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
	// explicit restart. The same applies when an earlier rebuild was never
	// followed by a restart: the process is older than the binary.
	processStale := recordProcessIdentity(&status, host.OS, loop, unitRunState(host.OS, name))
	if rebuiltLoop || definitionChanged || processStale {
		if err := restartScheduler(host.OS, name, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "autoheal scheduler restart after rebuild failed: "+err.Error(), recoveryNote())
			return status, nil
		}
		status.Notes = append(status.Notes, "restarted autoheal scheduler onto the current binary and definition")
	}

	if err := ensureLingering(host, status.Config, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "Linux lingering enable failed: "+err.Error(), recoveryNote())
		return status, nil
	}
	verified := h.Inspect(host, hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required, Config: status.Config})
	// Keep the mutation record (what was rebuilt, what was restarted) in
	// front of the verification notes: an operator reading `setup --json`
	// should see what changed, not only that it now verifies.
	verified.Notes = append(append([]string(nil), status.Notes...), verified.Notes...)
	if !verified.Applied {
		verified.ExecutionState = hostreqkit.ExecutionFailed
		verified.Notes = append(verified.Notes, "scheduler mutation completed but verification did not prove protection", recoveryNote())
	}
	return verified, nil
}

// ensureLingering enables Linux lingering for the invoking user when the
// dedicated boot policy asks for it. It is a no-op outside Linux, under the
// shared policy, and when lingering is already on.
func ensureLingering(host hostreqkit.Host, config map[string]any, opts hostreqkit.EnsureOptions) error {
	if hostreqspec.PlatformFromGOOS(host.OS) != hostreqspec.PlatformLinux || bootPolicy(config) != handlerDedicated {
		return nil
	}
	user := hostreqkit.InvokingUser()
	if lingeringEnabledFn(user) {
		return nil
	}
	return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "loginctl", []string{"enable-linger", user}, opts)
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

func scenarioDir(root string) string {
	return filepath.Join(root, repocontractmeta.ScenarioDir, autohealScenario)
}

// loopModuleDir is the loop component's build.dir from the scenario's
// service.json: the nested Go module the lifecycle engine builds.
func loopModuleDir(root string) string {
	return filepath.Join(scenarioDir(root), "cli", "loop")
}

func loopPath(root, osName string) string {
	name := loopBinaryName
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformWindows {
		name += ".exe"
	}
	return filepath.Join(scenarioDir(root), "cli", name)
}

func nativeDefinition(osName, root, home string) (string, string, string, error) {
	artifact, path, name, err := nativeArtifact(osName, root, home)
	if err != nil {
		return "", path, name, err
	}
	return artifact.Primary().Content, path, name, nil
}

// nativeArtifact renders the loop's unit for the host OS through the shared
// ServiceDefinition seam and returns it with its install path and native
// unit identity.
func nativeArtifact(osName, root, home string) (platformgo.RenderedArtifact, string, string, error) {
	unit, _ := platformgo.CoreUnitByID(platformgo.CoreUnitAutohealLoop)
	name := unit.Systemd
	path := platformgo.SystemdUserUnitPath(home, name)
	renderTarget := osName
	if isMacOS(osName) {
		name = unit.Launchd
		path = platformgo.LaunchAgentPath(home, name)
		renderTarget = "darwin"
	}
	if hostreqspec.PlatformFromGOOS(osName) == hostreqspec.PlatformWindows {
		name = unit.Windows
		path = filepath.Join(os.TempDir(), "vrooli-autoheal.xml")
	}
	artifact, err := platformgo.RenderWatchdogArtifact(renderTarget, platformgo.WatchdogDefinitionOptions{
		Root:         root,
		Home:         home,
		LoopBinary:   loopPath(root, osName),
		VrooliBinary: filepath.Join(home, repocontractmeta.ProjectConfigDir, "bin", "vrooli"),
		Username:     hostreqkit.InvokingUser(),
	})
	return artifact, path, name, err
}

// recordVerdict stores the native validator's answer as evidence so the
// readiness inspection can read it back from `vrooli setup status --json`.
// An unavailable validator is noted: unproven is not accepted.
func recordVerdict(status *hostreqkit.ItemStatus, verdict platformgo.Verdict) {
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	status.Evidence["validator_verdict"] = verdict
	if verdict.State == platformgo.VerdictUnavailable {
		status.Notes = append(status.Notes, "native validator unavailable; rendered unit is unproven: "+verdict.Output)
	}
}

// recordLoopFreshness stores the engine's verdict on the loop binary as
// evidence and notes anything other than fresh. It returns the verdict.
func recordLoopFreshness(status *hostreqkit.ItemStatus, root, loop string) string {
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	verdict, reason := loopBinaryVerdict(root, loop)
	status.Evidence["loop_freshness"] = map[string]any{
		"verdict":  verdict,
		"reason":   reason,
		"binary":   loop,
		"manifest": cliutil.FreshnessManifestPath(loop),
	}
	switch verdict {
	case verdictStale:
		status.Notes = append(status.Notes, "autoheal loop binary is stale ("+reason+"); setup will rebuild it through the lifecycle engine")
	case verdictUnknown:
		status.Notes = append(status.Notes, "autoheal loop freshness is unknown ("+reason+"); setup will rebuild it through the lifecycle engine")
	}
	return verdict
}

// loopBinaryVerdict reports whether the loop binary is fresh, stale, or of
// unknown freshness, and why.
//
// The verdict comes from the freshness manifest the lifecycle engine stamps
// next to the component artifact when `vrooli scenario setup` builds it
// (`<binary>.freshness.json`). The manifest is evaluated with the engine's
// own reader against the same input set the engine recorded, so this is a
// second reading of one contract, not a second freshness implementation.
// There is no mtime comparison here and no `go build`: a binary without a
// manifest is unknown, not fresh, and setup is what turns unknown into a
// stamped verdict.
//
// The manifest's recorded key inputs (toolchain, GOOS, flags) are passed back
// as the current keys: the safeguard has no Go-environment resolver of its
// own, and reimplementing the engine's would be the drift this function
// exists to remove. A toolchain change is caught by the engine on the next
// `scenario setup`, which restamps the manifest.
func loopBinaryVerdict(root, loop string) (string, string) {
	if _, err := os.Stat(loop); err != nil {
		return verdictStale, "binary missing"
	}
	manifestPath := cliutil.FreshnessManifestPath(loop)
	manifest, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil {
		return verdictUnknown, "freshness manifest unreadable: " + err.Error()
	}
	if !ok {
		return verdictUnknown, "no freshness manifest at " + manifestPath + "; the lifecycle engine has not built the loop component"
	}
	spec := cliutil.FreshnessSpec{
		SourceRoot:   loopModuleDir(root),
		ContextRoot:  root,
		Inputs:       manifest.Inputs,
		SkipFiles:    []string{filepath.Base(loop)},
		SkipSuffixes: []string{"_test.go", cliutil.FreshnessManifestSuffix},
	}
	verdict, err := cliutil.EvaluateFreshness(spec, manifest, manifest.KeyInputs)
	if err != nil {
		return verdictUnknown, "freshness evaluation failed: " + err.Error()
	}
	if verdict.Stale {
		reason := verdict.Reason
		if verdict.File != "" {
			reason += " (" + verdict.File + ")"
		}
		return verdictStale, reason
	}
	return verdictFresh, ""
}

// recordProcessIdentity compares the executable the unit's main process is
// running with the binary on disk and records both digests as evidence. It
// returns true when they differ: the process is older than the binary and
// only a restart makes the fix real. On Linux the running image is read
// through procfs, which serves the original file even after it was replaced
// on disk. Elsewhere there is no equivalent read, so nothing is claimed.
func recordProcessIdentity(status *hostreqkit.ItemStatus, osName, loop string, runState map[string]string) bool {
	if hostreqspec.PlatformFromGOOS(osName) != hostreqspec.PlatformLinux {
		return false
	}
	pid := strings.TrimSpace(runState["MainPID"])
	if pid == "" || pid == "0" {
		return false
	}
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	evidence := map[string]any{"pid": pid}
	status.Evidence["process_identity"] = evidence
	processDigest, err := fileSHA256(filepath.Join(procRoot, pid, "exe"))
	if err != nil {
		evidence["error"] = "running executable unreadable: " + err.Error()
		return false
	}
	binaryDigest, err := fileSHA256(loop)
	if err != nil {
		evidence["error"] = "binary unreadable: " + err.Error()
		return false
	}
	evidence["process_sha256"] = processDigest
	evidence["binary_sha256"] = binaryDigest
	evidence["match"] = processDigest == binaryDigest
	if processDigest == binaryDigest {
		return false
	}
	status.Notes = append(status.Notes, "autoheal unit process older than binary: pid "+pid+" runs sha256 "+shortDigest(processDigest)+" while "+loop+" is "+shortDigest(binaryDigest)+"; setup will restart the unit")
	return true
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func shortDigest(digest string) string {
	if digest == "" {
		return "(absent)"
	}
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

// setupScenarioThroughLifecycle runs `vrooli scenario setup vrooli-autoheal`
// through the shared cliinvoke seam. The engine builds every stale component
// of the scenario, the loop included, and stamps each one's manifest. When
// setup itself is elevated the invocation drops to the invoking user so the
// build products and manifests are owned by the operator, like every other
// per-user install in hostreqkit.
func setupScenarioThroughLifecycle(root, home string, opts hostreqkit.EnsureOptions) error {
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{RuntimeHome: filepath.Join(home, repocontractmeta.ProjectConfigDir)})
	if err != nil {
		return err
	}
	args := cliinvoke.ScenarioSetup(autohealScenario)
	if invoking := hostreqkit.InvokingUser(); hostreqkit.RunningAsRootFn() && invoking != "" && invoking != "root" {
		args = append([]string{"-u", invoking, "-H", "--", binary}, args...)
		binary = "sudo"
	}
	result := cliinvoke.Run(context.Background(), cliinvoke.Invocation{
		Binary: binary,
		Args:   args,
		Dir:    root,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})
	return result.Error()
}

// unitRunState reads the systemd properties that distinguish a unit that is
// active from one that is active but crash-looping (NRestarts, Result) and
// the main PID whose executable identity the inspection verifies. It is
// evidence, not a verdict; the readiness check decides what "zero restarts
// since boot" means.
func unitRunState(osName, name string) map[string]string {
	if hostreqspec.PlatformFromGOOS(osName) != hostreqspec.PlatformLinux {
		return nil
	}
	cmd, args := asUserArgs("systemctl", "--user", "show", name, "-p", "ActiveState", "-p", "NRestarts", "-p", "Result", "-p", "MainPID")
	out, err := commandOutputFn(cmd, args...)
	if err != nil {
		return nil
	}
	state := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			state[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return state
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
