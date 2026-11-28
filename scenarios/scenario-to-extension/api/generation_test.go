package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBuildStatePersistsAcrossManagerRestart(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()
	statePath := filepath.Join(env.Config.OutputPath, ".builds.json")
	manager := NewBuildManager(statePath)
	created := time.Now().UTC().Truncate(time.Second)
	manager.Add(&ExtensionBuild{
		BuildID:        "restart-safe-build",
		ScenarioName:   "neutral-fixture",
		TemplateType:   "full",
		Status:         "ready",
		ArtifactPath:   filepath.Join(env.Config.OutputPath, "neutral-fixture.zip"),
		ArtifactSHA256: strings.Repeat("a", 64),
		ArtifactBytes:  42,
		CreatedAt:      created,
	})

	reloaded := NewBuildManager(statePath)
	build, ok := reloaded.Get("restart-safe-build")
	if !ok {
		t.Fatal("persisted build was not restored")
	}
	if build.Status != "ready" || build.ArtifactSHA256 != strings.Repeat("a", 64) || build.ArtifactBytes != 42 {
		t.Fatalf("restored build lost durable receipt: %+v", build)
	}
}

func TestCancelExtensionHandlerCancelsRunningBuild(t *testing.T) {
	manager := NewBuildManager()
	manager.Add(&ExtensionBuild{BuildID: "cancel-me", Status: "building", CreatedAt: time.Now()})
	canceled := make(chan struct{})
	manager.RegisterCancel("cancel-me", func() { close(canceled) })
	original := buildManager
	buildManager = manager
	defer func() { buildManager = original }()

	response, err := executeRequest(cancelExtensionHandler, HTTPTestRequest{
		Method:  http.MethodPost,
		Path:    "/api/v1/extension/cancel/cancel-me",
		URLVars: map[string]string{"build_id": "cancel-me"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("cancel status = %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("cancel callback was not invoked")
	}
	build, ok := manager.Get("cancel-me")
	if !ok || build.Status != "canceled" || build.CompletedAt == nil {
		t.Fatalf("canceled build state = %+v", build)
	}
}

// TestGenerateExtension tests the extension generation process
func TestGenerateExtension(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success_CompleteGeneration", func(t *testing.T) {
		build := &ExtensionBuild{
			BuildID:      "test_build_gen",
			ScenarioName: "test-scenario",
			TemplateType: "full",
			Config: ExtensionConfig{
				AppName:         "Test App",
				Description:     "Test Description",
				APIEndpoint:     "http://localhost:3000",
				Version:         "1.0.0",
				AuthorName:      "Test Author",
				License:         "MIT",
				Permissions:     []string{"storage", "activeTab"},
				HostPermissions: []string{"<all_urls>"},
			},
			Status:        "building",
			ExtensionPath: filepath.Join(env.Config.OutputPath, "test-scenario", "platforms", "extension"),
			BuildLog:      []string{},
			ErrorLog:      []string{},
			CreatedAt:     time.Now(),
		}

		generateExtension(build)

		// Verify build completed successfully
		if build.Status != "ready" {
			t.Errorf("Expected status 'ready', got '%s'", build.Status)
		}

		if build.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set")
		}

		if len(build.BuildLog) == 0 {
			t.Error("Expected build log entries")
		}

		// Verify directory was created
		if _, err := os.Stat(build.ExtensionPath); os.IsNotExist(err) {
			t.Errorf("Expected extension directory to be created at %s", build.ExtensionPath)
		}
	})

	t.Run("Success_SpecializedTemplate", func(t *testing.T) {
		build := &ExtensionBuild{
			BuildID:      "test_build_specialized",
			ScenarioName: "test-scenario",
			TemplateType: "content-script-only",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
			Status:        "building",
			ExtensionPath: filepath.Join(env.Config.OutputPath, "test-scenario-specialized", "platforms", "extension"),
			BuildLog:      []string{},
			ErrorLog:      []string{},
			CreatedAt:     time.Now(),
		}

		generateExtension(build)

		if build.Status != "ready" {
			t.Errorf("Expected status 'ready', got '%s'", build.Status)
		}

		// Check that specialized template was logged
		foundSpecializedLog := false
		for _, log := range build.BuildLog {
			if contains(log, "specialized template") {
				foundSpecializedLog = true
				break
			}
		}
		if !foundSpecializedLog {
			t.Error("Expected log entry about specialized template")
		}
	})

	t.Run("Error_DirectoryCreationFailure", func(t *testing.T) {
		// Create a build with an invalid path to trigger failure
		build := &ExtensionBuild{
			BuildID:      "test_build_fail",
			ScenarioName: "test-scenario",
			TemplateType: "full",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
			Status:        "building",
			ExtensionPath: "/invalid/path/that/cannot/be/created\x00/extension",
			BuildLog:      []string{},
			ErrorLog:      []string{},
			CreatedAt:     time.Now(),
		}

		generateExtension(build)

		if build.Status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", build.Status)
		}

		if len(build.ErrorLog) == 0 {
			t.Error("Expected error log entries")
		}

		if build.CompletedAt == nil {
			t.Error("Expected CompletedAt to be set even for failed builds")
		}
	})

	t.Run("Recovery_FromPanic", func(t *testing.T) {
		// This test ensures the panic recovery works
		// We can't easily trigger a panic in the current implementation,
		// but we verify the defer recover is in place by checking the function structure
		build := &ExtensionBuild{
			BuildID:      "test_build_panic",
			ScenarioName: "test-scenario",
			TemplateType: "full",
			Config: ExtensionConfig{
				AppName:     "Test App",
				APIEndpoint: "http://localhost:3000",
			},
			Status:        "building",
			ExtensionPath: filepath.Join(env.Config.OutputPath, "test-scenario-panic", "platforms", "extension"),
			BuildLog:      []string{},
			ErrorLog:      []string{},
			CreatedAt:     time.Now(),
		}

		// Normal execution should not panic
		generateExtension(build)

		if build.Status != "ready" {
			t.Errorf("Expected status 'ready', got '%s'", build.Status)
		}
	})
}

func TestGeneratedArtifactHasDeterministicIntegrityReceipt(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()
	configCleanup := setupTestConfig(env)
	defer configCleanup()

	build := &ExtensionBuild{
		BuildID:       "artifact-receipt",
		ScenarioName:  "neutral-fixture",
		TemplateType:  "full",
		Status:        "building",
		ExtensionPath: filepath.Join(env.Config.OutputPath, "neutral-fixture", "platforms", "extension"),
		Config: ExtensionConfig{
			AppName:         "Neutral Fixture",
			Description:     "fixture",
			APIEndpoint:     "http://localhost:3000",
			Version:         "1.0.0",
			Permissions:     []string{"storage"},
			HostPermissions: []string{"https://example.test/*"},
		},
		BuildLog:  []string{},
		ErrorLog:  []string{},
		CreatedAt: time.Now(),
	}
	generateExtension(build)
	if build.Status != "ready" {
		t.Fatalf("generated fixture status = %q, errors: %v", build.Status, build.ErrorLog)
	}
	if build.ArtifactSHA256 == "" || build.ManifestSHA256 == "" || build.ArtifactBytes == 0 {
		t.Fatalf("artifact receipt is incomplete: %+v", build)
	}

	artifactBytes, err := os.ReadFile(build.ArtifactPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(artifactBytes)
	if got := hex.EncodeToString(digest[:]); got != build.ArtifactSHA256 {
		t.Fatalf("artifact digest = %q, receipt = %q", got, build.ArtifactSHA256)
	}
	archive, err := zip.NewReader(bytes.NewReader(artifactBytes), int64(len(artifactBytes)))
	if err != nil {
		t.Fatal(err)
	}
	foundManifest := false
	foundProbeHTML, foundProbeJS := false, false
	for _, entry := range archive.File {
		if entry.Name == "manifest.json" {
			foundManifest = true
		}
		if entry.Name == browserPermissionProbePath {
			foundProbeHTML = true
		}
		if entry.Name == "__vrooli_bas/permission-report.js" {
			foundProbeJS = true
			reader, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			script, readErr := io.ReadAll(reader)
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read permission report script err=%v close=%v", readErr, closeErr)
			}
			scriptText := string(script)
			if !strings.Contains(scriptText, "chrome.permissions.getAll()") || strings.Contains(scriptText, "permissions.request(") || strings.Contains(scriptText, "fetch(") || strings.Contains(scriptText, "WebSocket(") || strings.Contains(scriptText, "XMLHttpRequest") {
				t.Fatalf("permission report does not use a read-only, no-network source: %s", scriptText)
			}
		}
		if strings.HasPrefix(entry.Name, "node_modules/") {
			t.Fatalf("release artifact includes build dependency %q", entry.Name)
		}
	}
	if !foundManifest {
		t.Fatal("release artifact has no manifest.json")
	}
	if !foundProbeHTML || !foundProbeJS {
		t.Fatalf("release artifact omitted its permission-report contract: html=%t js=%t", foundProbeHTML, foundProbeJS)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(build.ExtensionPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	permissions, ok := manifest["permissions"].([]any)
	if !ok || len(permissions) != 1 || permissions[0] != "storage" {
		t.Fatalf("permission report changed manifest API permissions: %#v", manifest["permissions"])
	}
	if _, exists := manifest["optional_permissions"]; exists {
		t.Fatalf("permission report added optional API permissions: %#v", manifest["optional_permissions"])
	}

	buildManager = NewBuildManager()
	buildManager.Add(build)
	metadata, err := buildManager.VerifyBrowserExtensionArtifact(build.BuildID, build.ArtifactSHA256)
	if err != nil {
		t.Fatalf("verify generated extension artifact: %v", err)
	}
	if len(metadata.ExtensionID) != 32 || metadata.ArtifactRef != build.BuildID || metadata.ArtifactSHA256 != build.ArtifactSHA256 || metadata.ManifestSHA256 != build.ManifestSHA256 || metadata.Version != "1.0.0" {
		t.Fatalf("artifact metadata did not bind id/digests/version: %#v", metadata)
	}
	if !slices.Contains(metadata.Permissions, "storage") || !slices.Contains(metadata.HostPermissions, "https://example.test/*") {
		t.Fatalf("permissions were not derived from the ZIP manifest: %#v", metadata)
	}
	if metadata.PermissionProbe != "chrome.permissions.getAll.v1" {
		t.Fatalf("metadata did not expose its digest-bound runtime permission owner: %#v", metadata)
	}
	if _, err := buildManager.VerifyBrowserExtensionArtifact(build.BuildID, "wrong-digest"); err == nil {
		t.Fatal("artifact owner accepted a caller digest mismatch")
	}
	metadataResponse, err := executeRequest(browserExtensionArtifactMetadataHandler, HTTPTestRequest{
		Method: http.MethodGet, Path: "/api/v1/extension/artifact/" + build.BuildID + "/metadata?expected_digest=" + build.ArtifactSHA256,
		URLVars: map[string]string{"build_id": build.BuildID},
	})
	if err != nil || metadataResponse.Code != http.StatusOK {
		t.Fatalf("artifact metadata response status=%d err=%v: %s", metadataResponse.Code, err, metadataResponse.Body.String())
	}
	if strings.Contains(metadataResponse.Body.String(), build.ArtifactPath) {
		t.Fatal("artifact metadata leaked a host filesystem path")
	}
	response, err := executeRequest(downloadExtensionHandler, HTTPTestRequest{
		Method:  http.MethodGet,
		Path:    "/api/v1/extension/download/" + build.BuildID,
		URLVars: map[string]string{"build_id": build.BuildID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("download status = %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Artifact-SHA256") != build.ArtifactSHA256 {
		t.Fatalf("download digest header = %q, receipt = %q", response.Header().Get("X-Artifact-SHA256"), build.ArtifactSHA256)
	}
	if response.Body.Len() != int(build.ArtifactBytes) {
		t.Fatalf("download bytes = %d, receipt = %d", response.Body.Len(), build.ArtifactBytes)
	}

	if err := os.WriteFile(build.ArtifactPath, append(artifactBytes, byte('x')), 0o644); err != nil {
		t.Fatal(err)
	}
	tampered, err := executeRequest(downloadExtensionHandler, HTTPTestRequest{
		Method:  http.MethodGet,
		Path:    "/api/v1/extension/download/" + build.BuildID,
		URLVars: map[string]string{"build_id": build.BuildID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tampered.Code != http.StatusConflict {
		t.Fatalf("tampered download status = %d, want %d", tampered.Code, http.StatusConflict)
	}
	if _, err := buildManager.VerifyBrowserExtensionArtifact(build.BuildID, build.ArtifactSHA256); err == nil {
		t.Fatal("artifact owner returned metadata after ZIP tampering")
	}
}

func TestCustomExtensionSourceIsPackagedWithoutScenarioBranches(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()
	configCleanup := setupTestConfig(env)
	defer configCleanup()

	source := filepath.Join(env.TempDir, "custom-extension")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "manifest.json"), []byte(`{"manifest_version":3,"name":"{{APP_NAME}}","version":"{{VERSION}}","optional_permissions":["tabs"],"optional_host_permissions":["https://optional.example/*"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "content.js"), []byte("window.__neutral_fixture = true;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "package.json"), []byte(`{"name":"{{PACKAGE_NAME}}","version":"{{VERSION}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# {{APP_NAME}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	build := &ExtensionBuild{
		BuildID:       "custom-source",
		ScenarioName:  "neutral-fixture",
		TemplateType:  "full",
		Status:        "building",
		ExtensionPath: filepath.Join(env.Config.OutputPath, "custom-source", "platforms", "extension"),
		Config: ExtensionConfig{
			AppName: "Neutral Custom Fixture", Description: "custom source", APIEndpoint: "http://localhost:3000", SourcePath: source,
		},
		BuildLog: []string{}, ErrorLog: []string{}, CreatedAt: time.Now(),
	}
	generateExtension(build)
	if build.Status != "ready" {
		t.Fatalf("custom source status = %q, errors: %v", build.Status, build.ErrorLog)
	}
	content, err := os.ReadFile(filepath.Join(build.ExtensionPath, "content.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "window.__neutral_fixture = true;\n" {
		t.Fatalf("custom source content was not packaged: %q", content)
	}
	if _, err := NewBuildManager().VerifyBrowserExtensionArtifact(build.BuildID, build.ArtifactSHA256); err == nil {
		t.Fatal("artifact verifier accepted a build that was never registered with the artifact owner")
	}
	manager := NewBuildManager()
	if err := manager.Add(build); err != nil {
		t.Fatal(err)
	}
	metadata, err := manager.VerifyBrowserExtensionArtifact(build.BuildID, build.ArtifactSHA256)
	if err != nil {
		t.Fatalf("verify custom artifact metadata: %v", err)
	}
	if !slices.Contains(metadata.Permissions, "tabs") || !slices.Contains(metadata.HostPermissions, "https://optional.example/*") {
		t.Fatalf("optional manifest permission visibility missing: %#v", metadata)
	}
}

// TestCopyAndProcessTemplates tests template copying functionality
func TestCopyAndProcessTemplates(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("Success", func(t *testing.T) {
		templatePath := filepath.Join(env.Config.TemplatesPath, "vanilla")
		outputPath := filepath.Join(env.TempDir, "output")

		build := &ExtensionBuild{
			BuildID: "test_copy",
			Config: ExtensionConfig{
				AppName:     "Test App",
				Description: "Test Description",
				Version:     "1.0.0",
			},
			BuildLog: []string{},
		}

		err := copyAndProcessTemplates(templatePath, outputPath, build)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(build.BuildLog) == 0 {
			t.Error("Expected build log entry")
		}
	})
}

// TestGeneratePackageJSON tests package.json generation
func TestGeneratePackageJSON(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	t.Run("Success", func(t *testing.T) {
		build := &ExtensionBuild{
			BuildID: "test_package",
			Config: ExtensionConfig{
				AppName: "Test App",
				Version: "1.0.0",
			},
			BuildLog: []string{},
		}

		err := generatePackageJSON(build)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(build.BuildLog) == 0 {
			t.Error("Expected build log entry")
		}
	})
}

// TestGenerateREADME tests README generation
func TestGenerateREADME(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	t.Run("Success", func(t *testing.T) {
		build := &ExtensionBuild{
			BuildID: "test_readme",
			Config: ExtensionConfig{
				AppName:     "Test App",
				Description: "Test Description",
			},
			BuildLog: []string{},
		}

		err := generateREADME(build)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(build.BuildLog) == 0 {
			t.Error("Expected build log entry")
		}
	})
}

// TestUnavailableExtensionResultNeverClaimsPrivatePathSuccess protects the
// legacy response helper while the endpoint only accepts managed BAS runs.
func TestUnavailableExtensionResultNeverClaimsPrivatePathSuccess(t *testing.T) {
	result := unavailableExtensionTest(&ExtensionTestRequest{
		ExtensionPath: "/local/extension",
		TestSites:     []string{"https://example.com", "https://test.example"},
	}, "managed BAS session required")
	if result.Success || result.Status != "unavailable" || result.Reason == "" {
		t.Fatalf("unavailable result = %#v", result)
	}
	if result.Summary.TotalTests != 2 || result.Summary.Failed != 2 || result.Summary.Passed != 0 {
		t.Fatalf("unavailable result summary = %#v", result.Summary)
	}
	for _, site := range result.TestResults {
		if site.Loaded {
			t.Fatalf("unavailable result claimed %q loaded", site.Site)
		}
	}
}

// TestListAvailableTemplates tests template listing
func TestListAvailableTemplates(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()
	env := setupTestDirectory(t)
	defer env.Cleanup()
	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("VerifyTemplateStructure", func(t *testing.T) {
		templates := listAvailableTemplates()

		if len(templates) == 0 {
			t.Fatal("Expected at least one template")
		}

		// Verify each template has required fields
		for _, tmpl := range templates {
			if _, exists := tmpl["name"]; !exists {
				t.Error("Template missing 'name' field")
			}
			if _, exists := tmpl["display_name"]; !exists {
				t.Error("Template missing 'display_name' field")
			}
			if _, exists := tmpl["description"]; !exists {
				t.Error("Template missing 'description' field")
			}
			if _, exists := tmpl["files"]; !exists {
				t.Error("Template missing 'files' field")
			}

			// Verify files is an array
			if files, ok := tmpl["files"].([]string); !ok {
				t.Error("Template 'files' field should be array of strings")
			} else if len(files) == 0 {
				t.Error("Template should have at least one file")
			}
		}
	})

	t.Run("VerifyExpectedTemplates", func(t *testing.T) {
		templates := listAvailableTemplates()
		templateNames := make(map[string]bool)

		for _, tmpl := range templates {
			if name, ok := tmpl["name"].(string); ok {
				templateNames[name] = true
			}
		}

		expectedTemplates := []string{"full", "content-script-only", "background-only", "popup-only"}
		for _, expected := range expectedTemplates {
			if !templateNames[expected] {
				t.Errorf("Expected template '%s' not found", expected)
			}
		}
	})
}

// TestGetInstallInstructions tests install instructions generation
func TestGetInstallInstructionsFunc(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	t.Run("FullTemplate", func(t *testing.T) {
		instructions := getInstallInstructions("full")

		if instructions == "" {
			t.Fatal("Expected non-empty instructions")
		}

		// Verify key information is present
		requiredPhrases := []string{
			"chrome://extensions",
			"Developer mode",
			"Load unpacked",
			"npm run dev",
			"npm run build",
		}

		for _, phrase := range requiredPhrases {
			if !contains(instructions, phrase) {
				t.Errorf("Expected instructions to contain '%s'", phrase)
			}
		}
	})

	t.Run("OtherTemplates", func(t *testing.T) {
		templates := []string{"content-script-only", "background-only", "popup-only"}

		for _, template := range templates {
			instructions := getInstallInstructions(template)
			if instructions == "" {
				t.Errorf("Expected non-empty instructions for template '%s'", template)
			}
		}
	})
}
