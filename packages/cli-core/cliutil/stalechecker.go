package cliutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
)

// StaleChecker compares the embedded build fingerprint against source files,
// optionally auto-rebuilding via cli-installer when a mismatch is detected.
type StaleChecker struct {
	AppName            string
	BuildFingerprint   string
	BuildTimestamp     string
	BuildSourceRoot    string
	SourceContextPath  string
	ManifestSourcePath string
	FreshnessInputs    []string
	SourceRootEnvVars  []string
	InstallerModule    string // relative path to cli-core (default: packages/cli-core)
	InstallKind        string // scenario|resource; used only by the owner refresh command
	InstallName        string // declared scenario/resource name, never a build path
	ReexecArgs         []string

	// RefreshStateDir holds the per-CLI lock and log of background refreshes
	// (default ~/.vrooli/cli-refresh).
	RefreshStateDir string
	// RefreshCooldown is the minimum time between background refresh launches
	// for one CLI, so a refresh that keeps failing is not relaunched by every
	// hook call (default 30s).
	RefreshCooldown time.Duration

	FingerprintFunc func(spec FreshnessSpec) (string, error)
	LookPathFunc    func(file string) (string, error)
	CommandRunner   func(cmd *exec.Cmd) error
	Logger          func(format string, args ...interface{})
	Reexec          func(executable string, args []string) error

	// BackgroundStarter launches the detached refresh (default cmd.Start then
	// release). Tests replace it to observe launches.
	BackgroundStarter func(cmd *exec.Cmd) error

	// InstalledManifest is the freshness manifest the installer wrote for this
	// binary (default: next to the running executable). Its watch lets an
	// unchanged tree be confirmed from metadata alone.
	InstalledManifest string

	// now stamps restamped manifests; tests move it to step past the racy window.
	now func() time.Time
}

// NewStaleChecker builds a StaleChecker with common defaults. Provide the app
// name (for warnings), build-time values, and source-root env var overrides.
func NewStaleChecker(appName, buildFingerprint, buildTimestamp, buildSourceRoot string, sourceRootEnvVars ...string) *StaleChecker {
	return &StaleChecker{
		AppName:           appName,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		SourceRootEnvVars: sourceRootEnvVars,
	}
}

// rebuildLoopEnvVar is set after a rebuild to detect infinite loops
const rebuildLoopEnvVar = "CLI_CORE_REBUILD_FINGERPRINT"

// debugStaleEnvVar enables per-file fingerprint diagnostics when set to a
// truthy value. When a fingerprint mismatch is detected, the StaleChecker
// will dump every file that participated (path, size, content hash) so that
// install-time vs run-time divergence can be identified file-by-file.
const debugStaleEnvVar = "VROOLI_CLI_DEBUG_STALE"

// artifactModeEnvVar marks a CLI invocation running from a source-reduced
// deployment artifact. Such a CLI must not attempt a development rebuild from
// the deployed tree; the release pipeline already verified the binary.
const artifactModeEnvVar = "VROOLI_CLI_ARTIFACT_MODE"

// LatencyBoundEnvVar marks the whole invocation as latency-bound when truthy:
// the stale checker then never waits on a rebuild (see CheckLatencyBound). It
// is the escape hatch for callers that cannot declare the command (a wrapper
// script, an ad-hoc hook); commands should prefer declaring themselves
// latency-bound (cliapp.Command.LatencyBound / manifest latency_bound).
const LatencyBoundEnvVar = "VROOLI_CLI_LATENCY_BOUND"

// defaultRefreshCooldown spaces background refresh launches for one CLI.
const defaultRefreshCooldown = 30 * time.Second

// CheckAndMaybeRebuild returns true when the process was restarted after a rebuild.
// When LatencyBoundEnvVar is truthy it behaves as CheckLatencyBound.
func (c *StaleChecker) CheckAndMaybeRebuild() (bool, error) {
	return c.check(truthy(os.Getenv(LatencyBoundEnvVar)))
}

// CheckLatencyBound is the stale check for a command that runs under a hard
// time limit (coding-agent hooks, status lines). It never blocks on a rebuild,
// never restarts the process and never fails the command: a stale CLI runs
// as-is while a detached background refresh (at most one per CLI, serialized by
// a lock file) brings the installed binary up to date for the next call.
func (c *StaleChecker) CheckLatencyBound() {
	_, _ = c.check(true)
}

func (c *StaleChecker) check(latencyBound bool) (bool, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv(artifactModeEnvVar)), "1") || strings.EqualFold(strings.TrimSpace(os.Getenv(artifactModeEnvVar)), "true") {
		return false, nil
	}
	if c.BuildFingerprint == "" || c.BuildFingerprint == "unknown" {
		return false, nil
	}
	srcRoot := ResolveSourceRoot(c.BuildSourceRoot, c.SourceRootEnvVars...)
	if srcRoot == "" {
		return false, nil
	}
	base := c.baseFreshnessSpec(srcRoot)
	if executable, err := os.Executable(); err == nil {
		base.SkipFiles = append(base.SkipFiles, filepath.Base(executable))
	}

	// Common path: the installed manifest's watch proves that neither the
	// replace closure nor any input changed since this binary's fingerprint
	// was recorded, from one lstat per watched object.
	installed, installedOK := c.readInstalledManifest()
	if installedOK && installed.Watch != nil && installed.Watch.ExpansionStable && installed.Watch.Holds(base, installed.WrittenAtNS) {
		return false, nil
	}

	writtenAt := c.clock().UnixNano()
	observer := newFreshnessObserver(base)
	observed := base
	observed.observer = observer
	spec := WithGoReplaceFreshnessInputs(observed)
	fingerprint, entries, err := c.fingerprint(spec)
	if err != nil {
		c.log("Warning: unable to verify CLI freshness: %v\n", err)
		return false, nil
	}

	if fingerprint == c.BuildFingerprint {
		// Something the watch covers moved without changing the inputs (an
		// excluded output, an editor's temporary file). Record the current
		// state so the next invocation takes the common path again.
		c.restampInstalledManifest(installed, installedOK, base, spec, entries, observer, writtenAt)
		return false, nil
	}
	spec.observer = nil

	if latencyBound {
		c.dumpFreshnessDebug(spec)
		c.refreshInBackground(spec)
		return false, nil
	}

	// Detect infinite rebuild loops: if we already rebuilt for this fingerprint
	// but still have a mismatch, something is wrong - don't rebuild again
	if prevFingerprint := os.Getenv(rebuildLoopEnvVar); prevFingerprint == fingerprint {
		c.log("Warning: %s CLI rebuild loop detected (fingerprint %s). Skipping auto-rebuild.\n", c.appLabel(), fingerprint)
		c.log("  This usually means stale checking and installation still disagree about the freshness inputs.\n")
		c.log("  Build fingerprint: %s, Source fingerprint: %s\n", c.BuildFingerprint, fingerprint)
		c.log("  Set %s=1 to dump per-file fingerprint inputs and identify the divergence.\n", debugStaleEnvVar)
		c.dumpFreshnessDebug(spec)
		return false, fmt.Errorf("%s CLI remains stale after a previous refresh attempt; refusing to dispatch with mismatched sources", c.appLabel())
	}
	c.dumpFreshnessDebug(spec)

	restarted, err := c.autoRebuild(spec, fingerprint)
	if err != nil {
		return false, err
	}
	return restarted, nil
}

// refreshCommand builds the capacity-admitted owner refresh command for this
// CLI. Stdio is left for the caller: a synchronous rebuild streams it, a
// background one redirects it to a log.
func (c *StaleChecker) refreshCommand(spec FreshnessSpec) (*exec.Cmd, string, error) {
	if _, err := c.lookPath()("go"); err != nil {
		return nil, "", fmt.Errorf("%s CLI is stale and the Go toolchain is unavailable: %w", c.appLabel(), err)
	}

	repoRoot, ok := findRepositoryRoot(spec.SourceRoot, c.installerModule())
	if !ok {
		return nil, "", fmt.Errorf("%s CLI is stale but its cli-core installer source root is unavailable", c.appLabel())
	}

	executable, err := os.Executable()
	if err != nil {
		return nil, "", fmt.Errorf("resolve running %s CLI path: %w", c.appLabel(), err)
	}
	controlPlane, err := c.controlPlaneBinary()
	if err != nil {
		return nil, "", fmt.Errorf("%s CLI is stale; capacity-admitted refresh is unavailable: %w", c.appLabel(), err)
	}
	if c.InstallKind != "scenario" && c.InstallKind != "resource" || strings.TrimSpace(c.InstallName) == "" {
		return nil, "", fmt.Errorf("%s CLI is stale but has no declared scenario/resource install identity", c.appLabel())
	}
	cmd := exec.Command(controlPlane, "cli-refresh", c.InstallKind, c.InstallName, executable)
	cmd.Dir = repoRoot
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	return cmd, executable, nil
}

func (c *StaleChecker) autoRebuild(spec FreshnessSpec, currentFingerprint string) (bool, error) {
	cmd, executable, err := c.refreshCommand(spec)
	if err != nil {
		return false, err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := c.commandRunner()(cmd); err != nil {
		return false, fmt.Errorf("%s CLI refresh was not admitted or did not complete; retry when available: %w", c.appLabel(), err)
	}

	c.log("%s CLI rebuilt from current sources (fingerprint %s); restarting command...\n", c.appLabel(), currentFingerprint)
	if err := c.reexec()(executable, c.ReexecArgs, currentFingerprint); err != nil {
		return false, fmt.Errorf("restart refreshed %s CLI: %w", c.appLabel(), err)
	}
	return true, nil
}

func (c *StaleChecker) controlPlaneBinary() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(LaunchRuntimeBinaryEnv)); configured != "" {
		info, err := os.Stat(configured)
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("configured control-plane binary %q is unavailable", configured)
		}
		return configured, nil
	}
	return c.lookPath()("vrooli")
}

func findRepositoryRoot(start, installerModule string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, installerModule)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// fingerprint returns the source fingerprint and, when cli-core computed it,
// the per-file entries it hashed.
func (c *StaleChecker) fingerprint(spec FreshnessSpec) (string, []FileManifestEntry, error) {
	if c.FingerprintFunc != nil {
		fingerprint, err := c.FingerprintFunc(spec)
		return fingerprint, nil, err
	}
	if strings.TrimSpace(spec.SourceRoot) == "" {
		return "", nil, fmt.Errorf("freshness source root must not be empty")
	}
	entries, err := collectFreshnessEntries(spec)
	if err != nil {
		return "", nil, err
	}
	return aggregateManifestDigest(entries, nil), entries, nil
}

func (c *StaleChecker) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *StaleChecker) installedManifestPath() string {
	if strings.TrimSpace(c.InstalledManifest) != "" {
		return c.InstalledManifest
	}
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	return FreshnessManifestPath(executable)
}

// readInstalledManifest returns the installed manifest only when it describes
// this binary's build. Per-file entries are not decoded: the watch decides the
// common path, and a restamp records freshly hashed entries.
func (c *StaleChecker) readInstalledManifest() (FreshnessManifest, bool) {
	path := c.installedManifestPath()
	if path == "" {
		return FreshnessManifest{}, false
	}
	manifest, ok, err := readFreshnessManifestHead(path)
	if err != nil || !ok || manifest.Version != freshnessManifestVersion || manifest.Digest != c.BuildFingerprint || len(manifest.KeyInputs) > 0 {
		return FreshnessManifest{}, false
	}
	return manifest, true
}

// restampInstalledManifest rewrites the installed manifest after a full
// enumeration confirmed the binary current. It writes only while the binary
// at the executable path is the one running, so a process outliving a
// reinstall never overwrites the new binary's manifest, and only for the input
// list the installer recorded. Best effort: failure just repeats the full
// check next time.
func (c *StaleChecker) restampInstalledManifest(installed FreshnessManifest, installedOK bool, base, spec FreshnessSpec, entries []FileManifestEntry, observer *freshnessObserver, writtenAtNS int64) {
	if entries == nil || (installedOK && !slices.Equal(installed.Inputs, spec.Inputs)) {
		return
	}
	executable, err := os.Executable()
	if err != nil || !runningExecutableIs(executable) {
		return
	}
	path := c.installedManifestPath()
	if path == "" {
		return
	}
	if !installedOK {
		// A manifest for another build (or none) must not be replaced unless it
		// is absent; the installer owns that transition.
		if _, present, _ := ReadFreshnessManifest(path); present {
			return
		}
	}
	checkType := installed.CheckType
	if checkType == "" {
		checkType = "cli"
	}
	manifest := newFreshnessManifest(spec, checkType, nil, writtenAtNS, entries, observer.watch(spec, slices.Equal(spec.Inputs, base.Inputs)))
	if err := WriteFreshnessManifest(path, manifest); err != nil {
		c.debugRefresh("freshness manifest restamp failed: %v\n", err)
	}
}

func (c *StaleChecker) lookPath() func(string) (string, error) {
	if c.LookPathFunc != nil {
		return c.LookPathFunc
	}
	return exec.LookPath
}

func (c *StaleChecker) commandRunner() func(cmd *exec.Cmd) error {
	if c.CommandRunner != nil {
		return c.CommandRunner
	}
	return func(cmd *exec.Cmd) error { return cmd.Run() }
}

func (c *StaleChecker) log(format string, args ...interface{}) {
	if c.Logger != nil {
		c.Logger(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

func (c *StaleChecker) reexec() func(string, []string, string) error {
	if c.Reexec != nil {
		// Wrap legacy Reexec that doesn't take fingerprint
		return func(executable string, args []string, fingerprint string) error {
			return c.Reexec(executable, args)
		}
	}
	return func(executable string, args []string, fingerprint string) error {
		cmd := exec.Command(executable, args...)
		cmd.Env = selfReexecEnv(envkit.Env(os.Environ()), fingerprint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			return err
		}
		os.Exit(0)
		return nil
	}
}

// selfReexecIdentityEnvVars are the agent identity variables envkit drops at
// every boundary except a delegated agent. A re-exec of the rebuilt binary is
// the same invocation by the same agent, so it keeps them; without this an
// agent's first CLI call after a source edit runs with no identity.
var selfReexecIdentityEnvVars = []string{EnvIdentityToken, EnvClaudeCodeSessionID, EnvCodexThreadID}

// selfReexecEnv derives the environment for re-executing the rebuilt CLI. It
// is the SameScenario environment plus the caller's own identity variables
// (only those present in parent) and the rebuild fingerprint that detects loops.
func selfReexecEnv(parent envkit.Env, fingerprint string) envkit.Env {
	platform := envkit.DefaultPlatform()
	overlay := envkit.Env{fmt.Sprintf("%s=%s", rebuildLoopEnvVar, fingerprint)}
	for _, entry := range parent {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		for _, identityKey := range selfReexecIdentityEnvVars {
			if key == identityKey || (platform.CaseInsensitive && strings.EqualFold(key, identityKey)) {
				overlay = append(overlay, entry)
				break
			}
		}
	}
	return envkit.WithOverlayWithPlatform(parent, envkit.SameScenario, overlay, platform)
}

func (c *StaleChecker) installerModule() string {
	if strings.TrimSpace(c.InstallerModule) != "" {
		return c.InstallerModule
	}
	return filepath.Join("packages", "cli-core")
}

func (c *StaleChecker) appLabel() string {
	if c.AppName != "" {
		return c.AppName
	}
	return "CLI"
}

func (c *StaleChecker) sourceContextRoot(srcRoot string) string {
	if strings.TrimSpace(c.SourceContextPath) == "" {
		return srcRoot
	}
	return filepath.Join(srcRoot, filepath.FromSlash(c.SourceContextPath))
}

// dumpFreshnessDebug prints the per-file fingerprint inputs when the
// debugStaleEnvVar env var is truthy. Used to diagnose install-time vs
// run-time divergence (typical cause: stray build artifacts left inside
// the freshness glob by ad-hoc `go build` runs).
func (c *StaleChecker) dumpFreshnessDebug(spec FreshnessSpec) {
	if !truthy(os.Getenv(debugStaleEnvVar)) {
		return
	}
	c.log("[%s] Freshness inputs (set %s=0 to silence):\n", c.appLabel(), debugStaleEnvVar)
	c.log("  SourceRoot:  %s\n", spec.SourceRoot)
	c.log("  ContextRoot: %s\n", spec.ContextRoot)
	c.log("  Inputs:      %v\n", spec.Inputs)
	c.log("  SkipFiles:   %v\n", spec.SkipFiles)
	if err := dumpFreshnessFiles(spec, c.log); err != nil {
		c.log("  (file enumeration failed: %v)\n", err)
	}
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

// freshnessSpec derives the same spec the installer recorded: declared inputs
// plus the local replace targets the CLI imports. Caller-provided inputs are
// not baked, so the runtime must expand them itself to agree with the build.
func (c *StaleChecker) freshnessSpec(srcRoot string) FreshnessSpec {
	return WithGoReplaceFreshnessInputs(c.baseFreshnessSpec(srcRoot))
}

// baseFreshnessSpec is the spec before replace-closure expansion. For an
// installed binary its inputs are the baked, already-expanded list.
func (c *StaleChecker) baseFreshnessSpec(srcRoot string) FreshnessSpec {
	return FreshnessSpec{
		SourceRoot:  srcRoot,
		ContextRoot: c.sourceContextRoot(srcRoot),
		Inputs:      append([]string(nil), c.FreshnessInputs...),
	}
}

func (c *StaleChecker) manifestSourcePath(contextRoot string) string {
	if strings.TrimSpace(c.ManifestSourcePath) == "" {
		return ""
	}
	return filepath.Join(contextRoot, filepath.FromSlash(c.ManifestSourcePath))
}

func dumpFreshnessFiles(spec FreshnessSpec, logf func(string, ...interface{})) error {
	entries, err := collectFreshnessEntries(spec)
	if err != nil {
		return err
	}
	logf("  Files (%d):\n", len(entries))
	for _, entry := range entries {
		hash := entry.Hash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		logf("    %s  %10d  %s\n", hash, entry.Size, entry.Rel)
	}
	return nil
}
