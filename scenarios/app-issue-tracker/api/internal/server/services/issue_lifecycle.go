package services

import (
	"fmt"
	"strings"
	"time"

	automationpkg "app-issue-tracker-api/internal/automation"
	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/server/metadata"
)

func ensureMetadataExtra(issue *issuespkg.Issue) map[string]string {
	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	return issue.Metadata.Extra
}

// ResetInvestigationForReopen clears investigation-derived fields when an issue moves backward.
func ResetInvestigationForReopen(issue *issuespkg.Issue) {
	issue.Investigation.AgentID = ""
	issue.Investigation.StartedAt = ""
	issue.Investigation.CompletedAt = ""
	issue.Investigation.Report = ""
	issue.Investigation.RootCause = ""
	issue.Investigation.SuggestedFix = ""
	issue.Investigation.ConfidenceScore = nil
	issue.Investigation.InvestigationDurationMinutes = nil
	issue.Investigation.TokensUsed = nil
	issue.Investigation.CostEstimate = nil

	issue.Fix.SuggestedFix = ""
	issue.Fix.ImplementationPlan = ""
	issue.Fix.Applied = false
	issue.Fix.AppliedAt = ""
	issue.Fix.CommitHash = ""
	issue.Fix.PrURL = ""
	issue.Fix.VerificationStatus = ""
	issue.Fix.RollbackPlan = ""
	issue.Fix.FixDurationMinutes = nil
	issue.Metadata.ResolvedAt = ""

	if issue.Metadata.Extra != nil {
		delete(issue.Metadata.Extra, metadata.AgentLastErrorKey)
		delete(issue.Metadata.Extra, metadata.AgentFailureTimeKey)
		delete(issue.Metadata.Extra, metadata.RateLimitUntilKey)
		delete(issue.Metadata.Extra, metadata.RateLimitAgentKey)
		delete(issue.Metadata.Extra, metadata.AgentCancelReasonKey)
		delete(issue.Metadata.Extra, metadata.AgentStatusExtraKey)
		delete(issue.Metadata.Extra, metadata.AgentStatusTimestampExtraKey)
		delete(issue.Metadata.Extra, metadata.AgentTranscriptPathKey)
		delete(issue.Metadata.Extra, metadata.AgentLastMessagePathKey)
		delete(issue.Metadata.Extra, metadata.AgentSessionIDKey)
		delete(issue.Metadata.Extra, metadata.AgentProviderKey)
		delete(issue.Metadata.Extra, "max_turns_exceeded")
	}
}

// MarkInvestigationStarted updates metadata to reflect an investigation start.
// agentID is the internal agent identifier (e.g., "unified-resolver")
// provider is the actual backend provider (e.g., "codex", "claude-code")
func MarkInvestigationStarted(issue *issuespkg.Issue, agentID, provider string, startedAt time.Time) {
	timestamp := startedAt.Format(time.RFC3339)
	issue.Investigation.AgentID = agentID
	issue.Investigation.StartedAt = timestamp
	issue.Metadata.UpdatedAt = timestamp

	extras := ensureMetadataExtra(issue)
	delete(extras, metadata.AgentLastErrorKey)
	delete(extras, metadata.AgentCancelReasonKey)
	delete(extras, metadata.AgentTranscriptPathKey)
	delete(extras, metadata.AgentLastMessagePathKey)
	extras[metadata.AgentStatusExtraKey] = automationpkg.AgentStatusRunning
	extras[metadata.AgentStatusTimestampExtraKey] = timestamp

	// Store the actual provider (codex/claude-code) for session resumption
	if trimmed := strings.TrimSpace(provider); trimmed != "" {
		extras[metadata.AgentProviderKey] = trimmed
	}
}

// MarkInvestigationCancelled stores cancellation metadata.
func MarkInvestigationCancelled(issue *issuespkg.Issue, reason string, cancelledAt time.Time) {
	timestamp := cancelledAt.Format(time.RFC3339)
	extras := ensureMetadataExtra(issue)
	delete(extras, metadata.AgentLastErrorKey)
	extras[metadata.AgentStatusExtraKey] = automationpkg.AgentStatusCancelled
	extras[metadata.AgentStatusTimestampExtraKey] = timestamp
	trimmed := strings.TrimSpace(reason)
	if trimmed != "" {
		extras[metadata.AgentCancelReasonKey] = trimmed
	} else {
		delete(extras, metadata.AgentCancelReasonKey)
	}
	issue.Metadata.UpdatedAt = timestamp
}

// MarkInvestigationFailure writes a failure report and metadata.
func MarkInvestigationFailure(issue *issuespkg.Issue, errorMsg, output string, occurredAt time.Time) {
	timestamp := occurredAt.Format(time.RFC3339)
	if output != "" {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s\n\nOutput:\n%s", errorMsg, output)
	} else {
		issue.Investigation.Report = fmt.Sprintf("Investigation failed: %s", errorMsg)
	}
	issue.Investigation.CompletedAt = timestamp
	issue.Metadata.UpdatedAt = timestamp

	extras := ensureMetadataExtra(issue)
	extras[metadata.AgentLastErrorKey] = errorMsg
	extras[metadata.AgentStatusExtraKey] = automationpkg.AgentStatusFailed
	extras[metadata.AgentStatusTimestampExtraKey] = timestamp
}

// RecordAgentExecutionFailure captures transient execution failure metadata.
func RecordAgentExecutionFailure(issue *issuespkg.Issue, errMsg, provider string, timestamp time.Time, transcriptPath, lastMessagePath, scenarioRoot string, maxTurnsExceeded bool) {
	extras := ensureMetadataExtra(issue)
	extras[metadata.AgentLastErrorKey] = errMsg
	if maxTurnsExceeded {
		extras["max_turns_exceeded"] = "true"
	} else {
		delete(extras, "max_turns_exceeded")
	}
	if trimmed := strings.TrimSpace(transcriptPath); trimmed != "" {
		extras[metadata.AgentTranscriptPathKey] = trimmed
		// Extract and store session ID from transcript path and marker files
		if sessionID := ExtractSessionIDFromPaths(trimmed, scenarioRoot); sessionID != "" {
			extras[metadata.AgentSessionIDKey] = sessionID
		}
	}
	if trimmed := strings.TrimSpace(lastMessagePath); trimmed != "" {
		extras[metadata.AgentLastMessagePathKey] = trimmed
	}
	// Store the actual provider for session resumption
	if trimmed := strings.TrimSpace(provider); trimmed != "" {
		extras[metadata.AgentProviderKey] = trimmed
	}
	extras[metadata.AgentStatusTimestampExtraKey] = timestamp.Format(time.RFC3339)
}

// MarkInvestigationSuccess stores completion metadata and report.
func MarkInvestigationSuccess(issue *issuespkg.Issue, report, provider string, completedAt time.Time, transcriptPath, lastMessagePath, scenarioRoot string) {
	completedAtUTC := completedAt.UTC()
	completedAtStr := completedAtUTC.Format(time.RFC3339)
	issue.Investigation.Report = strings.TrimSpace(report)
	issue.Investigation.CompletedAt = completedAtStr
	issue.Metadata.UpdatedAt = completedAtStr
	issue.Metadata.ResolvedAt = completedAtStr

	if issue.Investigation.StartedAt != "" {
		if parsedStart, err := time.Parse(time.RFC3339, issue.Investigation.StartedAt); err == nil {
			durationMinutes := int(completedAtUTC.Sub(parsedStart).Minutes())
			issue.Investigation.InvestigationDurationMinutes = &durationMinutes
		}
	}

	extras := ensureMetadataExtra(issue)
	delete(extras, metadata.AgentLastErrorKey)
	delete(extras, "max_turns_exceeded")
	extras[metadata.AgentStatusExtraKey] = automationpkg.AgentStatusCompleted
	extras[metadata.AgentStatusTimestampExtraKey] = completedAtStr
	if trimmed := strings.TrimSpace(transcriptPath); trimmed != "" {
		extras[metadata.AgentTranscriptPathKey] = trimmed
		// Extract and store session ID from transcript path and marker files
		if sessionID := ExtractSessionIDFromPaths(trimmed, scenarioRoot); sessionID != "" {
			extras[metadata.AgentSessionIDKey] = sessionID
		}
	}
	if trimmed := strings.TrimSpace(lastMessagePath); trimmed != "" {
		extras[metadata.AgentLastMessagePathKey] = trimmed
	}
	// Store the actual provider for session resumption
	if trimmed := strings.TrimSpace(provider); trimmed != "" {
		extras[metadata.AgentProviderKey] = trimmed
	}
}

// SetRateLimitMetadata records throttling metadata.
func SetRateLimitMetadata(issue *issuespkg.Issue, agentID, resetTime string) {
	extras := ensureMetadataExtra(issue)
	extras[metadata.RateLimitUntilKey] = resetTime
	extras[metadata.RateLimitAgentKey] = agentID
}

// ClearRateLimitMetadata removes throttling metadata on an issue.
func ClearRateLimitMetadata(issue *issuespkg.Issue) {
	if issue.Metadata.Extra == nil {
		return
	}
	delete(issue.Metadata.Extra, metadata.RateLimitUntilKey)
	delete(issue.Metadata.Extra, metadata.RateLimitAgentKey)
}
