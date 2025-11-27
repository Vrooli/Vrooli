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

	"github.com/google/uuid"
)

// InsightReport represents an analysis of execution patterns and suggested improvements
type InsightReport struct {
	ID             string              `json:"id"`
	TaskID         string              `json:"task_id"`
	GeneratedAt    time.Time           `json:"generated_at"`
	AnalysisWindow AnalysisWindow      `json:"analysis_window"`
	ExecutionCount int                 `json:"execution_count"`
	Patterns       []Pattern           `json:"patterns"`
	Suggestions    []Suggestion        `json:"suggestions"`
	Statistics     ExecutionStatistics `json:"statistics"`
	GeneratedBy    string              `json:"generated_by"` // task_id of insight-generator
}

// AnalysisWindow describes what executions were analyzed
type AnalysisWindow struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Limit        int       `json:"limit"`         // How many executions analyzed
	StatusFilter string    `json:"status_filter"` // e.g., "failed,timeout"
}

// Pattern represents a recurring issue or behavior
type Pattern struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`      // failure_mode, timeout, rate_limit, stuck_state
	Frequency   int      `json:"frequency"` // How many executions exhibit this
	Severity    string   `json:"severity"`  // critical, high, medium, low
	Description string   `json:"description"`
	Examples    []string `json:"examples"` // Execution IDs showing this pattern
	Evidence    []string `json:"evidence"` // Specific log excerpts
}

// Suggestion represents an actionable improvement
type Suggestion struct {
	ID          string           `json:"id"`
	PatternID   string           `json:"pattern_id"` // Which pattern this addresses
	Type        string           `json:"type"`       // prompt, timeout, code, autosteer_profile
	Priority    string           `json:"priority"`   // critical, high, medium, low
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Changes     []ProposedChange `json:"changes"`
	Impact      ImpactEstimate   `json:"impact"`
	Status      string           `json:"status"` // pending, applied, rejected, superseded
	AppliedAt   *time.Time       `json:"applied_at,omitempty"`
}

// ProposedChange describes a specific change to apply
type ProposedChange struct {
	File        string `json:"file"` // Relative path from scenario root
	Type        string `json:"type"` // edit, create, config_update
	Description string `json:"description"`
	Before      string `json:"before,omitempty"`       // For edits
	After       string `json:"after,omitempty"`        // For edits
	Content     string `json:"content,omitempty"`      // For creates
	ConfigPath  string `json:"config_path,omitempty"`  // For config updates (e.g., "phases[0].timeout")
	ConfigValue any    `json:"config_value,omitempty"` // New config value
}

// ImpactEstimate describes expected impact of a suggestion
type ImpactEstimate struct {
	SuccessRateImprovement string `json:"success_rate_improvement"` // e.g., "+15-25%"
	TimeReduction          string `json:"time_reduction,omitempty"` // e.g., "-10m avg"
	Confidence             string `json:"confidence"`               // high, medium, low
	Rationale              string `json:"rationale"`
}

// ExecutionStatistics provides aggregate stats for analyzed executions
type ExecutionStatistics struct {
	TotalExecutions      int     `json:"total_executions"`
	SuccessCount         int     `json:"success_count"`
	FailureCount         int     `json:"failure_count"`
	TimeoutCount         int     `json:"timeout_count"`
	RateLimitCount       int     `json:"rate_limit_count"`
	SuccessRate          float64 `json:"success_rate"`
	AvgDuration          string  `json:"avg_duration"`
	MedianDuration       string  `json:"median_duration"`
	MostCommonExitReason string  `json:"most_common_exit_reason"`
}

// SystemInsightReport represents system-wide analysis across all tasks
type SystemInsightReport struct {
	ID                string                   `json:"id"`
	GeneratedAt       time.Time                `json:"generated_at"`
	TimeWindow        AnalysisWindow           `json:"time_window"`
	TaskCount         int                      `json:"task_count"`
	TotalExecutions   int                      `json:"total_executions"`
	CrossTaskPatterns []CrossTaskPattern       `json:"cross_task_patterns"`
	SystemSuggestions []Suggestion             `json:"system_suggestions"`
	ByTaskType        map[string]TaskTypeStats `json:"by_task_type"`
	ByOperation       map[string]TaskTypeStats `json:"by_operation"`
}

// CrossTaskPattern represents a pattern affecting multiple tasks
type CrossTaskPattern struct {
	Pattern
	AffectedTasks []string `json:"affected_tasks"`
	TaskTypes     []string `json:"task_types"` // Which task types show this
}

// TaskTypeStats provides aggregate stats by task type or operation
type TaskTypeStats struct {
	Count       int     `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration string  `json:"avg_duration"`
	TopPattern  string  `json:"top_pattern"`
}

// getInsightDir returns the directory for a task's insight reports
func (qp *Processor) getInsightDir(taskID string) string {
	return filepath.Join(qp.taskLogsDir, taskID, "insights")
}

// getInsightReportDir returns the directory for a specific insight report
func (qp *Processor) getInsightReportDir(taskID, reportID string) string {
	return filepath.Join(qp.getInsightDir(taskID), reportID)
}

// SaveInsightReport persists an insight report to disk
func (qp *Processor) SaveInsightReport(report InsightReport) error {
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
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("create insight report directory: %w", err)
	}

	// Save report.json
	reportPath := filepath.Join(reportDir, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal insight report: %w", err)
	}

	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		return fmt.Errorf("write insight report: %w", err)
	}

	// Generate and save summary.md
	summaryPath := filepath.Join(reportDir, "summary.md")
	summary := generateInsightSummary(report)
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
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
	os.WriteFile(metadataPath, metadataData, 0644)

	log.Printf("Saved insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, report.TaskID, len(report.Patterns), len(report.Suggestions))

	return nil
}

// LoadInsightReports loads all insight reports for a task
func (qp *Processor) LoadInsightReports(taskID string) ([]InsightReport, error) {
	insightDir := qp.getInsightDir(taskID)
	entries, err := os.ReadDir(insightDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InsightReport{}, nil // No insights yet
		}
		return nil, fmt.Errorf("read insight directory: %w", err)
	}

	var reports []InsightReport
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

		var report InsightReport
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

// LoadInsightReport loads a specific insight report
func (qp *Processor) LoadInsightReport(taskID, reportID string) (*InsightReport, error) {
	reportPath := filepath.Join(qp.getInsightReportDir(taskID, reportID), "report.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("insight report not found")
		}
		return nil, fmt.Errorf("read insight report: %w", err)
	}

	var report InsightReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse insight report: %w", err)
	}

	return &report, nil
}

// UpdateSuggestionStatus updates the status of a specific suggestion
func (qp *Processor) UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error {
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

// LoadAllInsightReports loads all insight reports across all tasks since a given time
func (qp *Processor) LoadAllInsightReports(sinceTime time.Time) ([]InsightReport, error) {
	taskDirs, err := os.ReadDir(qp.taskLogsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InsightReport{}, nil
		}
		return nil, fmt.Errorf("read task logs directory: %w", err)
	}

	var allReports []InsightReport
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
func ComputeExecutionStatistics(executions []ExecutionHistory) ExecutionStatistics {
	stats := ExecutionStatistics{
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
func generateInsightSummary(report InsightReport) string {
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

// GenerateSystemInsightReport analyzes all insight reports and generates system-wide insights
func (qp *Processor) GenerateSystemInsightReport(sinceTime time.Time) (*SystemInsightReport, error) {
	// Load all insight reports since the given time
	reports, err := qp.LoadAllInsightReports(sinceTime)
	if err != nil {
		return nil, fmt.Errorf("load all insight reports: %w", err)
	}

	// Track unique tasks
	taskSet := make(map[string]bool)
	totalExecutions := 0

	// Aggregate patterns by type and identify cross-task patterns
	patternsByType := make(map[string][]Pattern)
	patternOccurrences := make(map[string]*CrossTaskPattern) // keyed by pattern type + description

	// Aggregate suggestions by type
	suggestionsByType := make(map[string]int)

	// Stats by task type and operation
	byTaskType := make(map[string]TaskTypeStats)
	byOperation := make(map[string]TaskTypeStats)

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
				patternOccurrences[key] = &CrossTaskPattern{
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
	var crossTaskPatterns []CrossTaskPattern
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
	sysReport := &SystemInsightReport{
		ID:          uuid.New().String(),
		GeneratedAt: time.Now(),
		TimeWindow: AnalysisWindow{
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
func aggregateTopSuggestions(reports []InsightReport, limit int) []Suggestion {
	var allSuggestions []Suggestion

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
