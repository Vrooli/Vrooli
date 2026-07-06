package planimport

import (
	"encoding/json"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"

	"github.com/gorilla/mux"
)

// Handler serves the plan-import bridge over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler creates a plan-import Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers POST /api/v1/plan-import.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/plan-import", h.Import).Methods("POST")
}

type importRequest struct {
	PlanID string `json:"plan_id"`
}

// Import handles POST /api/v1/plan-import {"plan_id": "<id-or-slug>"}.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[plan-import]", apierr.BadRequest("invalid request body"))
		return
	}
	planID := strings.TrimSpace(req.PlanID)
	if planID == "" {
		apierr.MapError(w, "[plan-import]", apierr.BadRequest("plan_id is required"))
		return
	}
	result, err := h.svc.Import(r.Context(), planID, identity.FromContext(r.Context()))
	if err != nil {
		apierr.MapError(w, "[plan-import]", err)
		return
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, result); err != nil {
		apierr.MapError(w, "[plan-import]", apierr.Internal("failed to encode response"))
	}
}
