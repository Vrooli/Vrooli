package graph

import (
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler serves the graph projection HTTP endpoint.
type Handler struct {
	projector Projector
}

// NewHandler creates a graph Handler.
func NewHandler(projector Projector) *Handler {
	return &Handler{projector: projector}
}

// RegisterRoutes registers graph endpoints on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/graph", h.GetGraph).Methods("GET")
}

// GetGraph handles GET /api/v1/graph?lens=topology.
func (h *Handler) GetGraph(w http.ResponseWriter, r *http.Request) {
	lensParam := r.URL.Query().Get("lens")
	if lensParam == "" {
		lensParam = "topology"
	}
	lens := Lens(lensParam)
	if !ValidateLens(lens) {
		apierr.MapError(w, "[graph]", apierr.BadRequest("invalid lens: must be topology"))
		return
	}

	params := ProjectionParams{
		Lens: lens,
	}

	resp, err := h.projector.Project(r.Context(), params)
	if err != nil {
		apierr.MapError(w, "[graph]", apierr.Internal("failed to build graph projection"))
		return
	}

	protoResp, err := encodeGraphResponse(resp)
	if err != nil {
		apierr.MapError(w, "[graph]", apierr.Internal("failed to encode graph projection"))
		return
	}

	_ = httputil.ProtoJSON(w, protoResp)
}
