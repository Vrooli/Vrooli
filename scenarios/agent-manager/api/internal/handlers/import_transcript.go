package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
)

func (h *Handler) ImportTranscriptHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path         string            `json:"path"`
		AttachmentID string            `json:"attachmentId"`
		RunnerType   domain.RunnerType `json:"runnerType"`
		Label        string            `json:"label"`
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
	run, err := h.svc.ImportTranscript(r.Context(), orchestration.ImportTranscriptRequest{Path: body.Path, RunnerType: body.RunnerType, Label: body.Label, LabelSource: labelSource})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}
