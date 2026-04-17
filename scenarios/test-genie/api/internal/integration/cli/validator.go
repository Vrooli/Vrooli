package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// CommandLookup resolves a command in PATH.
type CommandLookup func(file string) (string, error)

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
	lookup    CommandLookup
	logWriter io.Writer
}

type serviceManifest struct {
	CLI struct {
		Enabled bool   `json:"enabled"`
		Command string `json:"command"`
		Adapter struct {
			Kind      string `json:"kind"`
			ModuleDir string `json:"module_dir"`
		} `json:"adapter"`
		Install []struct {
			OS   []string `json:"os"`
			Kind string   `json:"kind"`
			Run  string   `json:"run"`
		} `json:"install"`
		Invoke struct {
			Kind    string `json:"kind"`
			Command string `json:"command"`
		} `json:"invoke"`
	} `json:"cli"`
}

// Option configures a validator.
type Option func(*validator)

// New creates a new CLI validator with the given config and options.
// Applies sensible defaults for any unset configuration values.
func New(config Config, opts ...Option) Validator {
	v := &validator{
		config:    config,
		lookup:    exec.LookPath,
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

// WithLookup sets the command lookup function (for testing).
func WithLookup(lookup CommandLookup) Option {
	return func(v *validator) {
		v.lookup = lookup
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

	// Step 1: Discover CLI invocation target
	binaryPath, err := v.discoverBinary(ctx)
	if err != nil {
		return ValidationResult{
			Result: types.FailMisconfiguration(err, "Declare a valid cli.invoke command and ensure the scenario can install or expose that executable."),
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

// discoverBinary resolves the CLI invocation target, preferring manifest-declared
// installed commands and falling back to local executables for legacy scenarios.
func (v *validator) discoverBinary(ctx context.Context) (string, error) {
	cliDir := filepath.Join(v.config.ScenarioDir, "cli")
	info, err := os.Stat(cliDir)
	if err != nil {
		return "", fmt.Errorf("cli directory missing: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cli path is not a directory: %s", cliDir)
	}

	manifest, manifestErr := v.loadServiceManifest()

	if manifestErr == nil {
		if command, ok := v.resolveInstalledCommand(ctx, manifest); ok {
			return command, nil
		}
	}

	// Build candidate list from the declared CLI contract first.
	var candidates []string
	if command := strings.TrimSpace(manifest.CLI.Invoke.Command); command != "" {
		candidates = append(candidates, v.commandCandidates(cliDir, command)...)
	}
	if command := strings.TrimSpace(manifest.CLI.Command); command != "" {
		candidates = append(candidates, v.commandCandidates(cliDir, command)...)
	}

	// Fall back to scenario-name heuristics for older manifests.
	name := strings.TrimSpace(v.config.ScenarioName)
	if name != "" {
		candidates = append(candidates, v.commandCandidates(cliDir, name)...)
	}

	// Also check for test-genie as fallback
	candidates = append(candidates, v.commandCandidates(cliDir, "test-genie")...)

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
		if v.isInstallerScript(path) {
			continue
		}
		if v.isExecutable(path) {
			return path, nil
		}
	}

	if manifestErr == nil && strings.TrimSpace(manifest.CLI.Invoke.Kind) == "installed_command" {
		return "", fmt.Errorf("unable to resolve installed CLI command %q and no local executable fallback found under %s", strings.TrimSpace(manifest.CLI.Invoke.Command), cliDir)
	}

	return "", fmt.Errorf("no executable CLI binary found under %s", cliDir)
}

func (v *validator) loadServiceManifest() (serviceManifest, error) {
	var manifest serviceManifest

	manifestPath := filepath.Join(v.config.ScenarioDir, ".vrooli", "service.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, fmt.Errorf("parse service manifest: %w", err)
	}

	return manifest, nil
}

func (v *validator) resolveInstalledCommand(ctx context.Context, manifest serviceManifest) (string, bool) {
	command := strings.TrimSpace(manifest.CLI.Invoke.Command)
	if !manifest.CLI.Enabled || strings.TrimSpace(manifest.CLI.Invoke.Kind) != "installed_command" || command == "" {
		return "", false
	}

	if v.commandAvailable(command) {
		return command, true
	}

	if err := v.runInstaller(ctx, manifest); err == nil && v.commandAvailable(command) {
		return command, true
	}

	return "", false
}

func (v *validator) commandCandidates(cliDir, command string) []string {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	candidates := []string{filepath.Join(cliDir, command)}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(cliDir, command+".exe"),
			filepath.Join(cliDir, command+".bat"),
			filepath.Join(cliDir, command+".cmd"),
		)
	}

	return candidates
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
	if v.isInstallerScript(path) {
		return false
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough.
		return true
	}
	return info.Mode()&0o111 != 0
}

func (v *validator) isInstallerScript(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "install.sh" || base == "install.ps1" || base == "install.bat" || base == "install.cmd"
}

func (v *validator) commandAvailable(command string) bool {
	if strings.TrimSpace(command) == "" || v.lookup == nil {
		return false
	}
	_, err := v.lookup(command)
	return err == nil
}

func (v *validator) runInstaller(ctx context.Context, manifest serviceManifest) error {
	if v.executor == nil {
		return fmt.Errorf("command executor not configured")
	}

	for _, step := range manifest.CLI.Install {
		if !installStepApplies(step.OS) {
			continue
		}
		if strings.TrimSpace(step.Kind) != "command" || strings.TrimSpace(step.Run) == "" {
			continue
		}
		v.logStep("Running CLI install step: %s", strings.TrimSpace(step.Run))
		return v.executor(ctx, v.config.ScenarioDir, v.logWriter, installerShell(), installerArgs(step.Run)...)
	}

	return fmt.Errorf("no applicable cli.install command for %s", runtime.GOOS)
}

func installStepApplies(targets []string) bool {
	if len(targets) == 0 {
		return true
	}
	for _, target := range targets {
		if strings.EqualFold(strings.TrimSpace(target), runtime.GOOS) {
			return true
		}
	}
	return false
}

func installerShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

func installerArgs(command string) []string {
	if runtime.GOOS == "windows" {
		return []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command}
	}
	return []string{"-lc", command}
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
