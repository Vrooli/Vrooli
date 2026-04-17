// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for agent uploads, messaging, events, run listing, and attachment.
package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// AttachAgentRunRequest is the request body for attaching an existing run to a chat.
type AttachAgentRunRequest struct {
	RunID  string `json:"run_id"`
	TaskID string `json:"task_id"`
}

// SendAgentMessageRequest is the request body for sending a message in agent mode.
type SendAgentMessageRequest struct {
	Message       string   `json:"message"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
}

// ListAgentRuns returns a paginated list of runs from agent-manager.
// GET /api/v1/agent-runs?status=X&tag_prefix=X&limit=N&offset=N
func (h *Handlers) ListAgentRuns(w http.ResponseWriter, r *http.Request) {
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	opts := integrations.ListRunsOptions{
		Status:    r.URL.Query().Get("status"),
		TagPrefix: r.URL.Query().Get("tag_prefix"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			opts.Limit = v
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			opts.Offset = v
		}
	}

	result, err := agentClient.ListRuns(r.Context(), opts)
	if err != nil {
		log.Printf("[ERROR] [%s] ListAgentRuns ListRuns failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	h.JSONResponse(w, result, http.StatusOK)
}

// AttachAgentRun attaches an existing agent-manager run to a chat.
// POST /api/v1/chats/{id}/agent-mode/attach
func (h *Handlers) AttachAgentRun(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req AttachAgentRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	if req.RunID == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("run_id is required"))
		return
	}
	if req.TaskID == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("task_id is required"))
		return
	}

	// Verify chat exists
	chat, err := h.Repo.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] AttachAgentRun GetChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}
	if chat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	// Check if already in agent mode with an active run
	if chat.ChatMode == domain.ChatModeAgent && chat.AgentRunID != "" {
		h.WriteAppError(w, r, domain.ErrAgentAlreadyActive(chatID))
		return
	}

	// Get agent-manager client
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Verify the run exists in agent-manager
	_, err = agentClient.GetRunStatus(r.Context(), req.RunID)
	if err != nil {
		log.Printf("[ERROR] [%s] AttachAgentRun GetRunStatus failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", "run not found: "+err.Error()))
		return
	}

	// Update chat to agent mode
	if err := h.Repo.SetAgentMode(r.Context(), chatID, req.TaskID, req.RunID); err != nil {
		log.Printf("[ERROR] [%s] AttachAgentRun SetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("update chat", err))
		return
	}

	h.JSONResponse(w, AgentModeResponse{
		ChatID: chatID,
		TaskID: req.TaskID,
		RunID:  req.RunID,
	}, http.StatusOK)
}

// ProxyAgentUpload proxies file upload to agent-manager's attachment endpoint.
// POST /api/v1/agent-attachments/upload
func (h *Handlers) ProxyAgentUpload(w http.ResponseWriter, r *http.Request) {
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Parse multipart form (32MB max)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidInput("invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidInput("missing file field"))
		return
	}
	defer file.Close()

	result, err := agentClient.UploadAttachment(r.Context(), file, header)
	if err != nil {
		log.Printf("[ERROR] [%s] ProxyAgentUpload failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	h.JSONResponse(w, result, http.StatusCreated)
}

// SendAgentMessage sends a follow-up message to an agent run.
// POST /api/v1/chats/{id}/agent-mode/message
func (h *Handlers) SendAgentMessage(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req SendAgentMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	if req.Message == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("message is required"))
		return
	}

	// Get chat and verify it's in agent mode
	chatMode, _, runID, err := h.Repo.GetAgentMode(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] SendAgentMessage GetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}

	if chatMode != domain.ChatModeAgent {
		h.WriteAppError(w, r, domain.ErrAgentNotInMode(chatID))
		return
	}

	if runID == "" {
		h.WriteAppError(w, r, domain.ErrAgentNoActiveRun(chatID))
		return
	}

	// Get agent-manager client
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Check run status before attempting continuation — the run must be completed
	// (session ID is only available after completion)
	runStatus, err := agentClient.GetRunStatus(r.Context(), runID)
	if err != nil {
		log.Printf("[WARN] [%s] SendAgentMessage GetRunStatus failed: %v", middleware.GetRequestID(r.Context()), err)
		// Fall through — let ContinueChat handle the error with its own validation
	} else if runStatus != nil && runStatus.Status != "" {
		switch runStatus.Status {
		case "pending", "starting", "running":
			h.WriteAppError(w, r, domain.ErrAgentRunBusy(chatID))
			return
		}
	}

	// Continue the run
	if err := agentClient.ContinueChat(r.Context(), runID, req.Message, req.AttachmentIDs); err != nil {
		log.Printf("[ERROR] [%s] SendAgentMessage ContinueChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	// Store the user message
	chat, _ := h.Repo.GetChat(r.Context(), chatID)
	parentMsgID := ""
	if chat != nil {
		parentMsgID = chat.ActiveLeafMessageID
	}
	_, err = h.Repo.CreateMessage(r.Context(), chatID, domain.RoleUser, req.Message, "", "", 0, parentMsgID, nil)
	if err != nil {
		log.Printf("[WARN] [%s] SendAgentMessage CreateMessage failed: %v", middleware.GetRequestID(r.Context()), err)
		// Non-fatal, continue
	} else {
		h.maybeAutoNameChat(r.Context(), chatID)
	}

	h.JSONResponse(w, map[string]interface{}{
		"success": true,
		"run_id":  runID,
	}, http.StatusOK)
}

// GetAgentEvents retrieves events for an agent run.
// GET /api/v1/chats/{id}/agent-mode/events?after_sequence=N
func (h *Handlers) GetAgentEvents(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	// Get optional after_sequence parameter
	afterSequence := int64(0)
	if seqStr := r.URL.Query().Get("after_sequence"); seqStr != "" {
		if seq, err := strconv.ParseInt(seqStr, 10, 64); err == nil {
			afterSequence = seq
		}
	}

	// Get chat and verify it's in agent mode
	chatMode, _, runID, err := h.Repo.GetAgentMode(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] GetAgentEvents GetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}

	if chatMode != domain.ChatModeAgent {
		h.WriteAppError(w, r, domain.ErrAgentNotInMode(chatID))
		return
	}

	if runID == "" {
		h.WriteAppError(w, r, domain.ErrAgentNoActiveRun(chatID))
		return
	}

	// Get agent-manager client
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Get events
	events, err := agentClient.GetEvents(r.Context(), runID, afterSequence)
	if err != nil {
		log.Printf("[ERROR] [%s] GetAgentEvents GetEvents failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"events": events,
		"run_id": runID,
	}, http.StatusOK)
}
