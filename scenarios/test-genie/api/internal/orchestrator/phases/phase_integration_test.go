package phases

import (
	"context"
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
			case strings.HasSuffix(name, filepath.Join("cli", "test-genie")):
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
			case strings.Contains(cmd, "test-genie help"):
				foundHelp = true
			case strings.Contains(cmd, "bats --tap test-genie.bats"):
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
