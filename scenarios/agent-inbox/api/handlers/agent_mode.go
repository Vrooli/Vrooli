// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains handlers for agent mode operations, integrating with agent-manager.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/middleware"

	"github.com/gorilla/mux"
)

// getAgentClient returns the agent-manager client or writes an error response if unavailable.
func (h *Handlers) getAgentClient(w http.ResponseWriter, r *http.Request) integrations.AgentManagerClientInterface {
	if h.AgentClient == nil {
		h.WriteAppError(w, r, domain.ErrAgentManagerUnavailable())
		return nil
	}
	return h.AgentClient
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

// AttachAgentRunRequest is the request body for attaching an existing run to a chat.
type AttachAgentRunRequest struct {
	RunID  string `json:"run_id"`
	TaskID string `json:"task_id"`
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

// StartAgentModeRequest is the request body for starting agent mode.
type StartAgentModeRequest struct {
	Message     string `json:"message"`      // Initial message to send to the agent
	RunnerType  string `json:"runner_type"`  // "claude-code", "codex", or "opencode"
	ProjectPath string `json:"project_path"` // Directory where the agent will operate
	Model       string `json:"model,omitempty"`
	MaxTurns    int    `json:"max_turns,omitempty"`
}

// AgentModeResponse is the response after starting or modifying agent mode.
type AgentModeResponse struct {
	ChatID    string `json:"chat_id"`
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id,omitempty"`
}

// StartAgentMode initializes agent mode for a chat.
// POST /api/v1/chats/{id}/agent-mode/start
func (h *Handlers) StartAgentMode(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	var req StartAgentModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteAppError(w, r, domain.ErrInvalidJSON())
		return
	}

	// Validate required fields
	if req.Message == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("message is required"))
		return
	}
	if req.RunnerType == "" {
		req.RunnerType = "claude-code" // Default to Claude Code
	}
	if req.ProjectPath == "" {
		h.WriteAppError(w, r, domain.ErrInvalidInput("project_path is required"))
		return
	}
	// Validate that the project path exists and is accessible
	if info, err := os.Stat(req.ProjectPath); err != nil {
		if os.IsNotExist(err) {
			h.WriteAppError(w, r, domain.ErrInvalidInput("project_path does not exist: "+req.ProjectPath))
		} else if os.IsPermission(err) {
			h.WriteAppError(w, r, domain.ErrInvalidInput("project_path is not accessible (permission denied): "+req.ProjectPath))
		} else {
			h.WriteAppError(w, r, domain.ErrInvalidInput("project_path is not valid: "+err.Error()))
		}
		return
	} else if !info.IsDir() {
		h.WriteAppError(w, r, domain.ErrInvalidInput("project_path is not a directory: "+req.ProjectPath))
		return
	}

	// Verify chat exists
	chat, err := h.Repo.GetChat(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] StartAgentMode GetChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}
	if chat == nil {
		h.WriteAppError(w, r, domain.ErrChatNotFound(chatID))
		return
	}

	// Check if already in agent mode
	if chat.ChatMode == domain.ChatModeAgent && chat.AgentRunID != "" {
		h.WriteAppError(w, r, domain.ErrAgentAlreadyActive(chatID))
		return
	}

	// Get agent-manager client
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Start agent chat
	cfg := integrations.AgentChatConfig{
		RunnerType:  integrations.RunnerType(req.RunnerType),
		ProjectPath: req.ProjectPath,
		Model:       req.Model,
		MaxTurns:    req.MaxTurns,
	}

	session, err := agentClient.StartAgentChat(r.Context(), req.Message, cfg)
	if err != nil {
		log.Printf("[ERROR] [%s] StartAgentMode StartAgentChat failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	// Update chat to agent mode
	if err := h.Repo.SetAgentMode(r.Context(), chatID, session.TaskID, session.RunID); err != nil {
		log.Printf("[ERROR] [%s] StartAgentMode SetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("update chat", err))
		return
	}

	// Store the initial message
	_, err = h.Repo.CreateMessage(r.Context(), chatID, domain.RoleUser, req.Message, "", "", 0, chat.ActiveLeafMessageID, nil)
	if err != nil {
		log.Printf("[WARN] [%s] StartAgentMode CreateMessage failed: %v", middleware.GetRequestID(r.Context()), err)
		// Non-fatal, continue
	} else {
		h.maybeAutoNameChat(r.Context(), chatID)
	}

	h.JSONResponse(w, AgentModeResponse{
		ChatID:    chatID,
		TaskID:    session.TaskID,
		RunID:     session.RunID,
		SessionID: session.SessionID,
	}, http.StatusOK)
}

// SendAgentMessageRequest is the request body for sending a message in agent mode.
type SendAgentMessageRequest struct {
	Message string `json:"message"`
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
	if err := agentClient.ContinueChat(r.Context(), runID, req.Message); err != nil {
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

// StopAgentMode stops the agent run for a chat.
// POST /api/v1/chats/{id}/agent-mode/stop
func (h *Handlers) StopAgentMode(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	// Get chat and verify it's in agent mode
	chatMode, _, runID, err := h.Repo.GetAgentMode(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] StopAgentMode GetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
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

	// Stop the run
	if err := agentClient.StopRun(r.Context(), runID); err != nil {
		log.Printf("[ERROR] [%s] StopAgentMode StopRun failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success": true,
		"run_id":  runID,
	}, http.StatusOK)
}

// GetAgentStatus gets the current status of an agent run.
// GET /api/v1/chats/{id}/agent-mode/status
func (h *Handlers) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	// Get chat and verify it's in agent mode
	chatMode, taskID, runID, err := h.Repo.GetAgentMode(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] GetAgentStatus GetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}

	// Return basic info even if not in agent mode
	if chatMode != domain.ChatModeAgent {
		h.JSONResponse(w, map[string]interface{}{
			"chat_mode": chatMode,
			"is_agent":  false,
		}, http.StatusOK)
		return
	}

	if runID == "" {
		h.JSONResponse(w, map[string]interface{}{
			"chat_mode": chatMode,
			"is_agent":  true,
			"task_id":   taskID,
			"run_id":    nil,
		}, http.StatusOK)
		return
	}

	// Get agent-manager client
	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	// Get run status
	status, err := agentClient.GetRunStatus(r.Context(), runID)
	if err != nil {
		log.Printf("[WARN] [%s] GetAgentStatus GetRunStatus failed: %v", middleware.GetRequestID(r.Context()), err)
		// Return what we know without the live status
		h.JSONResponse(w, map[string]interface{}{
			"chat_mode": chatMode,
			"is_agent":  true,
			"task_id":   taskID,
			"run_id":    runID,
			"error":     "unable to fetch live status",
		}, http.StatusOK)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"chat_mode":        chatMode,
		"is_agent":         true,
		"task_id":          taskID,
		"run_id":           runID,
		"status":           status.Status,
		"phase":            status.Phase,
		"progress_percent": status.ProgressPercent,
		"session_id":       status.SessionID,
		"error_msg":        status.ErrorMsg,
	}, http.StatusOK)
}

// GetRunEvents retrieves events for an agent-manager run directly by run ID.
// Unlike GetAgentEvents, this does not require the run to be attached to a chat.
// GET /api/v1/agent-runs/{run_id}/events?after_sequence=N
func (h *Handlers) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["run_id"]
	if runID == "" {
		h.WriteAppError(w, r, domain.ErrMissingField("run_id"))
		return
	}

	afterSequence := int64(0)
	if seqStr := r.URL.Query().Get("after_sequence"); seqStr != "" {
		if seq, err := strconv.ParseInt(seqStr, 10, 64); err == nil {
			afterSequence = seq
		}
	}

	agentClient := h.getAgentClient(w, r)
	if agentClient == nil {
		return
	}

	events, err := agentClient.GetEvents(r.Context(), runID, afterSequence)
	if err != nil {
		log.Printf("[ERROR] [%s] GetRunEvents GetEvents failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrExternalService("agent-manager", err.Error()))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"events": events,
		"run_id": runID,
	}, http.StatusOK)
}

// ClearAgentMode resets a chat back to LLM mode.
// POST /api/v1/chats/{id}/agent-mode/clear
func (h *Handlers) ClearAgentMode(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	// First try to stop any running agent
	chatMode, _, runID, err := h.Repo.GetAgentMode(r.Context(), chatID)
	if err != nil {
		log.Printf("[ERROR] [%s] ClearAgentMode GetAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("get chat", err))
		return
	}

	// If in agent mode with active run, try to stop it
	if chatMode == domain.ChatModeAgent && runID != "" && h.AgentClient != nil {
		if err := h.AgentClient.StopRun(r.Context(), runID); err != nil {
			log.Printf("[WARN] [%s] ClearAgentMode StopRun failed: %v", middleware.GetRequestID(r.Context()), err)
			// Non-fatal, continue to clear
		}
	}

	// Clear agent mode
	if err := h.Repo.ClearAgentMode(r.Context(), chatID); err != nil {
		log.Printf("[ERROR] [%s] ClearAgentMode ClearAgentMode failed: %v", middleware.GetRequestID(r.Context()), err)
		h.WriteAppError(w, r, domain.ErrDatabaseError("clear agent mode", err))
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"success":   true,
		"chat_mode": domain.ChatModeLLM,
	}, http.StatusOK)
}
