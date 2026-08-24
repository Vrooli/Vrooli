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
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
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
					if isCompiledBinary(content) {
						return nil
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
			if isCompiledBinary(content) {
				continue
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

// dumpFreshnessFiles re-walks the freshness spec and prints each
// participating file's relative path, byte size, and short hash prefix to
// the provided logger. Mirrors [computeFingerprintFromDeclaredInputs] so
// the listing is exactly what the fingerprint hash sees.
func dumpFreshnessFiles(spec FreshnessSpec, logf func(string, ...interface{})) error {
	root := spec.ContextRoot
	if strings.TrimSpace(root) == "" {
		root = spec.SourceRoot
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return fmt.Errorf("empty root")
	}
	type entry struct {
		rel  string
		size int64
		hash string
	}
	var listed []entry
	visit := func(rel string, info fs.FileInfo, content []byte) {
		listed = append(listed, entry{rel: rel, size: info.Size(), hash: fmt.Sprintf("%x", sha256.Sum256(content))})
	}
	for _, input := range spec.Inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		matches, err := expandDeclaredInputPaths(root, input)
		if err != nil {
			return err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				_ = filepath.WalkDir(match, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					rel, _ := filepath.Rel(root, path)
					rel = filepath.ToSlash(rel)
					if rel == "." {
						return nil
					}
					if buildinfoSkipDir(rel) && d.IsDir() {
						return filepath.SkipDir
					}
					if d.IsDir() || buildinfoSkipFile(rel, spec.SkipFiles) {
						return nil
					}
					content, _ := os.ReadFile(path)
					if isCompiledBinary(content) {
						return nil
					}
					fi, _ := d.Info()
					visit(rel, fi, content)
					return nil
				})
				continue
			}
			rel, _ := filepath.Rel(root, match)
			rel = filepath.ToSlash(rel)
			if buildinfoSkipFile(rel, spec.SkipFiles) {
				continue
			}
			content, _ := os.ReadFile(match)
			if isCompiledBinary(content) {
				continue
			}
			visit(rel, info, content)
		}
	}
	sort.Slice(listed, func(i, j int) bool { return listed[i].rel < listed[j].rel })
	logf("  Files (%d):\n", len(listed))
	for _, e := range listed {
		logf("    %s  %10d  %s\n", e.hash[:12], e.size, e.rel)
	}
	return nil
}

// isCompiledBinary reports whether content begins with a recognised compiled
// executable magic number. The freshness fingerprint excludes these because
// stray build artifacts (e.g., `go build` output sitting next to source files
// like `cli/<binary-name>` or `cli/<legacy-binary-name>`) would otherwise
// rewrite the fingerprint on every rebuild and trip the rebuild-loop guard.
//
// Recognised formats:
//
//   - ELF (Linux, BSD, etc.):       7F 45 4C 46
//   - Mach-O 32 / 64 / fat (macOS): FE ED FA CE / FE ED FA CF / CA FE BA BE
//     (and reverse-byte-order variants)
//   - PE / COFF (Windows):           4D 5A ("MZ")
//   - WebAssembly module:            00 61 73 6D
//
// Java .class files share the CA FE BA BE prefix, but those are also
// compiled artifacts and equally inappropriate as freshness inputs.
func isCompiledBinary(content []byte) bool {
	if len(content) < 4 {
		return false
	}
	prefix4 := [4]byte{content[0], content[1], content[2], content[3]}
	switch prefix4 {
	case [4]byte{0x7F, 0x45, 0x4C, 0x46}, // ELF
		[4]byte{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O 32
		[4]byte{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O 64
		[4]byte{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O 32 reverse
		[4]byte{0xCF, 0xFA, 0xED, 0xFE}, // Mach-O 64 reverse
		[4]byte{0xCA, 0xFE, 0xBA, 0xBE}, // Mach-O fat / Java class
		[4]byte{0x00, 0x61, 0x73, 0x6D}: // WebAssembly
		return true
	}
	if content[0] == 'M' && content[1] == 'Z' {
		return true
	}
	return false
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
		if pathHasComponent(path, skip) {
			return true
		}
	}
	return false
}

// buildinfoSkipFile reports whether a relative path should be excluded from
// the freshness fingerprint. A skip pattern matches when it equals any path
// component — so "swarm-manager" skips both `swarm-manager` (binary at the
// context root) AND `cli/swarm-manager` (stray binary that landed inside the
// freshness glob via `go build` in the cli dir). Without component-level
// matching, leftover binaries inside the source tree would rewrite the
// fingerprint on every rebuild and trip the rebuild-loop guard.
//
// Compiled-binary files are filtered separately by [isCompiledBinary] at
// read time; this function handles only path-based exclusions.
func buildinfoSkipFile(path string, extra []string) bool {
	path = strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	for _, skip := range []string{"build.meta"} {
		if pathHasComponent(path, skip) {
			return true
		}
	}
	for _, skip := range extra {
		if pathHasComponent(path, skip) {
			return true
		}
	}
	return false
}

// pathHasComponent reports whether the slash-separated path matches the
// skip pattern want. Single-component patterns (no `/`) match if any path
// segment is exactly equal to want — so `swarm-manager` excludes both
// `swarm-manager` (binary at the root) AND `cli/swarm-manager` (stray
// binary that landed inside a `cli/**` glob via `go build` in cli/).
// Multi-segment patterns (e.g., `custom/cache`) match by prefix —
// `custom/cache/index.json` is excluded but `other/custom/cache.json` is
// not. Empty want never matches.
func pathHasComponent(path, want string) bool {
	if want == "" {
		return false
	}
	if strings.Contains(want, "/") {
		return path == want || strings.HasPrefix(path, want+"/")
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == want {
			return true
		}
	}
	return false
}
