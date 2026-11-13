package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// collectDetailedFileMetricsForTest supplies temporary fixture files through
// the same inventory boundary used in production. The filesystem reader lives
// only in tests so production cannot silently create a second metric scanner.
func collectDetailedFileMetricsForTest(root string, files []string, langMetrics map[Language]*LanguageMetrics) ([]DetailedFileMetrics, error) {
	inventory := make([]FileMetric, 0, len(files))
	for _, relPath := range files {
		language, known := languageForTestExtension(filepath.Ext(relPath))
		if known {
			metric, err := fileMetricForTest(filepath.Join(root, relPath), relPath, language)
			if err != nil {
				return nil, err
			}
			inventory = append(inventory, metric)
			continue
		}
		count, err := countLines(filepath.Join(root, relPath))
		if err != nil {
			return nil, err
		}
		inventory = append(inventory, FileMetric{Path: relPath, Lines: count, Extension: filepath.Ext(relPath)})
	}
	return CollectDetailedFileMetricsFromInventory(root, inventory, langMetrics)
}

// countLines is retained for assertions against temporary test fixtures only.
func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count, scanner.Err()
}
