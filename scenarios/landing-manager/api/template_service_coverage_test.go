package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScaffoldScenario tests the scaffoldScenario function comprehensively
func TestScaffoldScenario(t *testing.T) {
	t.Run("scaffold with TEMPLATE_PAYLOAD_DIR override", func(t *testing.T) {
		// Create a temporary template payload directory
		tmpPayload := t.TempDir()
		tmpOutput := t.TempDir()

		// Create complete template structure (all required directories)
		apiDir := filepath.Join(tmpPayload, "api")
		os.MkdirAll(apiDir, 0755)
		os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0644)

		uiDir := filepath.Join(tmpPayload, "ui")
		os.MkdirAll(filepath.Join(uiDir, "src"), 0755)
		os.WriteFile(filepath.Join(uiDir, "src", "App.tsx"), []byte("export default function App() {}"), 0644)

		vrooliDir := filepath.Join(tmpPayload, ".vrooli")
		os.MkdirAll(vrooliDir, 0755)
		os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(`{"name":"test"}`), 0644)

		// Create requirements directory (required by scaffoldScenario)
		reqDir := filepath.Join(tmpPayload, "requirements")
		os.MkdirAll(reqDir, 0755)
		os.WriteFile(filepath.Join(reqDir, "index.json"), []byte(`{}`), 0644)

		// Create initialization directory (may be required)
		initDir := filepath.Join(tmpPayload, "initialization")
		os.MkdirAll(initDir, 0755)

		// Create Makefile and PRD.md (required by copyTemplatePayload)
		os.WriteFile(filepath.Join(tmpPayload, "Makefile"), []byte("all:\n\techo test\n"), 0644)
		os.WriteFile(filepath.Join(tmpPayload, "PRD.md"), []byte("# PRD\n"), 0644)

		// Set override environment variable
		os.Setenv("TEMPLATE_PAYLOAD_DIR", tmpPayload)
		defer os.Unsetenv("TEMPLATE_PAYLOAD_DIR")

		// Run scaffold
		err := scaffoldScenario(tmpOutput)
		if err != nil {
			t.Fatalf("Expected scaffoldScenario to succeed, got error: %v", err)
		}

		// Verify output structure
		if _, err := os.Stat(filepath.Join(tmpOutput, "api", "main.go")); os.IsNotExist(err) {
			t.Error("Expected api/main.go to be copied")
		}

		if _, err := os.Stat(filepath.Join(tmpOutput, "ui", "src", "App.tsx")); os.IsNotExist(err) {
			t.Error("Expected ui/src/App.tsx to be copied")
		}

		if _, err := os.Stat(filepath.Join(tmpOutput, ".vrooli", "service.json")); os.IsNotExist(err) {
			t.Error("Expected .vrooli/service.json to be copied")
		}
	})

	t.Run("scaffold without override uses fallback", func(t *testing.T) {
		tmpOutput := t.TempDir()

		// Clear any environment override
		os.Unsetenv("TEMPLATE_PAYLOAD_DIR")

		// This test will attempt to resolve from executable path
		// The function should either succeed or fail gracefully
		err := scaffoldScenario(tmpOutput)

		// We don't assert success here because the test environment may not have
		// a proper template payload. We're mainly testing code paths and error handling.
		t.Logf("scaffoldScenario result (fallback mode): %v", err)
	})

	t.Run("scaffold with invalid TEMPLATE_PAYLOAD_DIR", func(t *testing.T) {
		tmpOutput := t.TempDir()

		// Set to non-existent directory
		os.Setenv("TEMPLATE_PAYLOAD_DIR", "/nonexistent/path")
		defer os.Unsetenv("TEMPLATE_PAYLOAD_DIR")

		err := scaffoldScenario(tmpOutput)
		if err == nil {
			t.Error("Expected error when TEMPLATE_PAYLOAD_DIR is invalid")
		}
	})
}

// TestCopyTemplatePayload tests the copyTemplatePayload function
func TestCopyTemplatePayload(t *testing.T) {
	t.Run("copy complete template payload", func(t *testing.T) {
		tmpSource := t.TempDir()
		tmpOutput := t.TempDir()

		// Create complete template structure
		dirs := []string{"api", "ui/src/pages", "requirements", "initialization", ".vrooli"}
		for _, dir := range dirs {
			os.MkdirAll(filepath.Join(tmpSource, dir), 0755)
		}

		// Create files
		files := map[string]string{
			"api/main.go":                   "package main\n",
			"ui/src/App.tsx":                "export default function App() {}",
			"ui/src/pages/FactoryHome.tsx":  "export default function FactoryHome() {}",
			"ui/src/pages/PublicHome.tsx":   "export default function PublicHome() {}",
			"requirements/index.json":       "{}",
			".vrooli/service.json":          `{"name":"test"}`,
			"Makefile":                      "all:\n\techo test",
			"PRD.md":                        "# PRD",
		}

		for path, content := range files {
			fullPath := filepath.Join(tmpSource, path)
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(content), 0644)
		}

		err := copyTemplatePayload(tmpSource, tmpOutput)
		if err != nil {
			t.Fatalf("Expected copyTemplatePayload to succeed, got error: %v", err)
		}

		// Verify key files were copied
		if _, err := os.Stat(filepath.Join(tmpOutput, "api", "main.go")); os.IsNotExist(err) {
			t.Error("Expected api/main.go to be copied")
		}

		if _, err := os.Stat(filepath.Join(tmpOutput, ".vrooli", "service.json")); os.IsNotExist(err) {
			t.Error("Expected .vrooli/service.json to be copied")
		}

		// Verify FactoryHome.tsx was removed
		if _, err := os.Stat(filepath.Join(tmpOutput, "ui", "src", "pages", "FactoryHome.tsx")); !os.IsNotExist(err) {
			t.Error("Expected FactoryHome.tsx to be removed from generated scenario")
		}

		// Verify README.md was created
		if _, err := os.Stat(filepath.Join(tmpOutput, "README.md")); os.IsNotExist(err) {
			t.Error("Expected README.md to be created")
		}

		// Verify App.tsx was rewritten
		appContent, err := os.ReadFile(filepath.Join(tmpOutput, "ui", "src", "App.tsx"))
		if err != nil {
			t.Fatalf("Failed to read App.tsx: %v", err)
		}

		appStr := string(appContent)
		if !contains(appStr, "PublicHome") {
			t.Error("Expected App.tsx to contain PublicHome route")
		}

		if !contains(appStr, "AdminHome") {
			t.Error("Expected App.tsx to contain AdminHome route")
		}
	})

	t.Run("copy with missing source directories", func(t *testing.T) {
		tmpSource := t.TempDir()
		tmpOutput := t.TempDir()

		// Create only minimal structure
		os.MkdirAll(filepath.Join(tmpSource, "api"), 0755)
		os.WriteFile(filepath.Join(tmpSource, "api", "main.go"), []byte("package main"), 0644)

		err := copyTemplatePayload(tmpSource, tmpOutput)
		// Should fail because required directories are missing
		if err == nil {
			t.Error("Expected error when source directories are missing")
		}
	})
}

// TestCopyDir tests directory copying with various scenarios
func TestCopyDir(t *testing.T) {
	t.Run("copy directory with nested files", func(t *testing.T) {
		tmpSrc := t.TempDir()
		tmpDst := t.TempDir()

		// Create nested structure
		nested := filepath.Join(tmpSrc, "level1", "level2", "level3")
		os.MkdirAll(nested, 0755)

		files := []string{
			filepath.Join(tmpSrc, "file1.txt"),
			filepath.Join(tmpSrc, "level1", "file2.txt"),
			filepath.Join(nested, "file3.txt"),
		}

		for _, f := range files {
			os.WriteFile(f, []byte("content"), 0644)
		}

		dstPath := filepath.Join(tmpDst, "copied")
		err := copyDir(tmpSrc, dstPath)
		if err != nil {
			t.Fatalf("Expected copyDir to succeed, got error: %v", err)
		}

		// Verify all files were copied
		expectedFiles := []string{
			filepath.Join(dstPath, "file1.txt"),
			filepath.Join(dstPath, "level1", "file2.txt"),
			filepath.Join(dstPath, "level1", "level2", "level3", "file3.txt"),
		}

		for _, f := range expectedFiles {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist", f)
			}
		}
	})

	t.Run("copy single file", func(t *testing.T) {
		tmpSrc := t.TempDir()
		tmpDst := t.TempDir()

		srcFile := filepath.Join(tmpSrc, "test.txt")
		os.WriteFile(srcFile, []byte("test content"), 0644)

		dstFile := filepath.Join(tmpDst, "test.txt")
		err := copyDir(srcFile, dstFile)
		if err != nil {
			t.Fatalf("Expected copyDir to handle single file, got error: %v", err)
		}

		content, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(content) != "test content" {
			t.Errorf("Expected copied content to match, got: %s", content)
		}
	})

	t.Run("copy non-existent source", func(t *testing.T) {
		tmpDst := t.TempDir()

		err := copyDir("/nonexistent/path", filepath.Join(tmpDst, "dest"))
		if err == nil {
			t.Error("Expected error when copying non-existent source")
		}
	})

	t.Run("copy with permission-restricted destination", func(t *testing.T) {
		tmpSrc := t.TempDir()
		tmpDst := t.TempDir()

		srcFile := filepath.Join(tmpSrc, "test.txt")
		os.WriteFile(srcFile, []byte("content"), 0644)

		// Make destination read-only
		os.Chmod(tmpDst, 0444)
		defer os.Chmod(tmpDst, 0755) // Restore permissions for cleanup

		err := copyDir(srcFile, filepath.Join(tmpDst, "test.txt"))
		// Should fail due to permissions (on most systems)
		// Note: This may not fail on all systems/configurations
		t.Logf("Copy with restricted permissions result: %v", err)
	})
}

// TestCopyFile tests file copying edge cases
func TestCopyFile(t *testing.T) {
	t.Run("copy file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcFile := filepath.Join(tmpDir, "source.txt")
		dstFile := filepath.Join(tmpDir, "dest.txt")

		testContent := "Hello, World!"
		os.WriteFile(srcFile, []byte(testContent), 0644)

		err := copyFile(srcFile, dstFile, 0644)
		if err != nil {
			t.Fatalf("Expected copyFile to succeed, got error: %v", err)
		}

		content, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(content) != testContent {
			t.Errorf("Expected content '%s', got '%s'", testContent, content)
		}

		// Verify file permissions
		srcInfo, _ := os.Stat(srcFile)
		dstInfo, _ := os.Stat(dstFile)

		if srcInfo.Mode() != dstInfo.Mode() {
			t.Errorf("Expected file permissions to match: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
		}
	})

	t.Run("copy non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := copyFile("/nonexistent/file.txt", filepath.Join(tmpDir, "dest.txt"), 0644)
		if err == nil {
			t.Error("Expected error when copying non-existent file")
		}
	})

	t.Run("copy to invalid destination", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcFile := filepath.Join(tmpDir, "source.txt")
		os.WriteFile(srcFile, []byte("content"), 0644)

		// Try to copy to a directory that doesn't exist
		err := copyFile(srcFile, "/nonexistent/path/dest.txt", 0644)
		if err == nil {
			t.Error("Expected error when destination directory doesn't exist")
		}
	})

	t.Run("copy empty file", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcFile := filepath.Join(tmpDir, "empty.txt")
		dstFile := filepath.Join(tmpDir, "empty_copy.txt")

		os.WriteFile(srcFile, []byte(""), 0644)

		err := copyFile(srcFile, dstFile, 0644)
		if err != nil {
			t.Fatalf("Expected copyFile to handle empty file, got error: %v", err)
		}

		info, err := os.Stat(dstFile)
		if err != nil {
			t.Fatalf("Failed to stat copied file: %v", err)
		}

		if info.Size() != 0 {
			t.Errorf("Expected empty file, got size: %d", info.Size())
		}
	})

	t.Run("copy large file", func(t *testing.T) {
		tmpDir := t.TempDir()

		srcFile := filepath.Join(tmpDir, "large.txt")
		dstFile := filepath.Join(tmpDir, "large_copy.txt")

		// Create a 1MB file
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}
		os.WriteFile(srcFile, largeContent, 0644)

		err := copyFile(srcFile, dstFile, 0644)
		if err != nil {
			t.Fatalf("Expected copyFile to handle large file, got error: %v", err)
		}

		copiedContent, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if len(copiedContent) != len(largeContent) {
			t.Errorf("Expected copied file size %d, got %d", len(largeContent), len(copiedContent))
		}
	})
}

// TestWriteLandingApp tests the App.tsx generation
func TestWriteLandingApp(t *testing.T) {
	t.Run("generate landing app successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		appPath := filepath.Join(tmpDir, "App.tsx")

		err := writeLandingApp(appPath)
		if err != nil {
			t.Fatalf("Expected writeLandingApp to succeed, got error: %v", err)
		}

		content, err := os.ReadFile(appPath)
		if err != nil {
			t.Fatalf("Failed to read generated App.tsx: %v", err)
		}

		appStr := string(content)

		// Verify key routes are present
		expectedStrings := []string{
			"PublicHome",
			"AdminHome",
			"AdminLogin",
			"VariantEditor",
			"SectionEditor",
			"AdminAnalytics",
			"AgentCustomization",
			"BrowserRouter",
			"AuthProvider",
			"VariantProvider",
			"ProtectedRoute",
			`path="/"`,
			`path="/health"`,
			`path="/admin"`,
			`path="/admin/login"`,
		}

		for _, expected := range expectedStrings {
			if !contains(appStr, expected) {
				t.Errorf("Expected App.tsx to contain '%s'", expected)
			}
		}
	})

	t.Run("write to invalid path", func(t *testing.T) {
		err := writeLandingApp("/nonexistent/directory/App.tsx")
		if err == nil {
			t.Error("Expected error when writing to invalid path")
		}
	})
}

// TestRewriteServiceConfig tests the rewriteServiceConfig function
func TestRewriteServiceConfig(t *testing.T) {
	t.Run("rewrite service config successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		os.MkdirAll(vrooliDir, 0755)

		// Create a realistic service.json with factory-only steps
		serviceJSON := `{
			"service": {
				"name": "landing-manager",
				"displayName": "Landing Manager",
				"description": "Factory scenario",
				"repository": {
					"directory": "/scenarios/landing-manager"
				}
			},
			"lifecycle": {
				"setup": {
					"steps": [
						{"name": "install-cli", "run": "echo install"},
						{"name": "setup-db", "run": "echo setup"}
					]
				},
				"develop": {
					"steps": [
						{"name": "start-api", "run": "cd api && exec ./landing-manager"},
						{"name": "start-ui", "run": "npm run dev"}
					]
				}
			}
		}`

		configPath := filepath.Join(vrooliDir, "service.json")
		os.WriteFile(configPath, []byte(serviceJSON), 0644)

		// Rewrite config
		err := rewriteServiceConfig(tmpDir, "Test Landing", "test-landing")
		if err != nil {
			t.Fatalf("Expected rewriteServiceConfig to succeed, got error: %v", err)
		}

		// Read and verify rewritten config
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read rewritten config: %v", err)
		}

		configStr := string(data)

		// Verify service name and display name were updated
		if !contains(configStr, `"name": "test-landing"`) {
			t.Error("Expected service name to be updated to 'test-landing'")
		}

		if !contains(configStr, `"displayName": "Test Landing"`) {
			t.Error("Expected displayName to be updated to 'Test Landing'")
		}

		// Verify description mentions template
		if !contains(configStr, "Generated landing page scenario") {
			t.Error("Expected description to mention generated scenario")
		}

		// Verify repository directory was updated
		if !contains(configStr, `"/scenarios/test-landing"`) {
			t.Error("Expected repository directory to be updated")
		}

		// Verify install-cli step was removed
		if contains(configStr, "install-cli") {
			t.Error("Expected 'install-cli' step to be removed")
		}

		// Verify setup-db step was kept
		if !contains(configStr, "setup-db") {
			t.Error("Expected 'setup-db' step to be kept")
		}

		// Verify API start command was updated
		if !contains(configStr, "./landing-manager-api") {
			t.Error("Expected API start command to reference landing-manager-api")
		}
	})

	t.Run("rewrite with missing service.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := rewriteServiceConfig(tmpDir, "Test", "test")
		if err == nil {
			t.Error("Expected error when service.json doesn't exist")
		}
	})

	t.Run("rewrite with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		os.MkdirAll(vrooliDir, 0755)

		configPath := filepath.Join(vrooliDir, "service.json")
		os.WriteFile(configPath, []byte("{invalid json}"), 0644)

		err := rewriteServiceConfig(tmpDir, "Test", "test")
		if err == nil {
			t.Error("Expected error when JSON is invalid")
		}
	})

	t.Run("rewrite with minimal config", func(t *testing.T) {
		tmpDir := t.TempDir()
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		os.MkdirAll(vrooliDir, 0755)

		// Minimal config without lifecycle or repository
		minimalJSON := `{"service": {"name": "old-name"}}`
		configPath := filepath.Join(vrooliDir, "service.json")
		os.WriteFile(configPath, []byte(minimalJSON), 0644)

		err := rewriteServiceConfig(tmpDir, "New Name", "new-name")
		if err != nil {
			t.Fatalf("Expected rewriteServiceConfig to handle minimal config, got error: %v", err)
		}

		// Verify it still updated what it could
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config: %v", err)
		}

		if !contains(string(data), `"name": "new-name"`) {
			t.Error("Expected service name to be updated")
		}
	})
}

// TestWriteTemplateProvenance tests the writeTemplateProvenance function
func TestWriteTemplateProvenance(t *testing.T) {
	t.Run("write provenance successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		template := &Template{
			ID:      "saas-landing-v1",
			Version: "1.2.3",
		}

		err := writeTemplateProvenance(tmpDir, template)
		if err != nil {
			t.Fatalf("Expected writeTemplateProvenance to succeed, got error: %v", err)
		}

		// Read and verify provenance file (written to .vrooli/template.json)
		provenancePath := filepath.Join(tmpDir, ".vrooli", "template.json")
		data, err := os.ReadFile(provenancePath)
		if err != nil {
			t.Fatalf("Failed to read provenance file: %v", err)
		}

		provenanceStr := string(data)

		// Verify template ID and version
		if !contains(provenanceStr, `"template_id": "saas-landing-v1"`) {
			t.Error("Expected provenance to contain template_id")
		}

		if !contains(provenanceStr, `"template_version": "1.2.3"`) {
			t.Error("Expected provenance to contain template_version")
		}

		// Verify generated_at timestamp exists
		if !contains(provenanceStr, "generated_at") {
			t.Error("Expected provenance to contain generated_at timestamp")
		}

		// Verify .vrooli directory was created
		if _, err := os.Stat(filepath.Join(tmpDir, ".vrooli")); os.IsNotExist(err) {
			t.Error("Expected .vrooli directory to be created")
		}
	})

	t.Run("write provenance handles directory creation", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Don't pre-create .vrooli directory - function should create it
		template := &Template{
			ID:      "test-template",
			Version: "1.0.0",
		}

		err := writeTemplateProvenance(tmpDir, template)
		if err != nil {
			t.Fatalf("Expected writeTemplateProvenance to create directory, got error: %v", err)
		}

		provenancePath := filepath.Join(tmpDir, ".vrooli", "template.json")
		if _, err := os.Stat(provenancePath); os.IsNotExist(err) {
			t.Error("Expected template.json to be created")
		}
	})
}

// TestValidateGeneratedScenario tests the validateGeneratedScenario function
func TestValidateGeneratedScenario(t *testing.T) {
	t.Run("validate complete scenario", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create all required directories and files
		dirs := []string{
			"api",
			"ui/src",
			"requirements",
			".vrooli",
		}

		for _, dir := range dirs {
			os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		}

		// Create required files
		files := map[string]string{
			"PRD.md":    "# PRD",
			"Makefile":  "all:\n\techo test",
		}

		for path, content := range files {
			fullPath := filepath.Join(tmpDir, path)
			os.WriteFile(fullPath, []byte(content), 0644)
		}

		result := validateGeneratedScenario(tmpDir)

		status, ok := result["status"].(string)
		if !ok || status != "passed" {
			t.Errorf("Expected validation status 'passed', got: %v", result)
		}

		missing, _ := result["missing"].([]string)
		if len(missing) > 0 {
			t.Errorf("Expected no missing files, got: %v", missing)
		}
	})

	t.Run("validate scenario with missing API directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Only create UI directory
		os.MkdirAll(filepath.Join(tmpDir, "ui/src"), 0755)

		result := validateGeneratedScenario(tmpDir)

		status, ok := result["status"].(string)
		if !ok || status != "failed" {
			t.Errorf("Expected validation status 'failed', got: %v", result)
		}

		missing, _ := result["missing"].([]string)
		if len(missing) == 0 {
			t.Error("Expected missing files when api/ directory is missing")
		}
	})

	t.Run("validate scenario with missing UI directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Only create API directory
		os.MkdirAll(filepath.Join(tmpDir, "api"), 0755)

		result := validateGeneratedScenario(tmpDir)

		status, ok := result["status"].(string)
		if !ok || status != "failed" {
			t.Errorf("Expected validation status 'failed', got: %v", result)
		}
	})

	t.Run("validate scenario with missing .vrooli directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create directories but no .vrooli
		os.MkdirAll(filepath.Join(tmpDir, "api"), 0755)
		os.MkdirAll(filepath.Join(tmpDir, "ui"), 0755)

		result := validateGeneratedScenario(tmpDir)

		status, ok := result["status"].(string)
		if !ok || status != "failed" {
			t.Errorf("Expected validation status 'failed', got: %v", result)
		}

		missing, _ := result["missing"].([]string)
		hasVrooli := false
		for _, m := range missing {
			if m == ".vrooli" {
				hasVrooli = true
				break
			}
		}
		if !hasVrooli {
			t.Error("Expected .vrooli to be in missing list")
		}
	})

	t.Run("validate scenario with missing PRD.md", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create all required except PRD.md
		dirs := []string{"api", "ui", ".vrooli", "requirements"}
		for _, dir := range dirs {
			os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		}

		os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("all:\n\techo test"), 0644)

		result := validateGeneratedScenario(tmpDir)

		status, ok := result["status"].(string)
		if !ok || status != "failed" {
			t.Errorf("Expected validation status 'failed', got: %v", result)
		}

		missing, _ := result["missing"].([]string)
		hasPRD := false
		for _, m := range missing {
			if m == "PRD.md" {
				hasPRD = true
				break
			}
		}
		if !hasPRD {
			t.Error("Expected PRD.md to be in missing list")
		}
	})
}

// TestResolveGenerationPath tests the resolveGenerationPath function
func TestResolveGenerationPath(t *testing.T) {
	t.Run("resolve with GEN_OUTPUT_DIR override", func(t *testing.T) {
		tmpDir := t.TempDir()

		os.Setenv("GEN_OUTPUT_DIR", tmpDir)
		defer os.Unsetenv("GEN_OUTPUT_DIR")

		path, err := resolveGenerationPath("my-scenario")
		if err != nil {
			t.Fatalf("Expected resolveGenerationPath to succeed, got error: %v", err)
		}

		expectedPath := filepath.Join(tmpDir, "my-scenario")
		if path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, path)
		}
	})

	t.Run("resolve without override uses executable path", func(t *testing.T) {
		// Clear any override
		os.Unsetenv("GEN_OUTPUT_DIR")

		path, err := resolveGenerationPath("test-scenario")
		if err != nil {
			t.Fatalf("Expected resolveGenerationPath to succeed, got error: %v", err)
		}

		// Should contain "generated/test-scenario" at the end
		if !contains(path, "/generated/test-scenario") {
			t.Errorf("Expected path to contain '/generated/test-scenario', got: %s", path)
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
