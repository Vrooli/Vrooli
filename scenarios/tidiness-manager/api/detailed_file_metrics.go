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
// This is a convenience wrapper that doesn't include complexity/duplication data
func CollectDetailedFileMetrics(scenarioPath string, files []string) ([]DetailedFileMetrics, error) {
	return CollectDetailedFileMetricsWithLangMetrics(scenarioPath, files, nil)
}

// CollectDetailedFileMetricsWithLangMetrics analyzes files with optional language metrics
// for complexity and duplication data
func CollectDetailedFileMetricsWithLangMetrics(scenarioPath string, files []string, langMetrics map[Language]*LanguageMetrics) ([]DetailedFileMetrics, error) {
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

	// Build per-file complexity map from language metrics
	// Maps file path -> (avgComplexity, maxComplexity) for functions in that file
	fileComplexity := buildFileComplexityMap(langMetrics)

	// Build per-file duplication map from language metrics
	// Maps file path -> percentage of lines that are duplicated
	fileDuplication := buildFileDuplicationMap(langMetrics, languages)

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

		// Add complexity metrics if available for this file
		if complexity, ok := fileComplexity[relPath]; ok {
			detailed.ComplexityAvg = &complexity.avg
			detailed.ComplexityMax = &complexity.max
		}

		// Add duplication percentage if available for this file
		if dupPct, ok := fileDuplication[relPath]; ok {
			detailed.DuplicationPct = &dupPct
		}

		results = append(results, detailed)
	}

	return results, nil
}

// fileComplexityInfo holds per-file complexity statistics
type fileComplexityInfo struct {
	avg float64
	max int
}

// buildFileComplexityMap extracts per-file complexity from language metrics
// by aggregating function-level complexity data
func buildFileComplexityMap(langMetrics map[Language]*LanguageMetrics) map[string]fileComplexityInfo {
	result := make(map[string]fileComplexityInfo)

	if langMetrics == nil {
		return result
	}

	// Group complexity by file
	fileComplexities := make(map[string][]int)

	for _, lm := range langMetrics {
		if lm.Complexity == nil || lm.Complexity.Skipped {
			continue
		}

		// Process high complexity files (these have detailed per-function data)
		for _, cf := range lm.Complexity.HighComplexityFiles {
			fileComplexities[cf.Path] = append(fileComplexities[cf.Path], cf.Complexity)
		}
	}

	// Calculate avg and max per file
	for path, complexities := range fileComplexities {
		if len(complexities) == 0 {
			continue
		}

		sum := 0
		max := 0
		for _, c := range complexities {
			sum += c
			if c > max {
				max = c
			}
		}

		result[path] = fileComplexityInfo{
			avg: float64(sum) / float64(len(complexities)),
			max: max,
		}
	}

	return result
}

// buildFileDuplicationMap calculates per-file duplication percentage
// by analyzing duplicate blocks that involve each file
func buildFileDuplicationMap(langMetrics map[Language]*LanguageMetrics, languages map[Language]*LanguageInfo) map[string]float64 {
	result := make(map[string]float64)

	if langMetrics == nil {
		return result
	}

	// Track duplicated lines per file
	fileDupLines := make(map[string]int)

	// Get file line counts for percentage calculation
	fileLines := make(map[string]int)
	for _, langInfo := range languages {
		for _, file := range langInfo.Files {
			// We don't have per-file line counts in LanguageInfo, so we'll estimate
			// from the duplicate block data or skip percentage calculation
			fileLines[file] = 0
		}
	}

	for _, lm := range langMetrics {
		if lm.Duplicates == nil || lm.Duplicates.Skipped {
			continue
		}

		// Process each duplicate block
		for _, block := range lm.Duplicates.DuplicateBlocks {
			for _, loc := range block.Files {
				lines := loc.EndLine - loc.StartLine + 1
				fileDupLines[loc.Path] += lines
				// Track total lines seen in this file from duplication info
				if fileLines[loc.Path] < loc.EndLine {
					fileLines[loc.Path] = loc.EndLine
				}
			}
		}
	}

	// Calculate percentage for each file with duplicates
	for path, dupLines := range fileDupLines {
		totalLines := fileLines[path]
		if totalLines > 0 && dupLines > 0 {
			pct := (float64(dupLines) / float64(totalLines)) * 100.0
			// Cap at 100% (duplicates can overlap)
			if pct > 100.0 {
				pct = 100.0
			}
			result[path] = pct
		}
	}

	return result
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
