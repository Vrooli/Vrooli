package nodejs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Validator benchmarks Node.js builds and validates them against time thresholds.
type Validator interface {
	// Benchmark runs a Node.js build and returns the benchmark result.
	Benchmark(ctx context.Context, maxDuration time.Duration) BenchmarkResult
}

// CommandExecutor runs shell commands. Injectable for testing.
type CommandExecutor func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

// CommandLookup checks if a command exists. Injectable for testing.
type CommandLookup func(name string) (string, error)

// validator is the default implementation of Validator.
type validator struct {
	scenarioDir     string
	logWriter       io.Writer
	commandExecutor CommandExecutor
	commandLookup   CommandLookup
}

// Option configures a validator.
type Option func(*validator)

// New creates a new Node.js build benchmark validator.
func New(scenarioDir string, opts ...Option) Validator {
	v := &validator{
		scenarioDir:     scenarioDir,
		logWriter:       io.Discard,
		commandExecutor: defaultCommandExecutor,
		commandLookup:   exec.LookPath,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(v *validator) {
		v.logWriter = w
	}
}

// WithCommandExecutor sets a custom command executor (for testing).
func WithCommandExecutor(executor CommandExecutor) Option {
	return func(v *validator) {
		v.commandExecutor = executor
	}
}

// WithCommandLookup sets a custom command lookup (for testing).
func WithCommandLookup(lookup CommandLookup) Option {
	return func(v *validator) {
		v.commandLookup = lookup
	}
}

// Benchmark implements Validator.
func (v *validator) Benchmark(ctx context.Context, maxDuration time.Duration) BenchmarkResult {
	// Detect Node.js workspace
	nodeDir := v.detectWorkspaceDir()
	if nodeDir == "" {
		logInfo(v.logWriter, "UI workspace not detected")
		return BenchmarkResult{
			Result:  OK().WithObservations(NewSkipObservation("ui workspace not detected")),
			Skipped: true,
		}
	}

	// Load package.json
	manifest, err := v.loadPackageManifest(filepath.Join(nodeDir, "package.json"))
	if err != nil {
		return BenchmarkResult{
			Result: FailMisconfiguration(
				err,
				"Fix package.json so the Node workspace can be parsed.",
			),
		}
	}

	// Check for build script
	buildScript := ""
	if manifest != nil {
		buildScript = manifest.Scripts["build"]
	}
	if buildScript == "" {
		logInfo(v.logWriter, "UI workspace lacks build script")
		return BenchmarkResult{
			Result:  OK().WithObservations(NewSkipObservation("ui workspace lacks build script")),
			Skipped: true,
		}
	}

	// Detect package manager
	manager := v.detectPackageManager(manifest, nodeDir)

	// Check that package manager is available
	if _, err := v.commandLookup(manager); err != nil {
		logError(v.logWriter, "%s not found", manager)
		return BenchmarkResult{
			Result: FailMissingDependency(
				fmt.Errorf("%s command not found: %w", manager, err),
				fmt.Sprintf("Install %s to run UI build benchmarks.", manager),
			),
			PackageManager: manager,
		}
	}

	// Install dependencies if needed
	logInfo(v.logWriter, "Running UI build via %s", manager)
	if _, err := os.Stat(filepath.Join(nodeDir, "node_modules")); os.IsNotExist(err) {
		if installErr := v.installDependencies(ctx, nodeDir, manager); installErr != nil {
			return BenchmarkResult{
				Result: FailSystem(
					fmt.Errorf("%s install failed: %w", manager, installErr),
					"Resolve dependency installation issues before benchmarking the UI build.",
				),
				PackageManager: manager,
			}
		}
	}

	// Run the build
	start := time.Now()
	if err := v.commandExecutor(ctx, nodeDir, v.logWriter, manager, "run", "build"); err != nil {
		duration := time.Since(start)
		logError(v.logWriter, "UI build failed after %s", duration.Round(time.Second))
		return BenchmarkResult{
			Result: FailSystem(
				fmt.Errorf("ui build failed: %w", err),
				"Inspect the UI build output above, fix failures, and rerun the suite.",
			),
			Duration:       duration,
			PackageManager: manager,
		}
	}
	duration := time.Since(start)
	seconds := int(duration.Round(time.Second) / time.Second)
	logSuccess(v.logWriter, "UI build completed in %ds", seconds)

	// Check against threshold
	if duration > maxDuration {
		return BenchmarkResult{
			Result: FailSystem(
				fmt.Errorf("ui build exceeded %s (took %s)", maxDuration, duration.Round(time.Second)),
				"Investigate slow front-end builds or trim unused dependencies.",
			).WithObservations(NewSuccessObservation(fmt.Sprintf("ui build duration: %ds", seconds))),
			Duration:       duration,
			PackageManager: manager,
		}
	}

	return BenchmarkResult{
		Result:         OK().WithObservations(NewSuccessObservation(fmt.Sprintf("ui build duration: %ds", seconds))),
		Duration:       duration,
		PackageManager: manager,
	}
}

// detectWorkspaceDir finds the Node.js workspace directory.
func (v *validator) detectWorkspaceDir() string {
	candidates := []string{
		filepath.Join(v.scenarioDir, "ui"),
		v.scenarioDir,
	}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(candidate, "package.json")) {
			return candidate
		}
	}
	return ""
}

// packageManifest represents package.json structure.
type packageManifest struct {
	Scripts        map[string]string `json:"scripts"`
	PackageManager string            `json:"packageManager"`
}

// loadPackageManifest reads and parses package.json.
func (v *validator) loadPackageManifest(path string) (*packageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc packageManifest
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Scripts == nil {
		doc.Scripts = make(map[string]string)
	}
	return &doc, nil
}

// detectPackageManager determines which package manager to use.
func (v *validator) detectPackageManager(manifest *packageManifest, dir string) string {
	if manifest != nil {
		if mgr := parsePackageManager(manifest.PackageManager); mgr != "" {
			return mgr
		}
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

// parsePackageManager extracts the package manager name from packageManager field.
func parsePackageManager(raw string) string {
	if raw == "" {
		return ""
	}
	lowered := strings.ToLower(raw)
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

// installDependencies installs Node.js dependencies.
func (v *validator) installDependencies(ctx context.Context, dir, manager string) error {
	switch manager {
	case "pnpm":
		return v.commandExecutor(ctx, dir, v.logWriter, "pnpm", "install", "--frozen-lockfile", "--ignore-scripts")
	case "yarn":
		return v.commandExecutor(ctx, dir, v.logWriter, "yarn", "install", "--frozen-lockfile")
	default:
		return v.commandExecutor(ctx, dir, v.logWriter, "npm", "install")
	}
}

// defaultCommandExecutor runs commands using os/exec.
func defaultCommandExecutor(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] %s\n", msg)
}

func logError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] %s\n", msg)
}
