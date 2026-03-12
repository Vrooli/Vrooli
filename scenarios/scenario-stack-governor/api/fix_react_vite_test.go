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

	// The fix requires ui/package.json to exist (matching the rule check behavior).
	if err := os.WriteFile(filepath.Join(scenarioDir, "ui", "package.json"), []byte(`{"name":"test"}`), 0o644); err != nil {
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

// TestCheckUIInstall_MalformedLifecycle verifies that a service.json with a
// malformed lifecycle structure (e.g., null or wrong type) returns a parseErr
// rather than silently passing.
func TestCheckUIInstall_MalformedLifecycle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "lifecycle_null",
			content: `{"lifecycle": null}`,
			wantErr: "lifecycle field is not an object",
		},
		{
			name:    "lifecycle_string",
			content: `{"lifecycle": "broken"}`,
			wantErr: "lifecycle field is not an object",
		},
		{
			name:    "lifecycle_missing",
			content: `{"version": "1.0.0"}`,
			wantErr: "lifecycle field missing",
		},
		{
			name:    "setup_null",
			content: `{"lifecycle": {"setup": null}}`,
			wantErr: "lifecycle.setup field is not an object",
		},
		{
			name:    "steps_string",
			content: `{"lifecycle": {"setup": {"steps": "not-array"}}}`,
			wantErr: "lifecycle.setup.steps field is not an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sjPath := setupReactViteTestDir(t, "malformed-"+tt.name)
			writeServiceJSONRaw(t, sjPath, tt.content)

			result := checkUIInstall(sjPath)
			if result.parseErr == "" {
				t.Fatal("expected parseErr to be set")
			}
			if !strings.Contains(result.parseErr, tt.wantErr) {
				t.Errorf("expected parseErr containing %q, got %q", tt.wantErr, result.parseErr)
			}
			if result.ok {
				t.Error("expected ok=false for malformed structure")
			}
		})
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

func TestFixReactVite_SkipsNonUIScenario(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "no-ui-scenario"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	// Create scenario with service.json but NO ui/package.json.
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeServiceJSON(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), map[string]any{
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
	if results[0].Fixed {
		t.Error("expected fixed=false for scenario without ui/package.json")
	}
	if results[0].Error != "" {
		t.Errorf("expected no error, got: %s", results[0].Error)
	}

	// Verify service.json was NOT modified (no UI install step added).
	content, _ := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if strings.Contains(string(content), "pnpm install") {
		t.Error("service.json should not have been modified for non-UI scenario")
	}
}

func TestFixReactVite_SkipsNoUIDirectory(t *testing.T) {
	root := setupFixTestDir(t)
	scenarioName := "no-ui-dir"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	// Create scenario with no ui/ directory at all.
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false for scenario without ui/ directory")
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

// TestFixReactVite_InsertedStepMatchesExistingIndentation verifies that when a
// new step is added (Phase 3), the inserted JSON matches the indentation of
// existing steps rather than using hardcoded whitespace.
func TestFixReactVite_InsertedStepMatchesExistingIndentation(t *testing.T) {
	scenarioName := "test-indent"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	// Write with 8-space indentation for step objects (realistic service.json).
	rawJSON := `{
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "build-api",
          "run": "cd api && go build ./...",
          "description": "Build Go API binary"
        },
        {
          "name": "show-urls",
          "run": "echo 'Ready'",
          "description": "Display info"
        }
      ]
    }
  }
}`
	writeServiceJSONRaw(t, sjPath, rawJSON)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	text := string(content)

	// The new step should be valid JSON.
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("inserted step produced invalid JSON: %v\nContent:\n%s", err, text)
	}

	// The new step's opening brace should have the same indentation as existing steps.
	// Existing steps start with 8 spaces of indent.
	if !strings.Contains(text, "install-ui-deps") {
		t.Fatal("expected install-ui-deps step to be added")
	}

	// Find the line with "install-ui-deps" and check its indentation matches
	// existing step names.
	lines := strings.Split(text, "\n")
	var existingStepIndent, newStepIndent string
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.Contains(trimmed, `"name": "build-api"`) {
			existingStepIndent = line[:len(line)-len(trimmed)]
		}
		if strings.Contains(trimmed, `"name": "install-ui-deps"`) {
			newStepIndent = line[:len(line)-len(trimmed)]
		}
	}
	if existingStepIndent == "" {
		t.Fatal("could not find existing step indentation")
	}
	if newStepIndent == "" {
		t.Fatal("could not find new step indentation")
	}
	if existingStepIndent != newStepIndent {
		t.Errorf("indentation mismatch: existing step uses %q, new step uses %q",
			existingStepIndent, newStepIndent)
	}
}

// TestFixReactVite_InsertedStepValidJSON verifies that inserting a step into
// various JSON structures always produces valid JSON.
func TestFixReactVite_InsertedStepValidJSON(t *testing.T) {
	tests := []struct {
		name    string
		rawJSON string
	}{
		{
			name:    "compact_steps",
			rawJSON: `{"lifecycle":{"setup":{"steps":[{"name":"build","run":"go build"}]}}}`,
		},
		{
			name: "2_space_indent",
			rawJSON: `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "build", "run": "go build"}
      ]
    }
  }
}`,
		},
		{
			name: "4_space_indent",
			rawJSON: `{
    "lifecycle": {
        "setup": {
            "steps": [
                {"name": "build", "run": "go build"}
            ]
        }
    }
}`,
		},
		{
			name: "many_existing_steps",
			rawJSON: `{
  "lifecycle": {
    "setup": {
      "steps": [
        {"name": "install-cli", "run": "cd cli && ./install.sh"},
        {"name": "build-api", "run": "cd api && go build ."},
        {"name": "setup-db", "run": "scripts/setup-db.sh"},
        {"name": "seed-data", "run": "scripts/seed.sh"},
        {"name": "show-urls", "run": "echo ready"}
      ]
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioName := "test-valid-json-" + tt.name
			root, sjPath := setupReactViteTestDir(t, scenarioName)
			writeServiceJSONRaw(t, sjPath, tt.rawJSON)

			results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
			if !results[0].Fixed {
				t.Fatalf("expected fixed=true; error=%s", results[0].Error)
			}

			content, _ := os.ReadFile(sjPath)
			var doc map[string]any
			if err := json.Unmarshal(content, &doc); err != nil {
				t.Fatalf("produced invalid JSON: %v\nContent:\n%s", err, string(content))
			}

			// Verify the step was actually added.
			lifecycle, _ := doc["lifecycle"].(map[string]any)
			setup, _ := lifecycle["setup"].(map[string]any)
			steps, _ := setup["steps"].([]any)
			found := false
			for _, s := range steps {
				step, _ := s.(map[string]any)
				if step["name"] == "install-ui-deps" {
					found = true
					if step["run"] != "cd ui && pnpm install --ignore-workspace" {
						t.Errorf("unexpected run value: %v", step["run"])
					}
				}
			}
			if !found {
				t.Error("install-ui-deps step not found in parsed JSON")
			}
		})
	}
}

// TestFixReactVite_NullLifecycleRecovery verifies the fixer handles a
// service.json with lifecycle: null by constructing the correct structure
// via marshal fallback.
func TestFixReactVite_NullLifecycleRecovery(t *testing.T) {
	scenarioName := "test-null-lifecycle"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{"lifecycle": null}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// The fixer should reconstruct the lifecycle structure and add the step.
	if !results[0].Fixed {
		t.Fatalf("expected fixed=true; error=%s", results[0].Error)
	}

	content, _ := os.ReadFile(sjPath)
	if !strings.Contains(string(content), "pnpm install --ignore-workspace") {
		t.Error("expected pnpm install step in fixed service.json")
	}

	// Verify the output is valid JSON.
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Errorf("fixed service.json is invalid JSON: %v", err)
	}
}

// TestFixReactVite_InvalidJSON verifies the fixer returns an error for
// completely unparseable JSON.
func TestFixReactVite_InvalidJSON(t *testing.T) {
	scenarioName := "test-bad-json"
	root, sjPath := setupReactViteTestDir(t, scenarioName)

	writeServiceJSONRaw(t, sjPath, `{not valid json}`)

	results := FixReactViteUIInstallsDependencies(t.Context(), root, scenarioName, false)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fixed {
		t.Error("expected fixed=false for invalid JSON")
	}
	if results[0].Error == "" {
		t.Error("expected error message for invalid JSON")
	}
}

// TestDetectStepIndent verifies indentation detection for various formats.
func TestDetectStepIndent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "8_space_indent",
			text:     `  "steps": [` + "\n" + `        {"name": "build"}`,
			expected: "        ",
		},
		{
			name:     "4_space_indent",
			text:     `  "steps": [` + "\n" + `    {"name": "build"}`,
			expected: "    ",
		},
		{
			name:     "tab_indent",
			text:     `  "steps": [` + "\n" + "\t\t" + `{"name": "build"}`,
			expected: "\t\t",
		},
		{
			name:     "no_steps_key",
			text:     `{"lifecycle": {}}`,
			expected: "        ", // 8-space default fallback
		},
		{
			name:     "compact_json",
			text:     `{"lifecycle":{"setup":{"steps":[{"name":"build"}]}}}`,
			expected: "", // empty = compact JSON, signal to use marshal fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectStepIndent(tt.text)
			if got != tt.expected {
				t.Errorf("expected indent %q, got %q", tt.expected, got)
			}
		})
	}
}
