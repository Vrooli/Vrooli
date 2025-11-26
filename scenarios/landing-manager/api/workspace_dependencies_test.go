package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixWorkspaceDependencies_NoPackageJSON tests behavior when package.json doesn't exist
func TestFixWorkspaceDependencies_NoPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-scenario")

	// Create output directory but no ui/package.json
	if err := os.MkdirAll(filepath.Join(outputDir, "ui"), 0o755); err != nil {
		t.Fatalf("Failed to create ui directory: %v", err)
	}

	// Should not error if package.json doesn't exist (API-only scenario)
	err := fixWorkspaceDependencies(outputDir)
	if err != nil {
		t.Errorf("Expected no error for missing package.json, got: %v", err)
	}
}

// TestFixWorkspaceDependencies_ValidPackageJSON tests successful dependency rewriting
func TestFixWorkspaceDependencies_ValidPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-scenario")
	uiDir := filepath.Join(outputDir, "ui")

	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("Failed to create ui directory: %v", err)
	}

	// Create package.json with workspace dependencies
	packageJSON := map[string]interface{}{
		"name": "test-scenario",
		"dependencies": map[string]interface{}{
			"@vrooli/api-base":     "file:../../../packages/api-base",
			"@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge",
			"react":                "^18.0.0",
		},
	}

	packageData, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	packagePath := filepath.Join(uiDir, "package.json")
	if err := os.WriteFile(packagePath, packageData, 0o644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Run fixWorkspaceDependencies
	err = fixWorkspaceDependencies(outputDir)
	if err != nil {
		t.Fatalf("fixWorkspaceDependencies() returned error: %v", err)
	}

	// Verify package.json was updated
	updatedData, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("Failed to read updated package.json: %v", err)
	}

	var updatedPkg map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedPkg); err != nil {
		t.Fatalf("Failed to unmarshal updated package.json: %v", err)
	}

	deps, ok := updatedPkg["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("Dependencies section missing or wrong type")
	}

	// Verify workspace dependencies were converted to absolute paths
	apiBase, ok := deps["@vrooli/api-base"].(string)
	if !ok {
		t.Fatalf("Expected string for @vrooli/api-base, got: %T", deps["@vrooli/api-base"])
	}
	// Should start with "file:" and not be the original relative path
	if !strings.HasPrefix(apiBase, "file:") || apiBase == "file:../../../packages/api-base" {
		t.Errorf("Expected absolute file: path for @vrooli/api-base, got: %v", apiBase)
	}
	// The path after "file:" should contain "packages/api-base"
	if !strings.Contains(apiBase, "packages/api-base") {
		t.Errorf("Expected path to contain packages/api-base, got: %v", apiBase)
	}

	// Verify non-workspace dependencies were unchanged
	react, ok := deps["react"].(string)
	if !ok || react != "^18.0.0" {
		t.Errorf("Expected react dependency unchanged, got: %v", deps["react"])
	}
}

// TestFixWorkspaceDependencies_InvalidJSON tests handling of corrupted package.json
func TestFixWorkspaceDependencies_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-scenario")
	uiDir := filepath.Join(outputDir, "ui")

	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("Failed to create ui directory: %v", err)
	}

	// Write invalid JSON
	packagePath := filepath.Join(uiDir, "package.json")
	if err := os.WriteFile(packagePath, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid package.json: %v", err)
	}

	// Should return error for invalid JSON
	err := fixWorkspaceDependencies(outputDir)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestFixWorkspaceDependencies_NoDependencies tests package.json without dependencies
func TestFixWorkspaceDependencies_NoDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-scenario")
	uiDir := filepath.Join(outputDir, "ui")

	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("Failed to create ui directory: %v", err)
	}

	// Create package.json without dependencies
	packageJSON := map[string]interface{}{
		"name": "test-scenario",
	}

	packageData, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	packagePath := filepath.Join(uiDir, "package.json")
	if err := os.WriteFile(packagePath, packageData, 0o644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Should not error if dependencies section is missing
	err = fixWorkspaceDependencies(outputDir)
	if err != nil {
		t.Errorf("Expected no error for missing dependencies, got: %v", err)
	}
}

// TestFixWorkspaceDependencies_MixedDependencyTypes tests various dependency formats
func TestFixWorkspaceDependencies_MixedDependencyTypes(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-scenario")
	uiDir := filepath.Join(outputDir, "ui")

	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatalf("Failed to create ui directory: %v", err)
	}

	// Create package.json with various dependency types
	packageJSON := map[string]interface{}{
		"name": "test-scenario",
		"dependencies": map[string]interface{}{
			"@vrooli/api-base":     "file:../../../packages/api-base",     // workspace - should be fixed
			"@vrooli/iframe-bridge": "file:../../../packages/iframe-bridge", // workspace - should be fixed
			"react":                "^18.0.0",                             // npm - unchanged
			"lodash":               "4.17.21",                             // npm - unchanged
			"custom-lib":           "file:../../custom-lib",               // other file: - unchanged
		},
	}

	packageData, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	packagePath := filepath.Join(uiDir, "package.json")
	if err := os.WriteFile(packagePath, packageData, 0o644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Run fixWorkspaceDependencies
	err = fixWorkspaceDependencies(outputDir)
	if err != nil {
		t.Fatalf("fixWorkspaceDependencies() returned error: %v", err)
	}

	// Verify package.json was updated correctly
	updatedData, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("Failed to read updated package.json: %v", err)
	}

	var updatedPkg map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedPkg); err != nil {
		t.Fatalf("Failed to unmarshal updated package.json: %v", err)
	}

	deps, ok := updatedPkg["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("Dependencies section missing or wrong type")
	}

	// Verify workspace dependencies were fixed
	apiBase, ok := deps["@vrooli/api-base"].(string)
	if !ok {
		t.Fatalf("Expected string for @vrooli/api-base, got: %T", deps["@vrooli/api-base"])
	}
	// Should start with "file:" and not be the original relative path
	if !strings.HasPrefix(apiBase, "file:") || apiBase == "file:../../../packages/api-base" {
		t.Errorf("Expected absolute file: path for @vrooli/api-base, got: %v", apiBase)
	}
	// The path after "file:" should contain "packages/api-base"
	if !strings.Contains(apiBase, "packages/api-base") {
		t.Errorf("Expected path to contain packages/api-base, got: %v", apiBase)
	}

	// Verify non-workspace file: dependencies were unchanged
	customLib, ok := deps["custom-lib"].(string)
	if !ok || customLib != "file:../../custom-lib" {
		t.Errorf("Expected custom-lib unchanged, got: %v", customLib)
	}

	// Verify npm dependencies unchanged
	if deps["react"] != "^18.0.0" {
		t.Errorf("Expected react unchanged")
	}
}
