package aisearch

import (
	"errors"
	"log/slog"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

// Handler exposes the AI search Service over HTTP. The Service handles search
// + status; the Reconciler handles reconcile + reconcile-status + reconcile-cancel.
// Splitting them keeps responsibilities clear: search reads, reconcile writes.
type Handler struct {
	svc        *Service
	reconciler *Reconciler
}

// NewHandler constructs an HTTP handler around an aisearch Service and
// Reconciler. Both are required: nil reconciler causes the reconcile routes
// to return 503 explaining the misconfiguration.
func NewHandler(svc *Service, reconciler *Reconciler) *Handler {
	return &Handler{svc: svc, reconciler: reconciler}
}

// RegisterRoutes mounts the aisearch routes under /api/v1/search/ai.
//
// Greenfield rename: /reindex* → /reconcile*. The new verb reflects the
// post-refactor semantics (diff-and-converge, not rebuild-from-scratch).
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/search/ai", h.Search).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/search/ai/status", h.Status).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/search/ai/reconcile", h.Reconcile).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/search/ai/reconcile/status", h.ReconcileStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/search/ai/reconcile/cancel", h.CancelReconcile).Methods(http.MethodPost)
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

// Reconcile handles POST /api/v1/search/ai/reconcile.
//
// Honors the dry-run header (set by `swarm-manager ai-search reconcile --dry-run`):
// when set, the handler calls Reconciler.Plan and returns the resulting
// DriftReport without applying anything. No mutation, no embeds.
//
// Live (non-dry-run) call spawns Reconciler.RunOnce in a background goroutine
// (with a fresh context.Background — the reconcile must not abort if the user
// disconnects) and returns 202 Accepted with the current ReconcileStatus.
// Returns 409 Conflict if a previous RunOnce is still in flight.
func (h *Handler) Reconcile(w http.ResponseWriter, r *http.Request) {
	if h.reconciler == nil {
		apierr.MapError(w, "[aisearch] reconcile", apierr.Unavailable("reconciler not configured"))
		return
	}

	if cliutil.IsDryRun(r) {
		plan, err := h.reconciler.Plan(r.Context())
		if err != nil {
			apierr.MapError(w, "[aisearch] reconcile dry-run", apierr.Internal("plan failed: %s", err.Error()))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := httputil.JSON(w, map[string]any{"dry_run": true, "plan": plan}); err != nil {
			slog.Error("aisearch reconcile dry-run encode", "err", err)
		}
		return
	}

	// Live path: StartAsync acquires the singleton synchronously, so the
	// Status() snapshot below is guaranteed to reflect Running=true. If a
	// prior pass is still in flight, return 409.
	if err := h.reconciler.StartAsync(); err != nil {
		if errors.Is(err, ErrReconcileBusy) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if encErr := httputil.JSON(w, h.reconciler.Status()); encErr != nil {
				slog.Error("aisearch reconcile conflict encode", "err", encErr)
			}
			return
		}
		apierr.MapError(w, "[aisearch] reconcile", apierr.Internal("start failed: %s", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := httputil.JSON(w, h.reconciler.Status()); err != nil {
		slog.Error("aisearch reconcile encode", "err", err)
	}
}

// ReconcileStatus handles GET /api/v1/search/ai/reconcile/status.
func (h *Handler) ReconcileStatus(w http.ResponseWriter, _ *http.Request) {
	if h.reconciler == nil {
		apierr.MapError(w, "[aisearch] reconcile-status", apierr.Unavailable("reconciler not configured"))
		return
	}
	status := h.reconciler.Status()
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch reconcile-status encode", "err", err)
	}
}

// CancelReconcile handles POST /api/v1/search/ai/reconcile/cancel.
func (h *Handler) CancelReconcile(w http.ResponseWriter, _ *http.Request) {
	if h.reconciler == nil {
		apierr.MapError(w, "[aisearch] reconcile-cancel", apierr.Unavailable("reconciler not configured"))
		return
	}
	h.reconciler.Cancel()
	status := h.reconciler.Status()
	if err := httputil.JSON(w, status); err != nil {
		slog.Error("aisearch reconcile-cancel encode", "err", err)
	}
}
