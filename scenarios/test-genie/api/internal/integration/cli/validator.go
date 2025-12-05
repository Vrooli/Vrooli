package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"test-genie/internal/structure/types"
)

// Validator validates CLI functionality.
type Validator interface {
	// Validate performs all CLI validation checks.
	Validate(ctx context.Context) ValidationResult
}

// CommandExecutor executes commands and returns any error.
type CommandExecutor func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

// CommandCapture executes commands and captures output.
type CommandCapture func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)

// AdaptExecutor adapts an integration.CommandExecutor to cli.CommandExecutor.
// Since they have the same signature, this is just a type conversion helper.
func AdaptExecutor(exec func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error) CommandExecutor {
	return CommandExecutor(exec)
}

// AdaptCapture adapts an integration.CommandCapture to cli.CommandCapture.
func AdaptCapture(cap func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)) CommandCapture {
	return CommandCapture(cap)
}

// ValidationResult contains the outcome of CLI validation.
type ValidationResult struct {
	types.Result

	// BinaryPath is the discovered CLI binary path.
	BinaryPath string

	// VersionOutput is the captured version output.
	VersionOutput string
}

// Config holds configuration for CLI validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string
}

// validator implements the Validator interface.
type validator struct {
	config    Config
	executor  CommandExecutor
	capture   CommandCapture
	logWriter io.Writer
}

// Option configures a validator.
type Option func(*validator)

// New creates a new CLI validator.
func New(config Config, opts ...Option) Validator {
	v := &validator{
		config:    config,
		logWriter: io.Discard,
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

// WithExecutor sets the command executor (for testing).
func WithExecutor(exec CommandExecutor) Option {
	return func(v *validator) {
		v.executor = exec
	}
}

// WithCapture sets the command capture function (for testing).
func WithCapture(cap CommandCapture) Option {
	return func(v *validator) {
		v.capture = cap
	}
}

// Validate performs all CLI validation checks.
func (v *validator) Validate(ctx context.Context) ValidationResult {
	if err := ctx.Err(); err != nil {
		return ValidationResult{
			Result: types.FailSystem(err, "Context cancelled"),
		}
	}

	var observations []types.Observation
	observations = append(observations, types.NewSectionObservation("🖥️", "Validating CLI..."))

	// Step 1: Discover CLI binary
	binaryPath, err := v.discoverBinary()
	if err != nil {
		return ValidationResult{
			Result: types.FailMisconfiguration(err, "Add an executable CLI binary under cli/ so operators can invoke the scenario."),
		}
	}
	v.logStep("CLI binary verified: %s", binaryPath)
	observations = append(observations, types.NewSuccessObservation("CLI binary executable"))

	// Step 2: Validate help command
	if err := v.validateHelp(ctx, binaryPath); err != nil {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI help command failed: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Run `%s help` manually to inspect the error output.", binaryPath),
				Observations: observations,
			},
			BinaryPath: binaryPath,
		}
	}
	v.logStep("CLI help command succeeded")
	observations = append(observations, types.NewSuccessObservation("CLI help verified"))

	// Step 3: Validate version command
	versionOutput, err := v.validateVersion(ctx, binaryPath)
	if err != nil {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI version command failed: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Ensure %s version works without interactive prompts.", binaryPath),
				Observations: observations,
			},
			BinaryPath: binaryPath,
		}
	}
	if !strings.Contains(strings.ToLower(versionOutput), "version") {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI version output malformed: %s", strings.TrimSpace(versionOutput)),
				FailureClass: types.FailureClassMisconfiguration,
				Remediation:  fmt.Sprintf("Update %s to print the version string used by operator tooling.", binaryPath),
				Observations: observations,
			},
			BinaryPath:    binaryPath,
			VersionOutput: versionOutput,
		}
	}
	v.logStep("CLI version output: %s", strings.TrimSpace(versionOutput))
	observations = append(observations, types.NewSuccessObservation("CLI version reported"))

	return ValidationResult{
		Result: types.Result{
			Success:      true,
			Observations: observations,
		},
		BinaryPath:    binaryPath,
		VersionOutput: versionOutput,
	}
}

// discoverBinary finds the CLI binary in the scenario's cli/ directory.
func (v *validator) discoverBinary() (string, error) {
	cliDir := filepath.Join(v.config.ScenarioDir, "cli")
	info, err := os.Stat(cliDir)
	if err != nil {
		return "", fmt.Errorf("cli directory missing: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cli path is not a directory: %s", cliDir)
	}

	// Build candidate list based on scenario name
	var candidates []string
	name := strings.TrimSpace(v.config.ScenarioName)
	if name != "" {
		candidates = append(candidates,
			filepath.Join(cliDir, name),
			filepath.Join(cliDir, name+".sh"),
			filepath.Join(cliDir, name+".exe"),
		)
	}
	// Also check for test-genie as fallback
	candidates = append(candidates,
		filepath.Join(cliDir, "test-genie"),
		filepath.Join(cliDir, "test-genie.exe"),
	)

	// Check preferred candidates first
	for _, candidate := range candidates {
		if v.isExecutable(candidate) {
			return candidate, nil
		}
	}

	// Fall back to first executable found in cli directory
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to list cli directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(cliDir, entry.Name())
		if v.isExecutable(path) {
			return path, nil
		}
	}

	return "", fmt.Errorf("no executable CLI binary found under %s", cliDir)
}

// isExecutable checks if a file exists and is executable.
func (v *validator) isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough.
		return true
	}
	return info.Mode()&0o111 != 0
}

// validateHelp runs the help command.
func (v *validator) validateHelp(ctx context.Context, binaryPath string) error {
	if v.executor == nil {
		return fmt.Errorf("command executor not configured")
	}
	return v.executor(ctx, "", v.logWriter, binaryPath, "help")
}

// validateVersion runs the version command and captures output.
func (v *validator) validateVersion(ctx context.Context, binaryPath string) (string, error) {
	if v.capture == nil {
		return "", fmt.Errorf("command capture not configured")
	}
	return v.capture(ctx, "", v.logWriter, binaryPath, "version")
}

// logStep writes a step message to the log.
func (v *validator) logStep(format string, args ...interface{}) {
	if v.logWriter == nil {
		return
	}
	fmt.Fprintf(v.logWriter, format+"\n", args...)
}
