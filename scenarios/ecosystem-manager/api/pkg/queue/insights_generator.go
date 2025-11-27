package queue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// GenerateInsightReportForTask analyzes execution history and generates an insight report
func (qp *Processor) GenerateInsightReportForTask(taskID string, limit int, statusFilter string) (*InsightReport, error) {
	log.Printf("Generating insight report for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)
	systemlog.Infof("Starting insight generation for task %s", taskID)

	// Load execution history
	history, err := qp.LoadExecutionHistory(taskID)
	if err != nil {
		return nil, fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	// Load execution file contents (output and last message)
	executionDetails, err := qp.loadExecutionDetails(filtered)
	if err != nil {
		log.Printf("Warning: Failed to load some execution details: %v", err)
		// Continue with what we have
	}

	// Assemble the insight-generator prompt
	prompt, err := qp.assembleInsightPrompt(taskID, filtered, stats, executionDetails)
	if err != nil {
		return nil, fmt.Errorf("assemble insight prompt: %w", err)
	}

	log.Printf("Assembled insight prompt (%d chars) for task %s", len(prompt), taskID)

	// Call Claude Code with the prompt
	claudeResponse, err := qp.callClaudeCodeForInsight(prompt, taskID)
	if err != nil {
		return nil, fmt.Errorf("call Claude Code: %w", err)
	}

	if !claudeResponse.Success {
		return nil, fmt.Errorf("Claude Code execution failed: %s", claudeResponse.Error)
	}

	// Parse JSON response into InsightReport
	report, err := parseInsightResponse(claudeResponse.Output, taskID, filtered, stats)
	if err != nil {
		// Try to extract JSON from the output if it's wrapped in markdown
		cleanedOutput := extractJSONFromMarkdown(claudeResponse.Output)
		report, err = parseInsightResponse(cleanedOutput, taskID, filtered, stats)
		if err != nil {
			return nil, fmt.Errorf("parse insight response: %w\nOutput: %s", err, claudeResponse.Output)
		}
	}

	// Enrich report with metadata
	report.GeneratedBy = "insight-generator"
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}
	if report.ID == "" {
		// ID will be generated when saved
	}

	// Save the report
	if err := qp.SaveInsightReport(*report); err != nil {
		return nil, fmt.Errorf("save insight report: %w", err)
	}

	systemlog.Infof("Generated insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, taskID, len(report.Patterns), len(report.Suggestions))

	return report, nil
}

// filterExecutionsByStatus filters execution history by exit reason
func filterExecutionsByStatus(history []ExecutionHistory, statusFilter string, limit int) []ExecutionHistory {
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

// executionDetailData holds file contents for an execution
type executionDetailData struct {
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

// loadExecutionDetails loads file contents for executions
func (qp *Processor) loadExecutionDetails(executions []ExecutionHistory) ([]executionDetailData, error) {
	details := make([]executionDetailData, 0, len(executions))

	for _, exec := range executions {
		detail := executionDetailData{
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
			if content, err := os.ReadFile(filepath.Join(qp.taskLogsDir, exec.CleanOutputPath)); err == nil {
				detail.Output = string(content)
			}
		}
		if detail.Output == "" && exec.OutputPath != "" {
			if content, err := os.ReadFile(filepath.Join(qp.taskLogsDir, exec.OutputPath)); err == nil {
				detail.Output = string(content)
			}
		}

		// Load last message
		if exec.LastMessagePath != "" {
			if content, err := os.ReadFile(filepath.Join(qp.taskLogsDir, exec.LastMessagePath)); err == nil {
				detail.LastMessage = string(content)
			}
		}

		// Optionally load prompt (commented out to reduce prompt size, can enable if needed)
		// if exec.PromptPath != "" {
		// 	if content, err := os.ReadFile(filepath.Join(qp.taskLogsDir, exec.PromptPath)); err == nil {
		// 		detail.Prompt = string(content)
		// 	}
		// }

		details = append(details, detail)
	}

	return details, nil
}

// assembleInsightPrompt creates the prompt for insight generation
func (qp *Processor) assembleInsightPrompt(taskID string, executions []ExecutionHistory, stats ExecutionStatistics, details []executionDetailData) (string, error) {
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
	assembly, err := qp.assembler.AssemblePromptForTask(insightTask)
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
		"{{STATUS_FILTER}}":           determineStatusFilter(executions),
		"{{SUCCESS_RATE}}":            fmt.Sprintf("%.1f", stats.SuccessRate),
		"{{AVG_DURATION}}":            stats.AvgDuration,
		"{{MOST_COMMON_EXIT_REASON}}": stats.MostCommonExitReason,
		"{{EXECUTIONS_SUMMARY}}":      buildExecutionSummary(executions),
		"{{EXECUTION_DETAILS}}":       buildExecutionDetails(details),
	}

	for key, value := range replacements {
		prompt = strings.ReplaceAll(prompt, key, value)
	}

	return prompt, nil
}

// determineStatusFilter figures out what status filter was used
func determineStatusFilter(executions []ExecutionHistory) string {
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

// buildExecutionSummary creates a brief summary of executions
func buildExecutionSummary(executions []ExecutionHistory) string {
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

// buildExecutionDetails creates detailed execution information
func buildExecutionDetails(details []executionDetailData) string {
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

// callClaudeCodeForInsight calls Claude Code via the resource API
func (qp *Processor) callClaudeCodeForInsight(prompt string, taskID string) (*tasks.ClaudeCodeResponse, error) {
	// Find claude-code resource port
	claudeCodePort, err := qp.getResourcePort("claude-code")
	if err != nil {
		return nil, fmt.Errorf("get claude-code port: %w", err)
	}

	// Prepare request
	request := tasks.ClaudeCodeRequest{
		Prompt: prompt,
		Context: map[string]any{
			"task_id":   taskID,
			"operation": "insight-generation",
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Call claude-code via HTTP (similar to how other resources are called)
	url := fmt.Sprintf("http://localhost:%d/execute", claudeCodePort)

	log.Printf("Calling Claude Code for insight generation (task: %s)", taskID)

	// Use a simple HTTP POST (you may need to adjust based on actual claude-code API)
	response, err := qp.httpPost(url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("HTTP POST to claude-code: %w", err)
	}

	// Parse response
	var claudeResp tasks.ClaudeCodeResponse
	if err := json.Unmarshal(response, &claudeResp); err != nil {
		return nil, fmt.Errorf("parse claude-code response: %w", err)
	}

	return &claudeResp, nil
}

// httpPost performs an HTTP POST request
func (qp *Processor) httpPost(url string, body []byte) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// getResourcePort looks up the port for a resource from the port registry
func (qp *Processor) getResourcePort(resourceName string) (int, error) {
	// Read the port registry file
	portRegistryPath := os.Getenv("VROOLI_PORT_REGISTRY")
	if portRegistryPath == "" {
		// Default location
		vrooliRoot := os.Getenv("VROOLI_ROOT")
		if vrooliRoot == "" {
			vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
		}
		portRegistryPath = filepath.Join(vrooliRoot, "scripts", "resources", "port-registry.json")
	}

	data, err := os.ReadFile(portRegistryPath)
	if err != nil {
		// Fallback to known defaults if registry is unavailable
		log.Printf("Warning: Could not read port registry (%s), using defaults: %v", portRegistryPath, err)
		switch resourceName {
		case "claude-code":
			return 8100, nil
		case "ollama":
			return 11434, nil
		case "postgres":
			return 5432, nil
		default:
			return 0, fmt.Errorf("resource %s not found (port registry unavailable)", resourceName)
		}
	}

	var registry map[string]any
	if err := json.Unmarshal(data, &registry); err != nil {
		return 0, fmt.Errorf("parse port registry: %w", err)
	}

	// Look for the resource in the registry
	if resourceData, ok := registry[resourceName].(map[string]any); ok {
		if port, ok := resourceData["port"].(float64); ok {
			return int(port), nil
		}
	}

	return 0, fmt.Errorf("resource %s not found in port registry", resourceName)
}

// parseInsightResponse parses the LLM response into an InsightReport
func parseInsightResponse(output string, taskID string, executions []ExecutionHistory, stats ExecutionStatistics) (*InsightReport, error) {
	// Try to parse as JSON
	var rawReport struct {
		Patterns    []Pattern    `json:"patterns"`
		Suggestions []Suggestion `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(output), &rawReport); err != nil {
		return nil, fmt.Errorf("JSON unmarshal: %w", err)
	}

	// Build full report
	report := &InsightReport{
		TaskID:         taskID,
		GeneratedAt:    time.Now(),
		ExecutionCount: len(executions),
		AnalysisWindow: AnalysisWindow{
			Limit:        len(executions),
			StatusFilter: determineStatusFilter(executions),
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

// extractJSONFromMarkdown tries to extract JSON from markdown code blocks
func extractJSONFromMarkdown(output string) string {
	// Remove markdown code fences
	output = strings.TrimSpace(output)

	// Look for ```json or ``` blocks
	if strings.Contains(output, "```json") {
		start := strings.Index(output, "```json") + 7
		end := strings.Index(output[start:], "```")
		if end != -1 {
			return strings.TrimSpace(output[start : start+end])
		}
	}

	if strings.Contains(output, "```") {
		start := strings.Index(output, "```") + 3
		end := strings.Index(output[start:], "```")
		if end != -1 {
			return strings.TrimSpace(output[start : start+end])
		}
	}

	return output
}

// BuildInsightPrompt builds and returns the prompt that would be used for insight generation
// without actually generating the report. Useful for preview/editing workflows.
func (qp *Processor) BuildInsightPrompt(taskID string, limit int, statusFilter string) (string, error) {
	log.Printf("Building insight prompt for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)

	// Load execution history
	history, err := qp.LoadExecutionHistory(taskID)
	if err != nil {
		return "", fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return "", fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return "", fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	// Load execution file contents (output and last message)
	executionDetails, err := qp.loadExecutionDetails(filtered)
	if err != nil {
		log.Printf("Warning: Failed to load some execution details: %v", err)
		// Continue with what we have
	}

	// Assemble the insight-generator prompt
	prompt, err := qp.assembleInsightPrompt(taskID, filtered, stats, executionDetails)
	if err != nil {
		return "", fmt.Errorf("assemble insight prompt: %w", err)
	}

	log.Printf("Built insight prompt (%d chars) for task %s", len(prompt), taskID)

	return prompt, nil
}

// GenerateInsightReportWithCustomPrompt generates an insight report using a custom prompt
// instead of the default template. This allows users to customize the analysis focus.
func (qp *Processor) GenerateInsightReportWithCustomPrompt(taskID string, limit int, statusFilter string, customPrompt string) (*InsightReport, error) {
	log.Printf("Generating insight report with custom prompt for task %s (limit: %d, filter: %s)", taskID, limit, statusFilter)
	systemlog.Infof("Starting custom insight generation for task %s", taskID)

	if customPrompt == "" {
		return nil, fmt.Errorf("custom prompt is required")
	}

	// Load execution history for metadata
	history, err := qp.LoadExecutionHistory(taskID)
	if err != nil {
		return nil, fmt.Errorf("load execution history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no execution history available for analysis")
	}

	// Filter executions based on status
	filtered := filterExecutionsByStatus(history, statusFilter, limit)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no executions matching filter: %s", statusFilter)
	}

	// Compute statistics
	stats := ComputeExecutionStatistics(filtered)

	log.Printf("Using custom insight prompt (%d chars) for task %s", len(customPrompt), taskID)

	// Call Claude Code with the custom prompt
	claudeResponse, err := qp.callClaudeCodeForInsight(customPrompt, taskID)
	if err != nil {
		return nil, fmt.Errorf("call Claude Code: %w", err)
	}

	if !claudeResponse.Success {
		return nil, fmt.Errorf("Claude Code execution failed: %s", claudeResponse.Error)
	}

	// Parse JSON response into InsightReport
	report, err := parseInsightResponse(claudeResponse.Output, taskID, filtered, stats)
	if err != nil {
		// Try to extract JSON from the output if it's wrapped in markdown
		cleanedOutput := extractJSONFromMarkdown(claudeResponse.Output)
		report, err = parseInsightResponse(cleanedOutput, taskID, filtered, stats)
		if err != nil {
			return nil, fmt.Errorf("parse insight response: %w\nOutput: %s", err, claudeResponse.Output)
		}
	}

	// Enrich report with metadata
	report.GeneratedBy = "insight-generator"
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}
	if report.ID == "" {
		// ID will be generated when saved
	}

	// Save the report
	if err := qp.SaveInsightReport(*report); err != nil {
		return nil, fmt.Errorf("save insight report: %w", err)
	}

	systemlog.Infof("Generated custom insight report %s for task %s (%d patterns, %d suggestions)",
		report.ID, taskID, len(report.Patterns), len(report.Suggestions))

	return report, nil
}
