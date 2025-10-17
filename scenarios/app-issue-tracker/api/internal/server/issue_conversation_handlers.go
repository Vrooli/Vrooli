package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"

	"github.com/gorilla/mux"
)

// getIssueAttachmentHandler serves an issue attachment file
func (s *Server) getIssueAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	attachmentPath := strings.TrimSpace(vars["attachment"])
	if issueID == "" || attachmentPath == "" {
		http.Error(w, "Issue ID and attachment path are required", http.StatusBadRequest)
		return
	}

	decodedPath, err := url.PathUnescape(attachmentPath)
	if err != nil {
		http.Error(w, "Invalid attachment path", http.StatusBadRequest)
		return
	}

	normalized := issuespkg.NormalizeAttachmentPath(decodedPath)
	if normalized == "" || strings.HasPrefix(normalized, "../") {
		http.Error(w, "Invalid attachment path", http.StatusBadRequest)
		return
	}

	issueDir, _, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	fsPath, err := resolveAttachmentPath(issueDir, normalized)
	if err != nil {
		http.Error(w, "Invalid attachment path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(fsPath)
	if err != nil || info.IsDir() {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	loadedIssue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		LogErrorErr("Failed to load issue metadata for attachment lookup", err, "issue_id", issueID, "path", issueDir)
	}

	var contentType string
	var downloadName string
	targetMetaPath := issuespkg.NormalizeAttachmentPath(normalized)
	var attachmentMeta *Attachment
	if loadedIssue != nil {
		for idx := range loadedIssue.Attachments {
			attachment := &loadedIssue.Attachments[idx]
			candidate := issuespkg.NormalizeAttachmentPath(attachment.Path)
			if candidate == targetMetaPath {
				contentType = strings.TrimSpace(attachment.Type)
				if strings.TrimSpace(attachment.Name) != "" {
					downloadName = strings.TrimSpace(attachment.Name)
				}
				attachmentMeta = attachment
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

	file, err := os.Open(fsPath)
	if err != nil {
		LogErrorErr("Failed to open attachment", err, "issue_id", issueID, "attachment", normalized)
		http.Error(w, "Failed to open attachment", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", contentType)
	disposition := "inline"
	if !(strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "text/") || strings.HasSuffix(contentType, "+json") || contentType == "application/json") {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, downloadName))
	if attachmentMeta != nil && attachmentMeta.Category != "" {
		w.Header().Set("X-Attachment-Category", attachmentMeta.Category)
	}
	w.Header().Set("Cache-Control", "no-store")

	http.ServeContent(w, r, downloadName, info.ModTime(), file)
}

// getIssueAgentConversationHandler streams the captured agent transcript for UI rendering
func (s *Server) getIssueAgentConversationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issue, _, _, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		if strings.Contains(err.Error(), "issue not found") {
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}
		LogErrorErr("Failed to load issue for transcript lookup", err, "issue_id", issueID)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	payload := AgentConversationPayload{
		IssueID:   issueID,
		Available: false,
	}

	if issue.Investigation.AgentID != "" {
		payload.Provider = issue.Investigation.AgentID
	}

	var transcriptRaw, lastMessageRaw string
	if issue.Metadata.Extra != nil {
		if provider := strings.TrimSpace(issue.Metadata.Extra["agent_provider"]); provider != "" && payload.Provider == "" {
			payload.Provider = provider
		}
		transcriptRaw = issue.Metadata.Extra["agent_transcript_path"]
		lastMessageRaw = issue.Metadata.Extra["agent_last_message_path"]
	}

	if trimmed := strings.TrimSpace(lastMessageRaw); trimmed != "" {
		if resolved, resolveErr := s.resolveAgentFilePath(trimmed); resolveErr == nil {
			if data, readErr := os.ReadFile(resolved); readErr == nil {
				payload.LastMessage = strings.TrimSpace(string(data))
			}
		} else {
			LogWarn("Invalid last message path", "issue_id", issueID, "path", trimmed, "error", resolveErr)
		}
	}

	transcriptRaw = strings.TrimSpace(transcriptRaw)
	if transcriptRaw != "" {
		resolvedPath, resolveErr := s.resolveAgentFilePath(transcriptRaw)
		if resolveErr != nil {
			LogWarn("Transcript path rejected", "issue_id", issueID, "path", transcriptRaw, "error", resolveErr)
			payload.Available = false
		} else {
			metadata, prompt, entries, parseErr := parseAgentTranscript(resolvedPath, 750)
			if parseErr != nil {
				LogErrorErr("Failed to parse agent transcript", parseErr, "issue_id", issueID, "path", resolvedPath)
				http.Error(w, "Failed to parse agent transcript", http.StatusInternalServerError)
				return
			}

			payload.Available = true
			payload.Metadata = metadata
			payload.Prompt = prompt
			payload.Entries = entries
			if info, statErr := os.Stat(resolvedPath); statErr == nil {
				payload.TranscriptTimestamp = info.ModTime().UTC().Format(time.RFC3339)
			}
		}
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversation": payload,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		LogErrorErr("Failed to encode conversation response", err, "issue_id", issueID)
	}
}

func (s *Server) resolveAgentFilePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("empty path")
	}

	candidate := trimmed
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.config.ScenarioRoot, candidate)
	}
	candidate = filepath.Clean(candidate)

	allowedBases := []string{
		filepath.Join(s.config.ScenarioRoot, "tmp"),
		s.config.ScenarioRoot,
	}

	for _, base := range allowedBases {
		base = filepath.Clean(base)
		rel, err := filepath.Rel(base, candidate)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, fmt.Sprintf("..%c", os.PathSeparator))) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("path %s is outside the allowed scenario directories", candidate)
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

	lineCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lineCount++

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

		// Retain compact raw payload for debugging when helpful
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
