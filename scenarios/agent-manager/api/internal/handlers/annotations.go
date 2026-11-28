package handlers

// Annotation HTTP surface (WS-D). The annotation vocabulary is versioned data;
// these routes are a plain JSON contract so the pack can change without a proto
// regeneration. Errors use the same {"error": "..."} shape as the rest of the
// untagged JSON surface.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/annotationfacets"
	"agent-manager/internal/runannotations"

	"github.com/gorilla/mux"
)

// AnnotationsHandler exposes pack, label, coverage, run and batch routes.
type AnnotationsHandler struct {
	service *runannotations.Service
	facets  *annotationfacets.Refresher
}

// NewAnnotationsHandler wires the handler over an annotation service.
func NewAnnotationsHandler(service *runannotations.Service) *AnnotationsHandler {
	return &AnnotationsHandler{service: service}
}

// WithFacets projects written annotations into the conversation-search catalog
// tags. It is best-effort: the annotation store remains authoritative.
func (h *AnnotationsHandler) WithFacets(facets *annotationfacets.Refresher) *AnnotationsHandler {
	h.facets = facets
	return h
}

func (h *AnnotationsHandler) refreshFacets(ctx context.Context, views []runannotations.RunView) {
	if h == nil || h.facets == nil || len(views) == 0 {
		return
	}
	_ = h.facets.RefreshViews(ctx, views)
}

// RegisterRoutes registers the annotation JSON surface.
func (h *AnnotationsHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/annotations/packs", h.Packs).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/labels", h.Labels).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/coverage", h.Coverage).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/run/{id}", h.Run).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/run/{id}", h.Annotate).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/annotations/evaluate", h.Evaluate).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/annotations/gold", h.GoldDraft).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/benchmark", h.Benchmark).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/annotations/provenance/refresh", h.RefreshProvenance).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/annotations/series", h.Series).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotations/gate", h.Gate).Methods(http.MethodGet)
}

type packsResponse struct {
	Packs []runannotations.PackSummary `json:"packs"`
}

type labelsResponse struct {
	Pack    string                     `json:"pack"`
	Version string                     `json:"version"`
	Window  string                     `json:"window"`
	Labels  []runannotations.LabelView `json:"labels"`
}

type annotateRequest struct {
	ForceLabels     []string `json:"force_labels"`
	IncludeOptional bool     `json:"include_optional"`
	Labels          []string `json:"labels"`
	MaxLabels       int      `json:"max_labels"`
	DryRun          bool     `json:"dry_run"`
}

type evaluateRequest struct {
	Since           string   `json:"since"`
	Until           string   `json:"until"`
	Limit           int      `json:"limit"`
	RunIDs          []string `json:"run_ids"`
	Labels          []string `json:"labels"`
	IncludeOptional bool     `json:"include_optional"`
	ForceLabels     []string `json:"force_labels"`
	DryRun          bool     `json:"dry_run"`
	// PinResourceRole restricts every decision to one declared candidate and
	// stores the answers apart from production; Role is the role walked for it.
	PinResourceRole string `json:"pin_resource_role"`
	Role            string `json:"role"`
	// MessageIDs restricts message-scope labels to the named human messages.
	MessageIDs []string `json:"message_ids"`
}

// Packs returns the loaded pack identity.
func (h *AnnotationsHandler) Packs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, packsResponse{Packs: []runannotations.PackSummary{h.service.PackSummary()}})
}

// Labels returns declarations, gates and window coverage.
func (h *AnnotationsHandler) Labels(w http.ResponseWriter, r *http.Request) {
	if !annotationWindowOK(w, r) {
		return
	}
	summary, err := h.service.PackSummaryFor(strings.TrimSpace(r.URL.Query().Get("pack")))
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	labels, err := h.service.Labels(r.Context(), r.URL.Query().Get("window"))
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, labelsResponse{Pack: summary.Name, Version: summary.Version, Window: normalizedWindow(r), Labels: labels})
}

// Coverage returns per-label observed/compatible/not_observed/unavailable counts.
func (h *AnnotationsHandler) Coverage(w http.ResponseWriter, r *http.Request) {
	if !annotationWindowOK(w, r) {
		return
	}
	coverage, err := h.service.Coverage(r.Context(), r.URL.Query().Get("window"))
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, coverage)
}

// Run returns the latest annotations for one run without evaluating.
func (h *AnnotationsHandler) Run(w http.ResponseWriter, r *http.Request) {
	runID, ok := annotationRunID(w, r)
	if !ok {
		return
	}
	read := h.service.Run
	if source := strings.TrimSpace(r.URL.Query().Get("source")); source == runannotations.SourcePinned {
		read = h.service.RunTrial
	} else if source != "" && source != runannotations.SourceProduction {
		writeJSONError(w, http.StatusBadRequest, "source must be production or pinned")
		return
	}
	view, err := read(r.Context(), runID)
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// Annotate evaluates one run on demand.
func (h *AnnotationsHandler) Annotate(w http.ResponseWriter, r *http.Request) {
	runID, ok := annotationRunID(w, r)
	if !ok {
		return
	}
	var request annotateRequest
	if !decodeAnnotationBody(w, r, &request) {
		return
	}
	view, err := h.service.Annotate(r.Context(), runID, runannotations.AnnotateOptions{
		ForceLabels: request.ForceLabels, IncludeOptional: request.IncludeOptional,
		Labels: request.Labels, MaxLabels: request.MaxLabels, DryRun: request.DryRun,
	})
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	if !request.DryRun {
		h.refreshFacets(r.Context(), []runannotations.RunView{view})
	}
	writeJSON(w, http.StatusOK, view)
}

// GoldDraft returns a blind review packet of bounded states for one label. It
// deliberately omits the stored answer so a labelled set stays independent.
func (h *AnnotationsHandler) GoldDraft(w http.ResponseWriter, r *http.Request) {
	label := strings.TrimSpace(r.URL.Query().Get("label"))
	if label == "" {
		writeJSONError(w, http.StatusBadRequest, "label is required")
		return
	}
	if _, ok := h.service.Pack().Label(label); !ok {
		writeJSONError(w, http.StatusBadRequest, "unknown label "+label)
		return
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeJSONError(w, http.StatusBadRequest, "limit must be a non-negative integer")
			return
		}
		limit = parsed
	}
	draft, err := h.service.GoldDraft(r.Context(), label, r.URL.Query().Get("window"), limit)
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

// Evaluate runs the bounded batch over selected or named runs.
func (h *AnnotationsHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	var request evaluateRequest
	if !decodeAnnotationBody(w, r, &request) {
		return
	}
	options := runannotations.EvaluateOptions{
		Limit: request.Limit, RunIDs: request.RunIDs, Labels: request.Labels,
		IncludeOptional: request.IncludeOptional, ForceLabels: request.ForceLabels, DryRun: request.DryRun,
		PinResourceRole: request.PinResourceRole, Role: request.Role, MessageIDs: request.MessageIDs,
	}
	if strings.TrimSpace(request.Since) != "" {
		since, err := time.Parse(time.RFC3339, request.Since)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "since must be RFC3339")
			return
		}
		options.Since = since
	}
	if strings.TrimSpace(request.Until) != "" {
		until, err := time.Parse(time.RFC3339, request.Until)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "until must be RFC3339")
			return
		}
		options.Until = until
	}
	result, err := h.service.Evaluate(r.Context(), options)
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	// A pinned pass writes the trials table; its answers are never facets.
	if !request.DryRun && request.PinResourceRole == "" && len(result.Runs) > 0 {
		h.refreshFacets(r.Context(), result.Runs)
	}
	writeJSON(w, http.StatusOK, result)
}

// RefreshProvenance recomputes the stored authorship of annotated runs whose
// provenance predates the pack's current policy. It evaluates no label.
func (h *AnnotationsHandler) RefreshProvenance(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Limit int `json:"limit"`
	}
	if !decodeAnnotationBody(w, r, &request) {
		return
	}
	result, err := h.service.RefreshProvenance(r.Context(), request.Limit)
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Benchmark measures a label against gold rows from stored probability vectors.
// It runs no model.
func (h *AnnotationsHandler) Benchmark(w http.ResponseWriter, r *http.Request) {
	var request runannotations.BenchmarkRequest
	if !decodeAnnotationBodyLimit(w, r, &request, 4<<20) {
		return
	}
	result, err := h.service.Benchmark(r.Context(), request)
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Series returns a message-scope label's per-message series in time buckets.
func (h *AnnotationsHandler) Series(w http.ResponseWriter, r *http.Request) {
	if !annotationWindowOK(w, r) {
		return
	}
	query := r.URL.Query()
	label := strings.TrimSpace(query.Get("label"))
	if label == "" {
		writeJSONError(w, http.StatusBadRequest, "label is required")
		return
	}
	result, err := h.service.Series(r.Context(), label, query.Get("window"), query.Get("bucket"))
	if err != nil {
		writeAnnotationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func annotationRunID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "run id is required")
		return "", false
	}
	if len(id) > 200 {
		writeJSONError(w, http.StatusBadRequest, "run id is too long")
		return "", false
	}
	return id, true
}

func decodeAnnotationBody(w http.ResponseWriter, r *http.Request, target any) bool {
	return decodeAnnotationBodyLimit(w, r, target, 64<<10)
}

func decodeAnnotationBodyLimit(w http.ResponseWriter, r *http.Request, target any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return true // an empty body is the defaults
		}
		writeJSONError(w, http.StatusBadRequest, "invalid annotation request: "+err.Error())
		return false
	}
	return true
}

func writeAnnotationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, runannotations.ErrScopeNotFound):
		writeJSONError(w, http.StatusNotFound, "annotation scope not found")
	case errors.Is(err, runannotations.ErrPackNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, runannotations.ErrInvalidRequest):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, runannotations.ErrSelectionUnavailable):
		writeJSONError(w, http.StatusBadRequest, "run selection unavailable; pass run_ids or wire a selector")
	default:
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func normalizedWindow(r *http.Request) string {
	window := strings.TrimSpace(r.URL.Query().Get("window"))
	if window == "" {
		return "7d"
	}
	return window
}

func annotationWindowOK(w http.ResponseWriter, r *http.Request) bool {
	if _, _, err := runannotations.ParseWindow(r.URL.Query().Get("window"), time.Now()); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}
