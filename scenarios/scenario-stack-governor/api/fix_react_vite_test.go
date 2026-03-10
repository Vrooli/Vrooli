package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupReactViteTestDir(t *testing.T, scenarioName string) (repoRoot string, serviceJSONPath string) {
	t.Helper()
	root := setupFixTestDir(t)

	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if err := os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}

	return root, filepath.Join(scenarioDir, ".vrooli", "service.json")
}

func writeServiceJSON(t *testing.T, path string, doc map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFixReactVite_PatchesExistingStep(t *testing.T) {
	scenarioName := "test-react-vite"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSON(t, sjPath, map[string]any{
		"lifecycle": map[string]any{
			"setup": map[string]any{
				"steps": []any{
					map[string]any{
						"name": "install-ui",
						"run":  "cd ui && pnpm install",
					},
				},
			},
		},
	})

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}
	if results[0].Diff != nil {
		t.Error("expected Diff to be nil on non-dry-run")
	}

	content, _ := os.ReadFile(sjPath)
	if !strings.Contains(string(content), "--ignore-workspace") {
		t.Error("expected --ignore-workspace in patched service.json")
	}
}

func TestFixReactVite_AddsNewStep(t *testing.T) {
	scenarioName := "test-add-step"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSON(t, sjPath, map[string]any{
		"lifecycle": map[string]any{
			"setup": map[string]any{
				"steps": []any{
					map[string]any{
						"name": "build-api",
						"run":  "cd api && go build ./...",
					},
				},
			},
		},
	})

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)
	if !strings.Contains(text, "install-ui-deps") {
		t.Error("expected install-ui-deps step to be added")
	}
	if !strings.Contains(text, "pnpm install --ignore-workspace") {
		t.Error("expected pnpm install --ignore-workspace in new step")
	}
}

func TestFixReactVite_Idempotent(t *testing.T) {
	scenarioName := "test-idempotent-vite"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSON(t, sjPath, map[string]any{
		"lifecycle": map[string]any{
			"setup": map[string]any{
				"steps": []any{
					map[string]any{
						"name": "install-ui",
						"run":  "cd ui && pnpm install --ignore-workspace",
					},
				},
			},
		},
	})

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false when already correct")
	}
}

func TestFixReactVite_DryRunDoesNotWrite(t *testing.T) {
	scenarioName := "test-dryrun-vite"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	doc := map[string]any{
		"lifecycle": map[string]any{
			"setup": map[string]any{
				"steps": []any{
					map[string]any{
						"name": "install-ui",
						"run":  "cd ui && pnpm install",
					},
				},
			},
		},
	}
	writeServiceJSON(t, sjPath, doc)

	original, _ := os.ReadFile(sjPath)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, true)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Error("expected fixed=true in dry-run mode")
	}

	// File should be unchanged.
	current, _ := os.ReadFile(sjPath)
	if string(current) != string(original) {
		t.Error("expected service.json to be unchanged in dry-run mode")
	}

	// Diff should be populated.
	if results[0].Diff == nil {
		t.Fatal("expected Diff to be populated in dry-run")
	}
	if results[0].Diff.Before != string(original) {
		t.Error("expected Diff.Before to equal original service.json content")
	}
	if !strings.Contains(results[0].Diff.After, "--ignore-workspace") {
		t.Error("expected Diff.After to contain '--ignore-workspace'")
	}
}
