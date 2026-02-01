package aisearch

import (
	"encoding/json"
	"log"
	"net/http"
)

// Handlers provides HTTP handlers for AI search operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new AI search handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// Search handles POST /api/v1/search/ai - AI semantic search.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	var req AISearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	output := normalizeSearchOutput(req.Output)
	if output == "" {
		http.Error(w, "Output must be 'results', 'combined', or 'both'", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SearchWithOptions(r.Context(), req.Query, limit, SearchOptions{
		Output:      output,
		Format:      req.Format,
		RenderLimit: req.RenderLimit,
	})
	if err != nil {
		log.Printf("[aisearch] Search error: %v", err)
		if outputIncludesCombined(output) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Status handles GET /api/v1/search/ai/status - check AI availability.
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	status := h.service.GetStatus(r.Context())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// Reindex handles POST /api/v1/search/ai/reindex - rebuild vector index.
func (h *Handlers) Reindex(w http.ResponseWriter, r *http.Request) {
	// Check if AI services are available first
	status := h.service.GetStatus(r.Context())
	if !status.Available {
		http.Error(w, status.Message, http.StatusServiceUnavailable)
		return
	}

	resp, started := h.service.StartReindex()

	w.Header().Set("Content-Type", "application/json")
	if started {
		w.WriteHeader(http.StatusAccepted)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ReindexStatus handles GET /api/v1/search/ai/reindex/status - check reindex status.
func (h *Handlers) ReindexStatus(w http.ResponseWriter, r *http.Request) {
	status := h.service.ReindexStatus()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// CancelReindex handles POST /api/v1/search/ai/reindex/cancel - cancel active reindex.
func (h *Handlers) CancelReindex(w http.ResponseWriter, r *http.Request) {
	status := h.service.CancelReindex()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
