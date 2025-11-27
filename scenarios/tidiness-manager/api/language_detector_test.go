package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageDetector_DetectLanguages(t *testing.T) {
	// Create temporary scenario structure
	tmpDir := t.TempDir()

	// Create api/ directory with Go files
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(apiDir, "main.go"), "package main\n\nfunc main() {}\n")
	createTestFile(t, filepath.Join(apiDir, "handlers.go"), "package main\n\nfunc handleRequest() {}\n")

	// Create ui/src/ directory with TypeScript files
	uiSrcDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(uiSrcDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(uiSrcDir, "App.tsx"), "export default function App() {}\n")
	createTestFile(t, filepath.Join(uiSrcDir, "utils.ts"), "export const helper = () => {}\n")

	// Create cli/ directory with Python files
	cliDir := filepath.Join(tmpDir, "cli")
	if err := os.MkdirAll(cliDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(cliDir, "main.py"), "def main():\n    pass\n")

	detector := NewLanguageDetector(tmpDir)
	languages, err := detector.DetectLanguages()
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Should detect Go, TypeScript, and Python
	if len(languages) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(languages))
	}

	// Check Go detection
	goInfo, hasGo := languages[LanguageGo]
	if !hasGo {
		t.Error("Go language not detected")
	} else {
		if goInfo.FileCount != 2 {
			t.Errorf("Expected 2 Go files, got %d", goInfo.FileCount)
		}
		if goInfo.PrimaryDir != "api" {
			t.Errorf("Expected Go primary dir 'api', got '%s'", goInfo.PrimaryDir)
		}
	}

	// Check TypeScript detection
	tsInfo, hasTS := languages[LanguageTypeScript]
	if !hasTS {
		t.Error("TypeScript language not detected")
	} else {
		if tsInfo.FileCount != 2 {
			t.Errorf("Expected 2 TypeScript files, got %d", tsInfo.FileCount)
		}
		if tsInfo.PrimaryDir != "ui" {
			t.Errorf("Expected TypeScript primary dir 'ui', got '%s'", tsInfo.PrimaryDir)
		}
	}

	// Check Python detection
	pyInfo, hasPy := languages[LanguagePython]
	if !hasPy {
		t.Error("Python language not detected")
	} else {
		if pyInfo.FileCount != 1 {
			t.Errorf("Expected 1 Python file, got %d", pyInfo.FileCount)
		}
	}
}

func TestLanguageDetector_HasLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(apiDir, "main.go"), "package main\n")

	detector := NewLanguageDetector(tmpDir)

	hasGo, err := detector.HasLanguage(LanguageGo)
	if err != nil {
		t.Fatalf("HasLanguage failed: %v", err)
	}
	if !hasGo {
		t.Error("Should have detected Go")
	}

	hasRust, err := detector.HasLanguage(LanguageRust)
	if err != nil {
		t.Fatalf("HasLanguage failed: %v", err)
	}
	if hasRust {
		t.Error("Should not have detected Rust")
	}
}

func TestLanguageDetector_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create ui/src/node_modules with TypeScript files (should be ignored)
	nodeModulesDir := filepath.Join(tmpDir, "ui", "src", "node_modules")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(nodeModulesDir, "ignored.ts"), "// should be ignored\n")

	// Create ui/src with actual TypeScript file
	uiSrcDir := filepath.Join(tmpDir, "ui", "src")
	createTestFile(t, filepath.Join(uiSrcDir, "App.tsx"), "export default function App() {}\n")

	detector := NewLanguageDetector(tmpDir)
	languages, err := detector.DetectLanguages()
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	tsInfo, hasTS := languages[LanguageTypeScript]
	if !hasTS {
		t.Fatal("TypeScript not detected")
	}

	// Should only count the App.tsx, not the file in node_modules
	if tsInfo.FileCount != 1 {
		t.Errorf("Expected 1 TypeScript file (node_modules ignored), got %d", tsInfo.FileCount)
	}
}

func createTestFile(t *testing.T, path string, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
}
