package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"app-issue-tracker-api/internal/agents"
	"app-issue-tracker-api/internal/logging"
	services "app-issue-tracker-api/internal/server/services"
	"app-issue-tracker-api/internal/utils"
)

type InvestigationService struct {
	server          *Server
	now             func() time.Time
	prompts         *PromptBuilder
	rateLimiter     *RateLimitManager
	restartScenario scenarioRestarter
}

type scenarioRestarter func(ctx context.Context, scenario string) (string, error)

func NewInvestigationService(server *Server) *InvestigationService {
	svc := &InvestigationService{
		server: server,
		now:    time.Now,
	}
	svc.prompts = NewPromptBuilder(server.config.ScenarioRoot)
	svc.rateLimiter = NewRateLimitManager(server, func() time.Time { return svc.now() })
	svc.restartScenario = svc.defaultScenarioRestarter
	return svc
}

func (svc *InvestigationService) defaultScenarioRestarter(ctx context.Context, scenario string) (string, error) {
	workingDir := svc.scenarioRestartWorkingDir()
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)
	cmd.Dir = workingDir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (svc *InvestigationService) scenarioRestartWorkingDir() string {
	scenarioRoot := strings.TrimSpace(svc.server.config.ScenarioRoot)
	if scenarioRoot == "" {
		return "."
	}
	cleanRoot := filepath.Clean(scenarioRoot)
	parent := filepath.Dir(cleanRoot)
	if filepath.Base(parent) == "scenarios" {
		repoRoot := filepath.Dir(parent)
		if repoRoot != "" && repoRoot != "." {
			return repoRoot
		}
	}
	return parent
}

// transitionIssueAndPublish moves an issue to a new status and publishes the corresponding event
// Returns the actual status the issue was moved to (empty string if move failed)
func (svc *InvestigationService) transitionIssueAndPublish(
	issueID, targetStatus string,
	eventType EventType,
	agentID string,
	success bool,
	endTime time.Time,
	scenarioRestart *string,
) string {
	actualStatus := targetStatus
	if moveErr := svc.server.moveIssue(issueID, targetStatus); moveErr != nil {
		logging.LogErrorErr("Failed to move issue to status", moveErr, "issue_id", issueID, "target_status", targetStatus)
		actualStatus = ""
	}

	svc.server.hub.Publish(NewEvent(eventType, AgentCompletedData{
		IssueID:         issueID,
		AgentID:         agentID,
		Success:         success,
		EndTime:         endTime,
		NewStatus:       actualStatus,
		ScenarioRestart: scenarioRestart,
	}))

	return actualStatus
}

// attemptScenarioRestart tries to restart the scenario and returns a status string for the event
// Returns: nil if no scenario to restart, "success" if restart succeeded, "failed:<reason>" if restart failed
func (svc *InvestigationService) attemptScenarioRestart(issue *Issue, issueID string) *string {
	scenarioName := strings.TrimSpace(issue.AppID)
	if scenarioName == "" {
		return nil // No scenario to restart
	}

	restartCtx, cancel := context.WithTimeout(context.Background(), ScenarioRestartTimeout)
	defer cancel()

	output, restartErr := svc.restartScenario(restartCtx, scenarioName)
	trimmedOutput := strings.TrimSpace(utils.StripANSI(output))

	if restartErr != nil {
		failureReason := fmt.Sprintf("Failed to restart scenario '%s': %v", scenarioName, restartErr)
		if trimmedOutput != "" {
			failureReason = fmt.Sprintf("%s\nOutput: %s", failureReason, trimmedOutput)
		}
		logging.LogWarn("Scenario restart failed", "issue_id", issueID, "scenario", scenarioName, "error", restartErr)
		result := fmt.Sprintf("failed:%s", failureReason)
		return &result
	}

	if trimmedOutput != "" {
		logging.LogInfo("Scenario restart completed", "issue_id", issueID, "scenario", scenarioName, "output", trimmedOutput)
	} else {
		logging.LogInfo("Scenario restart completed", "issue_id", issueID, "scenario", scenarioName)
	}
	result := "success"
	return &result
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

	if persistErr := svc.server.writeIssueMetadata(issueDir, issue); persistErr != nil {
		logging.LogErrorErr("Failed to persist investigation failure state", persistErr, "issue_id", issueID)
	}

	svc.transitionIssueAndPublish(issueID, StatusFailed, EventAgentFailed, issue.Investigation.AgentID, false, nowUTC, nil)
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
	if currentFolder == StatusActive {
		return nil
	}

	if err := svc.server.moveIssue(issueID, StatusActive); err != nil {
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

	var scenarioRestart *string
	if loadErr == nil {
		services.MarkInvestigationCancelled(issue, reason, nowUTC)
		if persistErr := svc.server.writeIssueMetadata(issueDir, issue); persistErr != nil {
			logging.LogWarn("Failed to persist cancellation metadata", "issue_id", issueID, "error", persistErr)
		}

		// Restart scenario after cancellation to recover from potentially broken state
		scenarioRestart = svc.attemptScenarioRestart(issue, issueID)
	} else {
		logging.LogWarn("Failed to reload issue during cancellation", "issue_id", issueID, "error", loadErr)
	}

	svc.transitionIssueAndPublish(issueID, StatusOpen, EventAgentFailed, agentID, false, nowUTC, scenarioRestart)
}

func (svc *InvestigationService) handleInvestigationRateLimit(issueID, agentID string, result *ClaudeExecutionResult) bool {
	return svc.rateLimiter.Handle(issueID, agentID, result)
}

func (svc *InvestigationService) handleInvestigationFailure(issueID, agentID string, result *ClaudeExecutionResult) {
	logging.LogWarn("Investigation failed", "issue_id", issueID, "error", result.Error)

	nowUTC := svc.utcNow()
	var scenarioRestart *string

	issue, issueDir, _, loadErr := svc.server.loadIssueWithStatus(issueID)
	if loadErr == nil {
		services.RecordAgentExecutionFailure(issue, result.Error, nowUTC, result.TranscriptPath, result.LastMessagePath, result.MaxTurnsExceeded)
		issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)
		svc.server.writeIssueMetadata(issueDir, issue)

		// Restart scenario after failure to recover from potentially broken state
		scenarioRestart = svc.attemptScenarioRestart(issue, issueID)
	}

	svc.transitionIssueAndPublish(issueID, StatusFailed, EventAgentFailed, agentID, false, nowUTC, scenarioRestart)
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
		reportContent = strings.TrimSpace(result.Output)
	}

	// Restart scenario after successful investigation
	scenarioRestart := svc.attemptScenarioRestart(issue, issueID)

	services.MarkInvestigationSuccess(issue, reportContent, nowUTC, result.TranscriptPath, result.LastMessagePath)

	if persistErr := svc.server.writeIssueMetadata(issueDir, issue); persistErr != nil {
		logging.LogErrorErr("Failed to persist investigation results", persistErr, "issue_id", issueID)
	}

	svc.transitionIssueAndPublish(issueID, StatusCompleted, EventAgentCompleted, agentID, true, nowUTC, scenarioRestart)
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

// CleanupOldTranscripts removes old transcript files based on retention policy
func (svc *InvestigationService) CleanupOldTranscripts() {
	config := DefaultTranscriptCleanupConfig()
	if err := CleanupOldTranscripts(svc.server.config.ScenarioRoot, config); err != nil {
		logging.LogErrorErr("Failed to cleanup old transcripts", err)
	}

	// Also cleanup marker and prompt files from scenario root
	if err := CleanupCodexMarkerFiles(svc.server.config.ScenarioRoot); err != nil {
		logging.LogErrorErr("Failed to cleanup Codex marker files", err)
	}
}
