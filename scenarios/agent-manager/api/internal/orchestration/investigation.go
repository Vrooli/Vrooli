// Package orchestration provides investigation service methods.
//
// Investigation enables agent-manager to analyze its own runs to diagnose
// errors, identify patterns, and recommend improvements. The investigation
// agent runs as a self-contained task/run pair that produces a structured report.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// InvestigationService provides investigation operations.
// These methods extend the core Orchestrator with self-investigation capabilities.
type InvestigationService interface {
	// TriggerInvestigation starts a new investigation of the specified runs.
	TriggerInvestigation(ctx context.Context, req *domain.CreateInvestigationRequest) (*domain.Investigation, error)

	// GetInvestigation retrieves an investigation by ID.
	GetInvestigation(ctx context.Context, id uuid.UUID) (*domain.Investigation, error)

	// ListInvestigations retrieves investigations with optional filtering.
	ListInvestigations(ctx context.Context, filter repository.InvestigationListFilter) ([]*domain.Investigation, error)

	// GetActiveInvestigation returns any currently running investigation.
	GetActiveInvestigation(ctx context.Context) (*domain.Investigation, error)

	// StopInvestigation stops a running investigation.
	StopInvestigation(ctx context.Context, id uuid.UUID) error

	// DeleteInvestigation removes an investigation by ID.
	DeleteInvestigation(ctx context.Context, id uuid.UUID) error

	// ApplyFixes creates a new investigation to apply selected recommendations.
	ApplyFixes(ctx context.Context, investigationID uuid.UUID, req *domain.ApplyFixesRequest) (*domain.Investigation, error)
}

// investigationOrchestrator extends the base orchestrator with investigation capabilities.
type investigationOrchestrator struct {
	*Orchestrator
	investigations repository.InvestigationRepository
}

// NewInvestigationOrchestrator creates an investigation orchestrator wrapper.
func NewInvestigationOrchestrator(base *Orchestrator, investigations repository.InvestigationRepository) InvestigationService {
	return &investigationOrchestrator{
		Orchestrator:   base,
		investigations: investigations,
	}
}

// TriggerInvestigation starts a new investigation.
func (o *investigationOrchestrator) TriggerInvestigation(ctx context.Context, req *domain.CreateInvestigationRequest) (*domain.Investigation, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check for existing active investigation
	active, err := o.investigations.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, domain.NewStateError("Investigation", string(active.Status), "create",
			"an investigation is already running")
	}

	// Verify all runs exist
	for _, runID := range req.RunIDs {
		if _, err := o.GetRun(ctx, runID); err != nil {
			return nil, err
		}
	}

	// Create investigation record
	investigation := &domain.Investigation{
		ID:             uuid.New(),
		RunIDs:         req.RunIDs,
		Status:         domain.InvestigationStatusPending,
		AnalysisType:   req.AnalysisType,
		ReportSections: req.ReportSections,
		CustomContext:  req.CustomContext,
		Progress:       0,
		CreatedAt:      time.Now(),
	}

	if err := o.investigations.Create(ctx, investigation); err != nil {
		return nil, err
	}

	// Start investigation in background
	go o.runInvestigation(context.Background(), investigation)

	return investigation, nil
}

// GetInvestigation retrieves an investigation by ID.
func (o *investigationOrchestrator) GetInvestigation(ctx context.Context, id uuid.UUID) (*domain.Investigation, error) {
	investigation, err := o.investigations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return investigation, nil
}

// ListInvestigations retrieves investigations with optional filtering.
func (o *investigationOrchestrator) ListInvestigations(ctx context.Context, filter repository.InvestigationListFilter) ([]*domain.Investigation, error) {
	return o.investigations.List(ctx, filter)
}

// GetActiveInvestigation returns any currently running investigation.
func (o *investigationOrchestrator) GetActiveInvestigation(ctx context.Context) (*domain.Investigation, error) {
	return o.investigations.GetActive(ctx)
}

// StopInvestigation stops a running investigation.
func (o *investigationOrchestrator) StopInvestigation(ctx context.Context, id uuid.UUID) error {
	investigation, err := o.investigations.Get(ctx, id)
	if err != nil {
		return err
	}

	if investigation.IsTerminal() {
		return domain.NewStateError("Investigation", string(investigation.Status), "stop",
			"investigation is already in a terminal state")
	}

	// Stop the agent run if one exists
	if investigation.AgentRunID != nil {
		if err := o.StopRun(ctx, *investigation.AgentRunID); err != nil {
			// Log but continue - we still want to mark investigation as cancelled
			_ = err
		}
	}

	return o.investigations.UpdateStatus(ctx, id, domain.InvestigationStatusCancelled, "cancelled by user")
}

// DeleteInvestigation removes an investigation by ID.
func (o *investigationOrchestrator) DeleteInvestigation(ctx context.Context, id uuid.UUID) error {
	investigation, err := o.investigations.Get(ctx, id)
	if err != nil {
		return err
	}

	if investigation.IsActive() {
		return domain.NewStateError("Investigation", string(investigation.Status), "delete",
			"cannot delete an active investigation - stop it first")
	}

	return o.investigations.Delete(ctx, id)
}

// ApplyFixes creates a new investigation to apply selected recommendations.
func (o *investigationOrchestrator) ApplyFixes(ctx context.Context, investigationID uuid.UUID, req *domain.ApplyFixesRequest) (*domain.Investigation, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get the source investigation
	source, err := o.investigations.Get(ctx, investigationID)
	if err != nil {
		return nil, err
	}

	if source.Status != domain.InvestigationStatusCompleted {
		return nil, domain.NewStateError("Investigation", string(source.Status), "apply_fixes",
			"can only apply fixes from a completed investigation")
	}

	if source.Findings == nil || len(source.Findings.Recommendations) == 0 {
		return nil, domain.NewValidationError("findings", "investigation has no recommendations")
	}

	// Verify requested recommendations exist
	recommendationMap := make(map[string]domain.Recommendation)
	for _, rec := range source.Findings.Recommendations {
		recommendationMap[rec.ID] = rec
	}

	var selectedRecs []domain.Recommendation
	for _, recID := range req.RecommendationIDs {
		rec, exists := recommendationMap[recID]
		if !exists {
			return nil, domain.NewValidationError("recommendationIds",
				fmt.Sprintf("recommendation %q not found in investigation", recID))
		}
		selectedRecs = append(selectedRecs, rec)
	}

	// Create a new investigation for applying fixes
	fixInvestigation := &domain.Investigation{
		ID:     uuid.New(),
		RunIDs: source.RunIDs, // Same runs as the source
		Status: domain.InvestigationStatusPending,
		AnalysisType: domain.AnalysisType{
			ErrorDiagnosis: false, // Fix application mode
		},
		ReportSections: domain.ReportSections{
			RootCauseEvidence: false,
			Recommendations:   true,
			MetricsSummary:    false,
		},
		CustomContext:         buildFixApplicationContext(selectedRecs, req.Note),
		Progress:              0,
		SourceInvestigationID: &investigationID,
		CreatedAt:             time.Now(),
	}

	if err := o.investigations.Create(ctx, fixInvestigation); err != nil {
		return nil, err
	}

	// Start fix application in background
	go o.runFixApplication(context.Background(), fixInvestigation, selectedRecs)

	return fixInvestigation, nil
}

// runInvestigation executes the investigation workflow in a background goroutine.
func (o *investigationOrchestrator) runInvestigation(ctx context.Context, investigation *domain.Investigation) {
	// Update status to running
	if err := o.investigations.UpdateStatus(ctx, investigation.ID, domain.InvestigationStatusRunning, ""); err != nil {
		return
	}

	// Fetch run data for all runs
	runs := make([]*domain.Run, 0, len(investigation.RunIDs))
	eventsMap := make(map[uuid.UUID][]*domain.RunEvent)

	for _, runID := range investigation.RunIDs {
		run, err := o.GetRun(ctx, runID)
		if err != nil {
			o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to fetch run %s: %v", runID, err))
			return
		}
		runs = append(runs, run)

		// Fetch events for this run
		if o.events != nil {
			events, err := o.events.Get(ctx, runID, event.GetOptions{Limit: 1000})
			if err != nil {
				o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to fetch events for run %s: %v", runID, err))
				return
			}
			eventsMap[runID] = events
		}
	}

	// Update progress
	_ = o.investigations.UpdateProgress(ctx, investigation.ID, 10)

	// Build the investigation prompt
	prompt := o.buildInvestigationPrompt(investigation, runs, eventsMap)

	// Create a task for the investigation
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       fmt.Sprintf("Investigation %s", investigation.ID.String()[:8]),
		Description: prompt,
		ScopePath:   ".", // Investigation doesn't need file access
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if _, err := o.CreateTask(ctx, task); err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to create investigation task: %v", err))
		return
	}

	// Update progress
	_ = o.investigations.UpdateProgress(ctx, investigation.ID, 20)

	// Create a run for the investigation (uses default investigator profile)
	runReq := CreateRunRequest{
		TaskID: task.ID,
		Tag:    fmt.Sprintf("investigation-%s", investigation.ID.String()[:8]),
		Force:  true, // Investigations always run
	}

	run, err := o.CreateRun(ctx, runReq)
	if err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to create investigation run: %v", err))
		return
	}

	// Update investigation with agent run ID
	investigation.AgentRunID = &run.ID
	if err := o.investigations.Update(ctx, investigation); err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to update investigation: %v", err))
		return
	}

	// Update progress
	_ = o.investigations.UpdateProgress(ctx, investigation.ID, 30)

	// Poll for run completion
	o.pollInvestigatorRun(ctx, investigation, run.ID)
}

// pollInvestigatorRun polls the agent run until completion.
func (o *investigationOrchestrator) pollInvestigatorRun(ctx context.Context, investigation *domain.Investigation, runID uuid.UUID) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			o.failInvestigation(ctx, investigation.ID, "investigation cancelled")
			return

		case <-timeout:
			o.failInvestigation(ctx, investigation.ID, "investigation timed out after 30 minutes")
			return

		case <-ticker.C:
			run, err := o.GetRun(ctx, runID)
			if err != nil {
				o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to fetch run status: %v", err))
				return
			}

			// Update progress based on run progress
			progress := 30 + (run.ProgressPercent * 60 / 100)
			if progress > 90 {
				progress = 90
			}
			_ = o.investigations.UpdateProgress(ctx, investigation.ID, progress)

			// Broadcast progress if broadcaster available
			if o.broadcaster != nil {
				o.broadcaster.BroadcastProgress(investigation.ID, domain.RunPhase(investigation.Status), progress, "Analyzing runs...")
			}

			switch run.Status {
			case domain.RunStatusComplete:
				o.completeInvestigation(ctx, investigation, run)
				return

			case domain.RunStatusFailed:
				errorMsg := "investigation agent failed"
				if run.ErrorMsg != "" {
					errorMsg = run.ErrorMsg
				}
				o.failInvestigation(ctx, investigation.ID, errorMsg)
				return

			case domain.RunStatusCancelled:
				o.failInvestigation(ctx, investigation.ID, "investigation agent was cancelled")
				return
			}
		}
	}
}

// completeInvestigation parses the agent output and stores findings.
func (o *investigationOrchestrator) completeInvestigation(ctx context.Context, investigation *domain.Investigation, run *domain.Run) {
	// Get the final events to extract the report
	var report *domain.InvestigationReport
	var metrics *domain.MetricsData

	if o.events != nil {
		events, err := o.events.Get(ctx, run.ID, event.GetOptions{
			EventTypes: []domain.RunEventType{domain.EventTypeMessage, domain.EventTypeToolResult},
			Limit:      100,
		})
		if err == nil {
			report, metrics = parseInvestigationOutput(events)
		}
	}

	// If no structured report found, create a basic one
	if report == nil {
		report = &domain.InvestigationReport{
			Summary: "Investigation completed. Review agent output for details.",
		}
	}

	if err := o.investigations.UpdateFindings(ctx, investigation.ID, report, metrics); err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to store findings: %v", err))
		return
	}
}

// failInvestigation marks the investigation as failed.
func (o *investigationOrchestrator) failInvestigation(ctx context.Context, id uuid.UUID, errorMsg string) {
	_ = o.investigations.UpdateStatus(ctx, id, domain.InvestigationStatusFailed, errorMsg)
}

// runFixApplication executes the fix application workflow.
func (o *investigationOrchestrator) runFixApplication(ctx context.Context, investigation *domain.Investigation, recommendations []domain.Recommendation) {
	// Update status to running
	if err := o.investigations.UpdateStatus(ctx, investigation.ID, domain.InvestigationStatusRunning, ""); err != nil {
		return
	}

	// Build fix application prompt
	prompt := o.buildFixApplicationPrompt(investigation, recommendations)

	// Create a task for applying fixes
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       fmt.Sprintf("Apply Fixes %s", investigation.ID.String()[:8]),
		Description: prompt,
		ScopePath:   ".", // May need access to files for code fixes
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if _, err := o.CreateTask(ctx, task); err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to create fix task: %v", err))
		return
	}

	// Create a run for applying fixes
	runReq := CreateRunRequest{
		TaskID: task.ID,
		Tag:    fmt.Sprintf("apply-fixes-%s", investigation.ID.String()[:8]),
		Force:  true,
	}

	run, err := o.CreateRun(ctx, runReq)
	if err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to create fix run: %v", err))
		return
	}

	// Update investigation with agent run ID
	investigation.AgentRunID = &run.ID
	if err := o.investigations.Update(ctx, investigation); err != nil {
		o.failInvestigation(ctx, investigation.ID, fmt.Sprintf("failed to update investigation: %v", err))
		return
	}

	// Poll for completion
	o.pollInvestigatorRun(ctx, investigation, run.ID)
}

// buildInvestigationPrompt constructs the prompt for the investigation agent.
func (o *investigationOrchestrator) buildInvestigationPrompt(
	investigation *domain.Investigation,
	runs []*domain.Run,
	eventsMap map[uuid.UUID][]*domain.RunEvent,
) string {
	var sb strings.Builder

	// Context section
	sb.WriteString("# Investigation Task\n\n")
	sb.WriteString("You are investigating agent-manager runs to identify issues and patterns. ")
	sb.WriteString("Analyze the following runs and provide a structured report.\n\n")

	// Analysis type instructions
	sb.WriteString("## Analysis Focus\n\n")
	if investigation.AnalysisType.ErrorDiagnosis {
		sb.WriteString("- **Error Diagnosis**: Identify root causes of failures, errors, and exceptions.\n")
	}
	if investigation.AnalysisType.EfficiencyAnalysis {
		sb.WriteString("- **Efficiency Analysis**: Analyze resource usage, timing, and optimization opportunities.\n")
	}
	if investigation.AnalysisType.ToolUsagePatterns {
		sb.WriteString("- **Tool Usage Patterns**: Examine which tools were used, how often, and effectiveness.\n")
	}
	sb.WriteString("\n")

	// Custom context
	if investigation.CustomContext != "" {
		sb.WriteString("## Additional Context\n\n")
		sb.WriteString(investigation.CustomContext)
		sb.WriteString("\n\n")
	}

	// Runs summary
	sb.WriteString("## Runs Under Investigation\n\n")
	for _, run := range runs {
		sb.WriteString(fmt.Sprintf("### Run %s\n", run.ID.String()[:8]))
		sb.WriteString(fmt.Sprintf("- **Status**: %s\n", run.Status))
		sb.WriteString(fmt.Sprintf("- **Phase**: %s\n", run.Phase))
		if run.ErrorMsg != "" {
			sb.WriteString(fmt.Sprintf("- **Error**: %s\n", run.ErrorMsg))
		}
		if run.StartedAt != nil && run.EndedAt != nil {
			duration := run.EndedAt.Sub(*run.StartedAt)
			sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", duration.Round(time.Second)))
		}
		if run.ResolvedConfig != nil {
			sb.WriteString(fmt.Sprintf("- **Runner**: %s\n", run.ResolvedConfig.RunnerType))
			if run.ResolvedConfig.Model != "" {
				sb.WriteString(fmt.Sprintf("- **Model**: %s\n", run.ResolvedConfig.Model))
			}
		}
		sb.WriteString("\n")

		// Include relevant events
		if events, ok := eventsMap[run.ID]; ok && len(events) > 0 {
			sb.WriteString("#### Events\n\n")
			for _, evt := range events {
				o.writeEventSummary(&sb, evt)
			}
			sb.WriteString("\n")
		}
	}

	// Metrics instructions
	sb.WriteString("## Metrics Recording\n\n")
	sb.WriteString("Use the `record_metric` tool to record any computed metrics. ")
	sb.WriteString("Do NOT compute metrics in your response text - always use the tool.\n\n")
	sb.WriteString("Available categories: rate, count, duration, cost, custom\n\n")
	sb.WriteString("Example metrics to record:\n")
	sb.WriteString("- success_rate (rate)\n")
	sb.WriteString("- avg_duration_seconds (duration)\n")
	sb.WriteString("- total_tokens_used (count)\n")
	sb.WriteString("- tool_call_count (count)\n\n")

	// Output format
	sb.WriteString("## Output Format\n\n")
	sb.WriteString("Provide your findings as a JSON object with this structure:\n\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"summary\": \"Brief overview of findings\",\n")
	if investigation.ReportSections.RootCauseEvidence {
		sb.WriteString("  \"rootCause\": {\n")
		sb.WriteString("    \"description\": \"Root cause description\",\n")
		sb.WriteString("    \"evidence\": [{\"runId\": \"...\", \"description\": \"...\", \"snippet\": \"...\"}],\n")
		sb.WriteString("    \"confidence\": \"high|medium|low\"\n")
		sb.WriteString("  },\n")
	}
	if investigation.ReportSections.Recommendations {
		sb.WriteString("  \"recommendations\": [\n")
		sb.WriteString("    {\n")
		sb.WriteString("      \"id\": \"rec-1\",\n")
		sb.WriteString("      \"priority\": \"critical|high|medium|low\",\n")
		sb.WriteString("      \"title\": \"Short title\",\n")
		sb.WriteString("      \"description\": \"Detailed recommendation\",\n")
		sb.WriteString("      \"actionType\": \"prompt_change|profile_config|code_fix\"\n")
		sb.WriteString("    }\n")
		sb.WriteString("  ]\n")
	}
	sb.WriteString("}\n")
	sb.WriteString("```\n")

	return sb.String()
}

// writeEventSummary writes a summarized event to the string builder.
func (o *investigationOrchestrator) writeEventSummary(sb *strings.Builder, evt *domain.RunEvent) {
	switch evt.EventType {
	case domain.EventTypeError:
		if data, ok := evt.Data.(*domain.ErrorEventData); ok {
			sb.WriteString(fmt.Sprintf("- [ERROR] %s: %s\n", data.Code, data.Message))
		}
	case domain.EventTypeToolCall:
		if data, ok := evt.Data.(*domain.ToolCallEventData); ok {
			sb.WriteString(fmt.Sprintf("- [TOOL] %s\n", data.ToolName))
		}
	case domain.EventTypeStatus:
		if data, ok := evt.Data.(*domain.StatusEventData); ok {
			sb.WriteString(fmt.Sprintf("- [STATUS] %s -> %s\n", data.OldStatus, data.NewStatus))
		}
	case domain.EventTypeMessage:
		if data, ok := evt.Data.(*domain.MessageEventData); ok {
			// Truncate long messages
			content := data.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("- [MSG] %s: %s\n", data.Role, content))
		}
	}
}

// buildFixApplicationPrompt constructs the prompt for applying fixes.
func (o *investigationOrchestrator) buildFixApplicationPrompt(
	investigation *domain.Investigation,
	recommendations []domain.Recommendation,
) string {
	var sb strings.Builder

	sb.WriteString("# Apply Investigation Recommendations\n\n")
	sb.WriteString("You are applying recommendations from a previous investigation. ")
	sb.WriteString("Implement the following fixes:\n\n")

	for i, rec := range recommendations {
		sb.WriteString(fmt.Sprintf("## Recommendation %d: %s\n\n", i+1, rec.Title))
		sb.WriteString(fmt.Sprintf("- **Priority**: %s\n", rec.Priority))
		sb.WriteString(fmt.Sprintf("- **Action Type**: %s\n", rec.ActionType))
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n\n", rec.Description))
	}

	if investigation.CustomContext != "" {
		sb.WriteString("## Additional Notes\n\n")
		sb.WriteString(investigation.CustomContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Implement each recommendation based on its action type\n")
	sb.WriteString("2. For prompt_change: Modify prompts or system instructions\n")
	sb.WriteString("3. For profile_config: Update agent profile settings\n")
	sb.WriteString("4. For code_fix: Make code changes to address the issue\n")
	sb.WriteString("5. Document what was changed and verify the fix\n")

	return sb.String()
}

// buildFixApplicationContext creates context for fix application investigations.
func buildFixApplicationContext(recommendations []domain.Recommendation, note string) string {
	var parts []string

	parts = append(parts, "Applying recommendations:")
	for _, rec := range recommendations {
		parts = append(parts, fmt.Sprintf("- [%s] %s", rec.Priority, rec.Title))
	}

	if note != "" {
		parts = append(parts, "", "User note:", note)
	}

	return strings.Join(parts, "\n")
}

// parseInvestigationOutput extracts structured report and metrics from events.
func parseInvestigationOutput(events []*domain.RunEvent) (*domain.InvestigationReport, *domain.MetricsData) {
	var report *domain.InvestigationReport
	metrics := &domain.MetricsData{
		ToolUsageBreakdown: make(map[string]int),
		ErrorTypeBreakdown: make(map[string]int),
		CustomMetrics:      make(map[string]float64),
	}

	for _, evt := range events {
		switch evt.EventType {
		case domain.EventTypeMessage:
			// Try to parse JSON report from message content
			if data, ok := evt.Data.(*domain.MessageEventData); ok && data.Role == "assistant" {
				if parsed := tryParseReport(data.Content); parsed != nil {
					report = parsed
				}
			}

		case domain.EventTypeToolCall:
			// Look for record_metric calls (input contains the metric data)
			if data, ok := evt.Data.(*domain.ToolCallEventData); ok {
				if metric := tryParseMetricFromToolCall(data); metric != nil {
					applyMetric(metrics, metric)
				}
			}
		}
	}

	if len(metrics.ToolUsageBreakdown) == 0 && len(metrics.CustomMetrics) == 0 {
		return report, nil
	}

	return report, metrics
}

// tryParseReport attempts to extract a JSON report from message content.
func tryParseReport(content string) *domain.InvestigationReport {
	// Look for JSON block in content
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	jsonStr := content[start : end+1]
	var report domain.InvestigationReport
	if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
		return nil
	}

	return &report
}

// tryParseMetricFromToolCall attempts to extract a metric from tool call data.
func tryParseMetricFromToolCall(data *domain.ToolCallEventData) *domain.RecordedMetric {
	if data.ToolName != "record_metric" {
		return nil
	}

	// Parse the input as a recorded metric
	var metric domain.RecordedMetric
	inputBytes, err := json.Marshal(data.Input)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(inputBytes, &metric); err != nil {
		return nil
	}

	return &metric
}

// applyMetric applies a recorded metric to the metrics data.
func applyMetric(metrics *domain.MetricsData, metric *domain.RecordedMetric) {
	switch metric.Category {
	case "rate":
		if metric.Name == "success_rate" {
			metrics.SuccessRate = metric.Value
		} else {
			metrics.CustomMetrics[metric.Name] = metric.Value
		}
	case "count":
		switch metric.Name {
		case "total_runs":
			metrics.TotalRuns = int(metric.Value)
		case "total_tokens_used":
			metrics.TotalTokensUsed = int64(metric.Value)
		default:
			metrics.CustomMetrics[metric.Name] = metric.Value
		}
	case "duration":
		if metric.Name == "avg_duration_seconds" {
			metrics.AvgDurationSeconds = metric.Value
		} else {
			metrics.CustomMetrics[metric.Name] = metric.Value
		}
	case "cost":
		if metric.Name == "total_cost" {
			metrics.TotalCost = metric.Value
		} else {
			metrics.CustomMetrics[metric.Name] = metric.Value
		}
	default:
		metrics.CustomMetrics[metric.Name] = metric.Value
	}
}
