package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/insights"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/google/uuid"
)

// InsightManagerDeps contains dependencies for creating an InsightManager.
type InsightManagerDeps struct {
	TaskLogsDir    string
	HistoryManager HistoryManagerAPI
	AgentSvc       agentmanager.AgentServiceAPI
	Assembler      prompts.AssemblerAPI
	Storage        tasks.StorageAPI
}

// InsightManager handles insight report generation, persistence, and retrieval.
type InsightManager struct {
	taskLogsDir    string
	historyManager HistoryManagerAPI
	agentSvc       agentmanager.AgentServiceAPI
	assembler      prompts.AssemblerAPI
	storage        tasks.StorageAPI
}

// NewInsightManager creates a new InsightManager with the specified dependencies.
func NewInsightManager(deps InsightManagerDeps) *InsightManager {
	return &InsightManager{
		taskLogsDir:    deps.TaskLogsDir,
		historyManager: deps.HistoryManager,
		agentSvc:       deps.AgentSvc,
		assembler:      deps.Assembler,
		storage:        deps.Storage,
	}
}

// getInsightDir returns the directory for a task's insight reports.
func (im *InsightManager) getInsightDir(taskID string) string {
	return filepath.Join(im.taskLogsDir, taskID, "insights")
}

// getInsightReportDir returns the directory for a specific insight report.
func (im *InsightManager) getInsightReportDir(taskID, reportID string) string {
	return filepath.Join(im.getInsightDir(taskID), reportID)
}

// SaveInsightReport persists an insight report to disk.
func (im *InsightManager) SaveInsightReport(report insights.InsightReport) error {
	// Generate ID if not set
	if report.ID == "" {
		report.ID = uuid.New().String()
	}

	// Set generated time if not set
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}

	// Create report directory
	reportDir := im.getInsightReportDir(report.TaskID, report.ID)
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
	summary := im.generateInsightSummary(report)
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
func (im *InsightManager) LoadInsightReports(taskID string) ([]insights.InsightReport, error) {
	insightDir := im.getInsightDir(taskID)
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
func (im *InsightManager) LoadInsightReport(taskID, reportID string) (*insights.InsightReport, error) {
	reportPath := filepath.Join(im.getInsightReportDir(taskID, reportID), "report.json")
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

// LoadAllInsightReports loads all insight reports across all tasks since a given time.
func (im *InsightManager) LoadAllInsightReports(sinceTime time.Time) ([]insights.InsightReport, error) {
	taskDirs, err := os.ReadDir(im.taskLogsDir)
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
		reports, err := im.LoadInsightReports(taskID)
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

// UpdateSuggestionStatus updates the status of a specific suggestion.
func (im *InsightManager) UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error {
	report, err := im.LoadInsightReport(taskID, reportID)
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
	return im.SaveInsightReport(*report)
}

// GenerateSystemInsightReport analyzes all insight reports and generates system-wide insights.
func (im *InsightManager) GenerateSystemInsightReport(sinceTime time.Time) (*insights.SystemInsightReport, error) {
	// Load all insight reports since the given time
	reports, err := im.LoadAllInsightReports(sinceTime)
	if err != nil {
		return nil, fmt.Errorf("load all insight reports: %w", err)
	}

	// Track unique tasks
	taskSet := make(map[string]bool)
	totalExecutions := 0

	// Aggregate patterns by type and identify cross-task patterns
	patternOccurrences := make(map[string]*insights.CrossTaskPattern) // keyed by pattern type + description

	// Stats by task type and operation
	byTaskType := make(map[string]insights.TaskTypeStats)
	byOperation := make(map[string]insights.TaskTypeStats)

	for _, report := range reports {
		taskSet[report.TaskID] = true
		totalExecutions += report.ExecutionCount

		// Process patterns
		for _, pattern := range report.Patterns {
			// Track cross-task patterns (patterns with similar descriptions)
			key := pattern.Type + ":" + pattern.Description
			if existing, ok := patternOccurrences[key]; ok {
				// Pattern exists across tasks
				existing.Frequency += pattern.Frequency
				if !containsString(existing.AffectedTasks, report.TaskID) {
					existing.AffectedTasks = append(existing.AffectedTasks, report.TaskID)
				}
			} else {
				// New cross-task pattern
				patternOccurrences[key] = &insights.CrossTaskPattern{
					Pattern:       pattern,
					AffectedTasks: []string{report.TaskID},
					TaskTypes:     []string{},
				}
			}
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

	// Generate system-level suggestions
	systemSuggestions := im.aggregateTopSuggestions(reports, 10)

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

// GenerateInsightReportForTask analyzes execution history and generates an insight report.
func (im *InsightManager) GenerateInsightReportForTask(taskID string, limit int, statusFilter string) (*insights.InsightReport, error) {
	log.Printf("Generating insight report for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)
	systemlog.Infof("Starting insight generation for task %s", taskID)

	// Get task title for process tracking
	var taskTitle string
	if im.storage != nil {
		if task, _, err := im.storage.GetTaskByID(taskID); err == nil && task != nil {
			taskTitle = task.Title
		}
	}
	if taskTitle == "" {
		taskTitle = taskID
	}

	// Register insight generation process
	RegisterInsightProcess(taskID, taskTitle)
	defer UnregisterInsightProcess(taskID)

	// Load execution history
	history, err := im.historyManager.LoadExecutionHistory(taskID)
	if err != nil {
		return nil, fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := im.filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	// Load execution file contents
	executionDetails, err := im.loadExecutionDetails(filtered)
	if err != nil {
		log.Printf("Warning: Failed to load some execution details: %v", err)
	}

	// Assemble the insight-generator prompt
	prompt, err := im.assembleInsightPrompt(taskID, filtered, stats, executionDetails)
	if err != nil {
		return nil, fmt.Errorf("assemble insight prompt: %w", err)
	}

	log.Printf("Assembled insight prompt (%d chars) for task %s", len(prompt), taskID)

	// Call Claude Code with the prompt
	claudeResponse, err := im.callClaudeCodeForInsight(prompt, taskID)
	if err != nil {
		return nil, fmt.Errorf("call Claude Code: %w", err)
	}

	if !claudeResponse.Success {
		return nil, fmt.Errorf("Claude Code execution failed: %s", claudeResponse.Error)
	}

	// Parse JSON response into InsightReport
	report, err := im.parseInsightResponse(claudeResponse.Output, taskID, filtered, stats)
	if err != nil {
		// Try to extract JSON from the output if it's wrapped in markdown
		cleanedOutput := extractJSONFromMarkdown(claudeResponse.Output)
		report, err = im.parseInsightResponse(cleanedOutput, taskID, filtered, stats)
		if err != nil {
			return nil, fmt.Errorf("parse insight response: %w\nOutput: %s", err, claudeResponse.Output)
		}
	}

	// Enrich report with metadata
	report.GeneratedBy = "insight-generator"
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}

	// Save the report
	if err := im.SaveInsightReport(*report); err != nil {
		return nil, fmt.Errorf("save insight report: %w", err)
	}

	systemlog.Infof("Generated insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, taskID, len(report.Patterns), len(report.Suggestions))

	return report, nil
}

// BuildInsightPrompt builds and returns the prompt that would be used for insight generation.
func (im *InsightManager) BuildInsightPrompt(taskID string, limit int, statusFilter string) (string, error) {
	log.Printf("Building insight prompt for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)

	// Load execution history
	history, err := im.historyManager.LoadExecutionHistory(taskID)
	if err != nil {
		return "", fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return "", fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := im.filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return "", fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	// Load execution file contents
	executionDetails, err := im.loadExecutionDetails(filtered)
	if err != nil {
		log.Printf("Warning: Failed to load some execution details: %v", err)
	}

	// Assemble the insight-generator prompt
	prompt, err := im.assembleInsightPrompt(taskID, filtered, stats, executionDetails)
	if err != nil {
		return "", fmt.Errorf("assemble insight prompt: %w", err)
	}

	log.Printf("Built insight prompt (%d chars) for task %s", len(prompt), taskID)

	return prompt, nil
}

// GenerateInsightReportWithCustomPrompt generates an insight report using a custom prompt.
func (im *InsightManager) GenerateInsightReportWithCustomPrompt(taskID string, limit int, statusFilter string, customPrompt string) (*insights.InsightReport, error) {
	log.Printf("Generating insight report with custom prompt for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)
	systemlog.Infof("Starting custom insight generation for task %s", taskID)

	if customPrompt == "" {
		return nil, fmt.Errorf("custom prompt is required")
	}

	// Get task title for process tracking
	var taskTitle string
	if im.storage != nil {
		if task, _, err := im.storage.GetTaskByID(taskID); err == nil && task != nil {
			taskTitle = task.Title
		}
	}
	if taskTitle == "" {
		taskTitle = taskID
	}

	// Register insight generation process
	RegisterInsightProcess(taskID, taskTitle)
	defer UnregisterInsightProcess(taskID)

	// Load execution history for metadata
	history, err := im.historyManager.LoadExecutionHistory(taskID)
	if err != nil {
		return nil, fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := im.filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	log.Printf("Using custom insight prompt (%d chars) for task %s", len(customPrompt), taskID)

	// Call Claude Code with the custom prompt
	claudeResponse, err := im.callClaudeCodeForInsight(customPrompt, taskID)
	if err != nil {
		return nil, fmt.Errorf("call Claude Code: %w", err)
	}

	if !claudeResponse.Success {
		return nil, fmt.Errorf("Claude Code execution failed: %s", claudeResponse.Error)
	}

	// Parse JSON response into InsightReport
	report, err := im.parseInsightResponse(claudeResponse.Output, taskID, filtered, stats)
	if err != nil {
		// Try to extract JSON from the output if it's wrapped in markdown
		cleanedOutput := extractJSONFromMarkdown(claudeResponse.Output)
		report, err = im.parseInsightResponse(cleanedOutput, taskID, filtered, stats)
		if err != nil {
			return nil, fmt.Errorf("parse insight response: %w\nOutput: %s", err, claudeResponse.Output)
		}
	}

	// Enrich report with metadata
	report.GeneratedBy = "insight-generator"
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}

	// Save the report
	if err := im.SaveInsightReport(*report); err != nil {
		return nil, fmt.Errorf("save insight report: %w", err)
	}

	systemlog.Infof("Generated custom insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, taskID, len(report.Patterns), len(report.Suggestions))

	return report, nil
}

// filterExecutionsByStatus filters execution history by exit reason.
func (im *InsightManager) filterExecutionsByStatus(history []ExecutionHistory, statusFilter string, limit int) []ExecutionHistory {
	if statusFilter == "" {
		statusFilter = "failed,timeout" // Default to failures
	}

	// If "all", return all executions with limit applied
	if statusFilter == "all" {
		if limit > 0 && len(history) > limit {
			return history[:limit]
		}
		return history
	}

	// Build status map
	statuses := make(map[string]bool)
	for _, s := range strings.Split(statusFilter, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			statuses[trimmed] = true
		}
	}

	// Filter by status
	filtered := make([]ExecutionHistory, 0, limit)
	for _, exec := range history {
		if statuses[exec.ExitReason] {
			filtered = append(filtered, exec)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}

	return filtered
}

// executionDetail holds file contents for an execution.
type executionDetail struct {
	ExecutionID  string
	ExitReason   string
	Duration     string
	StartTime    time.Time
	Output       string
	LastMessage  string
	Prompt       string
	SteerMode    string
	SteerProfile string
	SteerPhase   int
	SteerIter    int
}

// loadExecutionDetails loads file contents for executions.
func (im *InsightManager) loadExecutionDetails(executions []ExecutionHistory) ([]executionDetail, error) {
	details := make([]executionDetail, 0, len(executions))

	for _, exec := range executions {
		detail := executionDetail{
			ExecutionID:  exec.ExecutionID,
			ExitReason:   exec.ExitReason,
			Duration:     exec.Duration,
			StartTime:    exec.StartTime,
			SteerMode:    exec.SteerMode,
			SteerProfile: exec.AutoSteerProfileID,
			SteerPhase:   exec.SteerPhaseIndex,
			SteerIter:    exec.SteerPhaseIteration,
		}

		// Load output (prefer clean output)
		if exec.CleanOutputPath != "" {
			if content, err := os.ReadFile(filepath.Join(im.taskLogsDir, exec.CleanOutputPath)); err == nil {
				detail.Output = string(content)
			}
		}
		if detail.Output == "" && exec.OutputPath != "" {
			if content, err := os.ReadFile(filepath.Join(im.taskLogsDir, exec.OutputPath)); err == nil {
				detail.Output = string(content)
			}
		}

		// Load last message
		if exec.LastMessagePath != "" {
			if content, err := os.ReadFile(filepath.Join(im.taskLogsDir, exec.LastMessagePath)); err == nil {
				detail.LastMessage = string(content)
			}
		}

		details = append(details, detail)
	}

	return details, nil
}

// assembleInsightPrompt creates the prompt for insight generation.
func (im *InsightManager) assembleInsightPrompt(taskID string, executions []ExecutionHistory, stats insights.ExecutionStatistics, details []executionDetail) (string, error) {
	// Create a pseudo-task for prompt assembly
	insightTask := tasks.TaskItem{
		ID:        fmt.Sprintf("insight-generator-%s-%d", taskID, time.Now().Unix()),
		Title:     fmt.Sprintf("Analyze execution history for %s", taskID),
		Type:      "insight",
		Operation: "generator",
		Target:    taskID,
		Category:  "analysis",
		Priority:  "medium",
	}

	// Assemble base prompt using the template
	assembly, err := im.assembler.AssemblePromptForTask(insightTask)
	if err != nil {
		return "", fmt.Errorf("assemble base prompt: %w", err)
	}

	prompt := assembly.Prompt

	// Replace template variables
	replacements := map[string]string{
		"{{TASK_ID}}":                 taskID,
		"{{TITLE}}":                   fmt.Sprintf("Insight Generation for %s", taskID),
		"{{TARGET}}":                  taskID,
		"{{EXECUTION_COUNT}}":         fmt.Sprintf("%d", len(executions)),
		"{{STATUS_FILTER}}":           im.determineStatusFilter(executions),
		"{{SUCCESS_RATE}}":            fmt.Sprintf("%.1f", stats.SuccessRate),
		"{{AVG_DURATION}}":            stats.AvgDuration,
		"{{MOST_COMMON_EXIT_REASON}}": stats.MostCommonExitReason,
		"{{EXECUTIONS_SUMMARY}}":      im.buildExecutionSummary(executions),
		"{{EXECUTION_DETAILS}}":       im.buildExecutionDetails(details),
	}

	for key, value := range replacements {
		prompt = strings.ReplaceAll(prompt, key, value)
	}

	return prompt, nil
}

// determineStatusFilter figures out what status filter was used.
func (im *InsightManager) determineStatusFilter(executions []ExecutionHistory) string {
	statuses := make(map[string]bool)
	for _, exec := range executions {
		statuses[exec.ExitReason] = true
	}

	var filters []string
	for status := range statuses {
		filters = append(filters, status)
	}
	return strings.Join(filters, ", ")
}

// buildExecutionSummary creates a brief summary of executions.
func (im *InsightManager) buildExecutionSummary(executions []ExecutionHistory) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("**Total Executions**: %d\n\n", len(executions)))

	exitReasons := make(map[string]int)
	for _, exec := range executions {
		exitReasons[exec.ExitReason]++
	}

	buf.WriteString("**Breakdown by Exit Reason:**\n")
	for reason, count := range exitReasons {
		buf.WriteString(fmt.Sprintf("- %s: %d\n", reason, count))
	}

	return buf.String()
}

// buildExecutionDetails creates detailed execution information.
func (im *InsightManager) buildExecutionDetails(details []executionDetail) string {
	var buf bytes.Buffer

	for i, detail := range details {
		buf.WriteString(fmt.Sprintf("\n### Execution %d: %s\n\n", i+1, detail.ExecutionID))
		buf.WriteString(fmt.Sprintf("**Exit Reason**: %s\n", detail.ExitReason))
		buf.WriteString(fmt.Sprintf("**Duration**: %s\n", detail.Duration))
		buf.WriteString(fmt.Sprintf("**Started**: %s\n", detail.StartTime.Format(time.RFC3339)))

		if detail.SteerProfile != "" {
			buf.WriteString(fmt.Sprintf("**Auto Steer**: Profile %s, Phase %d, Iteration %d\n",
				detail.SteerProfile, detail.SteerPhase, detail.SteerIter))
		} else if detail.SteerMode != "" {
			buf.WriteString(fmt.Sprintf("**Steering**: %s\n", detail.SteerMode))
		}

		if detail.LastMessage != "" {
			buf.WriteString("\n**Last Agent Message:**\n```\n")
			// Truncate if too long
			lastMsg := detail.LastMessage
			if len(lastMsg) > 1000 {
				lastMsg = lastMsg[:1000] + "\n... (truncated)"
			}
			buf.WriteString(lastMsg)
			buf.WriteString("\n```\n")
		}

		if detail.Output != "" {
			buf.WriteString("\n**Output Excerpt:**\n```\n")
			// Extract last 1500 chars as they often contain the most relevant error info
			output := detail.Output
			if len(output) > 1500 {
				output = "... (truncated)\n" + output[len(output)-1500:]
			}
			buf.WriteString(output)
			buf.WriteString("\n```\n")
		}

		buf.WriteString("\n---\n")
	}

	return buf.String()
}

// callClaudeCodeForInsight executes insight generation via agent-manager.
func (im *InsightManager) callClaudeCodeForInsight(prompt string, taskID string) (*tasks.ClaudeCodeResponse, error) {
	log.Printf("Calling agent-manager for insight generation (task: %s)", taskID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Check agent-manager availability
	if !im.agentSvc.IsAvailable(ctx) {
		return nil, fmt.Errorf("agent-manager is not available")
	}

	// Execute via agent-manager insights profile
	result, err := im.agentSvc.ExecuteInsight(ctx, agentmanager.InsightRequest{
		TaskID:  taskID,
		Prompt:  prompt,
		Timeout: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("agent-manager execute insight: %w", err)
	}

	// Map result to ClaudeCodeResponse
	return &tasks.ClaudeCodeResponse{
		Success:          result.Success,
		Message:          result.Output,
		Output:           result.Output,
		Error:            result.ErrorMessage,
		RateLimited:      result.RateLimited,
		MaxTurnsExceeded: result.MaxTurnsExceeded,
	}, nil
}

// parseInsightResponse parses the LLM response into an InsightReport.
func (im *InsightManager) parseInsightResponse(output string, taskID string, executions []ExecutionHistory, stats insights.ExecutionStatistics) (*insights.InsightReport, error) {
	// Try to parse as JSON
	var rawReport struct {
		Patterns    []insights.Pattern    `json:"patterns"`
		Suggestions []insights.Suggestion `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(output), &rawReport); err != nil {
		return nil, fmt.Errorf("JSON unmarshal: %w", err)
	}

	// Build full report
	report := &insights.InsightReport{
		TaskID:         taskID,
		GeneratedAt:    time.Now(),
		ExecutionCount: len(executions),
		AnalysisWindow: insights.AnalysisWindow{
			Limit:        len(executions),
			StatusFilter: im.determineStatusFilter(executions),
		},
		Patterns:    rawReport.Patterns,
		Suggestions: rawReport.Suggestions,
		Statistics:  stats,
	}

	// Set time window
	if len(executions) > 0 {
		report.AnalysisWindow.StartTime = executions[len(executions)-1].StartTime
		report.AnalysisWindow.EndTime = executions[0].StartTime
	}

	return report, nil
}

// aggregateTopSuggestions collects the highest-priority suggestions across all reports.
func (im *InsightManager) aggregateTopSuggestions(reports []insights.InsightReport, limit int) []insights.Suggestion {
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

// generateInsightSummary creates a markdown summary of an insight report.
func (im *InsightManager) generateInsightSummary(report insights.InsightReport) string {
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

// containsString checks if a string slice contains a value.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
