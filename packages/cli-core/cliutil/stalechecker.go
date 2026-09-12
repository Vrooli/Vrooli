package cliutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	ReexecArgs         []string

	FingerprintFunc func(spec FreshnessSpec) (string, error)
	LookPathFunc    func(file string) (string, error)
	CommandRunner   func(cmd *exec.Cmd) error
	Logger          func(format string, args ...interface{})
	Reexec          func(executable string, args []string) error
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

// CheckAndMaybeRebuild returns true when the process was restarted after a rebuild.
func (c *StaleChecker) CheckAndMaybeRebuild() bool {
	if c.BuildFingerprint == "" || c.BuildFingerprint == "unknown" {
		return false
	}
	srcRoot := ResolveSourceRoot(c.BuildSourceRoot, c.SourceRootEnvVars...)
	if srcRoot == "" {
		return false
	}
	spec := c.freshnessSpec(srcRoot)

	if executable, err := os.Executable(); err == nil {
		spec.SkipFiles = append(spec.SkipFiles, filepath.Base(executable))
	}

	fingerprint, err := c.fingerprint(spec)
	if err != nil {
		c.log("Warning: unable to verify CLI freshness: %v\n", err)
		return false
	}

	if fingerprint == c.BuildFingerprint {
		return false
	}

	// Detect infinite rebuild loops: if we already rebuilt for this fingerprint
	// but still have a mismatch, something is wrong - don't rebuild again
	if prevFingerprint := os.Getenv(rebuildLoopEnvVar); prevFingerprint == fingerprint {
		c.log("Warning: %s CLI rebuild loop detected (fingerprint %s). Skipping auto-rebuild.\n", c.appLabel(), fingerprint)
		c.log("  This usually means stale checking and installation still disagree about the freshness inputs.\n")
		c.log("  Build fingerprint: %s, Source fingerprint: %s\n", c.BuildFingerprint, fingerprint)
		c.log("  Set %s=1 to dump per-file fingerprint inputs and identify the divergence.\n", debugStaleEnvVar)
		c.dumpFreshnessDebug(spec)
		return false
	}
	c.dumpFreshnessDebug(spec)

	if c.autoRebuild(spec, fingerprint) {
		return true
	}

	c.log("Warning: %s CLI binary built at %s (fingerprint %s) does not match the current sources (fingerprint %s).\n", c.appLabel(), c.BuildTimestamp, c.BuildFingerprint, fingerprint)
	return false
}

func (c *StaleChecker) autoRebuild(spec FreshnessSpec, currentFingerprint string) bool {
	if _, err := c.lookPath()("go"); err != nil {
		return false
	}

	repoRoot, ok := findRepositoryRoot(spec.SourceRoot, c.installerModule())
	if !ok {
		return false
	}

	executable, err := os.Executable()
	if err != nil {
		return false
	}

	// Pass the binary name to the installer so it uses the same skip file
	// as the stale checker when computing the fingerprint. Without this,
	// the installer would use the module directory name (e.g., "cli") while
	// the stale checker uses the executable name (e.g., "test-genie"),
	// causing different fingerprints and an infinite rebuild loop.
	binaryName := filepath.Base(executable)

	// IMPORTANT: bool flags must use `--flag=value` form. Passing them as
	// `--flag value` makes Go's flag library treat `value` as a positional
	// argument and silently stop parsing — which historically dropped
	// `--context-root`, `--manifest`, and `--freshness-input` and fell back
	// to walking the entire module root. That mismatched the runtime
	// freshness spec (which honors --context-root + --freshness-input) and
	// fired the rebuild-loop guard on every invocation.
	cmd := exec.Command("go", "run", "./cmd/cli-installer",
		"--module", spec.SourceRoot,
		"--output", executable,
		"--name", binaryName,
		"--force=true",
	)
	if manifestPath := c.manifestSourcePath(spec.ContextRoot); manifestPath != "" {
		cmd.Args = append(cmd.Args, "--manifest", manifestPath)
	}
	cmd.Args = append(cmd.Args, "--context-root", spec.ContextRoot)
	for _, input := range spec.Inputs {
		cmd.Args = append(cmd.Args, "--freshness-input", input)
	}
	cmd.Dir = filepath.Join(repoRoot, c.installerModule())
	cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := c.commandRunner()(cmd); err != nil {
		c.log("Warning: CLI auto-rebuild failed: %v\n", err)
		return false
	}

	c.log("%s CLI rebuilt from current sources (fingerprint %s); restarting command...\n", c.appLabel(), currentFingerprint)
	if err := c.reexec()(executable, c.ReexecArgs, currentFingerprint); err != nil {
		c.log("Warning: unable to restart CLI after rebuild: %v\n", err)
	}
	return true
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

func (c *StaleChecker) fingerprint(spec FreshnessSpec) (string, error) {
	if c.FingerprintFunc != nil {
		return c.FingerprintFunc(spec)
	}
	return ComputeFreshnessFingerprint(spec)
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
		// Set the rebuild fingerprint env var to detect loops
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{fmt.Sprintf("%s=%s", rebuildLoopEnvVar, fingerprint)})
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

func (c *StaleChecker) freshnessSpec(srcRoot string) FreshnessSpec {
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
