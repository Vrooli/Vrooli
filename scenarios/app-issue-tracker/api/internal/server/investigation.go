package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"app-issue-tracker-api/internal/agents"
	"app-issue-tracker-api/internal/logging"
	services "app-issue-tracker-api/internal/server/services"
)

type InvestigationService struct {
	server      *Server
	now         func() time.Time
	prompts     *PromptBuilder
	rateLimiter *RateLimitManager
}

func NewInvestigationService(server *Server) *InvestigationService {
	svc := &InvestigationService{
		server: server,
		now:    time.Now,
	}
	svc.prompts = NewPromptBuilder(server.config.ScenarioRoot)
	svc.rateLimiter = NewRateLimitManager(server, func() time.Time { return svc.now() })
	return svc
}

func (s *Server) loadPromptTemplate() string {
	return s.investigations.loadPromptTemplate()
}

func (s *Server) buildInvestigationPrompt(issue *Issue, issueDir, agentID, projectPath, timestamp string) string {
	return s.investigations.buildInvestigationPrompt(issue, issueDir, agentID, projectPath, timestamp)
}

func (s *Server) persistInvestigationStart(issue *Issue, issueDir, agentID, startedAt string) error {
	return s.investigations.persistInvestigationStart(issue, issueDir, agentID, startedAt)
}

func (s *Server) rateLimitStatus() RateLimitStatus {
	return s.investigations.RateLimitStatus()
}

func (s *Server) handleInvestigationRateLimit(issueID, agentID string, result *ClaudeExecutionResult) bool {
	return s.investigations.handleInvestigationRateLimit(issueID, agentID, result)
}

func (svc *InvestigationService) utcNow() time.Time {
	return svc.now().UTC()
}

func (svc *InvestigationService) loadPromptTemplate() string {
	return svc.prompts.LoadTemplate()
}

func (svc *InvestigationService) buildInvestigationPrompt(issue *Issue, issueDir, agentID, projectPath, timestamp string) string {
	return svc.prompts.BuildPrompt(issue, issueDir, agentID, projectPath, timestamp, "")
}

func (svc *InvestigationService) failInvestigation(issueID, errorMsg, output string) {
	issue, issueDir, _, err := svc.server.loadIssueWithStatus(issueID)
	if err != nil {
		logging.LogErrorErr("Cannot load issue to mark investigation failed", err, "issue_id", issueID)
		return
	}

	nowUTC := svc.utcNow()

	services.MarkInvestigationFailure(issue, errorMsg, output, nowUTC)

	if err := svc.server.writeIssueMetadata(issueDir, issue); err != nil {
		logging.LogErrorErr("Failed to persist investigation failure state", err, "issue_id", issueID)
	}

	newStatus := "failed"
	if moveErr := svc.server.moveIssue(issueID, "failed"); moveErr != nil {
		logging.LogErrorErr("Failed to move issue to failed status", moveErr, "issue_id", issueID)
		newStatus = ""
	}

	svc.server.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   issue.Investigation.AgentID,
		Success:   false,
		EndTime:   nowUTC,
		NewStatus: newStatus,
	}))
}

func (svc *InvestigationService) TriggerInvestigation(issueID, agentID string, autoResolve bool) error {
	issue, issueDir, currentFolder, err := svc.server.loadIssueWithStatus(issueID)
	if err != nil {
		if errors.Is(err, services.ErrIssueNotFound) {
			return err
		}
		return fmt.Errorf("failed to load issue: %w", err)
	}

	if agentID == "" {
		agentID = agents.UnifiedResolverID
	}

	logging.LogInfo("Starting investigation", "issue_id", issueID, "agent_id", agentID, "auto_resolve", autoResolve)

	startedAt := svc.utcNow().Format(time.RFC3339)
	if err := svc.persistInvestigationStart(issue, issueDir, agentID, startedAt); err != nil {
		return err
	}

	if err := svc.ensureIssueActive(issueID, currentFolder); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc.server.registerRunningProcess(issueID, agentID, startedAt, cancel)
	go svc.executeInvestigation(ctx, cancel, issueID, agentID, startedAt)

	return nil
}

func (svc *InvestigationService) persistInvestigationStart(issue *Issue, issueDir, agentID, startedAt string) error {
	startedAtTime, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		startedAtTime = svc.utcNow()
		startedAt = startedAtTime.Format(time.RFC3339)
	}

	services.MarkInvestigationStarted(issue, agentID, startedAtTime)

	if err := svc.server.writeIssueMetadata(issueDir, issue); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return nil
}

func (svc *InvestigationService) ensureIssueActive(issueID, currentFolder string) error {
	if currentFolder == "active" {
		return nil
	}

	if err := svc.server.moveIssue(issueID, "active"); err != nil {
		return fmt.Errorf("failed to move issue to active: %w", err)
	}

	return nil
}

func (svc *InvestigationService) executeInvestigation(ctx context.Context, cancel context.CancelFunc, issueID, agentID, startedAt string) {
	defer cancel()
	defer svc.server.unregisterRunningProcess(issueID)

	recordCompletion, flush := svc.newCompletionTracker()
	defer flush()

	prompt, err := svc.prepareInvestigationPrompt(issueID, agentID, startedAt)
	if err != nil {
		return
	}

	result, err := svc.runInvestigationAgent(ctx, prompt, issueID)
	if err != nil {
		logging.LogErrorErr("Claude execution error", err, "issue_id", issueID)
		svc.failInvestigation(issueID, fmt.Sprintf("Execution error: %v", err), "")
		return
	}

	svc.handleInvestigationOutcome(issueID, agentID, result)
	recordCompletion()
}

func (svc *InvestigationService) newCompletionTracker() (func(), func()) {
	recorded := false
	processedCount := 0
	record := func() {
		if recorded {
			return
		}
		processedCount = svc.server.incrementProcessedCount()
		recorded = true
	}
	flush := func() {
		if recorded {
			logging.LogInfo("Processor investigations updated", "count", processedCount)
		}
	}
	return record, flush
}

func (svc *InvestigationService) prepareInvestigationPrompt(issueID, agentID, startedAt string) (string, error) {
	issue, issueDir, _, loadErr := svc.server.loadIssueWithStatus(issueID)
	if loadErr != nil {
		logging.LogErrorErr("Investigation failed to load issue", loadErr, "issue_id", issueID)
		return "", loadErr
	}

	prompt := svc.buildInvestigationPrompt(issue, issueDir, agentID, svc.server.config.ScenarioRoot, startedAt)
	return prompt, nil
}

func (svc *InvestigationService) runInvestigationAgent(ctx context.Context, prompt, issueID string) (*ClaudeExecutionResult, error) {
	settings := GetAgentSettings()
	timeoutDuration := time.Duration(settings.TimeoutSeconds) * time.Second
	startTime := svc.now()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	result, err := svc.server.executeClaudeCode(timeoutCtx, prompt, issueID, startTime, timeoutDuration)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (svc *InvestigationService) handleInvestigationOutcome(issueID, agentID string, result *ClaudeExecutionResult) {
	if svc.wasInvestigationCancelled(issueID, agentID) {
		return
	}
	if svc.handleInvestigationRateLimit(issueID, agentID, result) {
		return
	}
	if !result.Success {
		svc.handleInvestigationFailure(issueID, agentID, result)
		return
	}
	svc.handleInvestigationSuccess(issueID, agentID, result)
}

func (svc *InvestigationService) wasInvestigationCancelled(issueID, agentID string) bool {
	if cancelled, reason := svc.server.processor.CancellationInfo(issueID); cancelled {
		svc.handleInvestigationCancellation(issueID, agentID, reason)
		return true
	}
	return false
}

func (svc *InvestigationService) handleInvestigationCancellation(issueID, agentID, reason string) {
	logging.LogInfo("Investigation cancelled", "issue_id", issueID, "agent_id", agentID, "reason", reason)

	issue, issueDir, _, loadErr := svc.server.loadIssueWithStatus(issueID)
	nowUTC := svc.utcNow()
	if loadErr == nil {
		services.MarkInvestigationCancelled(issue, reason, nowUTC)
		if err := svc.server.writeIssueMetadata(issueDir, issue); err != nil {
			logging.LogWarn("Failed to persist cancellation metadata", "issue_id", issueID, "error", err)
		}
	} else {
		logging.LogWarn("Failed to reload issue during cancellation", "issue_id", issueID, "error", loadErr)
	}

	if err := svc.server.moveIssue(issueID, "open"); err != nil {
		logging.LogWarn("Failed to move cancelled issue to open", "issue_id", issueID, "error", err)
	}

	svc.server.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   false,
		EndTime:   nowUTC,
		NewStatus: "open",
	}))
}

func (svc *InvestigationService) handleInvestigationRateLimit(issueID, agentID string, result *ClaudeExecutionResult) bool {
	return svc.rateLimiter.Handle(issueID, agentID, result)
}

func (svc *InvestigationService) handleInvestigationFailure(issueID, agentID string, result *ClaudeExecutionResult) {
	logging.LogWarn("Investigation failed", "issue_id", issueID, "error", result.Error)

	issue, issueDir, _, loadErr := svc.server.loadIssueWithStatus(issueID)
	if loadErr == nil {
		timestamp := svc.utcNow()
		services.RecordAgentExecutionFailure(issue, result.Error, timestamp, result.TranscriptPath, result.LastMessagePath, result.MaxTurnsExceeded)
		issue.Metadata.UpdatedAt = timestamp.Format(time.RFC3339)
		svc.server.writeIssueMetadata(issueDir, issue)
	}

	newStatus := "failed"
	if moveErr := svc.server.moveIssue(issueID, "failed"); moveErr != nil {
		logging.LogErrorErr("Failed to move issue to failed status after investigation error", moveErr, "issue_id", issueID)
		newStatus = ""
	}
	svc.server.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   false,
		EndTime:   svc.now(),
		NewStatus: newStatus,
	}))
}

func (svc *InvestigationService) handleInvestigationSuccess(issueID, agentID string, result *ClaudeExecutionResult) {
	logging.LogInfo("Investigation completed successfully", "issue_id", issueID, "execution_time", result.ExecutionTime.Round(time.Second))

	issue, issueDir, _, loadErr := svc.server.loadIssueWithStatus(issueID)
	if loadErr != nil {
		logging.LogErrorErr("Failed to reload issue after investigation", loadErr, "issue_id", issueID)
		return
	}

	nowUTC := svc.utcNow()

	reportContent := strings.TrimSpace(result.LastMessage)
	if reportContent == "" {
		reportContent = result.Output
	}
	services.MarkInvestigationSuccess(issue, reportContent, nowUTC, result.TranscriptPath, result.LastMessagePath)

	if saveErr := svc.server.writeIssueMetadata(issueDir, issue); saveErr != nil {
		logging.LogErrorErr("Failed to persist investigation results", saveErr, "issue_id", issueID)
	}

	if moveErr := svc.server.moveIssue(issueID, "completed"); moveErr != nil {
		logging.LogErrorErr("Failed to move issue to completed", moveErr, "issue_id", issueID)
	}

	svc.server.hub.Publish(NewEvent(EventAgentCompleted, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   true,
		EndTime:   nowUTC,
		NewStatus: "completed",
	}))
}

func (svc *InvestigationService) RateLimitStatus() RateLimitStatus {
	return svc.rateLimiter.Status()
}

func (svc *InvestigationService) ClearExpiredRateLimitMetadata() {
	svc.rateLimiter.ClearExpired()
}

func (svc *InvestigationService) ClearRateLimitMetadata(issueID string) {
	svc.rateLimiter.Clear(issueID)
}
