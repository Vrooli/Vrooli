package main

// search_surface.go is Phase 6 of the KO documentation search cutover: the
// converged REST surface for the hybrid engine (internal/aisearch). It mirrors
// cli-health's verb semantics — `search query` / `search status` and
// `reindex run` / `reindex status` / `reindex cancel` — over KO's established
// REST surface (KO is REST-everywhere; only DocHealth is Connect, and the
// search-hub federation contract requires the REST `/docs/search/unified`
// endpoint be preserved). The new hybrid engine becomes the implementation
// behind these endpoints, the re-pointed `/knowledge/search`, and the
// `/docs/search/unified` semantic leg.

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	pkg "github.com/vrooli/aisearch-go"

	"knowledge-observatory/internal/services/docsearch"
)

// defaultDocEmbedsPerTick caps embeds per background reconcile tick when
// KO_DOCS_MAX_EMBEDS_PER_TICK is unset, so the first full index of the large
// documentation corpus never starves Ollama (plan §4.2). Overridable via env;
// a one-shot `reindex run` plans uncapped.
const defaultDocEmbedsPerTick = 800

// searchTokenHolder caches the control token search-hub mints for the
// knowledge-observatory.docs provider at self-registration. The registration
// goroutine calls Set when the hub echoes the token; the reindex handler's
// gate reads it via Get on every authenticated request. A restart loses it
// (memory only) and the next boot's re-registration re-acquires it — search-hub
// persists the authoritative copy. Get returns "" until Set runs; the gate
// treats an empty server-side token as "deny all" (not yet registered).
type searchTokenHolder struct {
	mu    sync.RWMutex
	token string
}

func (h *searchTokenHolder) Set(token string) {
	h.mu.Lock()
	h.token = token
	h.mu.Unlock()
}

func (h *searchTokenHolder) Get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.token
}

// validateControlToken returns true when the presented token matches the
// server-side token using a constant-time comparison (prevents timing attacks).
// Returns false when either token is empty (server not yet registered, or
// caller omitted the token).
func validateControlToken(serverToken, presented string) bool {
	if serverToken == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(serverToken)) == 1
}

// docSearchEngine is the read-path seam the api package depends on so handlers
// stay testable with a fake (the concrete *aisearch.SearchService satisfies
// it). It is the shared pkg.Service surface. Search mirrors the engine's
// variadic SearchOption signature so the concrete *pkg.Service satisfies this
// interface (KO passes no options today — the query-time override channel is a
// cli-health adopter feature; KO keeps its measured default behavior).
type docSearchEngine interface {
	Search(ctx context.Context, q pkg.SearchQuery, opts ...pkg.SearchOption) (pkg.SearchResponse, error)
	Status(ctx context.Context) pkg.StatusReport
}

// docSemanticAdapter bridges the hybrid documentation engine to docsearch's
// SemanticSearcher seam so the unified endpoint's semantic leg returns real
// hybrid results (the §1 junk / content:"" failure is fixed). It queries in
// hybrid mode (the unified endpoint runs its own file/text legs), so it never
// recurses into the grep fallback.
type docSemanticAdapter struct {
	engine docSearchEngine
}

func (a docSemanticAdapter) Search(ctx context.Context, req docsearch.SemanticRequest) (docsearch.SemanticResponse, error) {
	if a.engine == nil {
		return docsearch.SemanticResponse{}, nil
	}
	resp, err := a.engine.Search(ctx, pkg.SearchQuery{
		Query: req.Query,
		Mode:  pkg.ModeHybrid,
		Scope: scopeFromDocsearch(req.Scope, req.Scenario, req.BasePath),
		Limit: req.Limit,
	})
	if err != nil {
		return docsearch.SemanticResponse{}, err
	}
	out := make([]docsearch.SemanticResult, 0, len(resp.Results))
	for _, h := range resp.Results {
		out = append(out, docsearch.SemanticResult{
			ID:       h.ID,
			Score:    h.Score,
			Content:  h.Snippet,
			Metadata: h.Payload,
		})
	}
	return docsearch.SemanticResponse{Results: out}, nil
}

// scopeFromDocsearch maps docsearch's string scope onto the engine's typed
// Scope (mirrors the grep fallback's scope handling, in reverse).
func scopeFromDocsearch(scope, scenario, basePath string) pkg.Scope {
	switch strings.TrimSpace(scope) {
	case docsearch.ScopeScenario:
		return pkg.Scope{Kind: pkg.ScopeScenario, Value: scenario}
	case docsearch.ScopePath:
		return pkg.Scope{Kind: pkg.ScopePath, Value: basePath}
	default:
		return pkg.Scope{Kind: pkg.ScopeGlobal}
	}
}

// ---------------------------------------------------------------------------
// search query / status
// ---------------------------------------------------------------------------

// docSearchQueryRequest is the body for POST /api/v1/search/query. Scope and
// Facets map directly onto the engine's typed filters; Mode selects the
// retrieval strategy (auto|hybrid|dense|text).
type docSearchQueryRequest struct {
	Query  string `json:"query"`
	Mode   string `json:"mode,omitempty"`
	Scope  string `json:"scope,omitempty"`  // global|scenario|path
	Target string `json:"target,omitempty"` // scenario name or path prefix
	Limit  int    `json:"limit,omitempty"`
	Facets struct {
		DocType      []string `json:"doc_type,omitempty"`
		Audience     []string `json:"audience,omitempty"`
		Maturity     []string `json:"maturity,omitempty"`
		CanonicalFor []string `json:"canonical_for,omitempty"`
	} `json:"facets,omitempty"`
}

// docSearchHit is one projected result; the first five fields are the
// search-hub federation contract ({id, relative_path, score, snippet, path}).
type docSearchHit struct {
	ID           string                 `json:"id"`
	RelativePath string                 `json:"relative_path"`
	Score        float64                `json:"score"`
	Snippet      string                 `json:"snippet"`
	Path         string                 `json:"path"`
	Scenario     string                 `json:"scenario,omitempty"`
	DocType      string                 `json:"doc_type,omitempty"`
	Title        string                 `json:"title,omitempty"`
	HeadingPath  string                 `json:"heading_path,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// docSearchQueryResponse echoes which leg answered and the active reranker so
// the UI can render a degraded-mode hint.
type docSearchQueryResponse struct {
	Results  []docSearchHit `json:"results"`
	Total    int            `json:"total"`
	Query    string         `json:"query"`
	Method   string         `json:"method"`
	Reranker string         `json:"reranker"`
}

func (s *Server) handleSearchQuery(w http.ResponseWriter, r *http.Request) {
	var req docSearchQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		s.respondError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}
	if s == nil || s.docSearch == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation search is unavailable")
		return
	}

	q := pkg.SearchQuery{
		Query: req.Query,
		Mode:  parseSearchMode(req.Mode),
		Scope: parseScope(req.Scope, req.Target),
		Limit: req.Limit,
		Facets: pkg.Facets{
			DocType:      req.Facets.DocType,
			Audience:     req.Facets.Audience,
			Maturity:     req.Facets.Maturity,
			CanonicalFor: req.Facets.CanonicalFor,
		},
	}
	resp, err := s.docSearch.Search(r.Context(), q)
	if err != nil {
		s.log("doc search failed", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to execute search")
		return
	}
	writeJSON(w, docSearchQueryResponse{
		Results:  projectDocHits(resp.Results),
		Total:    resp.Total,
		Query:    resp.Query,
		Method:   resp.Method,
		Reranker: resp.Reranker,
	})
}

// docSearchStatusResponse mirrors cli-health's status shape for the
// `search status` verb.
type docSearchStatusResponse struct {
	Available            bool   `json:"available"`
	Ollama               bool   `json:"ollama"`
	Qdrant               bool   `json:"qdrant"`
	Reranker             string `json:"reranker"`
	IndexedCount         int    `json:"indexed_count"`
	LastReconcileAt      string `json:"last_reconcile_at"`
	LastReconcileOutcome string `json:"last_reconcile_outcome"`
}

func (s *Server) handleSearchStatus(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docSearch == nil {
		writeJSON(w, docSearchStatusResponse{Available: false})
		return
	}
	st := s.docSearch.Status(r.Context())
	writeJSON(w, docSearchStatusResponse{
		Available:            st.Available,
		Ollama:               st.Ollama,
		Qdrant:               st.Qdrant,
		Reranker:             st.Reranker,
		IndexedCount:         st.IndexedCount,
		LastReconcileAt:      st.LastReconcileAt,
		LastReconcileOutcome: st.LastReconcileOutcome,
	})
}

// ---------------------------------------------------------------------------
// reindex run / status / cancel
// ---------------------------------------------------------------------------

type reindexRunRequest struct {
	DryRun bool `json:"dry_run,omitempty"`
	// ControlToken, when present, authenticates the request as a search-hub
	// triggered reindex. Validated against the in-memory token minted at
	// self-registration. Unauthenticated calls (empty token) are still accepted
	// as locally-initiated reindexes — search-hub is an optional dependency.
	ControlToken string `json:"control_token,omitempty"`
}

type reindexRunResponse struct {
	State     string `json:"state"` // "planned" | "running" | "busy"
	DryRun    bool   `json:"dry_run"`
	Planned   int    `json:"planned_upserts"` // chunks to upsert
	Deletes   int    `json:"planned_deletes"` // ghost chunks to delete
	Refreshed int    `json:"planned_refresh"` // payloads refreshed without re-embedding
}

func (s *Server) handleReindexRun(w http.ResponseWriter, r *http.Request) {
	var req reindexRunRequest
	if r.Body != nil {
		// An empty body is allowed (defaults to a full async run).
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// When a control_token is presented, validate it against the in-memory
	// token minted at self-registration. A mismatch is rejected with 403 so a
	// rogue actor cannot trigger a reindex by guessing the endpoint. Requests
	// that omit the token entirely are accepted as locally-initiated reindexes
	// (search-hub is an optional dependency — the endpoint must remain usable
	// without it). An empty server-side token (not yet registered) also rejects
	// a presented token, because we cannot verify it.
	if req.ControlToken != "" {
		var serverToken string
		if s != nil && s.searchToken != nil {
			serverToken = s.searchToken.Get()
		}
		if !validateControlToken(serverToken, req.ControlToken) {
			s.respondError(w, http.StatusForbidden, "Invalid control token")
			return
		}
	}

	if s == nil || s.docIndexer == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation indexer is unavailable")
		return
	}

	if req.DryRun {
		res, err := s.docIndexer.Reindex(r.Context(), true)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to plan reindex: "+err.Error())
			return
		}
		writeJSON(w, reindexRunResponse{
			State:     "planned",
			DryRun:    true,
			Planned:   res.Planned,
			Deletes:   res.Deletes,
			Refreshed: res.Refreshed,
		})
		return
	}

	rec := s.docIndexer.Reconciler()
	if rec.Status().Running {
		writeJSONCode(w, http.StatusAccepted, reindexRunResponse{State: "busy"})
		return
	}
	// Plan first so the caller gets the planned counts; then apply in the
	// background (detached context) so the request returns immediately.
	plan, err := s.docIndexer.Reindex(r.Context(), true)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to plan reindex: "+err.Error())
		return
	}
	go func() {
		if _, _, rerr := rec.RunOnce(context.Background()); rerr != nil {
			s.log("reindex run failed", map[string]interface{}{"error": rerr.Error()})
		}
	}()
	writeJSONCode(w, http.StatusAccepted, reindexRunResponse{
		State:     "running",
		Planned:   plan.Planned,
		Deletes:   plan.Deletes,
		Refreshed: plan.Refreshed,
	})
}

type reindexStatusResponse struct {
	State      string `json:"state"` // idle|running|succeeded|failed|cancelled
	Running    bool   `json:"running"`
	Processed  int    `json:"processed"`
	Total      int    `json:"total"`
	Deferred   int    `json:"deferred"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
}

func (s *Server) handleReindexStatus(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docIndexer == nil {
		writeJSON(w, reindexStatusResponse{State: "unavailable"})
		return
	}
	st := s.docIndexer.Reconciler().Status()
	resp := reindexStatusResponse{
		Running:    st.Running,
		Error:      st.LastError,
		StartedAt:  st.StartedAt,
		FinishedAt: st.FinishedAt,
	}
	if st.LastPlan != nil {
		for _, c := range st.LastPlan.Collections {
			resp.Total += len(c.ToUpsert)
		}
	}
	if st.LastResult != nil {
		for _, c := range st.LastResult.Collections {
			resp.Processed += c.Upserted
		}
		resp.Deferred = st.LastResult.Deferred
	}
	switch {
	case st.Running:
		resp.State = "running"
	case st.Canceled:
		resp.State = "cancelled"
	case st.LastError != "":
		resp.State = "failed"
	case st.FinishedAt != "":
		resp.State = "succeeded"
	default:
		resp.State = "idle"
	}
	writeJSON(w, resp)
}

func (s *Server) handleReindexCancel(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docIndexer == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation indexer is unavailable")
		return
	}
	running := s.docIndexer.Reconciler().Status().Running
	s.docIndexer.Reconciler().Cancel()
	writeJSON(w, map[string]bool{"cancelled": running})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func parseSearchMode(mode string) pkg.SearchMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "hybrid":
		return pkg.ModeHybrid
	case "dense":
		return pkg.ModeDense
	case "text":
		return pkg.ModeText
	case "auto", "":
		return pkg.ModeAuto
	default:
		return pkg.ModeAuto
	}
}

func parseScope(kind, target string) pkg.Scope {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "scenario":
		return pkg.Scope{Kind: pkg.ScopeScenario, Value: target}
	case "path":
		return pkg.Scope{Kind: pkg.ScopePath, Value: target}
	default:
		return pkg.Scope{Kind: pkg.ScopeGlobal}
	}
}

func projectDocHits(hits []pkg.SearchResult) []docSearchHit {
	out := make([]docSearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, docSearchHit{
			ID:           h.ID,
			RelativePath: h.RelativePath,
			Score:        h.Score,
			Snippet:      h.Snippet,
			Path:         h.Path,
			Scenario:     payloadString(h.Payload, "scenario"),
			DocType:      payloadString(h.Payload, "doc_type"),
			Title:        payloadString(h.Payload, "title"),
			HeadingPath:  payloadString(h.Payload, "heading_path"),
			Metadata:     h.Payload,
		})
	}
	return out
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeJSONCode sets the content type and status code before encoding the body.
func writeJSONCode(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
