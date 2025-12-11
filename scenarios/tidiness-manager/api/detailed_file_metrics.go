package main

import (
	"context"
	"fmt"
	"path/filepath"
)

// DetailedFileMetrics contains comprehensive per-file metrics for refactor prioritization
type DetailedFileMetrics struct {
	FilePath       string   `json:"file_path"`
	Language       string   `json:"language"`
	FileExtension  string   `json:"file_extension"`
	LineCount      int      `json:"line_count"`
	TodoCount      int      `json:"todo_count"`
	FixmeCount     int      `json:"fixme_count"`
	HackCount      int      `json:"hack_count"`
	ImportCount    int      `json:"import_count"`
	FunctionCount  int      `json:"function_count"`
	CodeLines      int      `json:"code_lines"`
	CommentLines   int      `json:"comment_lines"`
	CommentRatio   float64  `json:"comment_to_code_ratio"`
	HasTestFile    bool     `json:"has_test_file"`
	ComplexityAvg  *float64 `json:"complexity_avg,omitempty"`
	ComplexityMax  *int     `json:"complexity_max,omitempty"`
	DuplicationPct *float64 `json:"duplication_pct,omitempty"`
}

// CollectDetailedFileMetrics analyzes files and returns detailed per-file metrics
func CollectDetailedFileMetrics(scenarioPath string, files []string) ([]DetailedFileMetrics, error) {
	if len(files) == 0 {
		return []DetailedFileMetrics{}, nil
	}

	// Detect languages first
	detector := NewLanguageDetector(scenarioPath)
	languages, err := detector.DetectLanguages()
	if err != nil {
		return nil, fmt.Errorf("language detection failed: %w", err)
	}

	// Build file->language mapping
	fileLangMap := make(map[string]Language)
	for lang, langInfo := range languages {
		for _, file := range langInfo.Files {
			fileLangMap[file] = lang
		}
	}

	// Collect metrics per file
	results := make([]DetailedFileMetrics, 0, len(files))
	codeMetricsAnalyzer := NewCodeMetricsAnalyzer(scenarioPath)

	for _, relPath := range files {
		lang, hasLang := fileLangMap[relPath]
		if !hasLang {
			// Unknown language - skip detailed metrics, just store basic info
			lineCount, _ := countLines(filepath.Join(scenarioPath, relPath))
			results = append(results, DetailedFileMetrics{
				FilePath:      relPath,
				Language:      "unknown",
				FileExtension: filepath.Ext(relPath),
				LineCount:     lineCount,
			})
			continue
		}

		// Analyze individual file
		absPath := filepath.Join(scenarioPath, relPath)
		fileMetrics, err := codeMetricsAnalyzer.analyzeFile(absPath, lang)
		if err != nil {
			// Skip files we can't analyze
			continue
		}

		// Build test file map for this language to check test coverage
		allFilesForLang := languages[lang].Files
		testFiles := codeMetricsAnalyzer.buildTestFileMap(allFilesForLang, lang)

		// If this file itself is a test file, mark hasTest as true
		// Otherwise, check if it has a corresponding test file
		hasTest := testFiles[relPath] || codeMetricsAnalyzer.hasTestFile(relPath, testFiles, lang)

		detailed := DetailedFileMetrics{
			FilePath:      relPath,
			Language:      string(lang),
			FileExtension: filepath.Ext(relPath),
			LineCount:     fileMetrics.CodeLines + fileMetrics.CommentLines,
			TodoCount:     fileMetrics.TodoCount,
			FixmeCount:    fileMetrics.FixmeCount,
			HackCount:     fileMetrics.HackCount,
			ImportCount:   fileMetrics.ImportCount,
			FunctionCount: fileMetrics.FunctionCount,
			CodeLines:     fileMetrics.CodeLines,
			CommentLines:  fileMetrics.CommentLines,
			CommentRatio:  fileMetrics.CommentToCodeRatio,
			HasTestFile:   hasTest,
		}

		results = append(results, detailed)
	}

	return results, nil
}

// PersistDetailedFileMetrics stores detailed file metrics in the database
func (s *Server) persistDetailedFileMetrics(ctx context.Context, scenario string, metrics []DetailedFileMetrics) error {
	if len(metrics) == 0 {
		return nil
	}
	if s.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return s.store.PersistDetailedFileMetrics(ctx, scenario, metrics)
}
