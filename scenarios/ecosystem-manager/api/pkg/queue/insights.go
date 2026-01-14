package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/insights"
	"github.com/google/uuid"
)

// getInsightDir returns the directory for a task's insight reports.
// Delegates to InsightManager when available.
func (qp *Processor) getInsightDir(taskID string) string {
	if qp.insightManager != nil {
		return qp.insightManager.getInsightDir(taskID)
	}
	return filepath.Join(qp.taskLogsDir, taskID, "insights")
}

// getInsightReportDir returns the directory for a specific insight report.
// Delegates to InsightManager when available.
func (qp *Processor) getInsightReportDir(taskID, reportID string) string {
	if qp.insightManager != nil {
		return qp.insightManager.getInsightReportDir(taskID, reportID)
	}
	return filepath.Join(qp.getInsightDir(taskID), reportID)
}

// SaveInsightReport persists an insight report to disk.
// Delegates to InsightManager when available.
func (qp *Processor) SaveInsightReport(report insights.InsightReport) error {
	if qp.insightManager != nil {
		return qp.insightManager.SaveInsightReport(report)
	}
	// Fallback implementation
	// Generate ID if not set
	if report.ID == "" {
		report.ID = uuid.New().String()
	}

	// Set generated time if not set
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}

	// Create report directory
	reportDir := qp.getInsightReportDir(report.TaskID, report.ID)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return fmt.Errorf("create insight report directory: %w", err)
	}

	// Save report.json
	reportPath := filepath.Join(reportDir, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal insight report: %w", err)
	}

	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return fmt.Errorf("write insight report: %w", err)
	}

	// Generate and save summary.md
	summaryPath := filepath.Join(reportDir, "summary.md")
	summary := generateInsightSummary(report)
	if err := os.WriteFile(summaryPath, []byte(summary), 0o644); err != nil {
		log.Printf("Warning: failed to write insight summary: %v", err)
	}

	// Save metadata.json for quick listing
	metadataPath := filepath.Join(reportDir, "metadata.json")
	metadata := map[string]any{
		"id":               report.ID,
		"task_id":          report.TaskID,
		"generated_at":     report.GeneratedAt,
		"execution_count":  report.ExecutionCount,
		"pattern_count":    len(report.Patterns),
		"suggestion_count": len(report.Suggestions),
		"generated_by":     report.GeneratedBy,
	}
	metadataData, _ := json.MarshalIndent(metadata, "", "  ")
	os.WriteFile(metadataPath, metadataData, 0o644)

	log.Printf("Saved insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, report.TaskID, len(report.Patterns), len(report.Suggestions))

	return nil
}

// LoadInsightReports loads all insight reports for a task.
// Delegates to InsightManager when available.
func (qp *Processor) LoadInsightReports(taskID string) ([]insights.InsightReport, error) {
	if qp.insightManager != nil {
		return qp.insightManager.LoadInsightReports(taskID)
	}
	// Fallback implementation
	insightDir := qp.getInsightDir(taskID)
	entries, err := os.ReadDir(insightDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []insights.InsightReport{}, nil // No insights yet
		}
		return nil, fmt.Errorf("read insight directory: %w", err)
	}

	var reports []insights.InsightReport
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		reportPath := filepath.Join(insightDir, entry.Name(), "report.json")
		data, err := os.ReadFile(reportPath)
		if err != nil {
			log.Printf("Warning: could not read insight report %s: %v", reportPath, err)
			continue
		}

		var report insights.InsightReport
		if err := json.Unmarshal(data, &report); err != nil {
			log.Printf("Warning: could not parse insight report %s: %v", reportPath, err)
			continue
		}

		reports = append(reports, report)
	}

	// Sort by generated time (newest first)
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].GeneratedAt.After(reports[j].GeneratedAt)
	})

	return reports, nil
}

// LoadInsightReport loads a specific insight report.
// Delegates to InsightManager when available.
func (qp *Processor) LoadInsightReport(taskID, reportID string) (*insights.InsightReport, error) {
	if qp.insightManager != nil {
		return qp.insightManager.LoadInsightReport(taskID, reportID)
	}
	// Fallback implementation
	reportPath := filepath.Join(qp.getInsightReportDir(taskID, reportID), "report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("insight report not found")
		}
		return nil, fmt.Errorf("read insight report: %w", err)
	}

	var report insights.InsightReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse insight report: %w", err)
	}

	return &report, nil
}

// UpdateSuggestionStatus updates the status of a specific suggestion.
// Delegates to InsightManager when available.
func (qp *Processor) UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error {
	if qp.insightManager != nil {
		return qp.insightManager.UpdateSuggestionStatus(taskID, reportID, suggestionID, status)
	}
	// Fallback implementation
	report, err := qp.LoadInsightReport(taskID, reportID)
	if err != nil {
		return err
	}

	// Find and update the suggestion
	found := false
	for i := range report.Suggestions {
		if report.Suggestions[i].ID == suggestionID {
			report.Suggestions[i].Status = status
			if status == "applied" {
				now := time.Now()
				report.Suggestions[i].AppliedAt = &now
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("suggestion %s not found in report %s", suggestionID, reportID)
	}

	// Save updated report
	return qp.SaveInsightReport(*report)
}

// LoadAllInsightReports loads all insight reports across all tasks since a given time.
// Delegates to InsightManager when available.
func (qp *Processor) LoadAllInsightReports(sinceTime time.Time) ([]insights.InsightReport, error) {
	if qp.insightManager != nil {
		return qp.insightManager.LoadAllInsightReports(sinceTime)
	}
	// Fallback implementation
	taskDirs, err := os.ReadDir(qp.taskLogsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []insights.InsightReport{}, nil
		}
		return nil, fmt.Errorf("read task logs directory: %w", err)
	}

	var allReports []insights.InsightReport
	for _, taskDir := range taskDirs {
		if !taskDir.IsDir() {
			continue
		}

		taskID := taskDir.Name()
		reports, err := qp.LoadInsightReports(taskID)
		if err != nil {
			log.Printf("Warning: could not load insight reports for task %s: %v", taskID, err)
			continue
		}

		// Filter by time
		for _, report := range reports {
			if report.GeneratedAt.After(sinceTime) || report.GeneratedAt.Equal(sinceTime) {
				allReports = append(allReports, report)
			}
		}
	}

	// Sort by generated time (newest first)
	sort.Slice(allReports, func(i, j int) bool {
		return allReports[i].GeneratedAt.After(allReports[j].GeneratedAt)
	})

	return allReports, nil
}

// ComputeExecutionStatistics calculates aggregate statistics from execution history
func ComputeExecutionStatistics(executions []ExecutionHistory) insights.ExecutionStatistics {
	stats := insights.ExecutionStatistics{
		TotalExecutions: len(executions),
	}

	if len(executions) == 0 {
		return stats
	}

	// Count by exit reason
	exitReasons := make(map[string]int)
	var durations []time.Duration

	for _, exec := range executions {
		exitReasons[exec.ExitReason]++

		switch exec.ExitReason {
		case "completed":
			stats.SuccessCount++
		case "failed":
			stats.FailureCount++
		case "timeout":
			stats.TimeoutCount++
		case "rate_limited":
			stats.RateLimitCount++
		}

		// Parse duration
		if !exec.StartTime.IsZero() && !exec.EndTime.IsZero() {
			duration := exec.EndTime.Sub(exec.StartTime)
			durations = append(durations, duration)
		}
	}

	// Success rate
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
	}

	// Most common exit reason
	maxCount := 0
	for reason, count := range exitReasons {
		if count > maxCount {
			maxCount = count
			stats.MostCommonExitReason = reason
		}
	}

	// Duration statistics
	if len(durations) > 0 {
		// Average
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))
		stats.AvgDuration = formatDuration(avg)

		// Median
		sort.Slice(durations, func(i, j int) bool {
			return durations[i] < durations[j]
		})
		median := durations[len(durations)/2]
		stats.MedianDuration = formatDuration(median)
	}

	return stats
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// generateInsightSummary creates a markdown summary of an insight report
func generateInsightSummary(report insights.InsightReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Insight Report: %s\n\n", report.TaskID))
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Executions Analyzed**: %d\n", report.ExecutionCount))
	sb.WriteString(fmt.Sprintf("**Generated By**: %s\n\n", report.GeneratedBy))

	// Statistics
	sb.WriteString("## Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- **Success Rate**: %.1f%%\n", report.Statistics.SuccessRate))
	sb.WriteString(fmt.Sprintf("- **Total Executions**: %d\n", report.Statistics.TotalExecutions))
	sb.WriteString(fmt.Sprintf("- **Successful**: %d\n", report.Statistics.SuccessCount))
	sb.WriteString(fmt.Sprintf("- **Failed**: %d\n", report.Statistics.FailureCount))
	sb.WriteString(fmt.Sprintf("- **Timeouts**: %d\n", report.Statistics.TimeoutCount))
	sb.WriteString(fmt.Sprintf("- **Rate Limited**: %d\n", report.Statistics.RateLimitCount))
	sb.WriteString(fmt.Sprintf("- **Average Duration**: %s\n", report.Statistics.AvgDuration))
	sb.WriteString(fmt.Sprintf("- **Most Common Exit**: %s\n\n", report.Statistics.MostCommonExitReason))

	// Patterns
	if len(report.Patterns) > 0 {
		sb.WriteString("## Patterns Identified\n\n")
		for i, pattern := range report.Patterns {
			sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, pattern.Description))
			sb.WriteString(fmt.Sprintf("- **Type**: %s\n", pattern.Type))
			sb.WriteString(fmt.Sprintf("- **Severity**: %s\n", pattern.Severity))
			sb.WriteString(fmt.Sprintf("- **Frequency**: %d occurrences\n", pattern.Frequency))
			if len(pattern.Evidence) > 0 {
				sb.WriteString("- **Evidence**:\n")
				for _, evidence := range pattern.Evidence {
					sb.WriteString(fmt.Sprintf("  - %s\n", evidence))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		sb.WriteString("## Suggestions\n\n")
		for i, suggestion := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, suggestion.Title))
			sb.WriteString(fmt.Sprintf("- **Priority**: %s\n", suggestion.Priority))
			sb.WriteString(fmt.Sprintf("- **Type**: %s\n", suggestion.Type))
			sb.WriteString(fmt.Sprintf("- **Status**: %s\n", suggestion.Status))
			sb.WriteString(fmt.Sprintf("\n%s\n\n", suggestion.Description))
			sb.WriteString(fmt.Sprintf("**Expected Impact**: %s (confidence: %s)\n",
				suggestion.Impact.SuccessRateImprovement, suggestion.Impact.Confidence))
			sb.WriteString(fmt.Sprintf("**Rationale**: %s\n\n", suggestion.Impact.Rationale))
		}
	}

	return sb.String()
}

// GenerateSystemInsightReport analyzes all insight reports and generates system-wide insights.
// Delegates to InsightManager when available.
func (qp *Processor) GenerateSystemInsightReport(sinceTime time.Time) (*insights.SystemInsightReport, error) {
	if qp.insightManager != nil {
		return qp.insightManager.GenerateSystemInsightReport(sinceTime)
	}
	// Fallback implementation
	// Load all insight reports since the given time
	reports, err := qp.LoadAllInsightReports(sinceTime)
	if err != nil {
		return nil, fmt.Errorf("load all insight reports: %w", err)
	}

	// Track unique tasks
	taskSet := make(map[string]bool)
	totalExecutions := 0

	// Aggregate patterns by type and identify cross-task patterns
	patternsByType := make(map[string][]insights.Pattern)
	patternOccurrences := make(map[string]*insights.CrossTaskPattern) // keyed by pattern type + description

	// Aggregate suggestions by type
	suggestionsByType := make(map[string]int)

	// Stats by task type and operation
	byTaskType := make(map[string]insights.TaskTypeStats)
	byOperation := make(map[string]insights.TaskTypeStats)

	for _, report := range reports {
		taskSet[report.TaskID] = true
		totalExecutions += report.ExecutionCount

		// Process patterns
		for _, pattern := range report.Patterns {
			patternsByType[pattern.Type] = append(patternsByType[pattern.Type], pattern)

			// Track cross-task patterns (patterns with similar descriptions)
			key := pattern.Type + ":" + pattern.Description
			if existing, ok := patternOccurrences[key]; ok {
				// Pattern exists across tasks
				existing.Frequency += pattern.Frequency
				if !contains(existing.AffectedTasks, report.TaskID) {
					existing.AffectedTasks = append(existing.AffectedTasks, report.TaskID)
				}
			} else {
				// New cross-task pattern
				patternOccurrences[key] = &insights.CrossTaskPattern{
					Pattern:       pattern,
					AffectedTasks: []string{report.TaskID},
					TaskTypes:     []string{}, // Will be populated if needed
				}
			}
		}

		// Process suggestions
		for _, suggestion := range report.Suggestions {
			suggestionsByType[suggestion.Type]++
		}
	}

	// Extract cross-task patterns (patterns affecting 2+ tasks)
	var crossTaskPatterns []insights.CrossTaskPattern
	for _, pattern := range patternOccurrences {
		if len(pattern.AffectedTasks) >= 2 {
			crossTaskPatterns = append(crossTaskPatterns, *pattern)
		}
	}

	// Sort cross-task patterns by severity and frequency
	sort.Slice(crossTaskPatterns, func(i, j int) bool {
		// Priority: critical > high > medium > low
		severityOrder := map[string]int{
			"critical": 4,
			"high":     3,
			"medium":   2,
			"low":      1,
		}
		iSev := severityOrder[crossTaskPatterns[i].Severity]
		jSev := severityOrder[crossTaskPatterns[j].Severity]
		if iSev != jSev {
			return iSev > jSev
		}
		return crossTaskPatterns[i].Frequency > crossTaskPatterns[j].Frequency
	})

	// Generate system-level suggestions (aggregate top suggestions from reports)
	systemSuggestions := aggregateTopSuggestions(reports, 10)

	// Create system insight report
	sysReport := &insights.SystemInsightReport{
		ID:          uuid.New().String(),
		GeneratedAt: time.Now(),
		TimeWindow: insights.AnalysisWindow{
			StartTime: sinceTime,
			EndTime:   time.Now(),
		},
		TaskCount:         len(taskSet),
		TotalExecutions:   totalExecutions,
		CrossTaskPatterns: crossTaskPatterns,
		SystemSuggestions: systemSuggestions,
		ByTaskType:        byTaskType,
		ByOperation:       byOperation,
	}

	return sysReport, nil
}

// aggregateTopSuggestions collects the highest-priority suggestions across all reports
func aggregateTopSuggestions(reports []insights.InsightReport, limit int) []insights.Suggestion {
	var allSuggestions []insights.Suggestion

	for _, report := range reports {
		allSuggestions = append(allSuggestions, report.Suggestions...)
	}

	// Sort by priority
	priorityOrder := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}

	sort.Slice(allSuggestions, func(i, j int) bool {
		return priorityOrder[allSuggestions[i].Priority] > priorityOrder[allSuggestions[j].Priority]
	})

	// Return top N suggestions
	if len(allSuggestions) > limit {
		return allSuggestions[:limit]
	}
	return allSuggestions
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
