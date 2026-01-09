package nodejs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// coverageSummary represents the structure of coverage-summary.json.
type coverageSummary struct {
	Total struct {
		Statements struct {
			Pct float64 `json:"pct"`
		} `json:"statements"`
	} `json:"total"`
}

// DetectCoverage extracts test coverage from available sources.
// It first tries coverage-summary.json, then falls back to parsing output.
func DetectCoverage(dir string, output string) string {
	// Try coverage-summary.json first
	if pct := extractFromSummaryJSON(dir); pct != "" {
		return pct
	}

	// Fall back to parsing output
	return extractFromOutput(output)
}

// extractFromSummaryJSON reads coverage from coverage/coverage-summary.json.
func extractFromSummaryJSON(dir string) string {
	summaryPath := filepath.Join(dir, "coverage", "coverage-summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return ""
	}

	var summary coverageSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return ""
	}

	if summary.Total.Statements.Pct > 0 {
		return fmt.Sprintf("%.2f", summary.Total.Statements.Pct)
	}
	return ""
}

// extractFromOutput parses coverage percentage from test output.
func extractFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "%") {
			if pct := extractPercentage(line); pct != "" {
				return pct
			}
		}
	}
	return ""
}

// extractPercentage finds a percentage value in a line of text.
func extractPercentage(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	for _, token := range strings.Fields(line) {
		if strings.HasSuffix(token, "%") {
			return strings.TrimSuffix(token, "%")
		}
	}
	return ""
}
