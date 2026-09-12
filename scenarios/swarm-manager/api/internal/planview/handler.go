package planview

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler serves the plan-board projection over HTTP.
type Handler struct {
	service *Service
}

// NewHandler creates a plan Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the plan endpoint on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/plan", h.GetBoard).Methods("GET")
}

// GetBoard handles GET /api/v1/plan?window_seconds=86400.
func (h *Handler) GetBoard(w http.ResponseWriter, r *http.Request) {
	params := Params{}
	if raw := strings.TrimSpace(r.URL.Query().Get("window_seconds")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			apierr.MapError(w, "[plan]", apierr.BadRequest("invalid window_seconds: must be a positive integer"))
			return
		}
		params.WindowSeconds = seconds
	}
	params.Goal = strings.TrimSpace(r.URL.Query().Get("goal"))

	board, err := h.service.Build(r.Context(), params)
	if err != nil {
		if errors.Is(err, ErrGoalScope) {
			apierr.MapError(w, "[plan]", apierr.NotFound("goal %q not found or goal scoping unavailable", params.Goal))
			return
		}
		apierr.MapError(w, "[plan]", apierr.Internal("failed to build plan projection"))
		return
	}

	_ = httputil.ProtoJSON(w, encodeBoard(board))
}
