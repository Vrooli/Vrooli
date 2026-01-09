package issues

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ValidationError struct {
	message string
}

func (e ValidationError) Error() string {
	return e.message
}

func newValidationError(message string) error {
	return ValidationError{message: message}
}

func PrepareIssueForCreate(req *CreateIssueRequest, now time.Time) (*Issue, string, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, "", newValidationError("Title is required")
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = title
	}

	issueType := strings.TrimSpace(req.Type)
	if issueType == "" {
		issueType = "bug"
	}

	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = "medium"
	}

	// Validate and normalize targets
	if len(req.Targets) == 0 {
		return nil, "", newValidationError("At least one target is required")
	}

	normalizedTargets, err := NormalizeTargets(req.Targets)
	if err != nil {
		return nil, "", newValidationError(fmt.Sprintf("Invalid targets: %v", err))
	}

	targetStatus := "open"
	if trimmed := strings.TrimSpace(req.Status); trimmed != "" {
		normalized, ok := NormalizeStatus(trimmed)
		if !ok {
			return nil, "", newValidationError(fmt.Sprintf("Invalid status: %s", trimmed))
		}
		targetStatus = normalized
	}

	id := fmt.Sprintf("issue-%s", uuid.New().String()[:8])
	issue := &Issue{
		ID:          id,
		Title:       title,
		Description: description,
		Type:        issueType,
		Priority:    priority,
		Targets:     normalizedTargets,
		Status:      targetStatus,
		Notes:       strings.TrimSpace(req.Notes),
	}

	timestamp := now.UTC().Format(time.RFC3339)

	if req.Reporter != nil {
		issue.Reporter.Name = strings.TrimSpace(req.Reporter.Name)
		issue.Reporter.Email = strings.TrimSpace(req.Reporter.Email)
		issue.Reporter.UserID = strings.TrimSpace(req.Reporter.UserID)
		issue.Reporter.Timestamp = strings.TrimSpace(req.Reporter.Timestamp)
	}
	if trimmed := strings.TrimSpace(req.ReporterName); trimmed != "" {
		issue.Reporter.Name = trimmed
	}
	if trimmed := strings.TrimSpace(req.ReporterEmail); trimmed != "" {
		issue.Reporter.Email = trimmed
	}
	if trimmed := strings.TrimSpace(req.ReporterUserID); trimmed != "" {
		issue.Reporter.UserID = trimmed
	}
	if strings.TrimSpace(issue.Reporter.Timestamp) == "" {
		issue.Reporter.Timestamp = timestamp
	}

	issue.ErrorContext.ErrorMessage = strings.TrimSpace(req.ErrorMessage)
	if strings.TrimSpace(req.ErrorLogs) != "" {
		issue.ErrorContext.ErrorLogs = req.ErrorLogs
	}
	if strings.TrimSpace(req.StackTrace) != "" {
		issue.ErrorContext.StackTrace = req.StackTrace
	}
	if len(req.AffectedFiles) > 0 {
		issue.ErrorContext.AffectedFiles = NormalizeStringSlice(req.AffectedFiles)
	}
	if len(req.AffectedComponents) > 0 {
		issue.ErrorContext.AffectedComponents = NormalizeStringSlice(req.AffectedComponents)
	}
	if len(req.Environment) > 0 {
		issue.ErrorContext.EnvironmentInfo = CloneStringMap(req.Environment)
	}

	issue.Metadata.Tags = NormalizeStringSlice(req.Tags)
	issue.Metadata.Labels = CloneStringMap(req.Labels)
	issue.Metadata.Watchers = NormalizeStringSlice(req.Watchers)
	issue.Metadata.Extra = CloneStringMap(req.MetadataExtra)

	return issue, targetStatus, nil
}

func ApplyUpdateRequest(issue *Issue, req *UpdateIssueRequest, currentFolder string) (string, error) {
	targetStatus := currentFolder
	if req.Status != nil {
		reqStatus := strings.TrimSpace(*req.Status)
		if reqStatus == "" {
			return "", newValidationError("Status cannot be empty")
		}
		normalized, ok := NormalizeStatus(reqStatus)
		if !ok {
			return "", newValidationError(fmt.Sprintf("Invalid status: %s", reqStatus))
		}
		targetStatus = normalized
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return "", newValidationError("Title cannot be empty")
		}
		issue.Title = title
	}
	if req.Description != nil {
		issue.Description = strings.TrimSpace(*req.Description)
	}
	if req.Type != nil {
		issue.Type = strings.TrimSpace(*req.Type)
	}
	if req.Priority != nil {
		priority := strings.TrimSpace(*req.Priority)
		if priority != "" {
			issue.Priority = priority
		}
	}
	if req.Targets != nil {
		if *req.Targets == nil || len(*req.Targets) == 0 {
			return "", newValidationError("Targets cannot be empty")
		}
		normalizedTargets, err := NormalizeTargets(*req.Targets)
		if err != nil {
			return "", newValidationError(fmt.Sprintf("Invalid targets: %v", err))
		}
		issue.Targets = normalizedTargets
	}
	if req.Tags != nil {
		if *req.Tags == nil {
			issue.Metadata.Tags = nil
		} else {
			issue.Metadata.Tags = NormalizeStringSlice(*req.Tags)
		}
	}
	if req.Labels != nil {
		if *req.Labels == nil {
			issue.Metadata.Labels = nil
		} else {
			issue.Metadata.Labels = CloneStringMap(*req.Labels)
		}
	}
	if req.Watchers != nil {
		if *req.Watchers == nil {
			issue.Metadata.Watchers = nil
		} else {
			issue.Metadata.Watchers = NormalizeStringSlice(*req.Watchers)
		}
	}
	if req.MetadataExtra != nil {
		if *req.MetadataExtra == nil {
			issue.Metadata.Extra = nil
		} else {
			issue.Metadata.Extra = CloneStringMap(*req.MetadataExtra)
		}
	}
	if req.Notes != nil {
		issue.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.ResolvedAt != nil {
		issue.Metadata.ResolvedAt = strings.TrimSpace(*req.ResolvedAt)
	}

	if req.Reporter != nil {
		if req.Reporter.Name != nil {
			issue.Reporter.Name = strings.TrimSpace(*req.Reporter.Name)
		}
		if req.Reporter.Email != nil {
			issue.Reporter.Email = strings.TrimSpace(*req.Reporter.Email)
		}
		if req.Reporter.UserID != nil {
			issue.Reporter.UserID = strings.TrimSpace(*req.Reporter.UserID)
		}
		if req.Reporter.Timestamp != nil {
			issue.Reporter.Timestamp = strings.TrimSpace(*req.Reporter.Timestamp)
		}
	}

	if req.ErrorContext != nil {
		if req.ErrorContext.ErrorMessage != nil {
			issue.ErrorContext.ErrorMessage = strings.TrimSpace(*req.ErrorContext.ErrorMessage)
		}
		if req.ErrorContext.ErrorLogs != nil {
			issue.ErrorContext.ErrorLogs = *req.ErrorContext.ErrorLogs
		}
		if req.ErrorContext.StackTrace != nil {
			issue.ErrorContext.StackTrace = *req.ErrorContext.StackTrace
		}
		if req.ErrorContext.AffectedFiles != nil {
			if *req.ErrorContext.AffectedFiles == nil {
				issue.ErrorContext.AffectedFiles = nil
			} else {
				issue.ErrorContext.AffectedFiles = NormalizeStringSlice(*req.ErrorContext.AffectedFiles)
			}
		}
		if req.ErrorContext.AffectedComponents != nil {
			if *req.ErrorContext.AffectedComponents == nil {
				issue.ErrorContext.AffectedComponents = nil
			} else {
				issue.ErrorContext.AffectedComponents = NormalizeStringSlice(*req.ErrorContext.AffectedComponents)
			}
		}
		if req.ErrorContext.EnvironmentInfo != nil {
			if *req.ErrorContext.EnvironmentInfo == nil {
				issue.ErrorContext.EnvironmentInfo = nil
			} else {
				envCopy := make(map[string]string, len(*req.ErrorContext.EnvironmentInfo))
				for k, v := range *req.ErrorContext.EnvironmentInfo {
					envCopy[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
				if len(envCopy) == 0 {
					issue.ErrorContext.EnvironmentInfo = nil
				} else {
					issue.ErrorContext.EnvironmentInfo = envCopy
				}
			}
		}
	}

	if req.Investigation != nil {
		if req.Investigation.AgentID != nil {
			issue.Investigation.AgentID = strings.TrimSpace(*req.Investigation.AgentID)
		}
		if req.Investigation.StartedAt != nil {
			issue.Investigation.StartedAt = strings.TrimSpace(*req.Investigation.StartedAt)
		}
		if req.Investigation.CompletedAt != nil {
			issue.Investigation.CompletedAt = strings.TrimSpace(*req.Investigation.CompletedAt)
		}
		if req.Investigation.Report != nil {
			issue.Investigation.Report = strings.TrimSpace(*req.Investigation.Report)
		}
		if req.Investigation.RootCause != nil {
			issue.Investigation.RootCause = strings.TrimSpace(*req.Investigation.RootCause)
		}
		if req.Investigation.SuggestedFix != nil {
			issue.Investigation.SuggestedFix = strings.TrimSpace(*req.Investigation.SuggestedFix)
		}
		if req.Investigation.ConfidenceScore != nil {
			issue.Investigation.ConfidenceScore = req.Investigation.ConfidenceScore
		}
		if req.Investigation.InvestigationDurationMinutes != nil {
			issue.Investigation.InvestigationDurationMinutes = req.Investigation.InvestigationDurationMinutes
		}
		if req.Investigation.TokensUsed != nil {
			issue.Investigation.TokensUsed = req.Investigation.TokensUsed
		}
		if req.Investigation.CostEstimate != nil {
			issue.Investigation.CostEstimate = req.Investigation.CostEstimate
		}
	}

	if req.Fix != nil {
		if req.Fix.SuggestedFix != nil {
			issue.Fix.SuggestedFix = strings.TrimSpace(*req.Fix.SuggestedFix)
		}
		if req.Fix.ImplementationPlan != nil {
			issue.Fix.ImplementationPlan = strings.TrimSpace(*req.Fix.ImplementationPlan)
		}
		if req.Fix.Applied != nil {
			issue.Fix.Applied = *req.Fix.Applied
		}
		if req.Fix.AppliedAt != nil {
			issue.Fix.AppliedAt = strings.TrimSpace(*req.Fix.AppliedAt)
		}
		if req.Fix.CommitHash != nil {
			issue.Fix.CommitHash = strings.TrimSpace(*req.Fix.CommitHash)
		}
		if req.Fix.PrURL != nil {
			issue.Fix.PrURL = strings.TrimSpace(*req.Fix.PrURL)
		}
		if req.Fix.VerificationStatus != nil {
			issue.Fix.VerificationStatus = strings.TrimSpace(*req.Fix.VerificationStatus)
		}
		if req.Fix.RollbackPlan != nil {
			issue.Fix.RollbackPlan = strings.TrimSpace(*req.Fix.RollbackPlan)
		}
		if req.Fix.FixDurationMinutes != nil {
			issue.Fix.FixDurationMinutes = req.Fix.FixDurationMinutes
		}
	}

	if req.ManualReview != nil {
		if req.ManualReview.MarkedAsFailed != nil {
			issue.ManualReview.MarkedAsFailed = *req.ManualReview.MarkedAsFailed
		}
		if req.ManualReview.FailureReason != nil {
			issue.ManualReview.FailureReason = strings.TrimSpace(*req.ManualReview.FailureReason)
		}
		if req.ManualReview.ReviewedBy != nil {
			issue.ManualReview.ReviewedBy = strings.TrimSpace(*req.ManualReview.ReviewedBy)
		}
		if req.ManualReview.ReviewedAt != nil {
			issue.ManualReview.ReviewedAt = strings.TrimSpace(*req.ManualReview.ReviewedAt)
		}
		if req.ManualReview.ReviewNotes != nil {
			issue.ManualReview.ReviewNotes = strings.TrimSpace(*req.ManualReview.ReviewNotes)
		}
		if req.ManualReview.OriginalStatus != nil {
			issue.ManualReview.OriginalStatus = strings.TrimSpace(*req.ManualReview.OriginalStatus)
		}
	}

	return targetStatus, nil
}
