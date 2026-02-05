//go:build integration

package updates_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
	"scenario-to-desktop-api/updates/providers"
)

// TestIntegration_ManifestGeneration tests the complete manifest generation flow.
func TestIntegration_ManifestGeneration(t *testing.T) {
	// Create a temporary directory for test artifacts
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create a fake artifact file
	artifactPath := filepath.Join(tmpDir, "myapp-1.0.0.exe")
	artifactContent := []byte("fake executable content for testing")
	if err := os.WriteFile(artifactPath, artifactContent, 0o644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	// Create update config
	config := &generation.UpdateConfig{
		Provider: "generic",
		Channel:  "stable",
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	// Create provider factory with real implementations
	factory := providers.NewProviderFactory()

	// Create manifest generator
	generator := updates.NewManifestGenerator(
		updates.WithProviderFactory(factory),
	)

	// Generate manifests
	req := &updates.GenerateManifestsRequest{
		Config:    config,
		Version:   "1.0.0",
		Artifacts: map[string]string{"win": artifactPath},
		OutputDir: outputDir,
		Channel:   "stable",
	}

	result, err := generator.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("manifest generation failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.RequiresUpload {
		t.Error("generic provider should require upload")
	}

	if result.Provider.Name() != "generic" {
		t.Errorf("expected generic provider, got %s", result.Provider.Name())
	}

	// Verify manifest was generated
	if _, ok := result.Manifests["latest.yml"]; !ok {
		t.Fatal("expected latest.yml manifest")
	}

	// Verify manifest file was written to disk
	manifestPath := filepath.Join(outputDir, "latest.yml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatalf("manifest file not written to %s", manifestPath)
	}

	// Parse and validate manifest content
	manifestContent, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest updates.ElectronManifest
	if err := yaml.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatalf("failed to parse manifest YAML: %v", err)
	}

	// Validate manifest fields
	if manifest.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", manifest.Version)
	}

	if manifest.Sha512 == "" {
		t.Error("expected sha512 hash to be set")
	}

	// Verify sha512 is base64 encoded (should end with = or have valid base64 chars)
	if len(manifest.Sha512) < 80 {
		t.Errorf("sha512 hash seems too short: %s", manifest.Sha512)
	}

	if len(manifest.Files) != 1 {
		t.Errorf("expected 1 file entry, got %d", len(manifest.Files))
	}

	if len(manifest.Files) > 0 {
		file := manifest.Files[0]
		expectedURL := "https://updates.example.com/myapp/stable/myapp-1.0.0.exe"
		if file.URL != expectedURL {
			t.Errorf("expected URL %s, got %s", expectedURL, file.URL)
		}
		if file.Size != int64(len(artifactContent)) {
			t.Errorf("expected size %d, got %d", len(artifactContent), file.Size)
		}
	}

	// Check for warnings
	for _, w := range result.Warnings {
		t.Logf("Warning: [%s] %s", w.Code, w.Message)
	}
}

// TestIntegration_ManifestGeneration_MultiPlatform tests multi-platform manifest generation.
func TestIntegration_ManifestGeneration_MultiPlatform(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create fake artifacts for all platforms
	artifacts := map[string]string{}
	for _, platform := range []string{"win", "mac", "linux"} {
		ext := ".exe"
		switch platform {
		case "mac":
			ext = ".dmg"
		case "linux":
			ext = ".AppImage"
		}
		path := filepath.Join(tmpDir, "myapp-1.0.0"+ext)
		if err := os.WriteFile(path, []byte("fake "+platform+" binary"), 0o644); err != nil {
			t.Fatalf("failed to create %s artifact: %v", platform, err)
		}
		artifacts[platform] = path
	}

	config := &generation.UpdateConfig{
		Provider: "generic",
		Channel:  "beta",
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	factory := providers.NewProviderFactory()
	generator := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	req := &updates.GenerateManifestsRequest{
		Config:    config,
		Version:   "2.0.0-beta.1",
		Artifacts: artifacts,
		OutputDir: outputDir,
		Channel:   "beta",
	}

	result, err := generator.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("manifest generation failed: %v", err)
	}

	// Verify all platform manifests were generated
	expectedFiles := map[string]string{
		"latest.yml":       "win",
		"latest-mac.yml":   "mac",
		"latest-linux.yml": "linux",
	}

	for filename, platform := range expectedFiles {
		if _, ok := result.Manifests[filename]; !ok {
			t.Errorf("expected %s manifest for %s", filename, platform)
			continue
		}

		// Verify file was written
		manifestPath := filepath.Join(outputDir, filename)
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Errorf("manifest file not written: %s", manifestPath)
		}
	}

	if len(result.Manifests) != 3 {
		t.Errorf("expected 3 manifests, got %d", len(result.Manifests))
	}
}

// TestIntegration_ManifestGeneration_MissingURL tests warning when URL is missing.
func TestIntegration_ManifestGeneration_MissingURL(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake artifact
	artifactPath := filepath.Join(tmpDir, "myapp.exe")
	if err := os.WriteFile(artifactPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("failed to create artifact: %v", err)
	}

	config := &generation.UpdateConfig{
		Provider: "generic",
		// No URL configured - should generate warning
	}

	factory := providers.NewProviderFactory()
	generator := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	req := &updates.GenerateManifestsRequest{
		Config:    config,
		Version:   "1.0.0",
		Artifacts: map[string]string{"win": artifactPath},
	}

	result, err := generator.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a MISSING_URL warning
	hasMissingURLWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Code, "MISSING_URL") || strings.Contains(w.Code, "PROVIDER_INVALID") {
			hasMissingURLWarning = true
			break
		}
	}

	if !hasMissingURLWarning {
		t.Error("expected MISSING_URL or PROVIDER_INVALID warning for config without URL")
		for _, w := range result.Warnings {
			t.Logf("Got warning: [%s] %s", w.Code, w.Message)
		}
	}
}

// TestIntegration_GitHubProvider_NoManifestRequired tests that GitHub provider skips manifest generation.
func TestIntegration_GitHubProvider_NoManifestRequired(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "github",
		GitHub: &generation.GitHubUpdateConfig{
			Owner: "myorg",
			Repo:  "myapp",
		},
	}

	factory := providers.NewProviderFactory()
	generator := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	req := &updates.GenerateManifestsRequest{
		Config:    config,
		Version:   "1.0.0",
		Artifacts: map[string]string{"win": "/fake/path.exe"},
	}

	result, err := generator.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RequiresUpload {
		t.Error("github provider should not require manifest upload")
	}

	if result.Provider.Name() != "github" {
		t.Errorf("expected github provider, got %s", result.Provider.Name())
	}

	// GitHub provider delegates to electron-builder, so no manifests generated
	if result.Manifests != nil && len(result.Manifests) > 0 {
		t.Error("github provider should not generate manifests")
	}
}

// TestIntegration_NoneProvider tests disabled updates.
func TestIntegration_NoneProvider(t *testing.T) {
	config := &generation.UpdateConfig{
		Provider: "none",
	}

	factory := providers.NewProviderFactory()
	generator := updates.NewManifestGenerator(updates.WithProviderFactory(factory))

	req := &updates.GenerateManifestsRequest{
		Config:    config,
		Version:   "1.0.0",
		Artifacts: map[string]string{"win": "/fake/path.exe"},
	}

	result, err := generator.GenerateManifests(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RequiresUpload {
		t.Error("none provider should not require manifest upload")
	}

	if result.Provider.Name() != "none" {
		t.Errorf("expected none provider, got %s", result.Provider.Name())
	}
}
