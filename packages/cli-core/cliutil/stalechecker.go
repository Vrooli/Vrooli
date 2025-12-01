package cliutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
)

// StaleChecker compares the embedded build fingerprint against source files,
// optionally auto-rebuilding via cli-installer when a mismatch is detected.
type StaleChecker struct {
	AppName           string
	BuildFingerprint  string
	BuildTimestamp    string
	BuildSourceRoot   string
	SourceRootEnvVars []string
	InstallerModule   string // relative path to cli-core (default: packages/cli-core)
	ReexecArgs        []string

	FingerprintFunc func(root string, skip ...string) (string, error)
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

// CheckAndMaybeRebuild returns true when the process was restarted after a rebuild.
func (c *StaleChecker) CheckAndMaybeRebuild() bool {
	if c.BuildFingerprint == "" || c.BuildFingerprint == "unknown" {
		return false
	}
	srcRoot := ResolveSourceRoot(c.BuildSourceRoot, c.SourceRootEnvVars...)
	if srcRoot == "" {
		return false
	}

	var skipNames []string
	if executable, err := os.Executable(); err == nil {
		skipNames = append(skipNames, filepath.Base(executable))
	}

	fingerprint, err := c.fingerprint(srcRoot, skipNames...)
	if err != nil {
		c.log("Warning: unable to verify CLI freshness: %v\n", err)
		return false
	}

	if fingerprint == c.BuildFingerprint {
		return false
	}

	if c.autoRebuild(srcRoot, fingerprint) {
		return true
	}

	c.log("Warning: %s CLI binary built at %s (fingerprint %s) does not match the current sources (fingerprint %s).\n", c.appLabel(), c.BuildTimestamp, c.BuildFingerprint, fingerprint)
	return false
}

func (c *StaleChecker) autoRebuild(srcRoot, currentFingerprint string) bool {
	if !c.canAutoRebuild(srcRoot) {
		return false
	}

	repoRoot, ok := findRepositoryRoot(srcRoot, c.installerModule())
	if !ok {
		return false
	}

	executable, err := os.Executable()
	if err != nil {
		return false
	}

	cmd := exec.Command("go", "run", "./cmd/cli-installer",
		"--module", srcRoot,
		"--output", executable,
		"--force", "true",
	)
	cmd.Dir = filepath.Join(repoRoot, c.installerModule())
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := c.commandRunner()(cmd); err != nil {
		c.log("Warning: CLI auto-rebuild failed: %v\n", err)
		return false
	}

	c.log("%s CLI rebuilt from current sources (fingerprint %s); restarting command...\n", c.appLabel(), currentFingerprint)
	if err := c.reexec()(executable, c.ReexecArgs); err != nil {
		c.log("Warning: unable to restart CLI after rebuild: %v\n", err)
	}
	return true
}

func (c *StaleChecker) canAutoRebuild(srcRoot string) bool {
	if _, err := c.lookPath()("go"); err != nil {
		return false
	}
	if !strings.Contains(srcRoot, string(os.PathSeparator)+"scenarios"+string(os.PathSeparator)) {
		return false
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

func (c *StaleChecker) fingerprint(root string, skip ...string) (string, error) {
	if c.FingerprintFunc != nil {
		return c.FingerprintFunc(root, skip...)
	}
	return buildinfo.ComputeFingerprint(root, skip...)
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

func (c *StaleChecker) reexec() func(string, []string) error {
	if c.Reexec != nil {
		return c.Reexec
	}
	return func(executable string, args []string) error {
		cmd := exec.Command(executable, args...)
		cmd.Env = os.Environ()
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
