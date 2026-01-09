package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"test-genie/internal/structure/types"
)

// Default argument patterns for help and version commands.
// These accommodate both cli-core style subcommands and standard flag-based CLIs.
var (
	DefaultHelpArgs    = []string{"help", "--help", "-h"}
	DefaultVersionArgs = []string{"version", "--version", "-v"}
)

// Default timeouts and settings.
const (
	DefaultNoArgsTimeoutMs = 5000
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

	// HelpArgs specifies argument patterns to try for help command, in order of preference.
	// Default: ["help", "--help", "-h"]
	HelpArgs []string

	// VersionArgs specifies argument patterns to try for version command, in order of preference.
	// Default: ["version", "--version", "-v"]
	VersionArgs []string

	// RequireVersionKeyword controls whether version output must contain the word "version".
	// Default: false (relaxed - any non-empty output passes)
	RequireVersionKeyword bool

	// CheckUnknownCommand controls whether to verify the CLI handles unknown commands gracefully.
	// Default: true
	CheckUnknownCommand bool

	// CheckNoArgs controls whether to verify the CLI handles no arguments gracefully.
	// Default: true
	CheckNoArgs bool

	// NoArgsTimeoutMs is the maximum time to wait for the no-args check in milliseconds.
	// Default: 5000
	NoArgsTimeoutMs int64
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

// New creates a new CLI validator with the given config and options.
// Applies sensible defaults for any unset configuration values.
func New(config Config, opts ...Option) Validator {
	v := &validator{
		config:    config,
		logWriter: io.Discard,
	}

	// Apply defaults
	if len(v.config.HelpArgs) == 0 {
		v.config.HelpArgs = DefaultHelpArgs
	}
	if len(v.config.VersionArgs) == 0 {
		v.config.VersionArgs = DefaultVersionArgs
	}
	if v.config.NoArgsTimeoutMs == 0 {
		v.config.NoArgsTimeoutMs = DefaultNoArgsTimeoutMs
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

	// Step 2: Validate no-args behavior (if enabled)
	// Run this early since it can detect hangs
	if v.config.CheckNoArgs {
		if err := v.validateNoArgs(ctx, binaryPath); err != nil {
			return ValidationResult{
				Result: types.Result{
					Success:      false,
					Error:        fmt.Errorf("CLI no-args check failed: %w", err),
					FailureClass: types.FailureClassSystem,
					Remediation:  fmt.Sprintf("Ensure %s handles being called with no arguments gracefully (should print help and exit 0).", filepath.Base(binaryPath)),
					Observations: observations,
				},
				BinaryPath: binaryPath,
			}
		}
		v.logStep("CLI handles no-args gracefully")
		observations = append(observations, types.NewSuccessObservation("CLI no-args behavior verified"))
	}

	// Step 3: Validate help command
	helpArg, err := v.validateHelp(ctx, binaryPath)
	if err != nil {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI help command failed: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Run `%s %s` manually to inspect the error output. Tried: %v", filepath.Base(binaryPath), v.config.HelpArgs[0], v.config.HelpArgs),
				Observations: observations,
			},
			BinaryPath: binaryPath,
		}
	}
	v.logStep("CLI help command succeeded (using '%s')", helpArg)
	observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("CLI help verified (%s)", helpArg)))

	// Step 4: Validate version command
	versionOutput, versionArg, err := v.validateVersion(ctx, binaryPath)
	if err != nil {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI version command failed: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Ensure %s supports a version command. Tried: %v", filepath.Base(binaryPath), v.config.VersionArgs),
				Observations: observations,
			},
			BinaryPath: binaryPath,
		}
	}

	// Validate version output content
	if err := v.validateVersionOutput(versionOutput); err != nil {
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("CLI version output invalid: %w", err),
				FailureClass: types.FailureClassMisconfiguration,
				Remediation:  fmt.Sprintf("Update %s to print a valid version string (got: %q).", filepath.Base(binaryPath), strings.TrimSpace(versionOutput)),
				Observations: observations,
			},
			BinaryPath:    binaryPath,
			VersionOutput: versionOutput,
		}
	}

	v.logStep("CLI version output: %s (using '%s')", strings.TrimSpace(versionOutput), versionArg)
	observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("CLI version reported (%s)", versionArg)))

	// Step 5: Validate unknown command handling (if enabled)
	if v.config.CheckUnknownCommand {
		if err := v.validateUnknownCommand(ctx, binaryPath); err != nil {
			return ValidationResult{
				Result: types.Result{
					Success:      false,
					Error:        fmt.Errorf("CLI unknown command check failed: %w", err),
					FailureClass: types.FailureClassMisconfiguration,
					Remediation:  fmt.Sprintf("Update %s to return non-zero exit code for unknown commands.", filepath.Base(binaryPath)),
					Observations: observations,
				},
				BinaryPath:    binaryPath,
				VersionOutput: versionOutput,
			}
		}
		v.logStep("CLI handles unknown commands correctly")
		observations = append(observations, types.NewSuccessObservation("CLI unknown command handling verified"))
	}

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

// validateNoArgs runs the CLI with no arguments and verifies it doesn't hang and exits cleanly.
func (v *validator) validateNoArgs(ctx context.Context, binaryPath string) error {
	if v.executor == nil {
		return fmt.Errorf("command executor not configured")
	}

	// Create a context with timeout
	timeout := time.Duration(v.config.NoArgsTimeoutMs) * time.Millisecond
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := v.executor(timeoutCtx, "", v.logWriter, binaryPath)
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("CLI hung when called with no arguments (timeout after %v)", timeout)
	}
	// Note: We accept any exit code here - some CLIs exit 0 (print help), some exit 1 (error: no command)
	// The important thing is that it doesn't hang
	if err != nil {
		// Log but don't fail - the key test is that it doesn't hang
		v.logStep("CLI no-args returned error (acceptable): %v", err)
	}
	return nil
}

// validateHelp tries each help argument pattern until one succeeds.
// Returns the successful argument and any error.
func (v *validator) validateHelp(ctx context.Context, binaryPath string) (string, error) {
	if v.executor == nil {
		return "", fmt.Errorf("command executor not configured")
	}

	var lastErr error
	for _, arg := range v.config.HelpArgs {
		err := v.executor(ctx, "", v.logWriter, binaryPath, arg)
		if err == nil {
			return arg, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("all help arguments failed (tried %v): %w", v.config.HelpArgs, lastErr)
}

// validateVersion tries each version argument pattern until one succeeds.
// Returns the output, successful argument, and any error.
func (v *validator) validateVersion(ctx context.Context, binaryPath string) (string, string, error) {
	if v.capture == nil {
		return "", "", fmt.Errorf("command capture not configured")
	}

	var lastErr error
	for _, arg := range v.config.VersionArgs {
		output, err := v.capture(ctx, "", v.logWriter, binaryPath, arg)
		if err == nil {
			return output, arg, nil
		}
		lastErr = err
	}

	return "", "", fmt.Errorf("all version arguments failed (tried %v): %w", v.config.VersionArgs, lastErr)
}

// validateVersionOutput checks if the version output is valid.
func (v *validator) validateVersionOutput(output string) error {
	trimmed := strings.TrimSpace(output)

	// Must have some output
	if trimmed == "" {
		return fmt.Errorf("version output is empty")
	}

	// Optionally require "version" keyword
	if v.config.RequireVersionKeyword {
		if !strings.Contains(strings.ToLower(trimmed), "version") {
			return fmt.Errorf("version output missing 'version' keyword: %s", trimmed)
		}
	}

	return nil
}

// validateUnknownCommand verifies the CLI returns non-zero for unknown commands.
func (v *validator) validateUnknownCommand(ctx context.Context, binaryPath string) error {
	if v.executor == nil {
		return fmt.Errorf("command executor not configured")
	}

	// Use a clearly nonsensical command that no CLI should recognize
	nonsenseCmd := "__test_genie_nonexistent_command_12345__"

	err := v.executor(ctx, "", v.logWriter, binaryPath, nonsenseCmd)
	if err == nil {
		return fmt.Errorf("CLI returned success (exit 0) for unknown command %q - should return non-zero", nonsenseCmd)
	}

	// Error is expected - CLI correctly rejected unknown command
	return nil
}

// logStep writes a step message to the log.
func (v *validator) logStep(format string, args ...interface{}) {
	if v.logWriter == nil {
		return
	}
	fmt.Fprintf(v.logWriter, format+"\n", args...)
}
