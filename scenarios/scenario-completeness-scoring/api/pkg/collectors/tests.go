package collectors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// loadTestResults loads test results from phase outputs
// [REQ:SCS-CORE-001A] Test results loading
func loadTestResults(scenarioRoot string) TestResults {
	// Try phase-based results first (preferred format)
	phaseResults := loadPhaseBasedResults(scenarioRoot)
	if phaseResults != nil {
		return *phaseResults
	}

	// Fallback to single-file results
	singleResults := loadSingleFileResults(scenarioRoot)
	if singleResults != nil {
		return *singleResults
	}

	// No test results found
	return TestResults{
		Total:   0,
		Passing: 0,
		LastRun: "",
	}
}

// summaryPattern matches "N passed, M failed" or "N passed, M failed, K skipped"
var summaryPattern = regexp.MustCompile(`(\d+)\s+passed(?:,\s*(\d+)\s+failed)?`)

// loadPhaseBasedResults loads from coverage/phase-results/*.json
func loadPhaseBasedResults(scenarioRoot string) *TestResults {
	phaseDir := filepath.Join(scenarioRoot, "coverage", "phase-results")

	entries, err := os.ReadDir(phaseDir)
	if err != nil {
		return nil
	}

	totalPassing := 0
	totalTests := 0
	var latestTimestamp time.Time
	hasResults := false

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(phaseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Try the requirements array format first
		var phaseData struct {
			Requirements []struct {
				Status string `json:"status"`
			} `json:"requirements"`
			UpdatedAt string `json:"updated_at"`
			// Ecosystem-manager format fields
			Summary string `json:"summary"`
		}

		if err := json.Unmarshal(data, &phaseData); err != nil {
			continue
		}

		// Process requirements array format (preferred)
		if len(phaseData.Requirements) > 0 {
			hasResults = true
			for _, req := range phaseData.Requirements {
				// Only count actual test runs (passed or failed)
				// Skip "skipped", "not_run", "unknown" - these aren't actual test executions
				if req.Status == "passed" {
					totalTests++
					totalPassing++
				} else if req.Status == "failed" {
					totalTests++
				}
			}
		} else if phaseData.Summary != "" {
			// Fallback: parse ecosystem-manager summary format
			// e.g., "2 passed, 1 failed, 1 skipped"
			matches := summaryPattern.FindStringSubmatch(phaseData.Summary)
			if len(matches) >= 2 {
				hasResults = true
				if passed, err := strconv.Atoi(matches[1]); err == nil {
					totalPassing += passed
					totalTests += passed
				}
				if len(matches) >= 3 && matches[2] != "" {
					if failed, err := strconv.Atoi(matches[2]); err == nil {
						totalTests += failed
					}
				}
			}
		}

		if phaseData.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, phaseData.UpdatedAt); err == nil {
				if t.After(latestTimestamp) {
					latestTimestamp = t
				}
			}
		}
	}

	if !hasResults {
		return nil
	}

	lastRun := ""
	if !latestTimestamp.IsZero() {
		lastRun = latestTimestamp.Format(time.RFC3339)
	}

	return &TestResults{
		Total:   totalTests,
		Passing: totalPassing,
		LastRun: lastRun,
	}
}

// loadSingleFileResults loads from coverage/test-results.json
func loadSingleFileResults(scenarioRoot string) *TestResults {
	resultsPath := filepath.Join(scenarioRoot, "coverage", "test-results.json")

	data, err := os.ReadFile(resultsPath)
	if err != nil {
		return nil
	}

	var results struct {
		Passed    int    `json:"passed"`
		Failed    int    `json:"failed"`
		Timestamp string `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &results); err != nil {
		return nil
	}

	return &TestResults{
		Total:   results.Passed + results.Failed,
		Passing: results.Passed,
		LastRun: results.Timestamp,
	}
}
