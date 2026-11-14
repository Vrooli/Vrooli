package main

import (
	"path/filepath"
	"strings"
)

// Language represents a detected programming language in the scenario.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguagePython     Language = "python"
	LanguageRust       Language = "rust"
)

// LanguageInfo groups Code Facts' already-filtered per-file metrics.
type LanguageInfo struct {
	Language   Language `json:"language"`
	Files      []string `json:"files"`
	FileCount  int      `json:"file_count"`
	TotalLines int      `json:"total_lines"`
	PrimaryDir string   `json:"primary_dir"`
}

func languagesFromFileMetrics(files []FileMetric) map[Language]*LanguageInfo {
	languages := make(map[Language]*LanguageInfo)
	for _, file := range files {
		language, ok := languageForCodeFacts(file.Language)
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

func languageForCodeFacts(language string) (Language, bool) {
	switch Language(strings.ToLower(language)) {
	case LanguageGo:
		return LanguageGo, true
	case LanguageTypeScript:
		return LanguageTypeScript, true
	case LanguageJavaScript:
		return LanguageJavaScript, true
	case LanguagePython:
		return LanguagePython, true
	case LanguageRust:
		return LanguageRust, true
	default:
		return "", false
	}
}
