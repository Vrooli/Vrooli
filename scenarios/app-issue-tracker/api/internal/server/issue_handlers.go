package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
	services "app-issue-tracker-api/internal/server/services"
)

const (
	maxIssuePayloadBytes  int64 = 2 << 20 // 2 MiB
	defaultIssueListLimit       = 20
	defaultSearchLimit          = 10
	maxIssueListLimit           = 200
)

func normalizeListLimit(raw string, defaultLimit, maxLimit int) int {
	limit := defaultLimit
	if strings.TrimSpace(raw) != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit <= 0 {
		return defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		return maxLimit
	}
	return limit
}

// getIssuesHandler retrieves a list of issues with optional filters
func (s *Server) getIssuesHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")
	issueType := r.URL.Query().Get("type")
	appID := r.URL.Query().Get("app_id")
	limitStr := r.URL.Query().Get("limit")

	limit := normalizeListLimit(limitStr, defaultIssueListLimit, maxIssueListLimit)

	issues, err := s.issues.ListIssues(status, priority, issueType, appID, limit)
	if err != nil {
		if errors.Is(err, services.ErrInvalidStatusFilter) {
			handlers.WriteError(w, http.StatusBadRequest, "Invalid status filter")
			return
		}
		logging.LogErrorErr("Failed to load issues", err,
			"status", status,
			"priority", priority,
			"type", issueType,
			"app_id", appID,
		)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load issues")
		return
	}

	response := ApiResponse{
		Success: true,
		Data: IssueListData{
			Issues: issues,
			Count:  len(issues),
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write issues response", err)
	}
}

// createIssueHandler creates a new issue
func (s *Server) createIssueHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateIssueRequest
	if err := handlers.DecodeJSON(r, &req, maxIssuePayloadBytes); err != nil {
		handlers.WriteDecodeError(w, err)
		return
	}

	issue, storagePath, err := s.issues.CreateIssue(&req)
	if err != nil {
		var vErr issuespkg.ValidationError
		if errors.As(err, &vErr) {
			handlers.WriteError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		logging.LogErrorErr("Failed to create issue", err)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to create issue")
		return
	}

	s.hub.Publish(NewEvent(EventIssueCreated, IssueEventData{Issue: issue}))

	response := ApiResponse{
		Success: true,
		Message: "Issue created successfully",
		Data: IssueCreateData{
			Issue:       issue,
			IssueID:     issue.ID,
			StoragePath: storagePath,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write create issue response", err)
	}
}

// getIssueHandler retrieves a single issue by ID
func (s *Server) getIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	issue, _, _, err := s.issues.LoadIssueWithStatus(issueID)
	if err != nil {
		if errors.Is(err, services.ErrIssueNotFound) {
			handlers.WriteError(w, http.StatusNotFound, "Issue not found")
			return
		}
		logging.LogErrorErr("Failed to load issue", err, "issue_id", issueID)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load issue")
		return
	}

	response := ApiResponse{
		Success: true,
		Data:    IssueData{Issue: issue},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write issue detail response", err, "issue_id", issueID)
	}
}

// updateIssueHandler updates an existing issue
func (s *Server) updateIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	var req UpdateIssueRequest
	if err := handlers.DecodeJSON(r, &req, maxIssuePayloadBytes); err != nil {
		logging.LogErrorErr("Invalid update payload", err, "issue_id", issueID)
		handlers.WriteDecodeError(w, err)
		return
	}

	issue, statusChange, err := s.issues.UpdateIssue(issueID, &req)
	if err != nil {
		if errors.Is(err, services.ErrIssueNotFound) {
			handlers.WriteError(w, http.StatusNotFound, "Issue not found")
			return
		}
		var vErr issuespkg.ValidationError
		if errors.As(err, &vErr) {
			handlers.WriteError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		if errors.Is(err, services.ErrIssueRunning) {
			handlers.WriteError(w, http.StatusConflict, "Cannot change issue status while an agent is running")
			return
		}
		if errors.Is(err, services.ErrIssueTargetExists) {
			handlers.WriteError(w, http.StatusConflict, "Issue already exists in target status")
			return
		}
		logging.LogErrorErr("Failed to update issue", err, "issue_id", issueID)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to update issue")
		return
	}

	if statusChange != nil && statusChange.From != statusChange.To {
		s.hub.Publish(NewEvent(EventIssueStatusChanged, IssueStatusChangedData{
			IssueID:   issueID,
			OldStatus: statusChange.From,
			NewStatus: statusChange.To,
		}))
	}

	s.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	response := ApiResponse{
		Success: true,
		Message: "Issue updated successfully",
		Data:    IssueData{Issue: issue},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write issue update response", err, "issue_id", issueID)
	}
}

// deleteIssueHandler deletes an issue
func (s *Server) deleteIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	if err := s.issues.DeleteIssue(issueID); err != nil {
		if errors.Is(err, services.ErrIssueNotFound) {
			handlers.WriteError(w, http.StatusNotFound, "Issue not found")
			return
		}
		if errors.Is(err, services.ErrIssueRunning) {
			handlers.WriteError(w, http.StatusConflict, "Cannot delete issue while an agent is running")
			return
		}
		logging.LogErrorErr("Failed to delete issue", err, "issue_id", issueID)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to delete issue")
		return
	}

	s.hub.Publish(NewEvent(EventIssueDeleted, IssueDeletedData{IssueID: issueID}))

	response := ApiResponse{
		Success: true,
		Message: "Issue deleted successfully",
		Data:    IssueDeleteData{IssueID: issueID},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write issue delete response", err, "issue_id", issueID)
	}
}

// searchIssuesHandler searches for issues using text matching
func (s *Server) searchIssuesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := normalizeListLimit(limitStr, defaultSearchLimit, maxIssueListLimit)

	results, err := s.issues.SearchIssues(query, limit)
	if err != nil {
		logging.LogErrorErr("Failed to search issues", err, "query", query)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to search issues")
		return
	}

	response := ApiResponse{
		Success: true,
		Data: IssueSearchData{
			Results: results,
			Count:   len(results),
			Query:   query,
			Method:  "text_search",
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to write search response", err, "query", query)
	}
}
