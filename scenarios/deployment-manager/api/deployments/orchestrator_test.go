package deployments

import (
	"path/filepath"
	"testing"

	"deployment-manager/build"
	"deployment-manager/bundles"
)

// =============================================================================
// updateManifestBinaryPaths Tests
// =============================================================================

// TestUpdateManifestBinaryPaths_BasicPathUpdate verifies that binary paths are
// correctly updated from build results and made relative to the manifest directory.
func TestUpdateManifestBinaryPaths_BasicPathUpdate(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64":  {Path: "bin/api/linux-x64/test-api"},
					"darwin-x64": {Path: "bin/api/darwin-x64/test-api"},
					"win-x64":    {Path: "bin/api/win-x64/test-api.exe"},
				},
				Build: &bundles.BuildConfig{
					Type:      "go",
					SourceDir: "api",
				},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "linux-x64", "test-api"),
			Success:    true,
		},
		{
			Platform:   "darwin-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "darwin-x64", "test-api"),
			Success:    true,
		},
		{
			Platform:   "win-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "win-x64", "test-api.exe"),
			Success:    true,
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	// Verify paths are relative to manifest directory
	svc := &manifest.Services[0]
	expectedPaths := map[string]string{
		"linux-x64":  "bin/api/linux-x64/test-api",
		"darwin-x64": "bin/api/darwin-x64/test-api",
		"win-x64":    "bin/api/win-x64/test-api.exe",
	}

	for platform, expected := range expectedPaths {
		if actual := svc.Binaries[platform].Path; actual != expected {
			t.Errorf("Binary path for %s incorrect:\n  got:  %s\n  want: %s", platform, actual, expected)
		}
	}

	// Verify build config was cleared
	if svc.Build != nil {
		t.Error("Build config should be cleared after updating binary paths")
	}
}

// TestUpdateManifestBinaryPaths_ClearsBuildConfig verifies that the Build field
// is set to nil after binary paths are updated to prevent recompilation.
func TestUpdateManifestBinaryPaths_ClearsBuildConfig(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {Path: "bin/api/linux-x64/test-api"},
				},
				Build: &bundles.BuildConfig{
					Type:          "go",
					SourceDir:     "api",
					EntryPoint:    ".",
					OutputPattern: "bin/api/{{platform}}/test-api{{ext}}",
				},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "linux-x64", "test-api"),
			Success:    true,
		},
	}

	// Verify build config exists before
	if manifest.Services[0].Build == nil {
		t.Fatal("Setup error: Build config should exist before test")
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	// Verify build config is cleared
	if manifest.Services[0].Build != nil {
		t.Error("Build config should be nil after updateManifestBinaryPaths")
	}
}

// TestUpdateManifestBinaryPaths_SkipsServicesWithoutBuild verifies that services
// without a Build config are left unchanged.
func TestUpdateManifestBinaryPaths_SkipsServicesWithoutBuild(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	originalPath := "pre-compiled/binary"
	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "pre-compiled-svc",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {Path: originalPath},
				},
				Build: nil, // No build config
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "new-path", "binary"),
			Success:    true,
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	// Path should be unchanged since service has no Build config
	if manifest.Services[0].Binaries["linux-x64"].Path != originalPath {
		t.Errorf("Path should be unchanged for services without Build config:\n  got:  %s\n  want: %s",
			manifest.Services[0].Binaries["linux-x64"].Path, originalPath)
	}
}

// TestUpdateManifestBinaryPaths_IgnoresFailedBuilds verifies that failed build
// results don't update the manifest.
func TestUpdateManifestBinaryPaths_IgnoresFailedBuilds(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	originalPath := "bin/api/linux-x64/test-api"
	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {Path: originalPath},
				},
				Build: &bundles.BuildConfig{Type: "go"},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "new-path", "binary"),
			Success:    false, // Failed build
			Error:      "compilation error",
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	// Path should be unchanged since build failed
	if manifest.Services[0].Binaries["linux-x64"].Path != originalPath {
		t.Errorf("Path should be unchanged for failed builds:\n  got:  %s\n  want: %s",
			manifest.Services[0].Binaries["linux-x64"].Path, originalPath)
	}
}

// TestUpdateManifestBinaryPaths_PreservesArgs verifies that binary args, env, and cwd
// are preserved when updating paths.
func TestUpdateManifestBinaryPaths_PreservesArgs(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	originalArgs := []string{"--config", "production"}
	originalEnv := map[string]string{"LOG_LEVEL": "info"}
	originalCwd := "/opt/app"

	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {
						Path: "old/path",
						Args: originalArgs,
						Env:  originalEnv,
						Cwd:  originalCwd,
					},
				},
				Build: &bundles.BuildConfig{Type: "go"},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "linux-x64", "test-api"),
			Success:    true,
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	binary := manifest.Services[0].Binaries["linux-x64"]

	// Path should be updated
	if binary.Path == "old/path" {
		t.Error("Path should have been updated")
	}

	// Args should be preserved
	if len(binary.Args) != len(originalArgs) || binary.Args[0] != originalArgs[0] {
		t.Errorf("Args not preserved:\n  got:  %v\n  want: %v", binary.Args, originalArgs)
	}

	// Env should be preserved
	if binary.Env["LOG_LEVEL"] != "info" {
		t.Errorf("Env not preserved:\n  got:  %v\n  want: %v", binary.Env, originalEnv)
	}

	// Cwd should be preserved
	if binary.Cwd != originalCwd {
		t.Errorf("Cwd not preserved:\n  got:  %s\n  want: %s", binary.Cwd, originalCwd)
	}
}

// TestUpdateManifestBinaryPaths_ARMPlatforms verifies ARM platform handling.
func TestUpdateManifestBinaryPaths_ARMPlatforms(t *testing.T) {
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {Path: "bin/api/linux-x64/test-api"},
				},
				Build: &bundles.BuildConfig{Type: "go"},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "linux-x64", "test-api"),
			Success:    true,
		},
		{
			Platform:   "darwin-arm64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "darwin-arm64", "test-api"),
			Success:    true,
		},
		{
			Platform:   "linux-arm64",
			OutputPath: filepath.Join(manifestDir, "bin", "api", "linux-arm64", "test-api"),
			Success:    true,
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	// darwin-arm64 should be added
	if _, ok := manifest.Services[0].Binaries["darwin-arm64"]; !ok {
		t.Error("darwin-arm64 binary should have been added")
	}

	// linux-arm64 should be added
	if _, ok := manifest.Services[0].Binaries["linux-arm64"]; !ok {
		t.Error("linux-arm64 binary should have been added")
	}

	// Paths should be relative
	if manifest.Services[0].Binaries["darwin-arm64"].Path != "bin/api/darwin-arm64/test-api" {
		t.Errorf("darwin-arm64 path incorrect: %s", manifest.Services[0].Binaries["darwin-arm64"].Path)
	}
}

// TestUpdateManifestBinaryPaths_ForwardSlashes verifies paths use forward slashes
// for cross-platform JSON compatibility.
func TestUpdateManifestBinaryPaths_ForwardSlashes(t *testing.T) {
	// Use an absolute path that would have backslashes on Windows
	scenarioDir := "/home/user/scenarios/test-app"
	manifestDir := filepath.Join(scenarioDir, "platforms", "electron", "bundle")

	manifest := &bundles.Manifest{
		Services: []bundles.ServiceEntry{
			{
				ID:   "test-api",
				Type: "api-binary",
				Binaries: map[string]bundles.ServiceBinary{
					"linux-x64": {Path: "bin/api/linux-x64/test-api"},
				},
				Build: &bundles.BuildConfig{Type: "go"},
			},
		},
	}

	buildResults := []build.BuildResult{
		{
			Platform:   "linux-x64",
			OutputPath: filepath.Join(manifestDir, "bin", "deep", "nested", "path", "binary"),
			Success:    true,
		},
	}

	updateManifestBinaryPaths(manifest, buildResults, scenarioDir, manifestDir)

	path := manifest.Services[0].Binaries["linux-x64"].Path
	if filepath.Separator == '\\' && filepath.ToSlash(path) != path {
		t.Errorf("Path should use forward slashes: %s", path)
	}
}
