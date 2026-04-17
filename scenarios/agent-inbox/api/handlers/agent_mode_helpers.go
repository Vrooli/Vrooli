// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains helper functions and types for agent mode operations.
package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ValidatePathRequest is the request body for the path validation endpoint.
type ValidatePathRequest struct {
	Path string `json:"path"`
}

// ValidatePathResponse is the response body for the path validation endpoint.
type ValidatePathResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

// ValidatePath checks whether a given path is a valid, accessible directory.
// POST /api/v1/validate-path
func (h *Handlers) ValidatePath(w http.ResponseWriter, r *http.Request) {
	var req ValidatePathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "invalid request body"}, http.StatusOK)
		return
	}

	if req.Path == "" {
		h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "path is required"}, http.StatusOK)
		return
	}

	info, err := os.Stat(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "path does not exist"}, http.StatusOK)
		} else if os.IsPermission(err) {
			h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "path is not accessible (permission denied)"}, http.StatusOK)
		} else {
			h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "path is not valid: " + err.Error()}, http.StatusOK)
		}
		return
	}

	if !info.IsDir() {
		h.JSONResponse(w, ValidatePathResponse{Valid: false, Message: "path is not a directory"}, http.StatusOK)
		return
	}

	h.JSONResponse(w, ValidatePathResponse{Valid: true}, http.StatusOK)
}

// GetProjectRoot returns the VROOLI_ROOT or current working directory as a default project path.
// GET /api/v1/project-root
func (h *Handlers) GetProjectRoot(w http.ResponseWriter, r *http.Request) {
	root := ""
	if value := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); value != "" {
		if resolved, err := repocontract.FindRepoRootFromPath(value); err == nil {
			root = resolved
		}
	}
	if root == "" {
		if resolved, err := repocontract.ResolveRepoRoot(); err == nil {
			root = resolved
		}
	}
	h.JSONResponse(w, map[string]string{"project_root": root}, http.StatusOK)
}

// getAgentClient returns the agent-manager client or writes an error response if unavailable.
func (h *Handlers) getAgentClient(w http.ResponseWriter, r *http.Request) integrations.AgentManagerClientInterface {
	if h.AgentClient == nil {
		h.WriteAppError(w, r, domain.ErrAgentManagerUnavailable())
		return nil
	}
	return h.AgentClient
}

// decodeJSON decodes JSON from a request body into the provided destination.
func decodeJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(r.Body).Decode(dst)
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
