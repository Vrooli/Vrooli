package staleness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Environment variable names
const (
	// SkipEnvVar can be set to "true" to skip staleness checking entirely.
	SkipEnvVar = "VROOLI_API_SKIP_STALE_CHECK"
)

// CheckerConfig configures the staleness checker behavior.
type CheckerConfig struct {
	// APIDir is the directory containing the API source code.
	// Default: directory containing the binary (os.Executable())
	APIDir string

	// BinaryPath is the path to the API binary being checked.
	// Default: os.Executable()
	BinaryPath string

	// Logger for output messages. Default: fmt.Fprintf(os.Stderr, ...)
	Logger func(format string, args ...interface{})

	// Disabled completely skips all staleness checking.
	Disabled bool

	// SkipRebuild only logs staleness without attempting rebuild.
	// Useful for debugging or when rebuild should be handled externally.
	SkipRebuild bool

	// LifecycleManaged means the binary is supervised by the Vrooli lifecycle.
	// In this mode a lifecycle freshness manifest, when present, is authoritative:
	// preflight verifies it but does not rebuild/re-exec behind the supervisor.
	LifecycleManaged bool

	// ManifestPath overrides the lifecycle freshness manifest location.
	// Default: <BinaryPath>.freshness.json.
	ManifestPath string
}

// Checker detects stale API binaries and optionally rebuilds them.
type Checker struct {
	cfg CheckerConfig
}

// NewChecker creates a new staleness checker with the given configuration.
func NewChecker(cfg CheckerConfig) *Checker {
	return &Checker{cfg: cfg}
}

// CheckAndMaybeRebuild checks if the API binary is stale and optionally rebuilds.
// Returns true if the process was restarted after rebuild.
//
// This should be called at the very start of main(), before any initialization:
//
//	func main() {
//	    checker := staleness.NewChecker(staleness.CheckerConfig{})
//	    if checker.CheckAndMaybeRebuild() {
//	        return // Process was re-exec'd
//	    }
//	    // ... rest of initialization
//	}
func (c *Checker) CheckAndMaybeRebuild() bool {
	// Check if disabled via config or environment
	if c.cfg.Disabled || os.Getenv(SkipEnvVar) == "true" {
		return false
	}

	// Resolve paths
	binaryPath, apiDir, err := c.resolvePaths()
	if err != nil {
		c.log("api-core: unable to resolve paths for staleness check: %v\n", err)
		return false
	}

	// Check only the lifecycle manifest. Rebuilding and process replacement
	// belong to the lifecycle supervisor, which owns the complete contract.
	if c.cfg.LifecycleManaged {
		handled, stale, reason := c.checkLifecycleFreshnessManifest(binaryPath, apiDir)
		if handled && stale {
			c.log("api-core: lifecycle-managed binary is stale per freshness manifest (%s); deferring rebuild to Vrooli lifecycle\n", reason)
		}
		return false
	}
	return false
}

func (c *Checker) checkLifecycleFreshnessManifest(binaryPath, apiDir string) (handled bool, stale bool, reason string) {
	manifestPath := strings.TrimSpace(c.cfg.ManifestPath)
	if manifestPath == "" {
		manifestPath = cliutil.FreshnessManifestPath(binaryPath)
	}
	manifest, ok, err := cliutil.ReadFreshnessManifest(manifestPath)
	if err != nil {
		c.log("api-core: lifecycle freshness manifest unreadable (%v); deferring rebuild to Vrooli lifecycle\n", err)
		return true, false, ""
	}
	if !ok {
		return false, false, ""
	}

	spec, keyInputs, err := c.lifecycleFreshnessSpec(binaryPath, apiDir, manifest)
	if err != nil {
		c.log("api-core: lifecycle freshness manifest could not be evaluated (%v); deferring rebuild to Vrooli lifecycle\n", err)
		return true, false, ""
	}
	verdict, err := cliutil.EvaluateFreshness(spec, manifest, keyInputs)
	if err != nil {
		c.log("api-core: lifecycle freshness evaluation failed (%v); deferring rebuild to Vrooli lifecycle\n", err)
		return true, false, ""
	}
	if !verdict.Stale {
		return true, false, ""
	}
	return true, true, formatLifecycleFreshnessReason(verdict)
}

func (c *Checker) lifecycleFreshnessSpec(binaryPath, apiDir string, manifest cliutil.FreshnessManifest) (cliutil.FreshnessSpec, map[string]string, error) {
	contextRoot, err := inferManifestContextRoot(apiDir, manifest)
	if err != nil {
		return cliutil.FreshnessSpec{}, nil, err
	}
	inputs := manifestInputs(manifest)
	if len(inputs) == 0 {
		return cliutil.FreshnessSpec{}, nil, fmt.Errorf("freshness manifest has no file inputs")
	}
	return cliutil.FreshnessSpec{
		SourceRoot:   apiDir,
		ContextRoot:  contextRoot,
		Inputs:       inputs,
		SkipFiles:    []string{filepath.Base(binaryPath)},
		SkipSuffixes: []string{"_test.go", cliutil.FreshnessManifestSuffix},
	}, manifest.KeyInputs, nil
}

func inferManifestContextRoot(apiDir string, manifest cliutil.FreshnessManifest) (string, error) {
	// Lifecycle manifests for repo-root Go replacements express inputs relative
	// to the repository root. A scenario API commonly has its own go.mod too,
	// so inferring the root from the first file (often just "go.mod") can select
	// apiDir and compare the repository go.mod record with api/go.mod. Prefer the
	// ancestor that resolves the largest portion of the manifest's declared
	// input set; this distinguishes a repo-root manifest from a single-module
	// API while remaining tolerant of a deleted input that should be reported as
	// stale later by EvaluateFreshness.
	if inputs := manifest.Inputs; len(inputs) > 0 {
		bestRoot := ""
		bestScore := -1
		for dir := filepath.Clean(apiDir); ; dir = filepath.Dir(dir) {
			score := 0
			for _, input := range inputs {
				input = strings.TrimSpace(input)
				if input == "" || filepath.IsAbs(input) {
					continue
				}
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(input))); err == nil {
					score++
				}
			}
			// Prefer the higher ancestor on ties. This handles manifests whose
			// input list has only a shared go.mod and API-local files.
			if score >= bestScore {
				bestRoot = dir
				bestScore = score
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
		if bestScore > 0 {
			return bestRoot, nil
		}
	}

	var sample string
	for _, entry := range manifest.Files {
		if rel := strings.TrimSpace(entry.Rel); rel != "" {
			sample = filepath.FromSlash(rel)
			break
		}
	}
	if sample == "" {
		return "", fmt.Errorf("freshness manifest has no file inputs")
	}
	for dir := filepath.Clean(apiDir); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, sample)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("could not infer freshness context root from %s", sample)
}

func manifestInputs(manifest cliutil.FreshnessManifest) []string {
	if len(manifest.Inputs) > 0 {
		inputs := append([]string(nil), manifest.Inputs...)
		sort.Strings(inputs)
		return inputs
	}

	seen := map[string]struct{}{}
	for _, entry := range manifest.Files {
		rel := filepath.ToSlash(strings.TrimSpace(entry.Rel))
		if rel == "" {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(rel))
		if dir == "." {
			dir = rel
		}
		seen[dir] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for dir := range seen {
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

func formatLifecycleFreshnessReason(verdict cliutil.FreshnessVerdict) string {
	if strings.TrimSpace(verdict.File) == "" {
		return verdict.Reason
	}
	return verdict.Reason + ": " + verdict.File
}

// resolvePaths determines the binary path and API source directory.
func (c *Checker) resolvePaths() (binaryPath, apiDir string, err error) {
	// Get binary path
	binaryPath = c.cfg.BinaryPath
	if binaryPath == "" {
		binaryPath, err = os.Executable()
		if err != nil {
			return "", "", fmt.Errorf("get executable path: %w", err)
		}
	}

	// Symlink resolution is useful but not required for a manifest check. Some
	// portable filesystems do not expose the link target consistently.
	if resolved, resolveErr := filepath.EvalSymlinks(binaryPath); resolveErr == nil {
		binaryPath = resolved
	}

	// Get API directory
	apiDir = c.cfg.APIDir
	if apiDir == "" {
		apiDir = filepath.Dir(binaryPath)
	}

	return binaryPath, apiDir, nil
}

// checkStaleness compares file timestamps to determine if the binary is stale.
// log outputs a message using the configured logger or stderr.
func (c *Checker) log(format string, args ...interface{}) {
	if c.cfg.Logger != nil {
		c.cfg.Logger(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}
