package aisearch

import (
	"log/slog"
	"net/http"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

// Handler exposes the AI search Service over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler constructs an HTTP handler around an aisearch Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the aisearch routes under /api/v1/search/ai.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/search/ai", h.Search).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/search/ai/status", h.Status).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/search/ai/reindex", h.Reindex).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/search/ai/reindex/status", h.ReindexStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/search/ai/reindex/cancel", h.CancelReindex).Methods(http.MethodPost)
}

// Search handles POST /api/v1/search/ai.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	var req AISearchRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[aisearch] search", apierr.BadRequest("invalid request body: %s", err.Error()))
		return
	}

	resp, err := h.svc.Search(r.Context(), req)
	if err != nil {
		apierr.MapError(w, "[aisearch] search", apierr.BadRequest("%s", err.Error()))
		return
	}
	if err := httputil.JSON(w, resp); err != nil {
		slog.Error("aisearch search encode", "err", err)
	}
}

// Status handles GET /api/v1/search/ai/status.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	status := h.svc.GetStatus(r.Context())
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch status encode", "err", err)
	}
}

// Reindex handles POST /api/v1/search/ai/reindex. Honors the dry-run header:
// when set, no mutation is performed and a realistic ReindexStatus is returned
// with message=dry-run.
func (h *Handler) Reindex(w http.ResponseWriter, r *http.Request) {
	if cliutil.IsDryRun(r) {
		dryRun := ReindexStatus{
			Running: false,
			Message: "dry-run: reindex not started",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		dryRunResp := map[string]any{
			"dry_run": true,
			"status":  dryRun,
		}
		if err := httputil.JSON(w, dryRunResp); err != nil {
			slog.Error("aisearch reindex dry-run encode", "err", err)
		}
		return
	}

	status, started := h.svc.StartReindex()
	code := http.StatusAccepted
	if !started {
		code = http.StatusConflict
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch reindex encode", "err", err)
	}
}

// ReindexStatus handles GET /api/v1/search/ai/reindex/status.
func (h *Handler) ReindexStatus(w http.ResponseWriter, _ *http.Request) {
	status := h.svc.ReindexStatus()
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch reindex-status encode", "err", err)
	}
}

// CancelReindex handles POST /api/v1/search/ai/reindex/cancel.
func (h *Handler) CancelReindex(w http.ResponseWriter, _ *http.Request) {
	status := h.svc.CancelReindex()
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch cancel-reindex encode", "err", err)
	}
}
