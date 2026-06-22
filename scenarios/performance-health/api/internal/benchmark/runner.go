package benchmark

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"performance-health/internal/scenarioroot"
)

// Default build budgets (milliseconds), mirroring test-genie's native perf phase
// defaults so the migrated axis ① behaves identically when
// .vrooli/testing.json declares no `performance.budgets` block.
const (
	defaultGoBuildMaxMs = 90_000
	defaultUIBuildMaxMs = 180_000
)

// CommandExecutor runs a build command in a directory. Injectable for tests so
// the runner never shells out under unit test.
type CommandExecutor func(ctx context.Context, dir, name string, args ...string) error

// CommandLookup reports whether a command is on PATH. Injectable for tests.
type CommandLookup func(name string) (string, error)

// CLIRunner is the production benchmark Runner: it resolves the scenario root,
// loads thresholds from .vrooli/testing.json, times `go build ./...` (api/) and
// the UI package-manager build (ui/), and flags any surface over its budget.
//
// Early-exit semantics (migrated from test-genie): when the Go build itself
// FAILS to compile the run returns immediately as FAILED — the UI build is not
// attempted. A surface that is merely over budget does not early-exit; both
// surfaces are still timed so the caller sees the full picture.
type CLIRunner struct {
	// RepoRoot is the Vrooli repo root used to resolve a scenario by name; empty
	// resolves lazily from the repo contract.
	RepoRoot string

	// Resolve maps (scenario, path) → absolute scenario root. Injectable so
	// tests point at a fixture tree. nil uses the repo-contract resolver.
	Resolve func(scenario, path string) (string, error)

	// Exec runs build commands; nil uses os/exec.
	Exec CommandExecutor

	// Lookup checks command availability; nil uses exec.LookPath.
	Lookup CommandLookup
}

var _ Runner = (*CLIRunner)(nil)

// thresholds is the performance section of .vrooli/testing.json that the
// benchmark runner consumes (build budgets only).
type thresholds struct {
	goBuildMax time.Duration
	uiBuildMax time.Duration
}

type testingConfig struct {
	Performance struct {
		Budgets struct {
			GoBuildMaxMs *int64 `json:"go_build_max_ms"`
			UIBuildMaxMs *int64 `json:"ui_build_max_ms"`
		} `json:"budgets"`
	} `json:"performance"`
}

// loadThresholds reads the build budgets from the `performance.budgets` block of
// <root>/.vrooli/testing.json — the single source of truth the budgets domain
// gates on — falling back to the test-genie-parity defaults when the file or
// block is absent. These thresholds only mark the informational OverBudget flag
// on each timing; the budgets domain is the sole emitter of gating findings.
func loadThresholds(root string) thresholds {
	t := thresholds{
		goBuildMax: time.Duration(defaultGoBuildMaxMs) * time.Millisecond,
		uiBuildMax: time.Duration(defaultUIBuildMaxMs) * time.Millisecond,
	}
	raw, err := os.ReadFile(filepath.Join(root, ".vrooli", "testing.json"))
	if err != nil {
		return t
	}
	var cfg testingConfig
	if json.Unmarshal(raw, &cfg) != nil {
		return t
	}
	if cfg.Performance.Budgets.GoBuildMaxMs != nil {
		t.goBuildMax = time.Duration(*cfg.Performance.Budgets.GoBuildMaxMs) * time.Millisecond
	}
	if cfg.Performance.Budgets.UIBuildMaxMs != nil {
		t.uiBuildMax = time.Duration(*cfg.Performance.Budgets.UIBuildMaxMs) * time.Millisecond
	}
	return t
}

// Run times the build surfaces for one scenario.
func (r *CLIRunner) Run(ctx context.Context, scenario, path string) (Result, error) {
	root, err := r.resolveRoot(scenario, path)
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "could not resolve scenario root: " + err.Error()}, nil
	}
	th := loadThresholds(root)
	exec := r.exec()
	lookup := r.lookup()

	var timings []BuildTiming

	// Surface: go build ./... (api/). Skip cleanly when there is no api surface
	// or no Go toolchain; FAIL (early-exit) when the build itself errors.
	apiDir := filepath.Join(root, "api")
	if isDir(apiDir) {
		if _, lerr := lookup("go"); lerr != nil {
			return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "go toolchain not found"}, nil
		}
		dur, berr := timeBuild(ctx, exec, apiDir, "go", goBuildArgs()...)
		timings = append(timings, BuildTiming{
			Surface:    "go",
			DurationMs: dur.Milliseconds(),
			BudgetMs:   th.goBuildMax.Milliseconds(),
			OverBudget: th.goBuildMax > 0 && dur > th.goBuildMax,
		})
		if berr != nil {
			// Early-exit: a broken Go build short-circuits the benchmark.
			return Result{
				Scenario: scenario,
				Outcome:  OutcomeFailed,
				Timings:  MarkOverBudget(timings),
				Reason:   "go build failed: " + berr.Error(),
			}, nil
		}
	}

	// Surface: UI build (ui/). Skipped cleanly when there is no UI workspace,
	// no build script, or no package manager.
	uiTiming, bundleBytes, uiReason, uiFailed := r.benchmarkUI(ctx, root, th, exec, lookup)
	if uiTiming != nil {
		timings = append(timings, *uiTiming)
	}
	if uiFailed {
		return Result{Scenario: scenario, Outcome: OutcomeFailed, Timings: MarkOverBudget(timings), Reason: uiReason}, nil
	}

	if len(timings) == 0 {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "no buildable surfaces (no api/ or ui/)"}, nil
	}
	return Result{Scenario: scenario, Outcome: OutcomeMeasured, Timings: MarkOverBudget(timings), BundleBytes: bundleBytes}, nil
}

// benchmarkUI times the UI build, returning (timing, bundleBytes, reason,
// failed). A nil timing with empty reason means the UI surface was cleanly
// skipped. bundleBytes is the total size of the production build output dir,
// measured right after a successful build (0 when no output dir is found).
func (r *CLIRunner) benchmarkUI(ctx context.Context, root string, th thresholds, exec CommandExecutor, lookup CommandLookup) (*BuildTiming, int64, string, bool) {
	uiDir := detectWorkspaceDir(root)
	if uiDir == "" {
		return nil, 0, "", false
	}
	manifest, ok := loadPackageManifest(filepath.Join(uiDir, "package.json"))
	if !ok || strings.TrimSpace(manifest.Scripts["build"]) == "" {
		return nil, 0, "", false
	}
	pm := detectPackageManager(manifest, uiDir)
	if _, lerr := lookup(pm); lerr != nil {
		return nil, 0, "", false
	}
	dur, berr := timeBuild(ctx, exec, uiDir, pm, "run", "build")
	timing := &BuildTiming{
		Surface:    "ui",
		DurationMs: dur.Milliseconds(),
		BudgetMs:   th.uiBuildMax.Milliseconds(),
		OverBudget: th.uiBuildMax > 0 && dur > th.uiBuildMax,
	}
	if berr != nil {
		return timing, 0, "ui build failed: " + berr.Error(), true
	}
	return timing, measureBundleBytes(uiDir), "", false
}

// measureBundleBytes sums the file sizes under the UI build output dir. Vite's
// default output is <uiDir>/dist; it falls back to build/ and out/ for other
// toolchains. Returns 0 when no known output dir exists (cheap stat walk; never
// fails the benchmark).
func measureBundleBytes(uiDir string) int64 {
	for _, name := range []string{"dist", "build", "out"} {
		dir := filepath.Join(uiDir, name)
		if !isDir(dir) {
			continue
		}
		var total int64
		_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // best-effort sizing; skip unreadable entries
			}
			if info, ierr := d.Info(); ierr == nil {
				total += info.Size()
			}
			return nil
		})
		if total > 0 {
			return total
		}
	}
	return 0
}

func (r *CLIRunner) resolveRoot(scenario, path string) (string, error) {
	if r.Resolve != nil {
		return r.Resolve(scenario, path)
	}
	return scenarioroot.Resolve(r.RepoRoot, scenario, path)
}

func (r *CLIRunner) exec() CommandExecutor {
	if r.Exec != nil {
		return r.Exec
	}
	return defaultExec
}

func (r *CLIRunner) lookup() CommandLookup {
	if r.Lookup != nil {
		return r.Lookup
	}
	return exec.LookPath
}

// timeBuild runs one build command in dir and returns its wall-clock duration.
func timeBuild(ctx context.Context, run CommandExecutor, dir, name string, args ...string) (time.Duration, error) {
	start := time.Now()
	err := run(ctx, dir, name, args...)
	return time.Since(start), err
}

// goBuildArgs builds `go build -o <tmp> ./...` so the multi-package build does
// not write binaries into the source tree (matching test-genie's runner).
func goBuildArgs() []string {
	tmp, err := os.MkdirTemp("", "perf-health-bench-*")
	if err != nil {
		return []string{"build", "./..."}
	}
	// Best-effort cleanup is handled by the OS temp reaper; the dir is small.
	return []string{"build", "-o", tmp, "./..."}
}

func defaultExec(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if name == "go" && os.Getenv("GOWORK") == "" {
		cmd.Env = append(os.Environ(), "GOWORK=off")
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// --- UI workspace detection (mirrors test-genie's nodejs validator shape) ---

type packageManifest struct {
	Scripts        map[string]string `json:"scripts"`
	PackageManager string            `json:"packageManager"`
}

func detectWorkspaceDir(root string) string {
	for _, c := range []string{filepath.Join(root, "ui"), root} {
		if fileExists(filepath.Join(c, "package.json")) {
			return c
		}
	}
	return ""
}

func loadPackageManifest(path string) (packageManifest, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return packageManifest{}, false
	}
	var doc packageManifest
	if json.Unmarshal(raw, &doc) != nil {
		return packageManifest{}, false
	}
	if doc.Scripts == nil {
		doc.Scripts = map[string]string{}
	}
	return doc, true
}

func detectPackageManager(manifest packageManifest, dir string) string {
	if pm := parsePackageManager(manifest.PackageManager); pm != "" {
		return pm
	}
	switch {
	case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(dir, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

func parsePackageManager(raw string) string {
	lowered := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(lowered, "pnpm"):
		return "pnpm"
	case strings.HasPrefix(lowered, "yarn"):
		return "yarn"
	case strings.HasPrefix(lowered, "npm"):
		return "npm"
	default:
		return ""
	}
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
