package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"

	"github.com/gorilla/mux"
)

var (
	errIssueRunning      = errors.New("cannot change issue status while an agent is running")
	errIssueTargetExists = errors.New("issue already exists in target status")
)

func (s *Server) createIssueBundle(req *CreateIssueRequest) (*Issue, string, error) {
	issue, targetStatus, err := issuespkg.PrepareIssueForCreate(req, time.Now().UTC())
	if err != nil {
		return nil, "", err
	}

	issueDir := s.issueDir(targetStatus, issue.ID)
	if err := os.MkdirAll(issueDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("failed to prepare issue directory: %w", err)
	}

	artifactPayloads := mergeCreateArtifacts(req)
	if err := s.storeIssueArtifacts(issue, issueDir, artifactPayloads, true); err != nil {
		return nil, "", fmt.Errorf("failed to store issue artifacts: %w", err)
	}

	if _, err := s.saveIssue(issue, targetStatus); err != nil {
		return nil, "", fmt.Errorf("failed to persist issue: %w", err)
	}

	storagePath := filepath.Join(targetStatus, issue.ID)
	s.hub.Publish(NewEvent(EventIssueCreated, IssueEventData{Issue: issue}))

	return issue, storagePath, nil
}

func (s *Server) updateIssueBundle(issueID string, req *UpdateIssueRequest) (*Issue, error) {
	issue, issueDir, currentFolder, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		return nil, err
	}

	targetStatus, err := issuespkg.ApplyUpdateRequest(issue, req, currentFolder)
	if err != nil {
		return nil, err
	}

	updatedDir := issueDir
	updatedFolder := currentFolder

	if targetStatus != currentFolder {
		updatedDir, updatedFolder, err = s.transitionIssueStatus(issueID, issue, currentFolder, targetStatus)
		if err != nil {
			return nil, err
		}
	}

	if err := s.storeIssueArtifacts(issue, updatedDir, req.Artifacts, false); err != nil {
		return nil, fmt.Errorf("failed to store additional artifacts: %w", err)
	}

	issue.Status = updatedFolder

	if err := s.writeIssueMetadata(updatedDir, issue); err != nil {
		return nil, fmt.Errorf("failed to persist updated issue: %w", err)
	}

	s.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	return issue, nil
}

func (s *Server) transitionIssueStatus(issueID string, issue *Issue, currentFolder, targetStatus string) (string, string, error) {
	if s.isIssueRunning(issueID) && targetStatus != "active" {
		return "", "", errIssueRunning
	}

	now := time.Now().UTC().Format(time.RFC3339)
	isBackwardsTransition := (currentFolder == "completed" || currentFolder == "failed" || currentFolder == "active") &&
		targetStatus == "open"

	if isBackwardsTransition {
		LogInfo(
			"Issue status moved backwards, clearing investigation data",
			"issue_id", issueID,
			"from", currentFolder,
			"to", targetStatus,
		)
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

		issue.Fix.SuggestedFix = ""
		issue.Fix.ImplementationPlan = ""
		issue.Fix.Applied = false
		issue.Fix.AppliedAt = ""
		issue.Fix.CommitHash = ""
		issue.Fix.PrURL = ""
		issue.Fix.VerificationStatus = ""
		issue.Fix.RollbackPlan = ""
		issue.Fix.FixDurationMinutes = nil

		if issue.Metadata.Extra != nil {
			delete(issue.Metadata.Extra, "agent_last_error")
			delete(issue.Metadata.Extra, AgentStatusExtraKey)
			delete(issue.Metadata.Extra, AgentStatusTimestampExtraKey)
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
		return "", "", errIssueTargetExists
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", "", fmt.Errorf("failed to inspect target directory: %w", statErr)
	}

	if err := s.moveIssue(issueID, targetStatus); err != nil {
		return "", "", fmt.Errorf("failed to move issue: %w", err)
	}

	return targetDir, targetStatus, nil
}

func (s *Server) deleteIssueBundle(issueID string) error {
	issueDir, _, err := s.findIssueDirectory(issueID)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(issueDir); err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	s.hub.Publish(NewEvent(EventIssueDeleted, IssueDeletedData{IssueID: issueID}))
	return nil
}

// getIssuesHandler retrieves a list of issues with optional filters
func (s *Server) getIssuesHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	appID := r.URL.Query().Get("app_id")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			limit = parsed
		}
	}

	issues, err := s.getAllIssues(status, priority, issueType, appID, limit)
	if err != nil {
		LogErrorErr("Failed to load issues", err,
			"status", status,
			"priority", priority,
			"type", issueType,
			"app_id", appID,
		)
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

	issue, storagePath, err := s.createIssueBundle(&req)
	if err != nil {
		var vErr issuespkg.ValidationError
		if errors.As(err, &vErr) {
			http.Error(w, vErr.Error(), http.StatusBadRequest)
			return
		}
		LogErrorErr("Failed to create issue", err)
		http.Error(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}

	issueResponse := *issue
	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: map[string]interface{}{
			"issue":        issueResponse,
			"issue_id":     issue.ID,
			"storage_path": storagePath,
		},
	}

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

	issue, _, _, err := s.loadIssueWithStatus(issueID)
	if err != nil {
		if strings.Contains(err.Error(), "issue not found") {
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}
		LogErrorErr("Failed to load issue", err, "issue_id", issueID)
		http.Error(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateIssueHandler updates an existing issue
func (s *Server) updateIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		http.Error(w, "Issue ID is required", http.StatusBadRequest)
		return
	}

	var req UpdateIssueRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		LogErrorErr("Invalid update payload", err, "issue_id", issueID)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	issue, err := s.updateIssueBundle(issueID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "issue not found") {
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}
		var vErr issuespkg.ValidationError
		if errors.As(err, &vErr) {
			http.Error(w, vErr.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, errIssueRunning) {
			http.Error(w, "Cannot change issue status while an agent is running", http.StatusConflict)
			return
		}
		if errors.Is(err, errIssueTargetExists) {
			http.Error(w, "Issue already exists in target status", http.StatusConflict)
			return
		}
		LogErrorErr("Failed to update issue", err, "issue_id", issueID)
		http.Error(w, "Failed to update issue", http.StatusInternalServerError)
		return
	}

	response := ApiResponse{
		Success: true,
		Message: "Issue updated successfully",
		Data: map[string]interface{}{
			"issue": issue,
		},
	}

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

	if err := s.deleteIssueBundle(issueID); err != nil {
		if strings.Contains(err.Error(), "issue not found") {
			http.Error(w, "Issue not found", http.StatusNotFound)
			return
		}
		LogErrorErr("Failed to delete issue", err, "issue_id", issueID)
		http.Error(w, "Failed to delete issue", http.StatusInternalServerError)
		return
	}

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
	folders := ValidIssueStatuses()
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
