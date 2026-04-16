package cliutil

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
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

	FingerprintFunc      func(root string, skip ...string) (string, error)
	InputFingerprintFunc func(root string, inputs []string, skip ...string) (string, error)
	LookPathFunc         func(file string) (string, error)
	CommandRunner        func(cmd *exec.Cmd) error
	Logger               func(format string, args ...interface{})
	Reexec               func(executable string, args []string) error
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

// CheckAndMaybeRebuild returns true when the process was restarted after a rebuild.
func (c *StaleChecker) CheckAndMaybeRebuild() bool {
	if c.BuildFingerprint == "" || c.BuildFingerprint == "unknown" {
		return false
	}
	srcRoot := ResolveSourceRoot(c.BuildSourceRoot, c.SourceRootEnvVars...)
	if srcRoot == "" {
		return false
	}
	contextRoot := c.sourceContextRoot(srcRoot)

	var skipNames []string
	if executable, err := os.Executable(); err == nil {
		skipNames = append(skipNames, filepath.Base(executable))
	}

	fingerprint, err := c.fingerprint(contextRoot, srcRoot, skipNames...)
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
		c.log("  This usually means the binary name doesn't match between the stale checker and installer.\n")
		c.log("  Build fingerprint: %s, Source fingerprint: %s\n", c.BuildFingerprint, fingerprint)
		return false
	}

	if c.autoRebuild(srcRoot, contextRoot, fingerprint) {
		return true
	}

	c.log("Warning: %s CLI binary built at %s (fingerprint %s) does not match the current sources (fingerprint %s).\n", c.appLabel(), c.BuildTimestamp, c.BuildFingerprint, fingerprint)
	return false
}

func (c *StaleChecker) autoRebuild(srcRoot, contextRoot, currentFingerprint string) bool {
	if _, err := c.lookPath()("go"); err != nil {
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

	// Pass the binary name to the installer so it uses the same skip file
	// as the stale checker when computing the fingerprint. Without this,
	// the installer would use the module directory name (e.g., "cli") while
	// the stale checker uses the executable name (e.g., "test-genie"),
	// causing different fingerprints and an infinite rebuild loop.
	binaryName := filepath.Base(executable)

	cmd := exec.Command("go", "run", "./cmd/cli-installer",
		"--module", srcRoot,
		"--output", executable,
		"--name", binaryName,
		"--force", "true",
	)
	if manifestPath := c.manifestSourcePath(contextRoot); manifestPath != "" {
		cmd.Args = append(cmd.Args, "--manifest", manifestPath)
	}
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

func (c *StaleChecker) fingerprint(contextRoot, sourceRoot string, skip ...string) (string, error) {
	if len(c.FreshnessInputs) > 0 {
		if c.InputFingerprintFunc != nil {
			return c.InputFingerprintFunc(contextRoot, c.FreshnessInputs, skip...)
		}
		return computeFingerprintFromDeclaredInputs(contextRoot, c.FreshnessInputs, skip...)
	}
	if c.FingerprintFunc != nil {
		return c.FingerprintFunc(sourceRoot, skip...)
	}
	return buildinfo.ComputeFingerprint(sourceRoot, skip...)
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
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", rebuildLoopEnvVar, fingerprint))
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

func (c *StaleChecker) manifestSourcePath(contextRoot string) string {
	if strings.TrimSpace(c.ManifestSourcePath) == "" {
		return ""
	}
	return filepath.Join(contextRoot, filepath.FromSlash(c.ManifestSourcePath))
}

type staleFileEntry struct {
	rel  string
	size int64
	hash [32]byte
}

func computeFingerprintFromDeclaredInputs(root string, inputs []string, extraSkipFiles ...string) (string, error) {
	entries := make(map[string]staleFileEntry)
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		matches, err := expandDeclaredInputPaths(root, input)
		if err != nil {
			return "", err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", err
			}
			if info.IsDir() {
				if err := filepath.WalkDir(match, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					rel, relErr := filepath.Rel(root, path)
					if relErr != nil {
						return relErr
					}
					rel = filepath.ToSlash(rel)
					if rel == "." {
						return nil
					}
					if buildinfoSkipDir(rel) && d.IsDir() {
						return filepath.SkipDir
					}
					if d.IsDir() || buildinfoSkipFile(rel, extraSkipFiles) {
						return nil
					}
					content, readErr := os.ReadFile(path)
					if readErr != nil {
						return readErr
					}
					fileInfo, infoErr := d.Info()
					if infoErr != nil {
						return infoErr
					}
					entries[rel] = staleFileEntry{rel: rel, size: fileInfo.Size(), hash: sha256.Sum256(content)}
					return nil
				}); err != nil {
					return "", err
				}
				continue
			}
			rel, err := filepath.Rel(root, match)
			if err != nil {
				return "", err
			}
			rel = filepath.ToSlash(rel)
			if buildinfoSkipFile(rel, extraSkipFiles) {
				continue
			}
			content, err := os.ReadFile(match)
			if err != nil {
				return "", err
			}
			entries[rel] = staleFileEntry{rel: rel, size: info.Size(), hash: sha256.Sum256(content)}
		}
	}
	list := make([]staleFileEntry, 0, len(entries))
	for _, entry := range entries {
		list = append(list, entry)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].rel < list[j].rel })
	hasher := sha256.New()
	for _, entry := range list {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func expandDeclaredInputPaths(root, input string) ([]string, error) {
	if hasGlobPattern(input) {
		return filepath.Glob(filepath.Join(root, filepath.FromSlash(input)))
	}
	return []string{filepath.Join(root, filepath.FromSlash(input))}, nil
}

func hasGlobPattern(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func buildinfoSkipDir(path string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range []string{".git", ".vscode", ".idea", "coverage", "dist", "build", "tmp", "data", "node_modules"} {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func buildinfoSkipFile(path string, extra []string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range []string{"build.meta"} {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	for _, skip := range extra {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}
