package main

import (
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

// DetectLanguages classifies the same filtered inventory used for file metrics.
func (ld *LanguageDetector) DetectLanguages() (map[Language]*LanguageInfo, error) {
	files, err := NewLightScanner(ld.scenarioPath, 0).collectFileMetrics()
	if err != nil {
		return nil, err
	}
	return languagesFromFileMetrics(files), nil
}

// Language grouping never chooses directories or walks the tree independently.
// A scan passes its already-filtered inventory so every metric has one scope.
func languagesFromFileMetrics(files []FileMetric) map[Language]*LanguageInfo {
	languages := make(map[Language]*LanguageInfo)
	extensions := map[string]Language{
		".go": LanguageGo, ".ts": LanguageTypeScript, ".tsx": LanguageTypeScript,
		".js": LanguageJavaScript, ".jsx": LanguageJavaScript,
		".py": LanguagePython, ".rs": LanguageRust,
	}
	for _, file := range files {
		language, ok := extensions[file.Extension]
		if !ok {
			continue
		}
		info := languages[language]
		if info == nil {
			primaryDir := "."
			if first, _, nested := strings.Cut(filepath.ToSlash(file.Path), "/"); nested {
				primaryDir = first
			}
			info = &LanguageInfo{Language: language, PrimaryDir: primaryDir}
			languages[language] = info
		}
		info.Files = append(info.Files, file.Path)
		info.FileCount++
		info.TotalLines += file.Lines
	}
	return languages
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
