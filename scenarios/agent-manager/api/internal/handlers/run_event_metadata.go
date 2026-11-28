package handlers

import (
	"context"
	"net/http"

	"agent-manager/internal/adapters/event"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type runEventMetadataService interface {
	ListRunEventMetadata(context.Context, uuid.UUID, event.GetOptions) ([]event.Metadata, bool, error)
}

type runEventMetadataResponse struct {
	RunID    string           `json:"run_id"`
	Events   []event.Metadata `json:"events"`
	HasMore  bool             `json:"has_more"`
	Metadata bool             `json:"metadata_only"`
}

// GetRunEventMetadata returns a bounded, prompt-free caller-source projection.
// It uses a dedicated owner API instead of event pages so message, tool and
// artifact bodies are never serialized into the response.
func (h *Handler) GetRunEventMetadata(w http.ResponseWriter, r *http.Request) {
	runID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}
	service, ok := h.svc.EventService.(runEventMetadataService)
	if !ok || service == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "metadata-only event reader unavailable")
		return
	}
	opts := event.GetOptions{AfterSequence: -1, Limit: 100}
	if after, provided, err := parseQueryInt64Strict(r, "after_sequence", "afterSequence"); err != nil {
		writeSimpleError(w, r, "after_sequence", "must be a number")
		return
	} else if provided {
		opts.AfterSequence = after
	}
	if limit, provided, err := parseQueryIntStrict(r, "limit"); err != nil {
		writeSimpleError(w, r, "limit", "must be a number")
		return
	} else if provided {
		if limit < 1 || limit > 1000 {
			writeSimpleError(w, r, "limit", "must be between 1 and 1000")
			return
		}
		opts.Limit = limit
	}
	items, more, err := service.ListRunEventMetadata(r.Context(), runID, opts)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if items == nil {
		items = []event.Metadata{}
	}
	writeJSON(w, http.StatusOK, runEventMetadataResponse{RunID: runID.String(), Events: items, HasMore: more, Metadata: true})
}
