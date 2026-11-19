// Package decomposition exposes durable Music Tools results and listener review.
package decomposition

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"music-library/internal/decomposition"
	"music-library/internal/module"
)

var hashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

type handler struct{ store *decomposition.Store }

func Module(db *database.RoutedDB) module.Module {
	h := &handler{store: decomposition.NewStore(db)}
	return module.Module{Name: "decomposition", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/decompositions", h.list).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/decompositions/{track_sha256}", h.put).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/decompositions/{track_sha256}", h.get).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/decompositions/{track_sha256}/reviews", h.review).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/decompositions/{track_sha256}/audio", h.audio).Methods(http.MethodGet, http.MethodHead)
	}, Endpoints: Endpoints}
}

func Schema() string { return decomposition.Schema() }

func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	track := mux.Vars(r)["track_sha256"]
	if !hashRE.MatchString(track) {
		http.Error(w, "track_sha256 must be a lowercase SHA-256", http.StatusBadRequest)
		return
	}
	var input struct {
		AnalysisJob string                       `json:"analysis_job_id"`
		SourceSHA   string                       `json:"source_sha256"`
		Source      decomposition.SourceMetadata `json:"source,omitempty"`
		ToolResult  json.RawMessage              `json:"tool_result"`
	}
	if !decodeJSON(w, r, 65<<20, &input) {
		return
	}
	result, err := h.store.Put(r.Context(), decomposition.Result{TrackSHA256: track, AnalysisJob: input.AnalysisJob, SourceSHA256: input.SourceSHA, Source: input.Source, ToolResult: input.ToolResult})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) audio(w http.ResponseWriter, r *http.Request) {
	track := mux.Vars(r)["track_sha256"]
	if !hashRE.MatchString(track) {
		http.Error(w, "track_sha256 must be a lowercase SHA-256", http.StatusBadRequest)
		return
	}
	if _, err := h.store.Get(r.Context(), track, ""); err != nil {
		writeError(w, err)
		return
	}
	base, err := musicToolsBaseURL(r.Context())
	if err != nil {
		http.Error(w, "private source audio service is unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	target := strings.TrimRight(base, "/") + "/api/v1/sources/" + track + "/audio"
	req, err := http.NewRequestWithContext(ctx, r.Method, target, nil)
	if err != nil {
		http.Error(w, "could not create private audio request", http.StatusInternalServerError)
		return
	}
	for _, name := range []string{"Range", "If-Range", "If-Modified-Since"} {
		if value := r.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}
	client := &http.Client{Timeout: 0, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "private source audio service could not be reached", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	for _, name := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag", "Content-Disposition"} {
		if value := resp.Header.Get(name); value != "" {
			w.Header().Set(name, value)
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent) && !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "audio/") {
		http.Error(w, "source service returned a non-audio response", http.StatusBadGateway)
		return
	}
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead && resp.StatusCode != http.StatusNotModified {
		_, _ = io.Copy(w, resp.Body)
	}
}

func musicToolsBaseURL(ctx context.Context) (string, error) {
	base := strings.TrimSpace(os.Getenv("MUSIC_TOOLS_URL"))
	if base == "" {
		var err error
		base, err = discovery.ResolveScenarioURLDefault(ctx, "music-tools")
		if err != nil {
			return "", err
		}
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	ip := net.ParseIP(host)
	if u.Scheme != "http" || u.Port() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || !(host == "localhost" || (ip != nil && ip.IsLoopback())) {
		return "", errors.New("Music Tools URL must be explicit loopback HTTP")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "limit must be an integer between 1 and 50", http.StatusBadRequest)
			return
		}
		if parsed < 1 || parsed > 50 {
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

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	track := mux.Vars(r)["track_sha256"]
	if !hashRE.MatchString(track) {
		http.Error(w, "track_sha256 must be a lowercase SHA-256", http.StatusBadRequest)
		return
	}
	result, err := h.store.Get(r.Context(), track, r.URL.Query().Get("analysis_job_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) review(w http.ResponseWriter, r *http.Request) {
	track := mux.Vars(r)["track_sha256"]
	if !hashRE.MatchString(track) {
		http.Error(w, "track_sha256 must be a lowercase SHA-256", http.StatusBadRequest)
		return
	}
	var input struct {
		AnalysisJob string                     `json:"analysis_job_id"`
		Verdict     string                     `json:"verdict"`
		Note        string                     `json:"note"`
		Corrections []decomposition.Correction `json:"timestamped_corrections,omitempty"`
	}
	if !decodeJSON(w, r, 1<<20, &input) {
		return
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		http.Error(w, "could not allocate review identity", http.StatusInternalServerError)
		return
	}
	review, err := h.store.Review(r.Context(), decomposition.Review{ID: hex.EncodeToString(id[:]), TrackSHA256: track, AnalysisJob: input.AnalysisJob, Verdict: input.Verdict, Note: input.Note, Corrections: input.Corrections})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, out any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	if err := dec.Decode(out); err != nil {
		http.Error(w, "invalid JSON request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		http.Error(w, "request must contain one JSON value", http.StatusBadRequest)
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, decomposition.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, decomposition.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, decomposition.ErrConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "decomposition storage failed", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "decomposition_list", Path: "/api/v1/decompositions", Method: "GET", Summary: "List analyzed tracks for review", Description: "Return stable newest-first pages of track/job summaries, source title/artist and review counts without loading result blobs; use the opaque next cursor to continue.", Category: "decomposition", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "A browser review queue needs paginated JSON discovery and has no generated Connect RPC counterpart."}},
	{ID: "decomposition_put", Path: "/api/v1/decompositions/{track_sha256}", Method: "POST", Summary: "Save Music Tools analysis", Description: "Store the exact Music Tools result envelope and local source title/artist by verified source SHA-256 and analysis job identity; no audio is copied.", Category: "decomposition", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "A JSON envelope carries Music Tools' preserved result document; it has no generated Connect RPC counterpart."}},
	{ID: "decomposition_get", Path: "/api/v1/decompositions/{track_sha256}", Method: "GET", Summary: "Read a track decomposition", Description: "Read the latest stored Music Tools result and its human review history, or select a prior analysis job.", Category: "decomposition", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The review workbench reads the JSON decomposition envelope; it has no generated Connect RPC counterpart."}},
	{ID: "decomposition_review", Path: "/api/v1/decompositions/{track_sha256}/reviews", Method: "POST", Summary: "Review a decomposition", Description: "Persist a human verdict, note, and timestamped section corrections against one immutable analysis job.", Category: "decomposition", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "A listener-authored review submission is a JSON workbench action without a generated Connect RPC counterpart."}},
	{ID: "decomposition_audio", Path: "/api/v1/decompositions/{track_sha256}/audio", Method: "GET", Summary: "Stream local source audio for private review.", Description: "Proxy Music Tools source audio using loopback-only service discovery and byte-range support; the audio remains in Music Tools storage.", Category: "decomposition", RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "The browser needs an audio stream at the Library origin; this pass-through does not copy source bytes into Library."}},
}
