package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/metadata"
	services "app-issue-tracker-api/internal/server/services"
)

var (
	ErrAttachmentNotFound    = errors.New("attachment not found")
	ErrInvalidAttachmentPath = errors.New("invalid attachment path")
)

// IssueContentService centralizes attachment and transcript operations for issues.
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

	if issue.Investigation.AgentID != "" {
		payload.Provider = issue.Investigation.AgentID
	}

	if issue.Metadata.Extra != nil {
		if provider := strings.TrimSpace(issue.Metadata.Extra[metadata.AgentProviderKey]); provider != "" && payload.Provider == "" {
			payload.Provider = provider
		}
	}

	lastMessagePath := ""
	transcriptPath := ""
	if issue.Metadata.Extra != nil {
		transcriptPath = strings.TrimSpace(issue.Metadata.Extra[metadata.AgentTranscriptPathKey])
		lastMessagePath = strings.TrimSpace(issue.Metadata.Extra[metadata.AgentLastMessagePathKey])
	}

	if lastMessagePath != "" {
		if resolved, resolveErr := svc.server.resolveAgentFilePath(lastMessagePath); resolveErr == nil {
			if data, readErr := svc.readFile(resolved); readErr == nil {
				payload.LastMessage = strings.TrimSpace(string(data))
			}
		} else {
			logging.LogWarn("Invalid last message path", "issue_id", trimmedID, "path", lastMessagePath, "error", resolveErr)
		}
	}

	if transcriptPath != "" {
		resolvedPath, resolveErr := svc.server.resolveAgentFilePath(transcriptPath)
		if resolveErr != nil {
			logging.LogWarn("Transcript path rejected", "issue_id", trimmedID, "path", transcriptPath, "error", resolveErr)
		} else {
			metadata, prompt, entries, parseErr := parseAgentTranscript(resolvedPath, maxEntries)
			if parseErr != nil {
				logging.LogErrorErr("Failed to parse agent transcript", parseErr, "issue_id", trimmedID, "path", resolvedPath)
				return nil, fmt.Errorf("parse transcript: %w", parseErr)
			}

			payload.Available = true
			payload.Metadata = metadata
			payload.Prompt = prompt
			payload.Entries = entries
			if info, statErr := svc.stat(resolvedPath); statErr == nil {
				payload.TranscriptTimestamp = info.ModTime().UTC().Format(time.RFC3339)
			}
		}
	}

	return payload, nil
}

func parseAgentTranscript(path string, maxEntries int) (map[string]interface{}, string, []AgentConversationEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var (
		metadata map[string]interface{}
		prompt   string
		entries  []AgentConversationEntry
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			entries = append(entries, AgentConversationEntry{
				Kind: "unparsed",
				Text: line,
				Data: map[string]interface{}{"error": err.Error()},
			})
			continue
		}

		if _, ok := raw["sandbox"]; ok {
			metadata = raw
			continue
		}

		if rawPrompt, ok := raw["prompt"].(string); ok {
			prompt = rawPrompt
			entries = append(entries, AgentConversationEntry{
				Kind: "prompt",
				Role: "user",
				Text: rawPrompt,
			})
			continue
		}

		msg, ok := raw["msg"].(map[string]interface{})
		if !ok {
			entries = append(entries, AgentConversationEntry{
				Kind: "raw",
				Raw:  raw,
			})
			continue
		}

		entry := AgentConversationEntry{
			Kind: "event",
			ID:   asString(raw["id"]),
			Type: asString(msg["type"]),
		}

		if role := strings.TrimSpace(asString(msg["role"])); role != "" {
			entry.Role = role
		}

		switch entry.Type {
		case "agent_message", "final_response":
			if entry.Role == "" {
				entry.Role = "assistant"
			}
			entry.Kind = "message"
			entry.Text = asString(msg["message"])
		case "agent_reasoning", "reasoning":
			if entry.Role == "" {
				entry.Role = "assistant"
			}
			entry.Kind = "reasoning"
			entry.Text = asString(msg["text"])
		case "user_message":
			if entry.Role == "" {
				entry.Role = "user"
			}
			entry.Kind = "message"
			entry.Text = asString(msg["message"])
		case "tool_request", "tool_result", "tool_output", "tool_error":
			entry.Kind = "tool"
			entry.Text = asString(msg["summary"])
		case "token_count", "task_started":
			entry.Kind = "event"
		}

		data := sanitizeConversationData(msg, "type", "message", "text", "role")
		if timestamp := raw["timestamp"]; timestamp != nil {
			if data == nil {
				data = make(map[string]interface{})
			}
			data["timestamp"] = timestamp
		}
		entry.Data = data

		if entry.Data == nil {
			entry.Raw = msg
		}

		entries = append(entries, entry)

		if maxEntries > 0 && len(entries) >= maxEntries {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", nil, err
	}

	return metadata, prompt, entries, nil
}

func sanitizeConversationData(source map[string]interface{}, excludeKeys ...string) map[string]interface{} {
	if len(source) == 0 {
		return nil
	}

	exclude := make(map[string]struct{}, len(excludeKeys))
	for _, key := range excludeKeys {
		exclude[key] = struct{}{}
	}

	clean := make(map[string]interface{})
	for key, value := range source {
		if _, skip := exclude[key]; skip {
			continue
		}
		clean[key] = value
	}

	if len(clean) == 0 {
		return nil
	}

	return clean
}

func asString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
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
