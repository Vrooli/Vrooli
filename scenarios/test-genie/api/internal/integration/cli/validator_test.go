package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestValidator_Success(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithCapture(h.versionCapture("1.0.0")), // No "version" keyword - should pass with default config
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.BinaryPath == "" {
		t.Error("expected binary path to be set")
	}
	if result.VersionOutput != "1.0.0" {
		t.Errorf("expected version output '1.0.0', got: %s", result.VersionOutput)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestValidator_SuccessWithVersionKeyword(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:          h.scenarioDir,
			ScenarioName:         h.scenarioName,
			RequireVersionKeyword: true,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithCapture(h.versionCapture("demo version 1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !strings.Contains(result.VersionOutput, "version") {
		t.Errorf("expected version output to contain 'version', got: %s", result.VersionOutput)
	}
}

func TestValidator_NoCLIDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scenarios", "demo")
	// Don't create cli directory

	v := New(
		Config{
			ScenarioDir:  scenarioDir,
			ScenarioName: "demo",
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when cli directory missing")
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_NoExecutableBinary(t *testing.T) {
	h := newTestHarness(t)
	// Remove executable
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName))

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when no executable found")
	}
	if !strings.Contains(result.Error.Error(), "no executable CLI binary") {
		t.Errorf("expected 'no executable' error, got: %v", result.Error)
	}
}

func TestValidator_HelpCommandFails_AllPatternsTried(t *testing.T) {
	h := newTestHarness(t)
	helpAttempts := []string{}

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			CheckNoArgs:  false, // Disable to focus on help test
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) > 0 {
				helpAttempts = append(helpAttempts, args[0])
				return errors.New("command failed")
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when all help commands fail")
	}
	// Should have tried all default help args
	if len(helpAttempts) != len(DefaultHelpArgs) {
		t.Errorf("expected %d help attempts, got %d: %v", len(DefaultHelpArgs), len(helpAttempts), helpAttempts)
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system failure class, got: %s", result.FailureClass)
	}
}

func TestValidator_HelpCommandSucceeds_SecondPattern(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			HelpArgs:     []string{"help", "--help"}, // First will fail, second will succeed
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "--help" {
				return nil // Only --help succeeds
			}
			if len(args) > 0 && args[0] == "help" {
				return errors.New("unknown command: help")
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success with --help, got error: %v", result.Error)
	}
}

func TestValidator_VersionCommandFails(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
			return "", errors.New("version command failed")
		}),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when version command fails")
	}
	if !strings.Contains(result.Error.Error(), "version command failed") {
		t.Errorf("expected version error, got: %v", result.Error)
	}
}

func TestValidator_VersionOutputEmpty(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("")), // Empty output
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when version output is empty")
	}
	if !strings.Contains(result.Error.Error(), "empty") {
		t.Errorf("expected empty error, got: %v", result.Error)
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_VersionOutputMissingKeyword(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:          h.scenarioDir,
			ScenarioName:         h.scenarioName,
			RequireVersionKeyword: true, // Require "version" keyword
			CheckNoArgs:          false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")), // Missing "version" word
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when version output missing keyword")
	}
	if !strings.Contains(result.Error.Error(), "keyword") {
		t.Errorf("expected keyword error, got: %v", result.Error)
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_ContextCancelled(t *testing.T) {
	h := newTestHarness(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system failure class, got: %s", result.FailureClass)
	}
}

func TestValidator_FindsScenarioNamedBinary(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	expectedPath := filepath.Join(h.scenarioDir, "cli", h.scenarioName)
	if result.BinaryPath != expectedPath {
		t.Errorf("expected binary path %s, got: %s", expectedPath, result.BinaryPath)
	}
}

func TestValidator_FindsFallbackBinary(t *testing.T) {
	h := newTestHarness(t)
	// Remove scenario-named binary, add a different one
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName))
	writeExecutable(t, filepath.Join(h.scenarioDir, "cli", "other-cli"), "#!/bin/bash\necho cli")

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !strings.Contains(result.BinaryPath, "other-cli") {
		t.Errorf("expected fallback binary, got: %s", result.BinaryPath)
	}
}

func TestValidator_NoExecutorConfigured(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		// No executor set
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when executor not configured")
	}
	if !strings.Contains(result.Error.Error(), "executor not configured") {
		t.Errorf("expected executor error, got: %v", result.Error)
	}
}

func TestValidator_NoCaptureConfigured(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		// No capture set
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when capture not configured")
	}
	if !strings.Contains(result.Error.Error(), "capture not configured") {
		t.Errorf("expected capture error, got: %v", result.Error)
	}
}

func TestValidator_UnknownCommandCheck_Passes(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:         h.scenarioDir,
			ScenarioName:        h.scenarioName,
			CheckUnknownCommand: true,
			CheckNoArgs:         false,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) > 0 && strings.HasPrefix(args[0], "__test_genie") {
				return errors.New("unknown command") // Correctly rejects unknown command
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_UnknownCommandCheck_Fails(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:         h.scenarioDir,
			ScenarioName:        h.scenarioName,
			CheckUnknownCommand: true,
			CheckNoArgs:         false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor), // Returns success for everything - wrong behavior
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI accepts unknown commands")
	}
	if !strings.Contains(result.Error.Error(), "unknown command") {
		t.Errorf("expected unknown command error, got: %v", result.Error)
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestValidator_UnknownCommandCheck_Disabled(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:         h.scenarioDir,
			ScenarioName:        h.scenarioName,
			CheckUnknownCommand: false, // Disabled
			CheckNoArgs:         false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor), // Returns success for everything - but check is disabled
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success with check disabled, got error: %v", result.Error)
	}
}

func TestValidator_NoArgsCheck_Passes(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:     h.scenarioDir,
			ScenarioName:    h.scenarioName,
			CheckNoArgs:     true,
			NoArgsTimeoutMs: 1000,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidator_NoArgsCheck_AcceptsNonZeroExit(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:     h.scenarioDir,
			ScenarioName:    h.scenarioName,
			CheckNoArgs:     true,
			NoArgsTimeoutMs: 1000,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) == 0 {
				return errors.New("no command specified") // Non-zero exit for no args
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	// Should still pass - the important thing is it doesn't hang
	if !result.Success {
		t.Fatalf("expected success (non-zero exit is acceptable), got error: %v", result.Error)
	}
}

func TestValidator_NoArgsCheck_Timeout(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:     h.scenarioDir,
			ScenarioName:    h.scenarioName,
			CheckNoArgs:     true,
			NoArgsTimeoutMs: 100, // Very short timeout
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) == 0 {
				// Simulate hanging
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return nil
				}
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if result.Success {
		t.Fatal("expected failure when CLI hangs")
	}
	if !strings.Contains(result.Error.Error(), "hung") || !strings.Contains(result.Error.Error(), "timeout") {
		t.Errorf("expected timeout/hung error, got: %v", result.Error)
	}
}

func TestValidator_NoArgsCheck_Disabled(t *testing.T) {
	h := newTestHarness(t)

	noArgsCalled := false
	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			CheckNoArgs:  false, // Disabled
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) == 0 {
				noArgsCalled = true
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if noArgsCalled {
		t.Error("no-args check should not have been called when disabled")
	}
}

func TestValidator_CustomHelpArgs(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			HelpArgs:     []string{"--help"}, // Only --help, not "help" subcommand
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "--help" {
				return nil
			}
			if len(args) > 0 && args[0] == "help" {
				return errors.New("unknown command: help")
			}
			return nil
		}),
		WithCapture(versionCapture("1.0.0")),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success with custom --help, got error: %v", result.Error)
	}
}

func TestValidator_CustomVersionArgs(t *testing.T) {
	h := newTestHarness(t)

	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
			VersionArgs:  []string{"-V"}, // Custom version arg
			CheckNoArgs:  false,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "-V" {
				return "2.0.0", nil
			}
			return "", errors.New("unknown version flag")
		}),
	)

	result := v.Validate(context.Background())

	if !result.Success {
		t.Fatalf("expected success with custom -V, got error: %v", result.Error)
	}
	if result.VersionOutput != "2.0.0" {
		t.Errorf("expected version 2.0.0, got: %s", result.VersionOutput)
	}
}

func TestValidator_DefaultsApplied(t *testing.T) {
	h := newTestHarness(t)

	// Create validator with minimal config - defaults should be applied
	v := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(successExecutor),
		WithCapture(versionCapture("1.0.0")),
	)

	// Access internal config through the validator
	val := v.(*validator)

	if len(val.config.HelpArgs) == 0 {
		t.Error("expected default HelpArgs to be set")
	}
	if len(val.config.VersionArgs) == 0 {
		t.Error("expected default VersionArgs to be set")
	}
	if val.config.NoArgsTimeoutMs == 0 {
		t.Error("expected default NoArgsTimeoutMs to be set")
	}
}

// Test helpers

type testHarness struct {
	scenarioDir  string
	scenarioName string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	tmpDir := t.TempDir()
	scenarioName := "demo"
	scenarioDir := filepath.Join(tmpDir, "scenarios", scenarioName)

	// Create cli directory
	cliDir := filepath.Join(scenarioDir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create executable
	writeExecutable(t, filepath.Join(cliDir, scenarioName), "#!/bin/bash\necho cli")

	return &testHarness{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
	}
}

func (h *testHarness) successExecutor(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return nil
}

func (h *testHarness) versionCapture(output string) CommandCapture {
	return func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
		return output, nil
	}
}

func successExecutor(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return nil
}

func versionCapture(output string) CommandCapture {
	return func(ctx context.Context, dir string, w io.Writer, name string, args ...string) (string, error) {
		return output, nil
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	perm := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		perm = 0o644
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("failed to write executable: %v", err)
	}
}

// Ensure validator implements interface at compile time
var _ Validator = (*validator)(nil)
