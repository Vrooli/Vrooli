package content

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateJSON_AllValid(t *testing.T) {
	root := t.TempDir()

	// Create valid JSON files
	writeFileJSON(t, filepath.Join(root, "config.json"), `{"key": "value"}`)
	writeFileJSON(t, filepath.Join(root, "data.json"), `[1, 2, 3]`)
	mustMkdirJSON(t, filepath.Join(root, "sub"))
	writeFileJSON(t, filepath.Join(root, "sub", "nested.json"), `{"nested": true}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ItemsChecked != 3 {
		t.Errorf("expected 3 items checked, got %d", result.ItemsChecked)
	}
}

func TestValidateJSON_InvalidJSON(t *testing.T) {
	root := t.TempDir()

	// Create one valid and one invalid JSON file
	writeFileJSON(t, filepath.Join(root, "valid.json"), `{"ok": true}`)
	writeFileJSON(t, filepath.Join(root, "invalid.json"), `{"broken":`)

	result := ValidateJSON(root, io.Discard)
	if result.Success {
		t.Fatal("expected failure for invalid JSON")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidateJSON_SkipsNodeModules(t *testing.T) {
	root := t.TempDir()

	// Create valid JSON in root
	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in node_modules (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, "node_modules"))
	writeFileJSON(t, filepath.Join(root, "node_modules", "bad.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (node_modules should be skipped), got error: %v", result.Error)
	}
	if result.ItemsChecked != 1 {
		t.Errorf("expected 1 item checked (skipping node_modules), got %d", result.ItemsChecked)
	}
}

func TestValidateJSON_SkipsDist(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in dist (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, "dist"))
	writeFileJSON(t, filepath.Join(root, "dist", "bad.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (dist should be skipped), got error: %v", result.Error)
	}
}

func TestValidateJSON_SkipsBuild(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in build (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, "build"))
	writeFileJSON(t, filepath.Join(root, "build", "bad.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (build should be skipped), got error: %v", result.Error)
	}
}

func TestValidateJSON_SkipsCoverage(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in coverage (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, "coverage"))
	writeFileJSON(t, filepath.Join(root, "coverage", "bad.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (coverage should be skipped), got error: %v", result.Error)
	}
}

func TestValidateJSON_SkipsGit(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in .git (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, ".git"))
	writeFileJSON(t, filepath.Join(root, ".git", "config.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (.git should be skipped), got error: %v", result.Error)
	}
}

func TestValidateJSON_SkipsArtifacts(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "config.json"), `{"ok": true}`)

	// Create invalid JSON in artifacts (should be skipped)
	mustMkdirJSON(t, filepath.Join(root, "artifacts"))
	writeFileJSON(t, filepath.Join(root, "artifacts", "report.json"), `{broken}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success (artifacts should be skipped), got error: %v", result.Error)
	}
}

func TestValidateJSON_NoJSONFiles(t *testing.T) {
	root := t.TempDir()

	// Create non-JSON files only
	writeFileJSON(t, filepath.Join(root, "README.md"), "# Readme")
	writeFileJSON(t, filepath.Join(root, "main.go"), "package main")

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success with no JSON files, got error: %v", result.Error)
	}
	if result.ItemsChecked != 0 {
		t.Errorf("expected 0 items checked, got %d", result.ItemsChecked)
	}
}

func TestValidateJSON_CaseInsensitiveExtension(t *testing.T) {
	root := t.TempDir()

	// Create JSON files with different case extensions
	writeFileJSON(t, filepath.Join(root, "lower.json"), `{"case": "lower"}`)
	writeFileJSON(t, filepath.Join(root, "upper.JSON"), `{"case": "upper"}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ItemsChecked != 2 {
		t.Errorf("expected 2 items checked (case-insensitive), got %d", result.ItemsChecked)
	}
}

func TestValidateJSON_EmptyObject(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "empty.json"), `{}`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success for empty object, got error: %v", result.Error)
	}
}

func TestValidateJSON_EmptyArray(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "empty.json"), `[]`)

	result := ValidateJSON(root, io.Discard)
	if !result.Success {
		t.Fatalf("expected success for empty array, got error: %v", result.Error)
	}
}

func TestValidateJSONDetailed_Success(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "a.json"), `{}`)
	writeFileJSON(t, filepath.Join(root, "b.json"), `{}`)

	result, err := ValidateJSONDetailed(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalFiles != 2 {
		t.Errorf("expected 2 total files, got %d", result.TotalFiles)
	}
	if result.ValidFiles != 2 {
		t.Errorf("expected 2 valid files, got %d", result.ValidFiles)
	}
	if len(result.InvalidFiles) != 0 {
		t.Errorf("expected no invalid files, got %v", result.InvalidFiles)
	}
}

func TestValidateJSONDetailed_WithInvalid(t *testing.T) {
	root := t.TempDir()

	writeFileJSON(t, filepath.Join(root, "valid.json"), `{}`)
	writeFileJSON(t, filepath.Join(root, "invalid.json"), `{bad}`)

	result, err := ValidateJSONDetailed(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalFiles != 2 {
		t.Errorf("expected 2 total files, got %d", result.TotalFiles)
	}
	if result.ValidFiles != 1 {
		t.Errorf("expected 1 valid file, got %d", result.ValidFiles)
	}
	if len(result.InvalidFiles) != 1 {
		t.Errorf("expected 1 invalid file, got %d", len(result.InvalidFiles))
	}
	if len(result.InvalidFiles) > 0 && result.InvalidFiles[0] != "invalid.json" {
		t.Errorf("expected invalid.json, got %s", result.InvalidFiles[0])
	}
}

func TestJSONValidator_Interface(t *testing.T) {
	root := t.TempDir()
	writeFileJSON(t, filepath.Join(root, "config.json"), `{"test": true}`)

	v := NewJSONValidator(root, io.Discard)
	result := v.Validate()

	if !result.Success {
		t.Errorf("Validate() failed: %v", result.Error)
	}
	if result.ItemsChecked != 1 {
		t.Errorf("expected 1 item checked, got %d", result.ItemsChecked)
	}
}

func TestSkipDirs_Contents(t *testing.T) {
	// Verify expected skip directories are configured
	expectedSkips := []string{".git", "node_modules", "dist", "build", "coverage", "artifacts"}

	for _, dir := range expectedSkips {
		if _, ok := SkipDirs[dir]; !ok {
			t.Errorf("expected %q to be in SkipDirs", dir)
		}
	}
}

// Test helpers

func mustMkdirJSON(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFileJSON(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if dir != "." {
		mustMkdirJSON(t, dir)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
