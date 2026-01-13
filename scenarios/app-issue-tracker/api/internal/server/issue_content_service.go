package server

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/metadata"
	services "app-issue-tracker-api/internal/server/services"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrAttachmentNotFound    = errors.New("attachment not found")
	ErrInvalidAttachmentPath = errors.New("invalid attachment path")
)

// IssueContentService centralizes attachment and run event operations for issues.
type IssueContentService struct {
	server   *Server
	openFile func(string) (*os.File, error)
	stat     func(string) (os.FileInfo, error)
	readFile func(string) ([]byte, error)
}

func NewIssueContentService(server *Server) *IssueContentService {
	return &IssueContentService{
		server:   server,
		openFile: os.Open,
		stat:     os.Stat,
		readFile: os.ReadFile,
	}
}

type AttachmentResource struct {
	File         *os.File
	ContentType  string
	DownloadName string
	Meta         *Attachment
	ModTime      time.Time
}

func (svc *IssueContentService) ResolveAttachment(issueID, rawAttachment string) (*AttachmentResource, error) {
	trimmedID := strings.TrimSpace(issueID)
	trimmedPath := strings.TrimSpace(rawAttachment)
	if trimmedID == "" || trimmedPath == "" {
		return nil, ErrInvalidAttachmentPath
	}

	normalized := issuespkg.NormalizeAttachmentPath(trimmedPath)
	if normalized == "" || strings.HasPrefix(normalized, "../") {
		return nil, ErrInvalidAttachmentPath
	}

	issueDir, _, err := svc.server.findIssueDirectory(trimmedID)
	if err != nil {
		return nil, err
	}

	fsPath, err := services.ResolveAttachmentPath(issueDir, normalized)
	if err != nil {
		return nil, ErrInvalidAttachmentPath
	}

	info, err := svc.stat(fsPath)
	if err != nil || info.IsDir() {
		return nil, ErrAttachmentNotFound
	}

	var attachmentMeta *Attachment
	contentType := ""
	downloadName := ""

	if issue, loadErr := svc.server.loadIssueFromDir(issueDir); loadErr == nil && issue != nil {
		for idx := range issue.Attachments {
			attachment := &issue.Attachments[idx]
			candidate := issuespkg.NormalizeAttachmentPath(attachment.Path)
			if candidate == normalized {
				attachmentMeta = attachment
				contentType = strings.TrimSpace(attachment.Type)
				if strings.TrimSpace(attachment.Name) != "" {
					downloadName = strings.TrimSpace(attachment.Name)
				}
				break
			}
		}
	}

	if contentType == "" {
		if detected := mime.TypeByExtension(strings.ToLower(filepath.Ext(fsPath))); detected != "" {
			contentType = detected
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if downloadName == "" {
		downloadName = filepath.Base(fsPath)
	} else if !strings.Contains(downloadName, ".") {
		if ext := filepath.Ext(fsPath); ext != "" {
			downloadName = fmt.Sprintf("%s%s", downloadName, ext)
		}
	}

	file, err := svc.openFile(fsPath)
	if err != nil {
		logging.LogErrorErr("Failed to open attachment", err, "issue_id", trimmedID, "attachment", normalized)
		return nil, fmt.Errorf("open attachment: %w", err)
	}

	return &AttachmentResource{
		File:         file,
		ContentType:  contentType,
		DownloadName: downloadName,
		Meta:         attachmentMeta,
		ModTime:      info.ModTime(),
	}, nil
}

func (svc *IssueContentService) AgentConversation(issueID string, maxEntries int) (*AgentConversationPayload, error) {
	trimmedID := strings.TrimSpace(issueID)
	if trimmedID == "" {
		return nil, errors.New("issue id is required")
	}

	issue, _, _, err := svc.server.loadIssueWithStatus(trimmedID)
	if err != nil {
		return nil, err
	}

	payload := &AgentConversationPayload{
		IssueID:   trimmedID,
		Available: false,
	}

	if issue.Metadata.Extra != nil {
		if runnerType := strings.TrimSpace(issue.Metadata.Extra[metadata.AgentRunnerTypeKey]); runnerType != "" {
			payload.RunnerType = runnerType
		}
		if profileKey := strings.TrimSpace(issue.Metadata.Extra[metadata.AgentProfileKey]); profileKey != "" {
			payload.ProfileKey = profileKey
		}
		if sessionID := strings.TrimSpace(issue.Metadata.Extra[metadata.AgentSessionIDKey]); sessionID != "" {
			payload.SessionID = sessionID
		}
	}

	runID := ""
	if issue.Metadata.Extra != nil {
		runID = strings.TrimSpace(issue.Metadata.Extra[metadata.AgentRunIDKey])
	}

	if runID == "" {
		return payload, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	run, err := svc.server.agentManager.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load run: %w", err)
	}
	if run == nil {
		return payload, nil
	}

	events, err := svc.server.agentManager.GetRunEvents(ctx, runID, 0)
	if err != nil {
		return nil, fmt.Errorf("load run events: %w", err)
	}

	entries := mapRunEvents(events)
	if maxEntries > 0 && len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	payload.Available = len(entries) > 0
	payload.RunID = runID
	if payload.RunnerType == "" {
		payload.RunnerType = inferRunnerType(run)
	}
	payload.SessionID = run.GetSessionId()
	payload.Metadata = mapRunMetadata(run)
	payload.Entries = entries
	if run.GetEndedAt() != nil {
		payload.TranscriptTimestamp = run.GetEndedAt().AsTime().UTC().Format(time.RFC3339)
	}

	return payload, nil
}

func mapRunEvents(events []*domainpb.RunEvent) []AgentConversationEntry {
	entries := make([]AgentConversationEntry, 0, len(events))
	for _, evt := range events {
		entry := AgentConversationEntry{
			ID:   evt.GetId(),
			Kind: "event",
		}

		timestamp := ""
		if evt.GetTimestamp() != nil {
			timestamp = evt.GetTimestamp().AsTime().UTC().Format(time.RFC3339)
		}

		switch evt.GetEventType() {
		case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
			msg := evt.GetMessage()
			entry.Kind = "message"
			entry.Role = strings.TrimSpace(msg.GetRole())
			entry.Text = msg.GetContent()
		case domainpb.RunEventType_RUN_EVENT_TYPE_LOG:
			logEvent := evt.GetLog()
			entry.Kind = "log"
			entry.Type = logEvent.GetLevel()
			entry.Text = logEvent.GetMessage()
		case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
			call := evt.GetToolCall()
			entry.Kind = "tool"
			entry.Type = "call"
			entry.Text = call.GetToolName()
			entry.Data = map[string]interface{}{
				"tool_name":    call.GetToolName(),
				"tool_call_id": call.GetToolCallId(),
				"input":        structToMap(call.GetInput()),
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
			result := evt.GetToolResult()
			entry.Kind = "tool"
			entry.Type = "result"
			entry.Text = result.GetOutput()
			if entry.Text == "" {
				entry.Text = result.GetError()
			}
			entry.Data = map[string]interface{}{
				"tool_name":    result.GetToolName(),
				"tool_call_id": result.GetToolCallId(),
				"success":      result.GetSuccess(),
				"output":       result.GetOutput(),
				"error":        result.GetError(),
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
			if limit := evt.GetRateLimit(); limit != nil {
				entry.Kind = "rate_limit"
				entry.Text = limit.GetMessage()
				entry.Data = map[string]interface{}{
					"limit_type":   limit.GetLimitType(),
					"retry_after":  limit.GetRetryAfter(),
					"current_used": limit.GetCurrentUsed(),
					"limit":        limit.GetLimit(),
				}
				if limit.GetResetTime() != nil {
					entry.Data["reset_time"] = limit.GetResetTime().AsTime().UTC().Format(time.RFC3339)
				}
				break
			}
			errEvent := evt.GetError()
			entry.Kind = "error"
			entry.Type = errEvent.GetCode()
			entry.Text = errEvent.GetMessage()
			entry.Data = map[string]interface{}{
				"code":       errEvent.GetCode(),
				"retryable":  errEvent.GetRetryable(),
				"recovery":   errEvent.GetRecovery().String(),
				"stacktrace": errEvent.GetStackTrace(),
			}
			if details := structToMap(errEvent.GetDetails()); details != nil {
				entry.Data["details"] = details
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
			if progress := evt.GetProgress(); progress != nil {
				entry.Kind = "progress"
				entry.Text = fmt.Sprintf("%d%%", progress.GetPercentComplete())
				entry.Data = map[string]interface{}{
					"phase":               progress.GetPhase().String(),
					"percent_complete":    progress.GetPercentComplete(),
					"current_action":      progress.GetCurrentAction(),
					"turns_completed":     progress.GetTurnsCompleted(),
					"turns_total":         progress.GetTurnsTotal(),
					"tokens_used":         progress.GetTokensUsed(),
					"elapsed_seconds":     progress.GetElapsedSeconds(),
					"estimated_remaining": progress.GetEstimatedRemaining(),
				}
				break
			}
			status := evt.GetStatus()
			entry.Kind = "status"
			entry.Type = status.GetNewStatus()
			entry.Text = status.GetReason()
			entry.Data = map[string]interface{}{
				"old_status": status.GetOldStatus(),
				"new_status": status.GetNewStatus(),
				"reason":     status.GetReason(),
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_METRIC:
			if cost := evt.GetCost(); cost != nil {
				entry.Kind = "cost"
				entry.Text = fmt.Sprintf("$%.4f", cost.GetTotalCostUsd())
				entry.Data = map[string]interface{}{
					"total_cost_usd":        cost.GetTotalCostUsd(),
					"input_tokens":          cost.GetInputTokens(),
					"output_tokens":         cost.GetOutputTokens(),
					"cache_creation_tokens": cost.GetCacheCreationTokens(),
					"cache_read_tokens":     cost.GetCacheReadTokens(),
					"service_tier":          cost.GetServiceTier(),
					"model":                 cost.GetModel(),
				}
				break
			}
			metric := evt.GetMetric()
			entry.Kind = "metric"
			entry.Type = metric.GetName()
			entry.Text = fmt.Sprintf("%v", metric.GetValue())
			entry.Data = map[string]interface{}{
				"name":  metric.GetName(),
				"value": metric.GetValue(),
				"unit":  metric.GetUnit(),
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
			artifact := evt.GetArtifact()
			entry.Kind = "artifact"
			entry.Type = artifact.GetType()
			entry.Text = artifact.GetPath()
			entry.Data = map[string]interface{}{
				"type":      artifact.GetType(),
				"path":      artifact.GetPath(),
				"size":      artifact.GetSize(),
				"mime_type": artifact.GetMimeType(),
			}
		case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
			deleted := evt.GetMessageDeleted()
			entry.Kind = "message_deleted"
			entry.Text = deleted.GetTargetEventId()
			entry.Data = map[string]interface{}{
				"target_event_id": deleted.GetTargetEventId(),
			}
		default:
			entry.Kind = "event"
		}

		if timestamp != "" {
			if entry.Data == nil {
				entry.Data = make(map[string]interface{})
			}
			entry.Data["timestamp"] = timestamp
		}

		entries = append(entries, entry)
	}

	return entries
}

func mapRunMetadata(run *domainpb.Run) map[string]interface{} {
	metadata := map[string]interface{}{
		"status":           run.GetStatus().String(),
		"phase":            run.GetPhase().String(),
		"progress_percent": run.GetProgressPercent(),
		"tag":              run.GetTag(),
	}

	if run.GetStartedAt() != nil {
		metadata["started_at"] = run.GetStartedAt().AsTime().UTC().Format(time.RFC3339)
	}
	if run.GetEndedAt() != nil {
		metadata["ended_at"] = run.GetEndedAt().AsTime().UTC().Format(time.RFC3339)
	}
	if run.ExitCode != nil {
		metadata["exit_code"] = run.GetExitCode()
	}
	if run.GetSummary() != nil {
		metadata["tokens_used"] = run.GetSummary().GetTokensUsed()
		metadata["cost_estimate"] = run.GetSummary().GetCostEstimate()
		metadata["files_modified"] = run.GetSummary().GetFilesModified()
		metadata["files_created"] = run.GetSummary().GetFilesCreated()
		metadata["files_deleted"] = run.GetSummary().GetFilesDeleted()
	}

	return metadata
}

func inferRunnerType(run *domainpb.Run) string {
	if run.GetResolvedConfig() != nil {
		switch run.GetResolvedConfig().GetRunnerType() {
		case domainpb.RunnerType_RUNNER_TYPE_CODEX:
			return "codex"
		case domainpb.RunnerType_RUNNER_TYPE_OPENCODE:
			return "opencode"
		case domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE:
			return "claude-code"
		}
	}
	return ""
}

func structToMap(value *structpb.Struct) map[string]interface{} {
	if value == nil {
		return nil
	}
	return value.AsMap()
}

func (svc *IssueContentService) disposition(contentType string) string {
	if strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "text/") ||
		strings.HasSuffix(contentType, "+json") ||
		contentType == "application/json" {
		return "inline"
	}
	return "attachment"
}
