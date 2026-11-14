package main

import (
	"context"
	"fmt"
	"path/filepath"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// DetailedFileMetrics contains comprehensive per-file metrics for refactor prioritization
type DetailedFileMetrics struct {
	FilePath              string   `json:"file_path"`
	Language              string   `json:"language"`
	FileExtension         string   `json:"file_extension"`
	LineCount             int      `json:"line_count"`
	TodoCount             int      `json:"todo_count"`
	FixmeCount            int      `json:"fixme_count"`
	HackCount             int      `json:"hack_count"`
	ImportCount           int      `json:"import_count"`
	FunctionCount         int      `json:"function_count"`
	CodeLines             int      `json:"code_lines"`
	CommentLines          int      `json:"comment_lines"`
	CommentRatio          float64  `json:"comment_to_code_ratio"`
	HasTestFile           bool     `json:"has_test_file"`
	ComplexityAvg         *float64 `json:"complexity_avg,omitempty"`
	ComplexityMax         *int     `json:"complexity_max,omitempty"`
	DuplicationPct        *float64 `json:"duplication_pct,omitempty"`
	AsAnyCount            int      `json:"as_any_count,omitempty"`
	AsTypeAssertionCount  int      `json:"as_type_assertion_count,omitempty"`
	TsIgnoreCount         int      `json:"ts_ignore_count,omitempty"`
	NonNullAssertionCount int      `json:"non_null_assertion_count,omitempty"`
}

// CollectDetailedFileMetricsFromInventory consumes the file inventory already
// supplied by Code Facts, avoiding a second path walk and line-count pass.
func CollectDetailedFileMetricsFromInventory(scenarioPath string, inventory []FileMetric, langMetrics map[Language]*LanguageMetrics) ([]DetailedFileMetrics, error) {
	files := make([]string, 0, len(inventory))
	lineCounts := make(map[string]int, len(inventory))
	metricsByPath := make(map[string]FileMetric, len(inventory))
	for _, metric := range inventory {
		files = append(files, metric.Path)
		lineCounts[metric.Path] = metric.Lines
		metricsByPath[metric.Path] = metric
	}
	return collectDetailedFileMetrics(scenarioPath, files, lineCounts, metricsByPath, langMetrics)
}

func collectDetailedFileMetrics(scenarioPath string, files []string, inventoryLines map[string]int, metricsByPath map[string]FileMetric, langMetrics map[Language]*LanguageMetrics) ([]DetailedFileMetrics, error) {
	if len(files) == 0 {
		return []DetailedFileMetrics{}, nil
	}

	// Code Facts owns file-language classification. Reuse that catalog result
	// instead of maintaining a second extension-to-language table here.
	fileLangMap := make(map[string]Language)
	languages := make(map[Language]*LanguageInfo)
	for _, file := range files {
		if metric, exists := metricsByPath[file]; exists {
			lang, ok := languageForCodeFacts(metric.Language)
			if !ok {
				continue
			}
			fileLangMap[file] = lang
			info := languages[lang]
			if info == nil {
				info = &LanguageInfo{Language: lang}
				languages[lang] = info
			}
			info.Files = append(info.Files, file)
			info.FileCount++
			info.TotalLines += inventoryLines[file]
		}
	}

	// Build per-file complexity map from language metrics
	// Maps file path -> (avgComplexity, maxComplexity) for functions in that file
	fileComplexity := buildFileComplexityMap(langMetrics)

	// Line counts come exclusively from the Code Facts inventory.
	fileLineCounts := inventoryLines

	// Build per-file duplication map from language metrics
	// Maps file path -> percentage of lines that are duplicated
	fileDuplication := buildFileDuplicationMap(langMetrics, languages, fileLineCounts)

	// Collect metrics per file
	results := make([]DetailedFileMetrics, 0, len(files))
	codeMetricsAnalyzer := NewCodeMetricsAnalyzer(scenarioPath)
	var graphReport *facts.CodeFactsReport
	for _, metrics := range langMetrics {
		if metrics != nil && metrics.facts != nil {
			graphReport = metrics.facts
			break
		}
	}
	graphMetrics := graphFileMetrics(graphReport)
	for _, relPath := range files {
		lang, hasLang := fileLangMap[relPath]
		if !hasLang {
			// Unknown language - skip detailed metrics, just store basic info
			lineCount := fileLineCounts[relPath]
			results = append(results, DetailedFileMetrics{
				FilePath:      relPath,
				Language:      "unknown",
				FileExtension: filepath.Ext(relPath),
				LineCount:     lineCount,
			})
			continue
		}

		metric, exists := metricsByPath[relPath]
		if !exists {
			continue
		}
		fileMetrics := codeMetricsFromFileMetric(metric)
		if graphMetric, ok := graphMetrics[relPath][lang]; ok && graphMetric.graphBacked {
			fileMetrics.ImportCount, fileMetrics.FunctionCount = graphMetric.imports, graphMetric.functions
		}

		// Build test file map for this language to check test coverage
		allFilesForLang := languages[lang].Files
		testFiles, sourceFiles := codeMetricsAnalyzer.buildTestFileMaps(allFilesForLang, lang)

		// If this file itself is a test file, mark hasTest as true
		// Otherwise, check if it has a corresponding test file
		hasTest := testFiles[relPath] || codeMetricsAnalyzer.hasTestFile(relPath, testFiles, sourceFiles, lang)

		detailed := DetailedFileMetrics{
			FilePath:              relPath,
			Language:              string(lang),
			FileExtension:         filepath.Ext(relPath),
			LineCount:             fileMetrics.CodeLines + fileMetrics.CommentLines,
			TodoCount:             fileMetrics.TodoCount,
			FixmeCount:            fileMetrics.FixmeCount,
			HackCount:             fileMetrics.HackCount,
			ImportCount:           fileMetrics.ImportCount,
			FunctionCount:         fileMetrics.FunctionCount,
			CodeLines:             fileMetrics.CodeLines,
			CommentLines:          fileMetrics.CommentLines,
			CommentRatio:          fileMetrics.CommentToCodeRatio,
			HasTestFile:           hasTest,
			AsAnyCount:            fileMetrics.AsAnyCount,
			AsTypeAssertionCount:  fileMetrics.AsTypeAssertionCount,
			TsIgnoreCount:         fileMetrics.TsIgnoreCount,
			NonNullAssertionCount: fileMetrics.NonNullAssertionCount,
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
// by using the FileComplexity map which contains ALL files, not just high-complexity ones
func buildFileComplexityMap(langMetrics map[Language]*LanguageMetrics) map[string]fileComplexityInfo {
	result := make(map[string]fileComplexityInfo)

	if langMetrics == nil {
		return result
	}

	for _, lm := range langMetrics {
		if lm.Complexity == nil || lm.Complexity.Skipped {
			continue
		}

		// Use the new FileComplexity map which has per-file stats for ALL files
		if lm.Complexity.FileComplexity != nil {
			for path, stats := range lm.Complexity.FileComplexity {
				result[path] = fileComplexityInfo{
					avg: stats.Avg,
					max: stats.Max,
				}
			}
		} else {
			// Fallback to old behavior for backwards compatibility
			// (in case older scan results don't have FileComplexity)
			fileComplexities := make(map[string][]int)
			for _, cf := range lm.Complexity.HighComplexityFiles {
				fileComplexities[cf.Path] = append(fileComplexities[cf.Path], cf.Complexity)
			}

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
		}
	}

	return result
}

// buildFileDuplicationMap calculates per-file duplication percentage
// by analyzing duplicate blocks that involve each file.
// fileLineCounts provides actual line counts for accurate percentage calculation.
func buildFileDuplicationMap(langMetrics map[Language]*LanguageMetrics, languages map[Language]*LanguageInfo, fileLineCounts map[string]int) map[string]float64 {
	result := make(map[string]float64)

	if langMetrics == nil {
		return result
	}

	// Track duplicated lines per file
	fileDupLines := make(map[string]int)

	// Get file line counts for percentage calculation
	// Use actual line counts when available
	fileLines := make(map[string]int)
	for path, count := range fileLineCounts {
		fileLines[path] = count
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
				// Only use EndLine as fallback if the file isn't in fileLineCounts
				if _, hasActual := fileLineCounts[loc.Path]; !hasActual {
					if fileLines[loc.Path] < loc.EndLine {
						fileLines[loc.Path] = loc.EndLine
					}
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
