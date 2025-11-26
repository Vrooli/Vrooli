package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TMPL-METADATA] Test loadTemplate with valid JSON
func TestLoadTemplate_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	template := Template{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Version:     "1.0.0",
		Metadata:    map[string]interface{}{"author": "Test Author"},
	}

	data, _ := json.Marshal(template)
	templatePath := filepath.Join(tmpDir, "test-template.json")
	os.WriteFile(templatePath, data, 0644)

	ts := &TemplateService{templatesDir: tmpDir}
	loaded, err := ts.loadTemplate(templatePath)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if loaded.ID != "test-template" {
		t.Errorf("Expected ID test-template, got %s", loaded.ID)
	}
	if loaded.Name != "Test Template" {
		t.Errorf("Expected Name 'Test Template', got %s", loaded.Name)
	}
}

// [REQ:TMPL-METADATA] Test loadTemplate with malformed JSON
func TestLoadTemplate_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	templatePath := filepath.Join(tmpDir, "bad-template.json")
	os.WriteFile(templatePath, []byte("{invalid json}"), 0644)

	ts := &TemplateService{templatesDir: tmpDir}
	_, err := ts.loadTemplate(templatePath)

	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}

// [REQ:TMPL-METADATA] Test loadTemplate with non-existent file
func TestLoadTemplate_NonExistentFile(t *testing.T) {
	ts := &TemplateService{templatesDir: "/nonexistent"}
	_, err := ts.loadTemplate("/nonexistent/path.json")

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// [REQ:TMPL-GENERATION] Test generationRoot with default behavior
func TestGenerationRoot_Default(t *testing.T) {
	os.Unsetenv("GEN_OUTPUT_DIR")

	root, err := generationRoot()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if root == "" {
		t.Error("Expected non-empty root")
	}
	if !filepath.IsAbs(root) {
		t.Errorf("Expected absolute path, got %s", root)
	}
}

// [REQ:TMPL-GENERATION] Test generationRoot with GEN_OUTPUT_DIR override
func TestGenerationRoot_WithOverride(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GEN_OUTPUT_DIR", tmpDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	root, err := generationRoot()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if root != tmpDir {
		t.Errorf("Expected root %s, got %s", tmpDir, root)
	}
}

// [REQ:TMPL-GENERATION] Test generationRoot with relative path override
func TestGenerationRoot_RelativePathOverride(t *testing.T) {
	os.Setenv("GEN_OUTPUT_DIR", "./relative/path")
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	root, err := generationRoot()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !filepath.IsAbs(root) {
		t.Errorf("Expected absolute path, got %s", root)
	}
}

// [REQ:TMPL-GENERATION] Test generationRoot with whitespace in override
func TestGenerationRoot_WhitespaceInOverride(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GEN_OUTPUT_DIR", "  "+tmpDir+"  ")
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	root, err := generationRoot()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if root != tmpDir {
		t.Errorf("Expected trimmed root %s, got %s", tmpDir, root)
	}
}

// [REQ:TMPL-PROVENANCE] Test writeTemplateProvenance with invalid path
func TestWriteTemplateProvenance_InvalidPath(t *testing.T) {
	template := &Template{
		ID:      "test-template",
		Version: "1.0.0",
	}

	// Try to write to a read-only directory
	err := writeTemplateProvenance("/nonexistent/readonly/path", template)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

// [REQ:TMPL-GENERATION] Test validateGeneratedScenario with complete structure
func TestValidateGeneratedScenario_Complete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create all required directories/files
	required := []string{"api", "ui", "requirements", ".vrooli", "Makefile", "PRD.md"}
	for _, name := range required {
		if name == "Makefile" || name == "PRD.md" {
			os.WriteFile(filepath.Join(tmpDir, name), []byte("test content"), 0644)
		} else {
			os.MkdirAll(filepath.Join(tmpDir, name), 0755)
		}
	}

	result := validateGeneratedScenario(tmpDir)

	if result["status"] != "passed" {
		t.Errorf("Expected status passed, got %v", result["status"])
	}

	if result["missing"] != nil {
		missing, ok := result["missing"].([]string)
		if ok && len(missing) > 0 {
			t.Errorf("Expected no missing items, got %v", missing)
		}
	}
}

// [REQ:TMPL-GENERATION] Test validateGeneratedScenario with missing items
func TestValidateGeneratedScenario_MissingItems(t *testing.T) {
	tmpDir := t.TempDir()

	// Only create some required items
	os.MkdirAll(filepath.Join(tmpDir, "api"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "ui"), 0755)

	result := validateGeneratedScenario(tmpDir)

	if result["status"] != "failed" {
		t.Errorf("Expected status failed, got %v", result["status"])
	}

	missing, ok := result["missing"].([]string)
	if !ok {
		t.Error("Expected missing to be a string array")
	} else if len(missing) == 0 {
		t.Error("Expected missing items, got none")
	}
}

// [REQ:TMPL-GENERATION] Test validateGeneratedScenario with empty directory
func TestValidateGeneratedScenario_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result := validateGeneratedScenario(tmpDir)

	if result["status"] != "failed" {
		t.Errorf("Expected status failed, got %v", result["status"])
	}

	missing, ok := result["missing"].([]string)
	if !ok {
		t.Error("Expected missing to be a string array")
	} else if len(missing) != 6 {
		t.Errorf("Expected 6 missing items, got %d", len(missing))
	}
}

// [REQ:TMPL-GENERATION] Test scaffoldScenario with TEMPLATE_PAYLOAD_DIR override
func TestScaffoldScenario_WithPayloadDirOverride(t *testing.T) {
	tmpDir := t.TempDir()
	payloadDir := filepath.Join(tmpDir, "payload")
	outputDir := filepath.Join(tmpDir, "output")

	// Create payload directory with complete required structure
	os.MkdirAll(filepath.Join(payloadDir, "api"), 0755)
	os.MkdirAll(filepath.Join(payloadDir, "ui", "src"), 0755)
	os.MkdirAll(filepath.Join(payloadDir, "requirements"), 0755)
	os.MkdirAll(filepath.Join(payloadDir, "initialization"), 0755)
	os.MkdirAll(filepath.Join(payloadDir, ".vrooli"), 0755)
	os.WriteFile(filepath.Join(payloadDir, "Makefile"), []byte("# Test makefile"), 0644)
	os.WriteFile(filepath.Join(payloadDir, "PRD.md"), []byte("# Test PRD"), 0644)
	os.WriteFile(filepath.Join(payloadDir, "ui", "src", "App.tsx"), []byte("// Test app"), 0644)

	os.Setenv("TEMPLATE_PAYLOAD_DIR", payloadDir)
	defer os.Unsetenv("TEMPLATE_PAYLOAD_DIR")

	err := scaffoldScenario(outputDir)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify structure was copied
	if _, err := os.Stat(filepath.Join(outputDir, "api")); os.IsNotExist(err) {
		t.Error("Expected api directory to be copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "PRD.md")); os.IsNotExist(err) {
		t.Error("Expected PRD.md to be copied")
	}
}

// [REQ:TMPL-GENERATION] Test NewTemplateService with custom directory
func TestNewTemplateService_WithCustomDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("TEMPLATES_DIR", tmpDir)
	defer os.Unsetenv("TEMPLATES_DIR")

	ts := NewTemplateService()

	if ts.templatesDir != tmpDir {
		t.Errorf("Expected templatesDir %s, got %s", tmpDir, ts.templatesDir)
	}
}

// [REQ:TMPL-GENERATION] Test NewTemplateService with default directory
func TestNewTemplateService_DefaultDirectory(t *testing.T) {
	os.Unsetenv("TEMPLATES_DIR")

	ts := NewTemplateService()

	if ts.templatesDir == "" {
		t.Error("Expected non-empty templatesDir")
	}
}

// [REQ:TMPL-GENERATION] Test copyDir with nested directories
func TestCopyDir_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	// Create nested directory structure
	os.MkdirAll(filepath.Join(srcDir, "level1", "level2", "level3"), 0755)
	os.WriteFile(filepath.Join(srcDir, "level1", "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "level1", "level2", "file2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(srcDir, "level1", "level2", "level3", "file3.txt"), []byte("content3"), 0644)

	err := copyDir(srcDir, destDir)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify nested structure was copied
	if _, err := os.Stat(filepath.Join(destDir, "level1", "file1.txt")); os.IsNotExist(err) {
		t.Error("Expected level1/file1.txt to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "level1", "level2", "file2.txt")); os.IsNotExist(err) {
		t.Error("Expected level1/level2/file2.txt to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "level1", "level2", "level3", "file3.txt")); os.IsNotExist(err) {
		t.Error("Expected level1/level2/level3/file3.txt to be copied")
	}
}

// [REQ:TMPL-GENERATION] Test copyDir with empty source directory
func TestCopyDir_EmptySourceDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")

	os.MkdirAll(srcDir, 0755)

	err := copyDir(srcDir, destDir)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify destination directory was created
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		t.Error("Expected destination directory to be created")
	}
}
