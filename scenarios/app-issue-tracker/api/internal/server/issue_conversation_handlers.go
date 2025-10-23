package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/handlers"
	services "app-issue-tracker-api/internal/server/services"
)

// getIssueAttachmentHandler serves an issue attachment file
func (s *Server) getIssueAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	attachmentPath := strings.TrimSpace(vars["attachment"])
	if issueID == "" || attachmentPath == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID and attachment path are required")
		return
	}

	decodedPath, err := url.PathUnescape(attachmentPath)
	if err != nil {
		handlers.WriteError(w, http.StatusBadRequest, "Invalid attachment path")
		return
	}

	resource, err := s.content.ResolveAttachment(issueID, decodedPath)
	if err != nil {
		if errors.Is(err, ErrInvalidAttachmentPath) {
			handlers.WriteError(w, http.StatusBadRequest, "Invalid attachment path")
			return
		}
		if errors.Is(err, ErrAttachmentNotFound) || errors.Is(err, services.ErrIssueNotFound) {
			handlers.WriteError(w, http.StatusNotFound, "Attachment not found")
			return
		}
		logging.LogErrorErr("Failed to resolve attachment", err, "issue_id", issueID, "attachment", decodedPath)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to open attachment")
		return
	}
	defer resource.File.Close()

	w.Header().Set("Content-Type", resource.ContentType)
	disposition := s.content.disposition(resource.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, resource.DownloadName))
	if resource.Meta != nil && resource.Meta.Category != "" {
		w.Header().Set("X-Attachment-Category", resource.Meta.Category)
	}
	w.Header().Set("Cache-Control", "no-store")

	http.ServeContent(w, r, resource.DownloadName, resource.ModTime, resource.File)
}

// getIssueAgentConversationHandler streams the captured agent transcript for UI rendering
func (s *Server) getIssueAgentConversationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := strings.TrimSpace(vars["id"])
	if issueID == "" {
		handlers.WriteError(w, http.StatusBadRequest, "Issue ID is required")
		return
	}

	payload, err := s.content.AgentConversation(issueID, 750)
	if err != nil {
		if errors.Is(err, services.ErrIssueNotFound) {
			handlers.WriteError(w, http.StatusNotFound, "Issue not found")
			return
		}
		if errors.Is(err, ErrInvalidAttachmentPath) {
			handlers.WriteError(w, http.StatusBadRequest, "Invalid conversation path")
			return
		}
		logging.LogErrorErr("Failed to load conversation", err, "issue_id", issueID)
		handlers.WriteError(w, http.StatusInternalServerError, "Failed to load issue")
		return
	}

	response := ApiResponse{
		Success: true,
		Data: map[string]interface{}{
			"conversation": payload,
		},
	}

	if err := handlers.WriteJSON(w, http.StatusOK, response); err != nil {
		logging.LogErrorErr("Failed to encode conversation response", err, "issue_id", issueID)
	}
}

func (s *Server) resolveAgentFilePath(raw string) (string, error) {
	return services.ResolveScenarioPath(s.config.ScenarioRoot, raw, "tmp")
}
