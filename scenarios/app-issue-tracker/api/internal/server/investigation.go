package server

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	promptTemplatePath = "prompts/unified-resolver.md"
	fallbackValue      = "(not provided)"
)

func (s *Server) loadPromptTemplate() string {
	templatePath := filepath.Join(s.config.ScenarioRoot, promptTemplatePath)
	data, err := os.ReadFile(templatePath)
	if err == nil {
		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			return string(data)
		}
	}
	return "You are an elite software engineer. Provide investigation findings based on the supplied metadata."
}

func safeValue(str string) string {
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return fallbackValue
	}
	return trimmed
}

func joinList(items []string) string {
	if len(items) == 0 {
		return fallbackValue
	}
	trimmed := make([]string, 0, len(items))
	for _, item := range items {
		if s := strings.TrimSpace(item); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		return fallbackValue
	}
	return strings.Join(trimmed, ", ")
}

func (s *Server) readIssueMetadataRaw(issueDir string, issue *Issue) string {
	if issueDir != "" {
		path := filepath.Join(issueDir, metadataFilename)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	if issue == nil {
		return fallbackValue
	}
	if marshaled, err := yaml.Marshal(issue); err == nil {
		return string(marshaled)
	}
	return fallbackValue
}

func (s *Server) listArtifactPaths(issueDir string, issue *Issue) string {
	var paths []string

	if issueDir != "" {
		artifactsDir := filepath.Join(issueDir, artifactsDirName)
		if info, err := os.Stat(artifactsDir); err == nil && info.IsDir() {
			filepath.WalkDir(artifactsDir, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				rel, relErr := filepath.Rel(issueDir, path)
				if relErr != nil {
					return nil
				}
				normalized := filepath.ToSlash(rel)
				paths = append(paths, normalized)
				return nil
			})
		}
	}

	if len(paths) == 0 && issue != nil {
		for _, att := range issue.Attachments {
			if p := strings.TrimSpace(att.Path); p != "" {
				normalized := filepath.ToSlash(p)
				paths = append(paths, normalized)
			}
		}
	}

	if len(paths) == 0 {
		return fallbackValue
	}

	sort.Strings(paths)
	for i, p := range paths {
		paths[i] = "- " + p
	}
	return strings.Join(paths, "\n")
}

func (s *Server) buildInvestigationPrompt(issue *Issue, issueDir, agentID, projectPath, timestamp string) string {
	template := s.loadPromptTemplate()

	replacements := map[string]string{
		"{{issue_id}}":          fallbackValue,
		"{{issue_title}}":       fallbackValue,
		"{{issue_description}}": fallbackValue,
		"{{issue_type}}":        fallbackValue,
		"{{issue_priority}}":    fallbackValue,
		"{{app_name}}":          fallbackValue,
		"{{error_message}}":     fallbackValue,
		"{{stack_trace}}":       fallbackValue,
		"{{affected_files}}":    fallbackValue,
		"{{issue_metadata}}":    fallbackValue,
		"{{issue_artifacts}}":   fallbackValue,
		"{{agent_id}}":          fallbackValue,
		"{{project_path}}":      fallbackValue,
		"{{timestamp}}":         fallbackValue,
	}

	if issue != nil {
		replacements["{{issue_id}}"] = safeValue(issue.ID)
		replacements["{{issue_title}}"] = safeValue(issue.Title)
		replacements["{{issue_description}}"] = safeValue(issue.Description)
		replacements["{{issue_type}}"] = safeValue(issue.Type)
		replacements["{{issue_priority}}"] = safeValue(issue.Priority)
		replacements["{{app_name}}"] = safeValue(issue.AppID)
		replacements["{{error_message}}"] = safeValue(issue.ErrorContext.ErrorMessage)
		replacements["{{stack_trace}}"] = safeValue(issue.ErrorContext.StackTrace)
		replacements["{{affected_files}}"] = safeValue(joinList(issue.ErrorContext.AffectedFiles))
	}

	replacements["{{issue_metadata}}"] = safeValue(s.readIssueMetadataRaw(issueDir, issue))
	replacements["{{issue_artifacts}}"] = safeValue(s.listArtifactPaths(issueDir, issue))
	replacements["{{agent_id}}"] = safeValue(agentID)
	replacements["{{project_path}}"] = safeValue(projectPath)
	replacements["{{timestamp}}"] = safeValue(timestamp)

	result := template
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func (s *Server) failInvestigation(issueID, errorMsg, output string) {
	issue, issueDir, _, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		LogErrorErr("Cannot load issue to mark investigation failed", err, "issue_id", issueID)
		return
	}

	nowUTC := time.Now().UTC()

	if output != "" {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s\n\nOutput:\n%s", errorMsg, output)
	} else {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s", errorMsg)
	}
	issue.Investigation.CompletedAt = nowUTC.Format(time.RFC3339)
	issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)

	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	issue.Metadata.Extra["agent_last_error"] = errorMsg
	issue.Metadata.Extra[AgentStatusExtraKey] = AgentStatusFailed
	issue.Metadata.Extra[AgentStatusTimestampExtraKey] = nowUTC.Format(time.RFC3339)

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		LogErrorErr("Failed to persist investigation failure state", err, "issue_id", issueID)
	}

	s.moveIssue(issueID, "failed")

	s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   issue.Investigation.AgentID,
		Success:   false,
		EndTime:   nowUTC,
		NewStatus: "failed",
	}))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGKHf]`)
	return ansiRegex.ReplaceAllString(str, "")
}

func extractJSON(str string) string {
	start := strings.Index(str, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escape := false

	for i := start; i < len(str); i++ {
		c := str[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return str[start : i+1]
			}
		}
	}

	return ""
}

func detectRateLimit(output string) (bool, string) {
	outputLower := strings.ToLower(output)

	if strings.Contains(outputLower, "rate limit") ||
		strings.Contains(outputLower, "rate_limit") ||
		strings.Contains(outputLower, "quota exceeded") ||
		strings.Contains(outputLower, "too many requests") ||
		strings.Contains(outputLower, "429") {

		resetTime := ""

		isoPattern := regexp.MustCompile(`20\d{2}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})?`)
		if match := isoPattern.FindString(output); match != "" {
			if _, err := time.Parse(time.RFC3339, match); err == nil {
				resetTime = match
			} else if parsed, err := time.Parse("2006-01-02T15:04:05", match[:19]); err == nil {
				resetTime = parsed.UTC().Format(time.RFC3339)
			}
		}

		if resetTime == "" {
			resetTime = time.Now().Add(5 * time.Minute).Format(time.RFC3339)
		}

		return true, resetTime
	}

	return false, ""
}

func (s *Server) triggerInvestigation(issueID, agentID string, autoResolve bool) error {
	issue, issueDir, currentFolder, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		if strings.Contains(err.Error(), "issue not found") {
			return fmt.Errorf("issue not found: %w", err)
		}
		return fmt.Errorf("failed to load issue: %w", err)
	}

	if agentID == "" {
		agentID = "unified-resolver"
	}

	LogInfo("Starting investigation", "issue_id", issueID, "agent_id", agentID, "auto_resolve", autoResolve)

	startedAt := time.Now().UTC().Format(time.RFC3339)
	if err := s.persistInvestigationStart(issue, issueDir, agentID, startedAt); err != nil {
		return err
	}

	if err := s.ensureIssueActive(issueID, currentFolder); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.registerRunningProcess(issueID, agentID, startedAt, cancel)
	go s.executeInvestigation(ctx, cancel, issueID, agentID, startedAt)

	return nil
}

func (s *Server) persistInvestigationStart(issue *Issue, issueDir, agentID, startedAt string) error {
	issue.Investigation.AgentID = agentID
	issue.Investigation.StartedAt = startedAt
	issue.Metadata.UpdatedAt = startedAt

	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	delete(issue.Metadata.Extra, "agent_last_error")
	delete(issue.Metadata.Extra, "agent_cancel_reason")
	delete(issue.Metadata.Extra, "agent_transcript_path")
	delete(issue.Metadata.Extra, "agent_last_message_path")
	issue.Metadata.Extra[AgentStatusExtraKey] = AgentStatusRunning
	issue.Metadata.Extra[AgentStatusTimestampExtraKey] = startedAt

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return nil
}

func (s *Server) ensureIssueActive(issueID, currentFolder string) error {
	if currentFolder == "active" {
		return nil
	}

	if err := s.moveIssue(issueID, "active"); err != nil {
		return fmt.Errorf("failed to move issue to active: %w", err)
	}

	return nil
}

func (s *Server) executeInvestigation(ctx context.Context, cancel context.CancelFunc, issueID, agentID, startedAt string) {
	defer cancel()
	defer s.unregisterRunningProcess(issueID)

	issue, issueDir, _, loadErr := s.loadIssueWithStatus(issueID)
	if loadErr != nil {
		LogErrorErr("Investigation failed to load issue", loadErr, "issue_id", issueID)
		return
	}

	prompt := s.buildInvestigationPrompt(issue, issueDir, agentID, s.config.ScenarioRoot, startedAt)

	settings := GetAgentSettings()
	timeoutDuration := time.Duration(settings.TimeoutSeconds) * time.Second
	startTime := time.Now()

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeoutDuration)
	defer timeoutCancel()

	result, err := s.executeClaudeCode(timeoutCtx, prompt, issueID, startTime, timeoutDuration)
	if err != nil {
		LogErrorErr("Claude execution error", err, "issue_id", issueID)
		s.failInvestigation(issueID, fmt.Sprintf("Execution error: %v", err), "")
		return
	}

	if cancelled, reason := s.processor.cancellationInfo(issueID); cancelled {
		s.handleInvestigationCancellation(issueID, agentID, reason)
		return
	}

	if s.handleInvestigationRateLimit(issueID, agentID, result) {
		return
	}

	if !result.Success {
		s.handleInvestigationFailure(issueID, agentID, result)
		return
	}

	s.handleInvestigationSuccess(issueID, agentID, result)
}

func (s *Server) handleInvestigationCancellation(issueID, agentID, reason string) {
	LogInfo("Investigation cancelled", "issue_id", issueID, "agent_id", agentID, "reason", reason)

	issue, issueDir, _, loadErr := s.loadIssueWithStatus(issueID)
	nowUTC := time.Now().UTC()
	if loadErr == nil {
		if issue.Metadata.Extra == nil {
			issue.Metadata.Extra = make(map[string]string)
		}
		delete(issue.Metadata.Extra, "agent_last_error")
		issue.Metadata.Extra[AgentStatusExtraKey] = AgentStatusCancelled
		issue.Metadata.Extra[AgentStatusTimestampExtraKey] = nowUTC.Format(time.RFC3339)
		if strings.TrimSpace(reason) != "" {
			issue.Metadata.Extra["agent_cancel_reason"] = reason
		} else {
			delete(issue.Metadata.Extra, "agent_cancel_reason")
		}
		issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)
		if err := s.writeIssueMetadata(issueDir, issue); err != nil {
			LogWarn("Failed to persist cancellation metadata", "issue_id", issueID, "error", err)
		}
	} else {
		LogWarn("Failed to reload issue during cancellation", "issue_id", issueID, "error", loadErr)
	}

	if err := s.moveIssue(issueID, "open"); err != nil {
		LogWarn("Failed to move cancelled issue to open", "issue_id", issueID, "error", err)
	}

	s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   false,
		EndTime:   nowUTC,
		NewStatus: "open",
	}))
}

func (s *Server) handleInvestigationRateLimit(issueID, agentID string, result *ClaudeExecutionResult) bool {
	if !strings.Contains(result.Error, "RATE_LIMIT") {
		return false
	}

	isRateLimit, resetTime := detectRateLimit(result.Output)
	if !isRateLimit {
		return false
	}

	LogWarn("Rate limit detected during investigation", "issue_id", issueID, "reset_time", resetTime)

	issue, issueDir, _, loadErr := s.loadIssueWithStatus(issueID)
	if loadErr == nil {
		if issue.Metadata.Extra == nil {
			issue.Metadata.Extra = make(map[string]string)
		}
		issue.Metadata.Extra["rate_limit_until"] = resetTime
		issue.Metadata.Extra["rate_limit_agent"] = agentID
		s.writeIssueMetadata(issueDir, issue)
	}

	if err := s.moveIssue(issueID, "failed"); err != nil {
		LogWarn("Failed to move rate-limited issue to failed status", "issue_id", issueID, "error", err)
	} else {
		s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
			IssueID:   issueID,
			AgentID:   agentID,
			Success:   false,
			EndTime:   time.Now(),
			NewStatus: "failed",
		}))
	}

	return true
}

func (s *Server) handleInvestigationFailure(issueID, agentID string, result *ClaudeExecutionResult) {
	LogWarn("Investigation failed", "issue_id", issueID, "error", result.Error)

	issue, issueDir, _, loadErr := s.loadIssueWithStatus(issueID)
	if loadErr == nil {
		if issue.Metadata.Extra == nil {
			issue.Metadata.Extra = make(map[string]string)
		}
		issue.Metadata.Extra["agent_last_error"] = result.Error
		if result.MaxTurnsExceeded {
			issue.Metadata.Extra["max_turns_exceeded"] = "true"
		}
		if result.TranscriptPath != "" {
			issue.Metadata.Extra["agent_transcript_path"] = result.TranscriptPath
		}
		if result.LastMessagePath != "" {
			issue.Metadata.Extra["agent_last_message_path"] = result.LastMessagePath
		}
		issue.Metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.writeIssueMetadata(issueDir, issue)
	}

	s.moveIssue(issueID, "failed")
	s.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   false,
		EndTime:   time.Now(),
		NewStatus: "failed",
	}))
}

func (s *Server) handleInvestigationSuccess(issueID, agentID string, result *ClaudeExecutionResult) {
	LogInfo("Investigation completed successfully", "issue_id", issueID, "execution_time", result.ExecutionTime.Round(time.Second))

	issue, issueDir, _, loadErr := s.loadIssueWithStatus(issueID)
	if loadErr != nil {
		LogErrorErr("Failed to reload issue after investigation", loadErr, "issue_id", issueID)
		return
	}

	nowUTC := time.Now().UTC()

	reportContent := strings.TrimSpace(result.LastMessage)
	if reportContent == "" {
		reportContent = result.Output
	}
	issue.Investigation.Report = reportContent
	issue.Investigation.CompletedAt = nowUTC.Format(time.RFC3339)
	issue.Metadata.UpdatedAt = nowUTC.Format(time.RFC3339)

	if issue.Investigation.StartedAt != "" {
		if parsedStart, err := time.Parse(time.RFC3339, issue.Investigation.StartedAt); err == nil {
			durationMinutes := int(time.Since(parsedStart).Minutes())
			issue.Investigation.InvestigationDurationMinutes = &durationMinutes
		}
	}

	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	delete(issue.Metadata.Extra, "agent_last_error")
	delete(issue.Metadata.Extra, "max_turns_exceeded")
	issue.Metadata.Extra[AgentStatusExtraKey] = AgentStatusCompleted
	issue.Metadata.Extra[AgentStatusTimestampExtraKey] = nowUTC.Format(time.RFC3339)
	if result.TranscriptPath != "" {
		issue.Metadata.Extra["agent_transcript_path"] = result.TranscriptPath
	}
	if result.LastMessagePath != "" {
		issue.Metadata.Extra["agent_last_message_path"] = result.LastMessagePath
	}

	issue.Metadata.ResolvedAt = nowUTC.Format(time.RFC3339)

	if saveErr := s.writeIssueMetadata(issueDir, issue); saveErr != nil {
		LogErrorErr("Failed to persist investigation results", saveErr, "issue_id", issueID)
	}

	if moveErr := s.moveIssue(issueID, "completed"); moveErr != nil {
		LogErrorErr("Failed to move issue to completed", moveErr, "issue_id", issueID)
	}

	LogInfo("Investigation state persisted", "issue_id", issueID)

	processedCount := s.incrementProcessedCount()
	LogInfo("Processor successful investigations updated", "count", processedCount)

	s.hub.Publish(NewEvent(EventAgentCompleted, AgentCompletedData{
		IssueID:   issueID,
		AgentID:   agentID,
		Success:   true,
		EndTime:   nowUTC,
		NewStatus: "completed",
	}))
}
