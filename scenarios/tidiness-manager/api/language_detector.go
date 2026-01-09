package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Language represents a detected programming language in the scenario
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguagePython     Language = "python"
	LanguageRust       Language = "rust"
)

// LanguageInfo contains files and metadata for a detected language
type LanguageInfo struct {
	Language   Language `json:"language"`
	Files      []string `json:"files"`
	FileCount  int      `json:"file_count"`
	TotalLines int      `json:"total_lines"`
	PrimaryDir string   `json:"primary_dir"` // api, ui, cli, etc.
}

// LanguageDetector scans a scenario and identifies programming languages present
type LanguageDetector struct {
	scenarioPath string
}

// NewLanguageDetector creates a detector for the specified scenario
func NewLanguageDetector(scenarioPath string) *LanguageDetector {
	return &LanguageDetector{
		scenarioPath: scenarioPath,
	}
}

// DetectLanguages walks the scenario directory and identifies all languages
func (ld *LanguageDetector) DetectLanguages() (map[Language]*LanguageInfo, error) {
	languages := make(map[Language]*LanguageInfo)

	// Directories to scan for source code
	scanDirs := []struct {
		path string
		name string
	}{
		{filepath.Join(ld.scenarioPath, "api"), "api"},
		{filepath.Join(ld.scenarioPath, "ui", "src"), "ui"},
		{filepath.Join(ld.scenarioPath, "cli"), "cli"},
	}

	// Extension to language mapping
	extMap := map[string]Language{
		".go":  LanguageGo,
		".ts":  LanguageTypeScript,
		".tsx": LanguageTypeScript,
		".js":  LanguageJavaScript,
		".jsx": LanguageJavaScript,
		".py":  LanguagePython,
		".rs":  LanguageRust,
	}

	for _, scanDir := range scanDirs {
		if _, err := os.Stat(scanDir.path); os.IsNotExist(err) {
			continue // Skip if directory doesn't exist
		}

		err := filepath.Walk(scanDir.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			if info.IsDir() {
				// Skip node_modules, vendor, and hidden directories
				dirName := info.Name()
				if dirName == "node_modules" || dirName == "vendor" || strings.HasPrefix(dirName, ".") {
					return filepath.SkipDir
				}
				return nil
			}

			ext := filepath.Ext(path)
			lang, exists := extMap[ext]
			if !exists {
				return nil // Not a language we track
			}

			// Count lines in file
			lines, err := countLines(path)
			if err != nil {
				return nil // Skip files we can't read
			}

			// Get relative path from scenario root
			relPath, _ := filepath.Rel(ld.scenarioPath, path)

			// Initialize language info if first occurrence
			if languages[lang] == nil {
				languages[lang] = &LanguageInfo{
					Language:   lang,
					Files:      []string{},
					PrimaryDir: scanDir.name,
				}
			}

			// Add file to language info
			languages[lang].Files = append(languages[lang].Files, relPath)
			languages[lang].FileCount++
			languages[lang].TotalLines += lines

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return languages, nil
}

// GetFilesByLanguage returns all files for a specific language
func (ld *LanguageDetector) GetFilesByLanguage(lang Language) ([]string, error) {
	languages, err := ld.DetectLanguages()
	if err != nil {
		return nil, err
	}

	if info, exists := languages[lang]; exists {
		return info.Files, nil
	}

	return []string{}, nil
}

// HasLanguage checks if a scenario contains a specific language
func (ld *LanguageDetector) HasLanguage(lang Language) (bool, error) {
	languages, err := ld.DetectLanguages()
	if err != nil {
		return false, err
	}

	_, exists := languages[lang]
	return exists, nil
}
