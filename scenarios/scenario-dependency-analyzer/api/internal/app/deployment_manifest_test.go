package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-dependency-analyzer/internal/deployment"

	types "scenario-dependency-analyzer/internal/types"
)

func TestBuildBundleManifestSkeletonValidatesAgainstSchema(t *testing.T) {
	scenarioDir := t.TempDir()

	writeBuildableUIFolder(t, scenarioDir)
	writeBuildableAPIFolder(t, scenarioDir, "test-scenario-api")

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.DisplayName = "Test Scenario"
	cfg.Service.Description = "Example scenario for bundle skeleton"
	cfg.Service.Version = "1.2.3"

	nodes := []types.DeploymentDependencyNode{
		{
			Name:         "postgres",
			Type:         "resource",
			Alternatives: []string{"sqlite"},
			Children:     nil,
		},
	}

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nodes, cfg)
	if manifest.Skeleton == nil {
		t.Fatalf("expected skeleton to be populated")
	}

	payload, err := json.Marshal(manifest.Skeleton)
	if err != nil {
		t.Fatalf("failed to marshal skeleton: %v", err)
	}
	if err := validateDesktopBundleManifestBytes(payload); err != nil {
		t.Fatalf("skeleton failed schema validation: %v", err)
	}

	if len(manifest.Skeleton.Swaps) == 0 {
		t.Fatalf("expected swaps to include at least one alternative")
	}

	var foundUI bool
	for _, svc := range manifest.Skeleton.Services {
		if svc.Type == "ui-bundle" {
			foundUI = true
			if svc.Health.Path != "/health" {
				t.Fatalf("ui skeleton should use /health for health checks")
			}
		}
		if svc.ID == "test-scenario-api" {
			if svc.Health.PortName != "api" {
				t.Fatalf("api skeleton should expose api port health check")
			}
			if svc.Readiness.Type != "health_success" {
				t.Fatalf("api skeleton should use health_success readiness")
			}
		}
	}
	if !foundUI {
		t.Fatalf("expected ui-bundle service to be included when ui/dist exists")
	}
}

func TestBuildBundleManifestWithoutUI(t *testing.T) {
	scenarioDir := t.TempDir()

	writeBuildableAPIFolder(t, scenarioDir, "test-scenario-api")

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.Version = "1.0.0"

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nil, cfg)

	if manifest.Skeleton == nil {
		t.Fatalf("expected skeleton to be populated")
	}

	// Verify no UI service
	for _, svc := range manifest.Skeleton.Services {
		if svc.Type == "ui-bundle" {
			t.Error("expected no ui-bundle service when ui/dist doesn't exist")
		}
	}

	// Should still have API service
	var foundAPI bool
	for _, svc := range manifest.Skeleton.Services {
		if svc.Type == "api-binary" {
			foundAPI = true
		}
	}
	if !foundAPI {
		t.Error("expected api-binary service to always be present")
	}
}

func TestBuildBundleManifestNilConfig(t *testing.T) {
	scenarioDir := t.TempDir()

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nil, nil)

	if manifest.Skeleton != nil {
		t.Error("expected nil skeleton when config is nil")
	}
}

func TestBuildBundleManifestDependencyFlattening(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.Version = "1.0.0"

	// Create nested dependency tree
	nodes := []types.DeploymentDependencyNode{
		{
			Name: "postgres",
			Type: "resource",
			Children: []types.DeploymentDependencyNode{
				{Name: "redis", Type: "resource"},
			},
		},
		{
			Name: "auth-service",
			Type: "scenario",
			Children: []types.DeploymentDependencyNode{
				{Name: "postgres", Type: "resource"}, // Duplicate
				{Name: "secrets-manager", Type: "scenario"},
			},
		},
	}

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nodes, cfg)

	// Check dependencies are flattened and deduplicated
	depNames := make(map[string]bool)
	for _, dep := range manifest.Dependencies {
		key := dep.Type + ":" + dep.Name
		if depNames[key] {
			t.Errorf("duplicate dependency found: %s", key)
		}
		depNames[key] = true
	}

	// Should have 4 unique dependencies
	expectedDeps := map[string]bool{
		"resource:postgres":        true,
		"resource:redis":           true,
		"scenario:auth-service":    true,
		"scenario:secrets-manager": true,
	}
	for key := range expectedDeps {
		if !depNames[key] {
			t.Errorf("expected dependency not found: %s", key)
		}
	}
}

func TestBuildBundleManifestSwapsGeneration(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.Version = "1.0.0"

	// Resources with alternatives
	nodes := []types.DeploymentDependencyNode{
		{
			Name:         "postgres",
			Type:         "resource",
			Alternatives: []string{"sqlite", "duckdb"},
		},
		{
			Name:         "redis",
			Type:         "resource",
			Alternatives: []string{"in-memory-cache"},
		},
		{
			Name:         "auth-service",
			Type:         "scenario", // Should not generate swap
			Alternatives: []string{"basic-auth"},
		},
	}

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nodes, cfg)

	if manifest.Skeleton == nil {
		t.Fatalf("expected skeleton to be populated")
	}

	// Check swaps are generated for resources only
	swapMap := make(map[string]string)
	for _, swap := range manifest.Skeleton.Swaps {
		swapMap[swap.Original] = swap.Replacement
	}

	// Postgres should have sqlite swap (first alternative)
	if swapMap["postgres"] != "sqlite" {
		t.Errorf("expected postgres->sqlite swap, got %v", swapMap["postgres"])
	}

	// Redis should have in-memory-cache swap
	if swapMap["redis"] != "in-memory-cache" {
		t.Errorf("expected redis->in-memory-cache swap, got %v", swapMap["redis"])
	}

	// auth-service should NOT have a swap (it's a scenario, not resource)
	if _, ok := swapMap["auth-service"]; ok {
		t.Error("scenarios should not generate swaps")
	}
}

func TestBuildBundleManifestFilesDiscovery(t *testing.T) {
	scenarioDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write service.json: %v", err)
	}

	writeBuildableAPIFolder(t, scenarioDir, "test-scenario-api")
	writeBuildableUIFolder(t, scenarioDir)

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.Version = "1.0.0"

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nil, cfg)

	// Check files are discovered
	fileExists := make(map[string]bool)
	for _, f := range manifest.Files {
		fileExists[f.Type] = f.Exists
	}

	if !fileExists["service-config"] {
		t.Error("expected service-config to be found")
	}
	if !fileExists["api-source"] {
		t.Error("expected api-source to be found")
	}
	if !fileExists["api-binary"] {
		t.Error("expected api-binary to be found")
	}
	if !fileExists["ui-bundle"] {
		t.Error("expected ui-bundle to be found")
	}
	if !fileExists["ui-entry"] {
		t.Error("expected ui-entry to be found")
	}
}

func TestComputeTierAggregates(t *testing.T) {
	supported := true
	nodes := []types.DeploymentDependencyNode{
		{
			Name: "postgres",
			Type: "resource",
			TierSupport: map[string]types.TierSupportSummary{
				"desktop": {Supported: nil, Reason: "needs sqlite"},
				"tier1":   {Supported: &supported, Reason: "docker available"},
			},
		},
		{
			Name: "redis",
			Type: "resource",
			TierSupport: map[string]types.TierSupportSummary{
				"desktop": {Supported: &supported, Reason: "embedded available"},
				"tier1":   {Supported: &supported, Reason: "docker available"},
			},
		},
	}

	aggregates := deployment.ComputeTierAggregates(nodes)

	// Desktop tier should have aggregates
	desktop := aggregates["desktop"]
	if desktop.DependencyCount != 2 {
		t.Errorf("desktop.DependencyCount = %d, want 2", desktop.DependencyCount)
	}

	// Tier1 should have aggregates
	tier1 := aggregates["tier1"]
	if tier1.DependencyCount != 2 {
		t.Errorf("tier1.DependencyCount = %d, want 2", tier1.DependencyCount)
	}
}

func TestDetectSecretRequirements(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "resource.json")
	if err := os.WriteFile(manifestPath, []byte(`{"credentials":{"descriptors":[{"logical_id":"custom/token","field":"token","label":"API token","description":"declared credential","required":true}]}}`), 0o644); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}
	nodes := []types.DeploymentDependencyNode{
		{Name: "custom-resource", Type: "resource", Path: manifestPath},
	}

	secrets := deployment.DetectSecretRequirements(nodes)

	var foundDeclaredSecret bool
	for _, secret := range secrets {
		if secret.DependencyName == "custom-resource" {
			foundDeclaredSecret = true
			if secret.SecretType == "" {
				t.Error("postgres secret should have a type")
			}
		}
	}
	if !foundDeclaredSecret {
		t.Error("expected declared credential requirement to be detected")
	}
}

func TestSuggestResourceSwaps(t *testing.T) {
	nodes := []types.DeploymentDependencyNode{
		{Name: "custom-resource", Type: "resource", Alternatives: []string{"declared-peer"}},
	}

	swaps := deployment.SuggestResourceSwaps(nodes)

	swapMap := make(map[string][]string)
	for _, swap := range swaps {
		swapMap[swap.OriginalResource] = append(swapMap[swap.OriginalResource], swap.AlternativeResource)
	}

	if len(swapMap["custom-resource"]) != 1 || swapMap["custom-resource"][0] != "declared-peer" {
		t.Errorf("expected declared alternative, got %v", swapMap)
	}
}

func TestBuildBundleManifestVersionDefaults(t *testing.T) {
	scenarioDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	// Config without version
	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.Version = "" // Empty version

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nil, cfg)

	if manifest.Skeleton == nil {
		t.Fatalf("expected skeleton to be populated")
	}

	// Should default to 0.1.0
	if manifest.Skeleton.App.Version != "0.1.0" {
		t.Errorf("expected default version 0.1.0, got %s", manifest.Skeleton.App.Version)
	}
}

func TestBuildBundleManifestAppNameFallbacks(t *testing.T) {
	tests := []struct {
		name         string
		displayName  string
		serviceName  string
		scenarioName string
		expectedName string
	}{
		{
			name:         "UsesDisplayName",
			displayName:  "My Display Name",
			serviceName:  "service-name",
			scenarioName: "scenario-name",
			expectedName: "My Display Name",
		},
		{
			name:         "FallsBackToServiceName",
			displayName:  "",
			serviceName:  "service-name",
			scenarioName: "scenario-name",
			expectedName: "service-name",
		},
		{
			name:         "FallsBackToScenarioName",
			displayName:  "",
			serviceName:  "",
			scenarioName: "scenario-name",
			expectedName: "scenario-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioDir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
				t.Fatalf("failed to create api dir: %v", err)
			}

			cfg := &types.ServiceConfig{}
			cfg.Service.DisplayName = tt.displayName
			cfg.Service.Name = tt.serviceName
			cfg.Service.Version = "1.0.0"

			manifest := deployment.BuildBundleManifest(tt.scenarioName, scenarioDir, time.Now(), nil, cfg)

			if manifest.Skeleton == nil {
				t.Fatalf("expected skeleton to be populated")
			}

			if manifest.Skeleton.App.Name != tt.expectedName {
				t.Errorf("expected app name %q, got %q", tt.expectedName, manifest.Skeleton.App.Name)
			}
		})
	}
}

// writeBuildableAPIFolder creates an api/ folder with Go markers so the folder
// scanner detects it as a Go API component.
func writeBuildableAPIFolder(t *testing.T, scenarioDir, binaryName string) {
	t.Helper()
	apiDir := filepath.Join(scenarioDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module test-scenario-api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("failed to write api go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write api main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, binaryName), []byte("#!/bin/bash\necho ok\n"), 0o755); err != nil {
		t.Fatalf("failed to write api binary placeholder: %v", err)
	}
}

// writeBuildableUIFolder creates a ui/ folder with package.json and a built
// dist/index.html so the scanner classifies it as a UI component.
func writeBuildableUIFolder(t *testing.T, scenarioDir string) {
	t.Helper()
	uiDir := filepath.Join(scenarioDir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "dist"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"ui"}`), 0o644); err != nil {
		t.Fatalf("failed to write ui package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write ui index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "dist", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write ui dist entry: %v", err)
	}
}
