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

// writeServiceJSONRaw writes raw JSON bytes to the service.json path.
func writeServiceJSONRaw(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFixReactVite_ReplacesNpmInstall(t *testing.T) {
	scenarioName := "test-npm-replace"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{
  "version": "1.0.0",
  "service": {"name": "test"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "cd ui && npm install"}
      ]
    }
  }
}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)
	if !strings.Contains(text, "pnpm install --ignore-workspace") {
		t.Error("expected pnpm install --ignore-workspace in fixed service.json")
	}
	// Check there's no standalone "npm install" (pnpm install contains "npm install" as substring).
	if strings.Contains(text, "npm install") && !strings.Contains(text, "pnpm install") {
		t.Error("npm install should have been replaced, but still present")
	}
}

func TestFixReactVite_ReplacesNpmInstallWithFlags(t *testing.T) {
	scenarioName := "test-npm-flags"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "cd ui && npm install --omit=dev"}
      ]
    }
  }
}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)
	if !strings.Contains(text, "pnpm install --ignore-workspace") {
		t.Error("expected pnpm install --ignore-workspace")
	}
	// The entire run field should be replaced, not just npm->pnpm.
	if strings.Contains(text, "--omit=dev") {
		t.Error("old npm flags should not remain after full replacement")
	}
}

func TestFixReactVite_ReplacesNpmInstallConditional(t *testing.T) {
	scenarioName := "test-npm-conditional"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "if [ -f ui/package.json ]; then cd ui && npm install; else echo 'skip'; fi"}
      ]
    }
  }
}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)
	if !strings.Contains(text, "pnpm install --ignore-workspace") {
		t.Error("expected pnpm install --ignore-workspace")
	}
	if strings.Contains(text, "npm install") && !strings.Contains(text, "pnpm install") {
		t.Error("npm install should have been fully replaced")
	}
}

func TestFixReactVite_PreservesKeyOrder(t *testing.T) {
	scenarioName := "test-key-order"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	// Write JSON with specific key order: version, service, lifecycle.
	rawJSON := `{
  "version": "1.0.0",
  "service": {"name": "test-app"},
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "cd ui && npm install"}
      ]
    }
  },
  "display": {"title": "Test"}
}`
	writeServiceJSONRaw(t, sjPath, rawJSON)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)

	// Verify the top-level key order is preserved: version before service before lifecycle before display.
	versionIdx := strings.Index(text, `"version"`)
	serviceIdx := strings.Index(text, `"service"`)
	lifecycleIdx := strings.Index(text, `"lifecycle"`)
	displayIdx := strings.Index(text, `"display"`)

	if versionIdx < 0 || serviceIdx < 0 || lifecycleIdx < 0 || displayIdx < 0 {
		t.Fatalf("expected all keys present; got:\n%s", text)
	}
	if versionIdx >= serviceIdx {
		t.Error("expected 'version' before 'service' in output")
	}
	if serviceIdx >= lifecycleIdx {
		t.Error("expected 'service' before 'lifecycle' in output")
	}
	if lifecycleIdx >= displayIdx {
		t.Error("expected 'lifecycle' before 'display' in output")
	}
}

func TestFixReactVite_NoDuplicateSteps(t *testing.T) {
	scenarioName := "test-no-dup"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build-api", "run": "cd api && go build ./..."},
        {"name": "install-ui-deps", "run": "cd ui && npm install"}
      ]
    }
  }
}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)

	// Count occurrences of install-ui-deps — should be exactly 1.
	count := strings.Count(text, "install-ui-deps")
	if count != 1 {
		t.Errorf("expected exactly 1 install-ui-deps step, found %d\n%s", count, text)
	}

	// Should not have a second pnpm install step appended.
	pnpmCount := strings.Count(text, "pnpm install")
	if pnpmCount != 1 {
		t.Errorf("expected exactly 1 pnpm install, found %d\n%s", pnpmCount, text)
	}
}

func TestFixReactVite_DryRunNpmInstall(t *testing.T) {
	scenarioName := "test-dryrun-npm"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	rawJSON := `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "cd ui && npm install"}
      ]
    }
  }
}`
	writeServiceJSONRaw(t, sjPath, rawJSON)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, true)

	if !results[0].Fixed {
		t.Fatal("expected fixed=true in dry-run")
	}

	// File should remain unchanged.
	current, _ := os.ReadFile(sjPath)
	if string(current) != rawJSON {
		t.Error("expected file unchanged in dry-run")
	}

	// Diff should show the replacement.
	if results[0].Diff == nil {
		t.Fatal("expected Diff populated in dry-run")
	}
	if !strings.Contains(results[0].Diff.After, "pnpm install --ignore-workspace") {
		t.Error("expected Diff.After to contain pnpm install --ignore-workspace")
	}
	// Ensure there's no standalone npm install (pnpm install contains the substring).
	if strings.Contains(results[0].Diff.After, "npm install") && !strings.Contains(results[0].Diff.After, "pnpm install") {
		t.Error("expected Diff.After to not contain standalone npm install")
	}
}

func TestHasUIInstallIgnoreWorkspace_NpmInstallShowsInEvidence(t *testing.T) {
	_, sjPath := setupReactViteTestDir(t, "test-npm-evidence")

	writeServiceJSONRaw(t, sjPath, `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "cd ui && npm install"}
      ]
    }
  }
}`)

	ok, evidence := hasUIInstallIgnoreWorkspace(sjPath)
	if ok {
		t.Error("expected hasUIInstallIgnoreWorkspace to return false for npm install")
	}
	if evidence == "" {
		t.Error("expected non-empty evidence string for npm install step")
	}
	if !strings.Contains(evidence, "npm install") {
		t.Errorf("expected evidence to contain 'npm install', got: %q", evidence)
	}
}

func TestHasUIInstallIgnoreWorkspace_NpmInstallByName(t *testing.T) {
	// Step name contains "ui" but run doesn't — should still be detected.
	_, sjPath := setupReactViteTestDir(t, "test-npm-name")

	writeServiceJSONRaw(t, sjPath, `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-ui-deps", "run": "npm install"}
      ]
    }
  }
}`)

	ok, evidence := hasUIInstallIgnoreWorkspace(sjPath)
	if ok {
		t.Error("expected false for npm install without --ignore-workspace")
	}
	if evidence == "" {
		t.Error("expected non-empty evidence for npm install step with ui in name")
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
