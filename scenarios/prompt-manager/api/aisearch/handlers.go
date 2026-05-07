package aisearch

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// Handlers provides HTTP handlers for AI search operations.
type Handlers struct {
	service                   *Service
	reconciler                *Reconciler
	budgetConfigStore         *BudgetConfigStore
	discoverFilterConfigStore *DiscoverFilterConfigStore
}

// NewHandlers creates new AI search handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// SetReconciler attaches the Reconciler used by the reconcile endpoints.
func (h *Handlers) SetReconciler(r *Reconciler) {
	h.reconciler = r
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

// Reconcile handles POST /api/v1/search/ai/reconcile.
//
// Behavior:
//   - X-Dry-Run: true (or ?dry_run=true) → 200 with the structured DriftReport,
//     no qdrant mutation, no embeds.
//   - default → 202 (Accepted); kicks off a background RunOnce. 409 if a
//     reconcile is already in progress.
//   - ?collection=skills|agents|teams|topics|actions filters to one
//     descriptor (still goes through the same Reconciler, against a per-call
//     filtered descriptor list).
func (h *Handlers) Reconcile(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		http.Error(w, "reconciler not configured", http.StatusServiceUnavailable)
		return
	}

	dryRun := isDryRun(r)
	collection := strings.TrimSpace(r.URL.Query().Get("collection"))

	rec := h.reconciler
	if collection != "" && collection != "all" {
		filtered, ok := filterByCollection(h.reconciler, collection)
		if !ok {
			http.Error(w, "unknown collection: "+collection, http.StatusBadRequest)
			return
		}
		rec = filtered
	}

	if dryRun {
		plan, err := rec.Plan(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"dry_run": true,
			"plan":    plan,
		})
		return
	}

	go func() {
		_, _, err := rec.RunOnce(context.Background())
		if err != nil && err != ErrReconcileBusy {
			log.Printf("[aisearch] background reconcile failed: %v", err)
		}
	}()

	// Brief non-blocking peek to see if we just collided with an in-flight run.
	// We use a tiny inline call: re-check status after a short wait — but to
	// keep this synchronous, just return 202; the caller polls /status.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(rec.Status())
}

// ReconcileStatus handles GET /api/v1/search/ai/reconcile/status.
func (h *Handlers) ReconcileStatus(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		http.Error(w, "reconciler not configured", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.reconciler.Status())
}

// CancelReconcile handles POST /api/v1/search/ai/reconcile/cancel.
func (h *Handlers) CancelReconcile(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		http.Error(w, "reconciler not configured", http.StatusServiceUnavailable)
		return
	}
	h.reconciler.Cancel()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.reconciler.Status())
}

func isDryRun(r *http.Request) bool {
	if v := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Dry-Run"))); v == "1" || v == "true" || v == "yes" {
		return true
	}
	if v := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("dry_run"))); v == "1" || v == "true" || v == "yes" {
		return true
	}
	return false
}

// filterByCollection returns a fresh Reconciler restricted to one descriptor
// kind. The new reconciler has independent state (mutex, last-plan/result), so
// per-collection ad-hoc requests don't serialize with the main full-corpus
// reconcile — that's intentional: a dry-run or scoped run shouldn't lock out
// the periodic loop.
func filterByCollection(r *Reconciler, name string) (*Reconciler, bool) {
	kind, ok := parseEntityKind(name)
	if !ok {
		return nil, false
	}
	for _, d := range r.Descriptors {
		if d.Kind == kind {
			return NewReconciler(r.Embedder, []CollectionDescriptor{d}, r.Parallelism), true
		}
	}
	return nil, false
}

func parseEntityKind(name string) (EntityKind, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "skill", "skills":
		return KindSkill, true
	case "agent", "agents":
		return KindAgent, true
	case "team", "teams":
		return KindTeam, true
	case "topic", "topics":
		return KindTopic, true
	case "action", "actions":
		return KindAction, true
	}
	return "", false
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

// SearchActions handles POST /api/v1/search/actions/ai - AI semantic action search.
func (h *Handlers) SearchActions(w http.ResponseWriter, r *http.Request) {
	var req AIActionSearchRequest
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

	resp, err := h.service.SearchActions(r.Context(), req.Query, limit)
	if err != nil {
		log.Printf("[aisearch] Action search error: %v", err)
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
