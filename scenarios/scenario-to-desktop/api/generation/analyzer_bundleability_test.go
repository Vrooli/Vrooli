package generation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCheckBundleability_NoResources tests that a scenario with no required resources is bundleable.
func TestCheckBundleability_NoResources(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with no required resources
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Bundleable {
		t.Errorf("expected Bundleable=true for scenario with no resources")
	}
	if len(result.RequiredResources) != 0 {
		t.Errorf("expected no required resources, got %v", result.RequiredResources)
	}
}

// TestCheckBundleability_UnbundleableResource_NoMetadata tests fallback behavior.
func TestCheckBundleability_UnbundleableResource_NoMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with postgres as required resource (no deployment metadata)
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"type":     "postgres",
					"enabled":  true,
					"required": true,
				},
			},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Bundleable {
		t.Errorf("expected Bundleable=false for scenario with postgres (no swap declared)")
	}
	if result.UnbundleableResource != "postgres" {
		t.Errorf("expected UnbundleableResource='postgres', got %q", result.UnbundleableResource)
	}
}

// TestCheckBundleability_WithDeploymentMetadata_NotSupported tests behavior when tier-2-desktop is not supported.
func TestCheckBundleability_WithDeploymentMetadata_NotSupported(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with deployment metadata showing tier-2-desktop not supported and no alternatives
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"type":     "postgres",
					"enabled":  true,
					"required": true,
				},
			},
		},
		"deployment": map[string]interface{}{
			"dependencies": map[string]interface{}{
				"resources": map[string]interface{}{
					"postgres": map[string]interface{}{
						"resource_type": "database",
						"platform_support": map[string]interface{}{
							"tier-2-desktop": map[string]interface{}{
								"supported":     false,
								"fitness_score": 0.2,
								"reason":        "Desktop install would need embedded database service",
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Bundleable {
		t.Errorf("expected Bundleable=false for scenario with postgres (not supported, no alternatives)")
	}
	if result.UnbundleableResource != "postgres" {
		t.Errorf("expected UnbundleableResource='postgres', got %q", result.UnbundleableResource)
	}
	if result.UnbundleableReason != "Desktop install would need embedded database service" {
		t.Errorf("expected reason from metadata, got %q", result.UnbundleableReason)
	}
}

// TestCheckBundleability_WithSwapAlternatives tests that swaps allow proceeding with a warning.
func TestCheckBundleability_WithSwapAlternatives(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with postgres not supported but with sqlite alternative
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"type":     "postgres",
					"enabled":  true,
					"required": true,
				},
			},
		},
		"deployment": map[string]interface{}{
			"dependencies": map[string]interface{}{
				"resources": map[string]interface{}{
					"postgres": map[string]interface{}{
						"resource_type": "database",
						"platform_support": map[string]interface{}{
							"tier-2-desktop": map[string]interface{}{
								"supported":     false,
								"fitness_score": 0.2,
								"reason":        "Desktop install would need embedded database service",
								"alternatives":  []string{"sqlite"},
							},
						},
						"swappable_with": []map[string]interface{}{
							{
								"id":           "sqlite",
								"relationship": "migration",
								"notes":        "Only viable for single-user bundles",
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Bundleable {
		t.Errorf("expected Bundleable=true for scenario with postgres (has sqlite swap)")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
	if len(result.Warnings) > 0 {
		warning := result.Warnings[0]
		if warning.Resource != "postgres" {
			t.Errorf("expected warning for 'postgres', got %q", warning.Resource)
		}
		if len(warning.Alternatives) == 0 || warning.Alternatives[0] != "sqlite" {
			t.Errorf("expected alternative 'sqlite', got %v", warning.Alternatives)
		}
	}
}

// TestCheckBundleability_BundleableResource tests a resource that is supported for tier-2-desktop.
func TestCheckBundleability_BundleableResource(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with a resource that is supported for tier-2-desktop
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"openrouter": map[string]interface{}{
					"type":     "openrouter",
					"enabled":  true,
					"required": true,
				},
			},
		},
		"deployment": map[string]interface{}{
			"dependencies": map[string]interface{}{
				"resources": map[string]interface{}{
					"openrouter": map[string]interface{}{
						"resource_type": "llm-api",
						"platform_support": map[string]interface{}{
							"tier-2-desktop": map[string]interface{}{
								"supported":     true,
								"fitness_score": 0.5,
								"reason":        "Requires user-provided API key and network",
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Bundleable {
		t.Errorf("expected Bundleable=true for scenario with openrouter (supported)")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(result.Warnings))
	}
}

// TestCheckBundleability_NonRequiredResource tests that non-required resources don't block bundling.
func TestCheckBundleability_NonRequiredResource(t *testing.T) {
	tmpDir := t.TempDir()
	scenarioName := "test-scenario"
	scenarioPath := filepath.Join(tmpDir, "scenarios", scenarioName)
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")

	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}

	// Create a service.json with postgres but not required
	serviceJSON := map[string]interface{}{
		"service": map[string]interface{}{
			"name": scenarioName,
		},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{
					"type":     "postgres",
					"enabled":  true,
					"required": false, // Not required
				},
			},
		},
	}

	data, _ := json.Marshal(serviceJSON)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.CheckBundleability(scenarioName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Bundleable {
		t.Errorf("expected Bundleable=true for scenario with non-required postgres")
	}
	if len(result.RequiredResources) != 0 {
		t.Errorf("expected no required resources, got %v", result.RequiredResources)
	}
}
