package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Test helper: creates a directory and its test files
func createTestDir(t *testing.T, basePath string, relPath string, files map[string]string) string {
	t.Helper()
	dirPath := filepath.Join(basePath, relPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dirPath, err)
	}
	for filename, content := range files {
		createTestFile(t, filepath.Join(dirPath, filename), content)
	}
	return dirPath
}

// Test helper: creates a test file with content
func createTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
}

// Test helper: asserts language info matches expectations
func assertLanguageInfo(t *testing.T, languages map[Language]*LanguageInfo, lang Language, expectedFileCount int, expectedPrimaryDir string) {
	t.Helper()
	info, found := languages[lang]
	if !found {
		t.Errorf("%s language not detected", lang)
		return
	}
	if info.FileCount != expectedFileCount {
		t.Errorf("%s: expected %d files, got %d", lang, expectedFileCount, info.FileCount)
	}
	if expectedPrimaryDir != "" && info.PrimaryDir != expectedPrimaryDir {
		t.Errorf("%s: expected primary dir %q, got %q", lang, expectedPrimaryDir, info.PrimaryDir)
	}
}

func TestLanguageDetector_DetectLanguages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create api/ directory with Go files
	createTestDir(t, tmpDir, "api", map[string]string{
		"main.go":     "package main\n\nfunc main() {}\n",
		"handlers.go": "package main\n\nfunc handleRequest() {}\n",
	})

	// Create ui/src/ directory with TypeScript files
	createTestDir(t, tmpDir, "ui/src", map[string]string{
		"App.tsx":  "export default function App() {}\n",
		"utils.ts": "export const helper = () => {}\n",
	})

	// Create cli/ directory with Python files
	createTestDir(t, tmpDir, "cli", map[string]string{
		"main.py": "def main():\n    pass\n",
	})

	detector := NewLanguageDetector(tmpDir)
	languages, err := detector.DetectLanguages()
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Should detect Go, TypeScript, and Python
	if len(languages) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(languages))
	}

	// Verify all detected languages
	assertLanguageInfo(t, languages, LanguageGo, 2, "api")
	assertLanguageInfo(t, languages, LanguageTypeScript, 2, "ui")
	assertLanguageInfo(t, languages, LanguagePython, 1, "")
}

func TestLanguageDetector_HasLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	createTestDir(t, tmpDir, "api", map[string]string{
		"main.go": "package main\n",
	})

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
	createTestDir(t, tmpDir, "ui/src/node_modules", map[string]string{
		"ignored.ts": "// should be ignored\n",
	})

	// Create ui/src with actual TypeScript file
	createTestDir(t, tmpDir, "ui/src", map[string]string{
		"App.tsx": "export default function App() {}\n",
	})

	detector := NewLanguageDetector(tmpDir)
	languages, err := detector.DetectLanguages()
	if err != nil {
		t.Fatalf("DetectLanguages failed: %v", err)
	}

	// Should only count the App.tsx, not the file in node_modules
	assertLanguageInfo(t, languages, LanguageTypeScript, 1, "ui")
}
