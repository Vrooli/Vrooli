package records

import (
	"errors"
	"net/http"
	"strconv"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler provides REST endpoints for records.
//
// REST surface (see PROBLEMS.md for the proto-conversion deferral):
//   POST   /api/v1/records
//   GET    /api/v1/records
//   GET    /api/v1/records/{id}
//   PATCH  /api/v1/records/{id}/narrative
//   POST   /api/v1/records/{id}/supersede
//   POST   /api/v1/records/search
type Handler struct {
	svc    *Service
	search Searcher
}

// Searcher is the semantic-search read seam used by POST /records/search.
// Production wiring is aisearch; tests substitute a fake. Nil is a valid
// dev-mode fallback that returns ErrSearchUnavailable.
//
// seam: records.Searcher
type Searcher interface {
	SearchRecords(query string, filter SearchFilter) ([]SearchHit, error)
}

// SearchFilter narrows a semantic query.
type SearchFilter struct {
	Kind     RecordKind
	Scenario string
	Limit    int
}

// SearchHit is one match returned from semantic search.
type SearchHit struct {
	Record Record  `json:"record"`
	Score  float64 `json:"score"`
}

// ErrSearchUnavailable is returned when no Searcher is wired.
var ErrSearchUnavailable = errors.New("records search is not configured")

// NewHandler constructs a Handler. `search` may be nil.
func NewHandler(svc *Service, search Searcher) *Handler {
	return &Handler{svc: svc, search: search}
}

// SetSearcher rewires the semantic-search seam post-construction. main.go uses
// this to install the aisearch-backed searcher after the aisearch service is
// constructed. Passing nil is a no-op (search endpoint stays unavailable).
func (h *Handler) SetSearcher(search Searcher) {
	h.search = search
}

// RegisterRoutes wires REST endpoints on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/records", h.create).Methods("POST")
	r.HandleFunc("/api/v1/records", h.list).Methods("GET")
	r.HandleFunc("/api/v1/records/search", h.search_).Methods("POST")
	r.HandleFunc("/api/v1/records/{id}", h.get).Methods("GET")
	r.HandleFunc("/api/v1/records/{id}/narrative", h.patchNarrative).Methods("PATCH")
	r.HandleFunc("/api/v1/records/{id}/supersede", h.supersede).Methods("POST")
}

type createRequest struct {
	Kind         string   `json:"kind"`
	Scenario     string   `json:"scenario"`
	BacklogRef   string   `json:"backlog_ref"`
	Supersedes   string   `json:"supersedes"`
	Trigger      string   `json:"trigger"`
	Approach     string   `json:"approach"`
	RuledOut     []string `json:"ruled_out"`
	Commit       string   `json:"commit"`
	FilesChanged []string `json:"files_changed"`
	Outcome      string   `json:"outcome"`
	CreatedBy    string   `json:"created_by"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[records] create", apierr.BadRequest("invalid body: %s", err))
		return
	}
	kind, err := ParseKind(req.Kind)
	if err != nil {
		apierr.MapError(w, "[records] create", apierr.BadRequest("%s", err))
		return
	}
	outcome, err := ParseOutcome(req.Outcome)
	if err != nil {
		apierr.MapError(w, "[records] create", apierr.BadRequest("%s", err))
		return
	}
	rec, err := h.svc.Create(r.Context(), CreateInput{
		Kind:         kind,
		Scenario:     req.Scenario,
		BacklogRef:   req.BacklogRef,
		Supersedes:   req.Supersedes,
		Trigger:      req.Trigger,
		Approach:     req.Approach,
		RuledOut:     req.RuledOut,
		Commit:       req.Commit,
		FilesChanged: req.FilesChanged,
		Outcome:      outcome,
		CreatedBy:    req.CreatedBy,
	})
	if err != nil {
		mapServiceErr(w, "[records] create", err)
		return
	}
	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{"record": rec})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := ListFilter{
		Scenario:     q.Get("scenario"),
		BacklogRef:   q.Get("backlog_ref"),
		IncludeStubs: q.Get("include_stubs") == "true",
	}
	if k := q.Get("kind"); k != "" {
		kind, err := ParseKind(k)
		if err != nil {
			apierr.MapError(w, "[records] list", apierr.BadRequest("%s", err))
			return
		}
		filter.Kind = kind
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			filter.Offset = n
		}
	}
	recs, err := h.svc.List(filter)
	if err != nil {
		apierr.MapError(w, "[records] list", apierr.Internal("list failed: %s", err))
		return
	}
	_ = httputil.JSON(w, map[string]any{"records": recs})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	rec, err := h.svc.Get(id)
	if err != nil {
		mapServiceErr(w, "[records] get", err)
		return
	}
	_ = httputil.JSON(w, map[string]any{"record": rec})
}

type narrativeRequest struct {
	Trigger      string   `json:"trigger"`
	Approach     string   `json:"approach"`
	RuledOut     []string `json:"ruled_out"`
	Commit       string   `json:"commit"`
	FilesChanged []string `json:"files_changed"`
	Outcome      string   `json:"outcome"`
}

func (h *Handler) patchNarrative(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req narrativeRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[records] narrative", apierr.BadRequest("invalid body: %s", err))
		return
	}
	var outcome Outcome
	if req.Outcome != "" {
		o, err := ParseOutcome(req.Outcome)
		if err != nil {
			apierr.MapError(w, "[records] narrative", apierr.BadRequest("%s", err))
			return
		}
		outcome = o
	}
	rec, err := h.svc.UpdateNarrative(r.Context(), id, Narrative{
		Trigger:      req.Trigger,
		Approach:     req.Approach,
		RuledOut:     req.RuledOut,
		Commit:       req.Commit,
		FilesChanged: req.FilesChanged,
		Outcome:      outcome,
	})
	if err != nil {
		mapServiceErr(w, "[records] narrative", err)
		return
	}
	_ = httputil.JSON(w, map[string]any{"record": rec})
}

type supersedeRequest struct {
	SuccessorID string `json:"successor_id"`
	Reason      string `json:"reason"`
}

func (h *Handler) supersede(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req supersedeRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[records] supersede", apierr.BadRequest("invalid body: %s", err))
		return
	}
	if req.SuccessorID == "" {
		apierr.MapError(w, "[records] supersede", apierr.BadRequest("successor_id is required"))
		return
	}
	rec, err := h.svc.Supersede(r.Context(), id, req.SuccessorID, req.Reason)
	if err != nil {
		mapServiceErr(w, "[records] supersede", err)
		return
	}
	_ = httputil.JSON(w, map[string]any{"record": rec})
}

type searchRequest struct {
	Query    string `json:"query"`
	Kind     string `json:"kind"`
	Scenario string `json:"scenario"`
	Limit    int    `json:"limit"`
}

func (h *Handler) search_(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[records] search", apierr.BadRequest("invalid body: %s", err))
		return
	}
	if req.Query == "" {
		apierr.MapError(w, "[records] search", apierr.BadRequest("query is required"))
		return
	}
	if h.search == nil {
		apierr.MapError(w, "[records] search", apierr.Internal("search not configured"))
		return
	}
	filter := SearchFilter{Scenario: req.Scenario, Limit: req.Limit}
	if req.Kind != "" {
		k, err := ParseKind(req.Kind)
		if err != nil {
			apierr.MapError(w, "[records] search", apierr.BadRequest("%s", err))
			return
		}
		filter.Kind = k
	}
	hits, err := h.search.SearchRecords(req.Query, filter)
	if err != nil {
		apierr.MapError(w, "[records] search", apierr.Internal("search failed: %s", err))
		return
	}
	_ = httputil.JSON(w, map[string]any{"hits": hits})
}

func mapServiceErr(w http.ResponseWriter, prefix string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, prefix, apierr.NotFound("record not found"))
	case errors.Is(err, ErrStubLocked):
		apierr.MapError(w, prefix, apierr.Conflict("%s", err))
	case errors.Is(err, ErrAlreadySuperseded):
		apierr.MapError(w, prefix, apierr.Conflict("%s", err))
	case errors.Is(err, ErrSupersedeCycle):
		apierr.MapError(w, prefix, apierr.BadRequest("%s", err))
	default:
		apierr.MapError(w, prefix, apierr.Internal("%s", err))
	}
}
