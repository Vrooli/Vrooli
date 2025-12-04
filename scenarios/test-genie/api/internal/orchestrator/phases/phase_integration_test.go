package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunIntegrationPhaseExecutesCliAndBats(t *testing.T) {
	t.Run("[REQ:TESTGENIE-ORCH-P0] integration phase exercises CLI + bats", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")
		cliTestDir := filepath.Join(scenarioDir, "cli", "test")
		if err := os.MkdirAll(cliTestDir, 0o755); err != nil {
			t.Fatalf("failed to create cli/test dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(cliTestDir, "test-genie-generate.bats"), []byte("#!/usr/bin/env bats\n"), 0o644); err != nil {
			t.Fatalf("failed to seed bats file: %v", err)
		}

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		var executed []string
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			executed = append(executed, fmt.Sprintf("%s %s", filepath.Base(name), strings.Join(args, " ")))
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			switch {
			case strings.Contains(name, filepath.Join("cli", "demo")):
				return "demo version 1.0.0", nil
			case strings.Contains(name, filepath.Join("cli", "test-genie")):
				return "test-genie version 1.0.0", nil
			default:
				return "", nil
			}
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}
		report := runIntegrationPhase(context.Background(), env, io.Discard)
		if report.Err != nil {
			t.Fatalf("integration phase failed: %v", report.Err)
		}

		if len(executed) < 3 {
			t.Fatalf("expected cli help and bats suites to run, got %v", executed)
		}
		foundHelp := false
		foundPrimary := false
		foundAdditional := false
		for _, cmd := range executed {
			switch {
			case strings.Contains(cmd, "demo help") || strings.Contains(cmd, "test-genie help"):
				foundHelp = true
			case strings.Contains(cmd, "bats --tap demo.bats") || strings.Contains(cmd, "bats --tap test-genie.bats"):
				foundPrimary = true
			case strings.Contains(cmd, "bats --tap test/test-genie-generate.bats"):
				foundAdditional = true
			}
		}
		if !foundHelp || !foundPrimary || !foundAdditional {
			t.Fatalf("expected cli help + bats invocations, got %v", executed)
		}
	})
}

func TestRunIntegrationPhaseContextCancelled(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P1] cancelled context returns system failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(ctx, env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error for cancelled context")
		}
		if report.FailureClassification != FailureClassSystem {
			t.Errorf("expected system failure class, got %s", report.FailureClassification)
		}
	})
}

func TestRunIntegrationPhaseCLIHelpFails(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P2] CLI help command failure is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if len(args) > 0 && args[0] == "help" {
				return errors.New("CLI help command failed")
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "demo version 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error when CLI help fails")
		}
		if report.FailureClassification != FailureClassSystem {
			t.Errorf("expected system failure class, got %s", report.FailureClassification)
		}
	})
}

func TestRunIntegrationPhaseCLIVersionFails(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P3] CLI version command failure is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil // help passes
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "version" {
				return "", errors.New("CLI version command failed")
			}
			return "", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error when CLI version fails")
		}
	})
}

func TestRunIntegrationPhaseBatsFails(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P4] BATS suite failure is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			// Fail on bats execution
			if strings.Contains(name, "bats") || (len(args) > 0 && strings.Contains(args[0], ".bats")) {
				return errors.New("bats test suite failed")
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "demo version 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error when BATS suite fails")
		}
	})
}

func TestRunIntegrationPhaseBatsMissing(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P5] missing bats command is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			if name == "bats" {
				return "", errors.New("bats not found")
			}
			return "/tmp/" + name, nil
		})

		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "demo version 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error when bats is not installed")
		}
	})
}

func TestRunIntegrationPhaseNoCLIDirectory(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P6] missing CLI directory is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		// Remove the CLI directory
		if err := os.RemoveAll(filepath.Join(scenarioDir, "cli")); err != nil {
			t.Fatalf("failed to remove cli dir: %v", err)
		}

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error when CLI directory is missing")
		}
	})
}

func TestRunIntegrationPhaseObservationsRecorded(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P7] observations are properly recorded", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "demo version 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err != nil {
			t.Fatalf("expected success, got error: %v", report.Err)
		}
		if len(report.Observations) == 0 {
			t.Fatal("expected observations to be recorded")
		}

		// Check for key observations - just verify some observations were recorded
		hasSuccess := false
		for _, obs := range report.Observations {
			if strings.Contains(obs.Text, "completed") || strings.Contains(obs.Text, "validated") || strings.Contains(obs.Text, "passed") {
				hasSuccess = true
			}
		}
		if !hasSuccess {
			t.Errorf("expected success observation, got observations: %+v", report.Observations)
		}
	})
}

func TestRunIntegrationPhaseCLIVersionMalformed(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P8] malformed version output is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			// Return output without "version" keyword
			return "demo 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error for malformed version output")
		}
		if report.FailureClassification != FailureClassMisconfiguration {
			t.Errorf("expected misconfiguration failure class, got %s", report.FailureClassification)
		}
	})
}

func TestRunIntegrationPhaseWithAdditionalBatsSuites(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P9] additional bats suites are executed", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		// Create additional bats files in cli/test
		cliTestDir := filepath.Join(scenarioDir, "cli", "test")
		if err := os.MkdirAll(cliTestDir, 0o755); err != nil {
			t.Fatalf("failed to create cli/test dir: %v", err)
		}
		for _, name := range []string{"extra1.bats", "extra2.bats", "extra3.bats"} {
			if err := os.WriteFile(filepath.Join(cliTestDir, name), []byte("#!/usr/bin/env bats\n"), 0o644); err != nil {
				t.Fatalf("failed to create %s: %v", name, err)
			}
		}

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})

		var batsRuns []string
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			if strings.HasSuffix(name, "bats") || (len(args) > 1 && strings.HasSuffix(args[1], ".bats")) {
				batsRuns = append(batsRuns, strings.Join(args, " "))
			}
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "demo version 1.0.0", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err != nil {
			t.Fatalf("expected success, got error: %v", report.Err)
		}

		// Should have primary suite + 3 additional
		if len(batsRuns) < 4 {
			t.Errorf("expected at least 4 bats runs (1 primary + 3 additional), got %d: %v", len(batsRuns), batsRuns)
		}
	})
}

func TestRunIntegrationPhaseNoExecutableCLI(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-P10] non-executable CLI binary is reported", func(t *testing.T) {
		root := t.TempDir()
		scenarioDir := createScenarioLayout(t, root, "demo")

		// Make CLI binary non-executable
		cliBinary := filepath.Join(scenarioDir, "cli", "demo")
		if err := os.Chmod(cliBinary, 0o644); err != nil {
			t.Fatalf("failed to chmod cli binary: %v", err)
		}

		stubCommandLookup(t, func(name string) (string, error) {
			return "/tmp/" + name, nil
		})
		stubPhaseCommandExecutor(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
			return nil
		})
		stubPhaseCommandCapture(t, func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
			return "", nil
		})

		env := workspace.Environment{
			ScenarioName: "demo",
			ScenarioDir:  scenarioDir,
			TestDir:      filepath.Join(scenarioDir, "test"),
		}

		report := runIntegrationPhase(context.Background(), env, io.Discard)

		if report.Err == nil {
			t.Fatal("expected error for non-executable CLI binary")
		}
	})
}
