package aisearch

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// Handlers provides HTTP handlers for AI search operations.
type Handlers struct {
	service                   *Service
	budgetConfigStore         *BudgetConfigStore
	discoverFilterConfigStore *DiscoverFilterConfigStore
}

// NewHandlers creates new AI search handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// SetBudgetConfigStore sets the budget config store for config endpoints.
func (h *Handlers) SetBudgetConfigStore(store *BudgetConfigStore) {
	h.budgetConfigStore = store
}

// SetDiscoverFilterConfigStore sets the discover filter config store for config endpoints.
func (h *Handlers) SetDiscoverFilterConfigStore(store *DiscoverFilterConfigStore) {
	h.discoverFilterConfigStore = store
}

// Search handles POST /api/v1/search/ai - AI semantic search.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	var req AISearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Support both single query and multi-query
	queries := req.Queries
	if len(queries) == 0 && req.Query != "" {
		queries = []string{req.Query}
	}
	if len(queries) == 0 {
		http.Error(w, "Query is required (provide 'query' or 'queries')", http.StatusBadRequest)
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

	// For multi-query, search each independently and merge results
	var resp *AISearchResponse
	var err error
	if len(queries) == 1 {
		resp, err = h.service.SearchWithOptions(r.Context(), queries[0], limit, SearchOptions{
			Output:      output,
			Format:      req.Format,
			RenderLimit: req.RenderLimit,
		})
	} else {
		resp, err = h.service.SearchMultiWithOptions(r.Context(), queries, limit, SearchOptions{
			Output:      output,
			Format:      req.Format,
			RenderLimit: req.RenderLimit,
		})
	}
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

// SearchAgents handles POST /api/v1/search/agents/ai - AI semantic agent search.
func (h *Handlers) SearchAgents(w http.ResponseWriter, r *http.Request) {
	var req AIAgentSearchRequest
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

	resp, err := h.service.SearchAgents(r.Context(), req.Query, limit)
	if err != nil {
		log.Printf("[aisearch] Agent search error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SearchTeams handles POST /api/v1/search/teams/ai - AI semantic team search.
func (h *Handlers) SearchTeams(w http.ResponseWriter, r *http.Request) {
	var req AITeamSearchRequest
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

	resp, err := h.service.SearchTeams(r.Context(), req.Query, limit)
	if err != nil {
		log.Printf("[aisearch] Team search error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Discover handles POST /api/v1/discover - unified topic + skill discovery.
func (h *Handlers) Discover(w http.ResponseWriter, r *http.Request) {
	var req DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Queries) == 0 {
		http.Error(w, "queries is required", http.StatusBadRequest)
		return
	}

	complexity := strings.ToLower(strings.TrimSpace(req.Complexity))
	if complexity != "" && !ValidComplexity(complexity) {
		http.Error(w, "complexity must be one of: minor, moderate, major, architectural", http.StatusBadRequest)
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	discoverType := strings.ToLower(strings.TrimSpace(req.Type))
	if discoverType != "" && normalizeDiscoverType(discoverType) == "" {
		http.Error(w, "type must be one of: skill, action, all", http.StatusBadRequest)
		return
	}

	resp, err := h.service.DiscoverTyped(r.Context(), req.Queries, complexity, limit, discoverType)
	if err != nil {
		log.Printf("[aisearch] Discover error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetBudgetConfig handles GET /api/v1/config/budgets.
func (h *Handlers) GetBudgetConfig(w http.ResponseWriter, r *http.Request) {
	if h.budgetConfigStore == nil {
		http.Error(w, "budget config store not configured", http.StatusServiceUnavailable)
		return
	}

	cfg, err := h.budgetConfigStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// PutBudgetConfig handles PUT /api/v1/config/budgets.
func (h *Handlers) PutBudgetConfig(w http.ResponseWriter, r *http.Request) {
	if h.budgetConfigStore == nil {
		http.Error(w, "budget config store not configured", http.StatusServiceUnavailable)
		return
	}

	var cfg BudgetConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := ValidateBudgetConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.budgetConfigStore.Put(r.Context(), cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// GetDiscoverFilterConfig handles GET /api/v1/config/discover-filters.
func (h *Handlers) GetDiscoverFilterConfig(w http.ResponseWriter, r *http.Request) {
	if h.discoverFilterConfigStore == nil {
		http.Error(w, "discover filter config store not configured", http.StatusServiceUnavailable)
		return
	}

	cfg, err := h.discoverFilterConfigStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// PutDiscoverFilterConfig handles PUT /api/v1/config/discover-filters.
func (h *Handlers) PutDiscoverFilterConfig(w http.ResponseWriter, r *http.Request) {
	if h.discoverFilterConfigStore == nil {
		http.Error(w, "discover filter config store not configured", http.StatusServiceUnavailable)
		return
	}

	var cfg DiscoverFilterConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := ValidateDiscoverFilterConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.discoverFilterConfigStore.Put(r.Context(), cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}
