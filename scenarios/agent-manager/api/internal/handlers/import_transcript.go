package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/transcriptindex"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (h *Handler) ListTranscriptReplacementIssues(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["id"]
	if _, err := uuid.Parse(runID); err != nil {
		writeSimpleError(w, r, "id", "valid run ID is required")
		return
	}
	owner, ok := h.transcriptImporter.(interface {
		ReplacementIssues(context.Context, string) (transcriptindex.ReplacementIssues, error)
	})
	if !ok {
		writeSimpleError(w, r, "service", "transcript replacement inspection is unavailable")
		return
	}
	issues, err := owner.ReplacementIssues(r.Context(), runID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

// ReconcileTranscriptReplacement retries one explicitly guarded replacement;
// uncertain event-recording outcomes are outside this operation.
func (h *Handler) ReconcileTranscriptReplacement(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["id"]
	if _, err := uuid.Parse(runID); err != nil {
		writeSimpleError(w, r, "id", "valid run ID is required")
		return
	}
	var body struct {
		FileKey string `json:"fileKey"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		writeSimpleError(w, r, "body", "one JSON object is required")
		return
	}
	seen := false
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil || key != "fileKey" || seen || decoder.Decode(&body.FileKey) != nil {
			writeSimpleError(w, r, "fileKey", "one exact file key is required")
			return
		}
		seen = true
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') || !seen || strings.TrimSpace(body.FileKey) == "" || len(body.FileKey) > 256 {
		writeSimpleError(w, r, "fileKey", "exact file key is required")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeSimpleError(w, r, "body", "one JSON object is required")
		return
	}
	owner, ok := h.transcriptImporter.(interface {
		ReconcileReplacement(context.Context, string, string) error
	})
	if !ok {
		writeSimpleError(w, r, "service", "transcript replacement reconciliation is unavailable")
		return
	}
	if err := owner.ReconcileReplacement(r.Context(), runID, body.FileKey); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"runId": runID, "fileKey": body.FileKey, "status": "reconciled"})
}

func (h *Handler) ImportTranscriptHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path            string            `json:"path"`
		AttachmentID    string            `json:"attachmentId"`
		RunnerType      domain.RunnerType `json:"runnerType"`
		Label           string            `json:"label"`
		SourceHarness   string            `json:"sourceHarness"`
		SourceSessionID string            `json:"sourceSessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeSimpleError(w, r, "body", "invalid import transcript request")
		return
	}
	if body.Path == "" && body.AttachmentID != "" {
		if h.storage == nil {
			writeSimpleError(w, r, "attachmentId", "attachment storage is not configured")
			return
		}
		attachment, err := h.storage.Get(r.Context(), body.AttachmentID)
		if err != nil {
			writeError(w, r, err)
			return
		}
		body.Path = h.storage.GetFilePath(attachment.StoragePath)
	}
	labelSource := domain.RunLabelSource("")
	if strings.TrimSpace(body.Label) != "" {
		labelSource = domain.RunLabelSourceManual
	}
	run, err := h.svc.ImportTranscript(r.Context(), orchestration.ImportTranscriptRequest{Path: body.Path, RunnerType: body.RunnerType, Label: body.Label, LabelSource: labelSource, SourceHarness: body.SourceHarness, SourceSessionID: body.SourceSessionID})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}
