package main

import (
	"fmt"
)

// IssueGeneratorConfig defines thresholds for issue generation
type IssueGeneratorConfig struct {
	LongFileThreshold      int     // Files with more lines than this generate issues (default: 500)
	LongFileThresholdTest  int     // Line count threshold for test files (default: 1250)
	HighComplexityMax      int     // Max complexity above this generates issues (default: 15)
	HighDuplicationPct     float64 // Duplication percentage above this generates issues (default: 10.0)
	HighDuplicationPctTest float64 // Duplication percentage threshold for test files (default: 30.0)
	HighTechDebtThreshold  int     // Total TODOs + FIXMEs + HACKs above this generates issues (default: 10)
	HighImportThreshold    int     // Import count above this generates coupling issues (default: 20)
}

// DefaultIssueGeneratorConfig returns sensible defaults for issue generation
func DefaultIssueGeneratorConfig() IssueGeneratorConfig {
	return IssueGeneratorConfig{
		LongFileThreshold:      500,
		LongFileThresholdTest:  1250,
		HighComplexityMax:      15,
		HighDuplicationPct:     10.0,
		HighDuplicationPctTest: 30.0,
		HighTechDebtThreshold:  10,
		HighImportThreshold:    20,
	}
}

// GenerateIssuesFromMetrics creates issues based on file metrics thresholds
// Categories: length, complexity, duplication, technical_debt, coupling
func GenerateIssuesFromMetrics(scenario string, metrics []DetailedFileMetrics, config IssueGeneratorConfig) []Issue {
	var issues []Issue

	for _, m := range metrics {
		// Length issues (long files) - use higher threshold for test files
		isTestFile := IsTestFilePath(m.FilePath)
		lengthThreshold := config.LongFileThreshold
		if isTestFile && config.LongFileThresholdTest > 0 {
			lengthThreshold = config.LongFileThresholdTest
		}
		if m.LineCount > lengthThreshold {
			issues = append(issues, Issue{
				Scenario: scenario,
				File:     m.FilePath,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("File has %d lines, exceeds threshold of %d lines", m.LineCount, lengthThreshold),
				Severity: severityForLineCount(m.LineCount, lengthThreshold),
				Tool:     "tidiness-manager",
				Category: "length",
			})
		}

		// Complexity issues (high cyclomatic complexity)
		if m.ComplexityMax != nil && *m.ComplexityMax > config.HighComplexityMax {
			issues = append(issues, Issue{
				Scenario: scenario,
				File:     m.FilePath,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("File has max cyclomatic complexity of %d, exceeds threshold of %d", *m.ComplexityMax, config.HighComplexityMax),
				Severity: severityForComplexity(*m.ComplexityMax, config.HighComplexityMax),
				Tool:     "gocyclo",
				Category: "complexity",
			})
		}

		// Duplication issues - use higher threshold for test files
		if m.DuplicationPct != nil {
			dupThreshold := config.HighDuplicationPct
			if isTestFile && config.HighDuplicationPctTest > 0 {
				dupThreshold = config.HighDuplicationPctTest
			}
			if *m.DuplicationPct > dupThreshold {
				issues = append(issues, Issue{
					Scenario: scenario,
					File:     m.FilePath,
					Line:     1,
					Column:   1,
					Message:  fmt.Sprintf("File has %.1f%% duplicated code, exceeds threshold of %.1f%%", *m.DuplicationPct, dupThreshold),
					Severity: severityForDuplication(*m.DuplicationPct, dupThreshold),
					Tool:     "dupl",
					Category: "duplication",
				})
			}
		}

		// Technical debt issues (TODOs, FIXMEs, HACKs)
		techDebtCount := m.TodoCount + m.FixmeCount + m.HackCount
		if techDebtCount > config.HighTechDebtThreshold {
			issues = append(issues, Issue{
				Scenario: scenario,
				File:     m.FilePath,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("File has %d tech debt markers (%d TODOs, %d FIXMEs, %d HACKs), exceeds threshold of %d", techDebtCount, m.TodoCount, m.FixmeCount, m.HackCount, config.HighTechDebtThreshold),
				Severity: severityForTechDebt(techDebtCount, config.HighTechDebtThreshold),
				Tool:     "tidiness-manager",
				Category: "technical_debt",
			})
		}

		// Coupling issues (excessive imports)
		if m.ImportCount > config.HighImportThreshold {
			issues = append(issues, Issue{
				Scenario: scenario,
				File:     m.FilePath,
				Line:     1,
				Column:   1,
				Message:  fmt.Sprintf("File has %d imports, exceeds threshold of %d (high coupling)", m.ImportCount, config.HighImportThreshold),
				Severity: severityForCoupling(m.ImportCount, config.HighImportThreshold),
				Tool:     "tidiness-manager",
				Category: "coupling",
			})
		}

		// Dangerous TS/JS pattern counts remain available as raw metrics for
		// historical score compatibility, but Quality Health owns static-quality
		// enforcement and finding production.
	}

	return issues
}

// severityForLineCount returns severity based on how much line count exceeds threshold
func severityForLineCount(lines, threshold int) string {
	ratio := float64(lines) / float64(threshold)
	if ratio > 3.0 {
		return "high"
	} else if ratio > 2.0 {
		return "medium"
	}
	return "low"
}

// severityForComplexity returns severity based on complexity level
func severityForComplexity(complexity, threshold int) string {
	if complexity > threshold*2 {
		return "high"
	} else if complexity > threshold+5 {
		return "medium"
	}
	return "low"
}

// severityForDuplication returns severity based on duplication percentage
func severityForDuplication(pct, threshold float64) string {
	if pct > threshold*3 {
		return "high"
	} else if pct > threshold*2 {
		return "medium"
	}
	return "low"
}

// severityForTechDebt returns severity based on tech debt count
func severityForTechDebt(count, threshold int) string {
	if count > threshold*3 {
		return "high"
	} else if count > threshold*2 {
		return "medium"
	}
	return "low"
}

// severityForCoupling returns severity based on import count
func severityForCoupling(imports, threshold int) string {
	if imports > threshold*2 {
		return "high"
	} else if imports > threshold+10 {
		return "medium"
	}
	return "low"
}
