package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"app-issue-tracker-api/internal/agents"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
	services "app-issue-tracker-api/internal/server/services"
)

const maxInvestigationPayloadBytes int64 = 1 << 20 // 1 MiB

// previewInvestigationPromptHandler generates an investigation prompt preview
func (s *Server) previewInvestigationPromptHandler(w http.ResponseWriter, r *http.Request) {
	var req PromptPreviewRequest
	if err := handlers.DecodeJSON(r, &req, maxInvestigationPayloadBytes); err != nil {
		handlers.WriteDecodeError(w, err)
		return
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		agentID = agents.UnifiedResolverID
	}

	var (
		issue  *Issue
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
			handlers.WriteError(w, http.StatusBadRequest, "Issue data is required")
			return
		}
		var loadErr error
		issue, issueDir, _, loadErr = s.loadIssueWithStatus(issueID)
		if loadErr != nil {
			if errors.Is(loadErr, services.ErrIssueNotFound) {
				handlers.WriteError(w, http.StatusNotFound, "Issue not found")
				return
			}
			logging.LogErrorErr("Failed to load issue for prompt preview", loadErr, "issue_id", issueID)
			handlers.WriteError(w, http.StatusInternalServerError, "Failed to load issue")
			return
		}
		source = "issue_directory"
	}

	if issue == nil {
		handlers.WriteError(w, http.StatusBadRequest, "Issue data is required")
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

	if err := handlers.WriteJSON(w, http.StatusOK, resp); err != nil {
		logging.LogErrorErr("Failed to encode prompt preview response", err, "issue_id", resp.IssueID)
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

	if err := handlers.DecodeJSON(r, &req, maxInvestigationPayloadBytes); err != nil {
		handlers.WriteDecodeError(w, err)
		return
	}

	issueID := strings.TrimSpace(req.IssueID)
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	autoResolve := true
	if req.AutoResolve != nil {
		autoResolve = *req.AutoResolve
	}

	agentID := strings.TrimSpace(req.AgentID)
	if agentID == "" {
		agentID = agents.UnifiedResolverID
	}

	if !req.Force {
		if s.processor.IsRunning(issueID) {
			handlers.WriteError(w, http.StatusConflict, "Agent is already running for the specified issue")
			return
		}

		state := s.currentProcessorState()
		slots := state.ConcurrentSlots
		if slots > 0 {
			running := len(s.processor.RunningProcesses())
			if running >= slots {
				handlers.WriteError(w, http.StatusTooManyRequests, fmt.Sprintf("Concurrent slot limit (%d) reached. Try again later or enable force.", slots))
				return
			}
		}
	}

	// Use the reusable triggerInvestigation method
	if err := s.TriggerInvestigation(issueID, agentID, autoResolve); err != nil {
		logging.LogErrorErr("Failed to trigger investigation", err, "issue_id", issueID, "agent_id", agentID)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to trigger investigation", err.Error())
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
			"issue_id":      issueID,
			"agent_id":      agentID,
			"status":        "active",
			"workflow":      "single-agent",
			"auto_resolve":  autoResolve,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write investigation response", err, "issue_id", issueID)
	}
}
