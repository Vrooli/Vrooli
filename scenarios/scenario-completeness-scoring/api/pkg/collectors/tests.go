package collectors

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		Failing: 0,
		LastRun: "",
	}
}

// loadPhaseBasedResults loads from coverage/phase-results/*.json
func loadPhaseBasedResults(scenarioRoot string) *TestResults {
	phaseDir := filepath.Join(scenarioRoot, "coverage", "phase-results")

	entries, err := os.ReadDir(phaseDir)
	if err != nil {
		return nil
	}

	totalPassing := 0
	totalFailing := 0
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

		var phaseData struct {
			Requirements []struct {
				Status string `json:"status"`
			} `json:"requirements"`
			UpdatedAt string `json:"updated_at"`
		}

		if err := json.Unmarshal(data, &phaseData); err != nil {
			continue
		}

		if len(phaseData.Requirements) > 0 {
			hasResults = true
			for _, req := range phaseData.Requirements {
				switch req.Status {
				case "passed":
					totalPassing++
				case "failed":
					totalFailing++
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
	}

	if !hasResults {
		return nil
	}

	lastRun := ""
	if !latestTimestamp.IsZero() {
		lastRun = latestTimestamp.Format(time.RFC3339)
	}

	return &TestResults{
		Total:   totalPassing + totalFailing,
		Passing: totalPassing,
		Failing: totalFailing,
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
		Failing: results.Failed,
		LastRun: results.Timestamp,
	}
}
