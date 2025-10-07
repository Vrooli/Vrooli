package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// healthHandler returns the API health status
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := ApiResponse{
		Success: true,
		Message: "App Issue Tracker API is healthy",
		Data: map[string]interface{}{
			"timestamp":  time.Now().UTC(),
			"version":    "2.0.0-file-based",
			"storage":    "file-based-yaml",
			"issues_dir": s.config.IssuesDir,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getIssuesHandler retrieves a list of issues with optional filters
func (s *Server) getIssuesHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	issues, err := s.getAllIssues(status, priority, issueType, limit)
	if err != nil {
		log.Printf("Error getting issues: %v", err)
		http.Error(w, "Failed to load issues", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issues": issues,
			"count":  len(issues),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createIssueHandler creates a new issue
func (s *Server) createIssueHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
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
	appID := strings.TrimSpace(req.AppID)
	if appID == "" {
		appID = "unknown"
	}

	targetStatus := "open"
	if trimmed := strings.ToLower(strings.TrimSpace(req.Status)); trimmed != "" {
		if _, ok := validIssueStatuses[trimmed]; !ok {
			http.Error(w, fmt.Sprintf("Invalid status: %s", trimmed), http.StatusBadRequest)
			return
		}
		targetStatus = trimmed
	}

	issueID := fmt.Sprintf("issue-%s", uuid.New().String()[:8])
	issue := Issue{
		ID:          issueID,
		Title:       title,
		Description: description,
		Type:        issueType,
		Priority:    priority,
		AppID:       appID,
		Status:      targetStatus,
		Notes:       strings.TrimSpace(req.Notes),
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if req.Reporter != nil {
		issue.Reporter.Name = strings.TrimSpace(req.Reporter.Name)
		issue.Reporter.Email = strings.TrimSpace(req.Reporter.Email)
		issue.Reporter.UserID = strings.TrimSpace(req.Reporter.UserID)
		issue.Reporter.Timestamp = strings.TrimSpace(req.Reporter.Timestamp)
	}
	if strings.TrimSpace(req.ReporterName) != "" {
		issue.Reporter.Name = strings.TrimSpace(req.ReporterName)
	}
	if strings.TrimSpace(req.ReporterEmail) != "" {
		issue.Reporter.Email = strings.TrimSpace(req.ReporterEmail)
	}
	if strings.TrimSpace(req.ReporterUserID) != "" {
		issue.Reporter.UserID = strings.TrimSpace(req.ReporterUserID)
	}
	if strings.TrimSpace(issue.Reporter.Timestamp) == "" {
		issue.Reporter.Timestamp = now
	}

	issue.ErrorContext.ErrorMessage = strings.TrimSpace(req.ErrorMessage)
	if strings.TrimSpace(req.ErrorLogs) != "" {
		issue.ErrorContext.ErrorLogs = req.ErrorLogs
	}
	if strings.TrimSpace(req.StackTrace) != "" {
		issue.ErrorContext.StackTrace = req.StackTrace
	}
	if len(req.AffectedFiles) > 0 {
		issue.ErrorContext.AffectedFiles = normalizeStringSlice(req.AffectedFiles)
	}
	if len(req.AffectedComponents) > 0 {
		issue.ErrorContext.AffectedComponents = normalizeStringSlice(req.AffectedComponents)
	}
	if len(req.Environment) > 0 {
		issue.ErrorContext.EnvironmentInfo = cloneStringMap(req.Environment)
	}

	issue.Metadata.Tags = normalizeStringSlice(req.Tags)
	issue.Metadata.Labels = cloneStringMap(req.Labels)
	issue.Metadata.Watchers = normalizeStringSlice(req.Watchers)
	issue.Metadata.Extra = cloneStringMap(req.MetadataExtra)

	issueDir := s.issueDir(targetStatus, issue.ID)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		log.Printf("Error preparing issue directory: %v", err)
		http.Error(w, "Failed to prepare storage", http.StatusInternalServerError)
		return
	}

	artifactPayloads := mergeCreateArtifacts(&req)
	attachments, err := s.persistArtifacts(issueDir, artifactPayloads)
	if err != nil {
		log.Printf("Error storing artifacts: %v", err)
		http.Error(w, "Failed to store artifacts", http.StatusInternalServerError)
		return
	}
	if len(attachments) > 0 {
		issue.Attachments = attachments
	}

	if _, err := s.saveIssue(&issue, targetStatus); err != nil {
		log.Printf("Error saving issue: %v", err)
		http.Error(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}

	storagePath := filepath.Join(targetStatus, issue.ID)
	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: map[string]interface{}{
			"issue":        issue,
			"issue_id":     issue.ID,
			"storage_path": storagePath,
		},
	}

	// Publish event for real-time updates
	s.hub.Publish(NewEvent(EventIssueCreated, IssueEventData{Issue: &issue}))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getIssueHandler retrieves a single issue by ID
func (s *Server) getIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, statusFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		log.Printf("Error loading issue %s: %v", issueID, err)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	issue.Status = statusFolder

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

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

	normalized := normalizeAttachmentPath(decodedPath)
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
		log.Printf("Failed to load issue metadata for attachment lookup: %v", err)
	}

	var contentType string
	var downloadName string
	targetMetaPath := normalizeAttachmentPath(normalized)
	var attachmentMeta *Attachment
	if loadedIssue != nil {
		for idx := range loadedIssue.Attachments {
			attachment := &loadedIssue.Attachments[idx]
			candidate := normalizeAttachmentPath(attachment.Path)
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
		log.Printf("Failed to open attachment %s for issue %s: %v", normalized, issueID, err)
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

	issueDir, _, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		log.Printf("Failed to load issue %s for transcript lookup: %v", issueID, err)
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
			log.Printf("Issue %s: ignoring last message path %q: %v", issueID, trimmed, resolveErr)
		}
	}

	transcriptRaw = strings.TrimSpace(transcriptRaw)
	if transcriptRaw != "" {
		resolvedPath, resolveErr := s.resolveAgentFilePath(transcriptRaw)
		if resolveErr != nil {
			log.Printf("Issue %s: transcript path rejected (%q): %v", issueID, transcriptRaw, resolveErr)
			payload.Available = false
		} else {
			metadata, prompt, entries, parseErr := parseAgentTranscript(resolvedPath, 750)
			if parseErr != nil {
				log.Printf("Issue %s: failed parsing transcript %s: %v", issueID, resolvedPath, parseErr)
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
		log.Printf("Failed to encode conversation response for issue %s: %v", issueID, err)
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

// updateIssueHandler updates an existing issue
func (s *Server) updateIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, currentFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		log.Printf("Error loading issue %s: %v", issueID, err)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	issue.Status = currentFolder

	var req UpdateIssueRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Invalid update payload for issue %s: %v", issueID, err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	targetStatus := currentFolder
	if req.Status != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.Status))
		if normalized == "" {
			http.Error(w, "Status cannot be empty", http.StatusBadRequest)
			return
		}
		if _, ok := validIssueStatuses[normalized]; !ok {
			http.Error(w, fmt.Sprintf("Invalid status: %s", normalized), http.StatusBadRequest)
			return
		}
		targetStatus = normalized
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			http.Error(w, "Title cannot be empty", http.StatusBadRequest)
			return
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
	if req.AppID != nil {
		appID := strings.TrimSpace(*req.AppID)
		if appID != "" {
			issue.AppID = appID
		}
	}
	if req.Tags != nil {
		if *req.Tags == nil {
			issue.Metadata.Tags = nil
		} else {
			issue.Metadata.Tags = normalizeStringSlice(*req.Tags)
		}
	}
	if req.Labels != nil {
		if *req.Labels == nil {
			issue.Metadata.Labels = nil
		} else {
			issue.Metadata.Labels = cloneStringMap(*req.Labels)
		}
	}
	if req.Watchers != nil {
		if *req.Watchers == nil {
			issue.Metadata.Watchers = nil
		} else {
			issue.Metadata.Watchers = normalizeStringSlice(*req.Watchers)
		}
	}
	if req.MetadataExtra != nil {
		if *req.MetadataExtra == nil {
			issue.Metadata.Extra = nil
		} else {
			issue.Metadata.Extra = cloneStringMap(*req.MetadataExtra)
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
				issue.ErrorContext.AffectedFiles = normalizeStringSlice(*req.ErrorContext.AffectedFiles)
			}
		}
		if req.ErrorContext.AffectedComponents != nil {
			if *req.ErrorContext.AffectedComponents == nil {
				issue.ErrorContext.AffectedComponents = nil
			} else {
				issue.ErrorContext.AffectedComponents = normalizeStringSlice(*req.ErrorContext.AffectedComponents)
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

	if targetStatus != currentFolder {
		if s.isIssueRunning(issueID) && targetStatus != "active" {
			http.Error(w, "Cannot change issue status while an agent is running", http.StatusConflict)
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)

		// Handle backwards status transitions (completed/failed -> open)
		// Clear investigation results for fresh execution (matches ecosystem-manager pattern)
		isBackwardsTransition := (currentFolder == "completed" || currentFolder == "failed" || currentFolder == "active") &&
			targetStatus == "open"

		if isBackwardsTransition {
			log.Printf("Issue %s moved backwards from %s to %s - clearing investigation data", issueID, currentFolder, targetStatus)
			// Clear investigation fields
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

			// Clear fix fields
			issue.Fix.SuggestedFix = ""
			issue.Fix.ImplementationPlan = ""
			issue.Fix.Applied = false
			issue.Fix.AppliedAt = ""
			issue.Fix.CommitHash = ""
			issue.Fix.PrURL = ""
			issue.Fix.VerificationStatus = ""
			issue.Fix.RollbackPlan = ""
			issue.Fix.FixDurationMinutes = nil

			// Clear agent execution error data from metadata.extra
			if issue.Metadata.Extra != nil {
				delete(issue.Metadata.Extra, "agent_last_error")
				delete(issue.Metadata.Extra, "agent_last_status")
				delete(issue.Metadata.Extra, "agent_failure_time")
				delete(issue.Metadata.Extra, "rate_limit_until")
				delete(issue.Metadata.Extra, "rate_limit_agent")
			}
		}

		if targetStatus == "active" && strings.TrimSpace(issue.Investigation.StartedAt) == "" {
			issue.Investigation.StartedAt = now
		}
		if targetStatus == "completed" && strings.TrimSpace(issue.Metadata.ResolvedAt) == "" {
			issue.Metadata.ResolvedAt = now
		}

		targetDir := s.issueDir(targetStatus, issue.ID)
		if _, statErr := os.Stat(targetDir); statErr == nil {
			http.Error(w, "Issue already exists in target status", http.StatusConflict)
			return
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			log.Printf("Failed to stat target directory for issue %s: %v", issueID, statErr)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}
		if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
			log.Printf("Failed to prepare target directory for issue %s: %v", issueID, err)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}
		if err := os.Rename(issueDir, targetDir); err != nil {
			log.Printf("Failed to move issue %s from %s to %s: %v", issueID, issueDir, targetDir, err)
			http.Error(w, "Failed to move issue", http.StatusInternalServerError)
			return
		}

		// Publish status change event
		s.hub.Publish(NewEvent(EventIssueStatusChanged, IssueStatusChangedData{
			IssueID:   issue.ID,
			OldStatus: currentFolder,
			NewStatus: targetStatus,
		}))

		issueDir = targetDir
		currentFolder = targetStatus
	}

	if len(req.Artifacts) > 0 {
		newAttachments, err := s.persistArtifacts(issueDir, req.Artifacts)
		if err != nil {
			log.Printf("Failed to store artifacts for issue %s: %v", issueID, err)
			http.Error(w, "Failed to store artifacts", http.StatusInternalServerError)
			return
		}
		if len(newAttachments) > 0 {
			issue.Attachments = append(issue.Attachments, newAttachments...)
		}
	}

	issue.Status = currentFolder

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		log.Printf("Failed to persist updated issue %s: %v", issueID, err)
		http.Error(w, "Failed to save issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue updated successfully",
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

	// Publish events for real-time updates
	s.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteIssueHandler deletes an issue
func (s *Server) deleteIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	issueDir, _, err := s.findIssueDirectory(issueID)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	if err := os.RemoveAll(issueDir); err != nil {
		log.Printf("Failed to delete issue %s: %v", issueID, err)
		http.Error(w, "Failed to delete issue", http.StatusInternalServerError)
		return
	}

	// Publish event for real-time updates
	s.hub.Publish(NewEvent(EventIssueDeleted, IssueDeletedData{IssueID: issueID}))

	response := ApiResponse{
		Success: true,
		Message: "Issue deleted successfully",
		Data: map[string]interface{}{
			"issue_id": issueID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// searchIssuesHandler searches for issues using text matching
func (s *Server) searchIssuesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	var results []Issue
	queryLower := strings.ToLower(query)

	// Search through all issues in all folders
	folders := []string{"open", "active", "waiting", "completed", "failed"}
	for _, folder := range folders {
		issues, err := s.loadIssuesFromFolder(folder)
		if err != nil {
			continue
		}

		for _, issue := range issues {
			// Simple text search
			searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
				issue.Title, issue.Description,
				issue.ErrorContext.ErrorMessage,
				strings.Join(issue.Metadata.Tags, " ")))

			if strings.Contains(searchText, queryLower) {
				results = append(results, issue)
			}
		}
	}

	// Sort by relevance (title matches first)
	sort.Slice(results, func(i, j int) bool {
		iTitleMatch := strings.Contains(strings.ToLower(results[i].Title), queryLower)
		jTitleMatch := strings.Contains(strings.ToLower(results[j].Title), queryLower)

		if iTitleMatch != jTitleMatch {
			return iTitleMatch
		}

		return results[i].Metadata.CreatedAt > results[j].Metadata.CreatedAt
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
			"query":   query,
			"method":  "text_search",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// previewInvestigationPromptHandler generates an investigation prompt preview
func (s *Server) previewInvestigationPromptHandler(w http.ResponseWriter, r *http.Request) {
	var req PromptPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		agentID = "unified-resolver"
	}

	var (
		issue  *Issue
		err    error
		source string
	)

	var issueDir string

	if req.Issue != nil {
		issueCopy := *req.Issue
		issue = &issueCopy
		source = "payload"
	} else {
		issueID := strings.TrimSpace(req.IssueID)
		if issueID == "" {
			http.Error(w, "Issue data is required", http.StatusBadRequest)
			return
		}
		issueDir, currentFolder, findErr := s.findIssueDirectory(issueID)
		if findErr != nil {
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}
		issue, err = s.loadIssueFromDir(issueDir)
		if err != nil {
			log.Printf("Failed to load issue for prompt preview: %v", err)
			http.Error(w, "Failed to load issue", http.StatusInternalServerError)
			return
		}
		issue.Status = currentFolder
		source = "issue_directory"
	}

	if issue == nil {
		http.Error(w, "Issue data is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(issue.ID) == "" {
		if trimmed := strings.TrimSpace(req.IssueID); trimmed != "" {
			issue.ID = trimmed
		} else {
			issue.ID = "preview-issue"
		}
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)
	promptTemplate := s.loadPromptTemplate()
	promptMarkdown := s.buildInvestigationPrompt(issue, issueDir, agentID, s.config.ScenarioRoot, generatedAt)

	resp := PromptPreviewResponse{
		IssueID:        issue.ID,
		AgentID:        agentID,
		IssueTitle:     strings.TrimSpace(issue.Title),
		IssueStatus:    strings.TrimSpace(issue.Status),
		PromptTemplate: promptTemplate,
		PromptMarkdown: promptMarkdown,
		GeneratedAt:    generatedAt,
		Source:         source,
	}

	if resp.Source == "" {
		resp.Source = "payload"
	}

	if msg := strings.TrimSpace(issue.ErrorContext.ErrorMessage); msg != "" {
		resp.ErrorMessage = msg
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode prompt preview response: %v", err)
	}
}

// triggerInvestigationHandler triggers an agent investigation for an issue
func (s *Server) triggerInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IssueID     string `json:"issue_id"`
		AgentID     string `json:"agent_id"`
		Priority    string `json:"priority"`
		AutoResolve *bool  `json:"auto_resolve"`
		Force       bool   `json:"force"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.IssueID) == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	autoResolve := true
	if req.AutoResolve != nil {
		autoResolve = *req.AutoResolve
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		agentID = "unified-resolver"
	}

	// Use the reusable triggerInvestigation method
	if err := s.triggerInvestigation(req.IssueID, agentID, autoResolve); err != nil {
		log.Printf("Failed to trigger investigation: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runID := fmt.Sprintf("run_%d", time.Now().Unix())
	resolutionID := fmt.Sprintf("resolve_%d", time.Now().Unix())

	response := ApiResponse{
		Success: true,
		Message: "Agent run started",
		Data: map[string]interface{}{
			"run_id":        runID,
			"resolution_id": resolutionID,
			"issue_id":      req.IssueID,
			"agent_id":      agentID,
			"status":        "active",
			"workflow":      "single-agent",
			"auto_resolve":  autoResolve,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getStatsHandler returns dashboard statistics
func (s *Server) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues by status
	var totalIssues, openIssues, inProgress, waiting, completedToday int

	allIssues, _ := s.getAllIssues("", "", "", 0)
	totalIssues = len(allIssues)

	today := time.Now().UTC().Format("2006-01-02")

	for _, issue := range allIssues {
		switch issue.Status {
		case "open":
			openIssues++
		case "active":
			inProgress++
		case "waiting":
			waiting++
		case "completed":
			if strings.HasPrefix(issue.Metadata.ResolvedAt, today) {
				completedToday++
			}
		}
	}

	// Count by app
	appCounts := make(map[string]int)
	for _, issue := range allIssues {
		appCounts[issue.AppID]++
	}

	// Convert to top apps list
	type appCount struct {
		AppName    string `json:"app_name"`
		IssueCount int    `json:"issue_count"`
	}
	var topApps []appCount
	for appID, count := range appCounts {
		topApps = append(topApps, appCount{AppName: appID, IssueCount: count})
	}

	// Sort by issue count
	sort.Slice(topApps, func(i, j int) bool {
		return topApps[i].IssueCount > topApps[j].IssueCount
	})

	// Limit to top 5
	if len(topApps) > 5 {
		topApps = topApps[:5]
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
				"total_issues":         totalIssues,
				"open_issues":          openIssues,
				"in_progress":          inProgress,
				"waiting":              waiting,
				"completed_today":      completedToday,
				"avg_resolution_hours": 24.5, // TODO: Calculate from resolved issues
				"top_apps":             topApps,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAgentsHandler returns available agents
func (s *Server) getAgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Single unified agent exposed for the simplified workflow
	agents := []Agent{
		{
			ID:             "unified-resolver",
			Name:           "unified-resolver",
			DisplayName:    "Unified Issue Resolver",
			Description:    "Single-pass agent that triages, investigates, and proposes fixes",
			Capabilities:   []string{"triage", "investigate", "fix", "test"},
			IsActive:       true,
			SuccessRate:    88.4,
			TotalRuns:      173,
			SuccessfulRuns: 153,
		},
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// triggerFixGenerationHandler returns deprecation notice
func (s *Server) triggerFixGenerationHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Fix generation now runs automatically as part of the unified /investigate workflow. Pass auto_resolve=false to /investigate for investigation-only runs.", http.StatusGone)
}

// getAppsHandler returns a list of applications with issue counts
func (s *Server) getAppsHandler(w http.ResponseWriter, r *http.Request) {
	// Count issues per app
	allIssues, _ := s.getAllIssues("", "", "", 0)
	appStats := make(map[string]struct {
		total int
		open  int
	})

	for _, issue := range allIssues {
		stats := appStats[issue.AppID]
		stats.total++
		if issue.Status == "open" || issue.Status == "active" {
			stats.open++
		}
		appStats[issue.AppID] = stats
	}

	var apps []App
	for appID, stats := range appStats {
		apps = append(apps, App{
			ID:          appID,
			Name:        appID,
			DisplayName: strings.Title(strings.ReplaceAll(appID, "-", " ")),
			Type:        "scenario",
			Status:      "active",
			TotalIssues: stats.total,
			OpenIssues:  stats.open,
		})
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"apps":  apps,
			"count": len(apps),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getAgentSettingsHandler returns the current agent backend settings
func (s *Server) getAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load settings from file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		log.Printf("Failed to read agent settings: %v", err)
		http.Error(w, "Failed to load agent settings", http.StatusInternalServerError)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Printf("Failed to parse agent settings: %v", err)
		http.Error(w, "Failed to parse agent settings", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Data:    settings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateAgentSettingsHandler updates agent backend settings
func (s *Server) updateAgentSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider        string  `json:"provider"`
		AutoFallback    bool    `json:"auto_fallback"`
		TimeoutSeconds  *int    `json:"timeout_seconds"`  // Optional timeout update
		MaxTurns        *int    `json:"max_turns"`        // Optional max turns update
		AllowedTools    *string `json:"allowed_tools"`    // Optional allowed tools update
		SkipPermissions *bool   `json:"skip_permissions"` // Optional skip permissions update
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate provider if provided
	if req.Provider != "" {
		validProviders := map[string]bool{
			"codex":       true,
			"claude-code": true,
		}

		if !validProviders[req.Provider] {
			http.Error(w, "Invalid provider. Must be 'codex' or 'claude-code'", http.StatusBadRequest)
			return
		}
	}

	settingsPath := filepath.Join(s.config.ScenarioRoot, "initialization/configuration/agent-settings.json")

	// Load current settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		log.Printf("Failed to read agent settings: %v", err)
		http.Error(w, "Failed to load agent settings", http.StatusInternalServerError)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		log.Printf("Failed to parse agent settings: %v", err)
		http.Error(w, "Failed to parse agent settings", http.StatusInternalServerError)
		return
	}

	agentBackendMap, _ := settings["agent_backend"].(map[string]interface{})
	if agentBackendMap == nil {
		agentBackendMap = map[string]interface{}{}
		settings["agent_backend"] = agentBackendMap
	}

	// Ensure fallback order exists for new configurations
	if _, ok := agentBackendMap["fallback_order"]; !ok {
		agentBackendMap["fallback_order"] = []string{"codex", "claude-code"}
	}

	// Determine current provider and apply updates
	currentProvider := "claude-code"
	if provider, ok := agentBackendMap["provider"].(string); ok && provider != "" {
		currentProvider = provider
	}

	if req.Provider != "" {
		agentBackendMap["provider"] = req.Provider
		currentProvider = req.Provider
	}

	agentBackendMap["auto_fallback"] = req.AutoFallback
	if req.SkipPermissions != nil {
		agentBackendMap["skip_permissions"] = *req.SkipPermissions
	}

	targetProvider := currentProvider

	// Update provider operation settings if provided
	if req.TimeoutSeconds != nil || req.MaxTurns != nil || req.AllowedTools != nil {
		if providers, ok := settings["providers"].(map[string]interface{}); ok {
			if providerConfig, ok := providers[targetProvider].(map[string]interface{}); ok {
				if operations, ok := providerConfig["operations"].(map[string]interface{}); ok {
					for _, opKey := range []string{"investigate", "fix"} {
						opMap, ok := operations[opKey].(map[string]interface{})
						if !ok {
							continue
						}

						if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
							opMap["timeout_seconds"] = *req.TimeoutSeconds
							log.Printf("Updated timeout for %s/%s to %d seconds (%d minutes)", targetProvider, opKey, *req.TimeoutSeconds, *req.TimeoutSeconds/60)
						}
						if req.MaxTurns != nil && *req.MaxTurns > 0 {
							opMap["max_turns"] = *req.MaxTurns
							log.Printf("Updated max_turns for %s/%s to %d", targetProvider, opKey, *req.MaxTurns)
						}
						if req.AllowedTools != nil {
							opMap["allowed_tools"] = strings.TrimSpace(*req.AllowedTools)
							log.Printf("Updated allowed_tools for %s/%s to %s", targetProvider, opKey, strings.TrimSpace(*req.AllowedTools))
						}
					}
				}
			}
		}
	}

	// Write back to file with proper formatting
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal agent settings: %v", err)
		http.Error(w, "Failed to save agent settings", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(settingsPath, updatedData, 0644); err != nil {
		log.Printf("Failed to write agent settings: %v", err)
		http.Error(w, "Failed to save agent settings", http.StatusInternalServerError)
		return
	}

	// CRITICAL: Reload settings into memory so changes take effect immediately
	ReloadAgentSettings()

	log.Printf("Agent settings updated and reloaded: provider=%s, auto_fallback=%v", targetProvider, req.AutoFallback)

	response := ApiResponse{
		Success: true,
		Message: "Agent settings updated successfully",
		Data: map[string]interface{}{
			"agent_backend": settings["agent_backend"],
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
