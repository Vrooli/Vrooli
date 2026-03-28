package graph

import (
	"net/http"

	"github.com/gorilla/mux"

	"swarm-manager/internal/httputil"
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

// GetGraph handles GET /api/v1/graph?lens=topology|flow|operations.
func (h *Handler) GetGraph(w http.ResponseWriter, r *http.Request) {
	lensParam := r.URL.Query().Get("lens")
	if lensParam == "" {
		lensParam = "topology"
	}
	lens := Lens(lensParam)
	if !ValidateLens(lens) {
		httputil.BadRequest(w, "[graph]", "invalid lens: must be topology, flow, or operations")
		return
	}

	resp, err := h.projector.Project(r.Context(), lens)
	if err != nil {
		httputil.InternalError(w, "[graph]", "failed to build graph projection")
		return
	}

	protoResp, err := encodeGraphResponse(resp)
	if err != nil {
		httputil.InternalError(w, "[graph]", "failed to encode graph projection")
		return
	}

	_ = httputil.ProtoJSON(w, protoResp)
}
