package releases

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"music-library/internal/module"
	domain "music-library/internal/releases"
)

type handler struct{ store *domain.Store }

func Module(db *database.RoutedDB) module.Module {
	h := &handler{store: domain.NewStoreWithLineageAdapter(db, nil, newImageToolsAssetVerifier(), newMusicToolsDerivationVerifier())}
	return module.Module{Name: "releases", Mount: h.mount, Endpoints: Endpoints}
}

func Schema() string { return domain.Schema() }

// NewHandler exposes the narrow HTTP surface for tests and local adapters that
// already own a domain store. Production composition still uses Module.
func NewHandler(store *domain.Store) http.Handler {
	h := &handler{store: store}
	r := mux.NewRouter()
	h.mount(r)
	return r
}

func (h *handler) mount(r *mux.Router) {
	r.HandleFunc("/api/v1/releases", h.list).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/releases", h.create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/releases/{release_id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/releases/{release_id}/rights-decisions", h.rightsDecision).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/releases/{release_id}/reviews", h.review).Methods(http.MethodPost)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 50 {
			http.Error(w, "limit must be an integer between 1 and 50", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	page, err := h.store.List(r.Context(), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var edition domain.Edition
	if !decodeJSON(w, r, &edition) {
		return
	}
	saved, err := h.store.SaveDraft(r.Context(), edition)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"edition": saved})
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	revision := int64(0)
	if raw := r.URL.Query().Get("revision"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 1 {
			http.Error(w, "revision must be a positive integer", http.StatusBadRequest)
			return
		}
		revision = parsed
	}
	edition, reviews, readiness, err := h.store.Assess(r.Context(), mux.Vars(r)["release_id"], revision)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edition": edition, "reviews": reviews, "eligibility": readiness})
}

func (h *handler) review(w http.ResponseWriter, r *http.Request) {
	var review domain.Review
	if !decodeJSON(w, r, &review) {
		return
	}
	if review.ReleaseID != "" && review.ReleaseID != mux.Vars(r)["release_id"] {
		http.Error(w, "release_id must match the route", http.StatusBadRequest)
		return
	}
	review.ReleaseID = mux.Vars(r)["release_id"]
	recorded, err := h.store.RecordReview(r.Context(), review)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"review": recorded})
}

func (h *handler) rightsDecision(w http.ResponseWriter, r *http.Request) {
	var decision domain.RightsDecision
	if !decodeJSON(w, r, &decision) {
		return
	}
	if decision.ReleaseID != "" && decision.ReleaseID != mux.Vars(r)["release_id"] {
		http.Error(w, "release_id must match the route", http.StatusBadRequest)
		return
	}
	decision.ReleaseID = mux.Vars(r)["release_id"]
	recorded, err := h.store.RecordRightsDecision(r.Context(), decision)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"rights_decision": recorded})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		http.Error(w, "request must contain exactly one JSON value", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrNotReady):
		status = http.StatusPreconditionFailed
	}
	http.Error(w, err.Error(), status)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "release_list", Path: "/api/v1/releases", Method: "GET", Summary: "List latest immutable release revisions", Description: "Return bounded newest-first release summaries and track counts without loading proof JSON or audio; use the opaque cursor to continue.", Category: "releases", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The release catalog UI needs keyset-paged JSON summaries with no generated Connect counterpart."}},
	{ID: "release_create", Path: "/api/v1/releases", Method: "POST", Summary: "Save immutable release draft", Description: "Create a versioned album or EP draft with exact catalog hashes, credits, artwork references, rights and derivation evidence; no audio is copied or published.", Category: "releases", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The release editor submits a structured edition with external asset and rights provenance; there is no generated Connect request for this evolving owner-bound payload."}},
	{ID: "release_get", Path: "/api/v1/releases/{release_id}", Method: "GET", Summary: "Read edition and eligibility", Description: "Read one immutable release revision and property-by-property evidence status, including unresolved canonical artist and artwork ownership proofs.", Category: "releases", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The editor reads a nested proof report with owner-resolved status that has no generated Connect counterpart."}},
	{ID: "release_rights_decision", Path: "/api/v1/releases/{release_id}/rights-decisions", Method: "POST", Summary: "Record exact-hash operator rights decision", Description: "Append a rights decision pinned to a track hash, release revision and the catalog receipt provenance; draft claims alone never satisfy rights readiness.", Category: "releases", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "This operator rights attestation is a browser workbench action without a generated Connect counterpart."}},
	{ID: "release_review", Path: "/api/v1/releases/{release_id}/reviews", Method: "POST", Summary: "Record one-batch EP review", Description: "Append an operator decision against a pinned edition revision after evidence eligibility checks; the decision is not a publish action.", Category: "releases", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "A listener-authored release review is a browser workbench action without a generated Connect counterpart."}},
}
