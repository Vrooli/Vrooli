package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
		var loadErr error
		issue, issueDir, _, loadErr = s.loadIssueWithStatus(issueID)
		if loadErr != nil {
			if strings.Contains(loadErr.Error(), "issue not found") {
				http.Error(w, "Issue not found", http.StatusNotFound)
				return
			}
			LogErrorErr("Failed to load issue for prompt preview", loadErr, "issue_id", issueID)
			http.Error(w, "Failed to load issue", http.StatusInternalServerError)
			return
		}
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
		LogErrorErr("Failed to encode prompt preview response", err, "issue_id", resp.IssueID)
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

	issueID := strings.TrimSpace(req.IssueID)
	if issueID == "" {
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

	if !req.Force {
		if s.processor.IsRunning(issueID) {
			http.Error(w, "Agent is already running for the specified issue", http.StatusConflict)
			return
		}

		state := s.currentProcessorState()
		slots := state.ConcurrentSlots
		if slots > 0 {
			running := len(s.processor.RunningProcesses())
			if running >= slots {
				http.Error(
					w,
					fmt.Sprintf("Concurrent slot limit (%d) reached. Try again later or enable force.", slots),
					http.StatusTooManyRequests,
				)
				return
			}
		}
	}

	// Use the reusable triggerInvestigation method
	if err := s.triggerInvestigation(issueID, agentID, autoResolve); err != nil {
		LogErrorErr("Failed to trigger investigation", err, "issue_id", issueID, "agent_id", agentID)
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
			"issue_id":      issueID,
			"agent_id":      agentID,
			"status":        "active",
			"workflow":      "single-agent",
			"auto_resolve":  autoResolve,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
