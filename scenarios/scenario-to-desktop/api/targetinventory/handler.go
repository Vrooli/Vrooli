package targetinventory

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct {
	probe         LocalProbe
	bridgeSources []BridgeSource
}

func NewHandler(probe LocalProbe, bridgeSources ...BridgeSource) *Handler {
	return &Handler{probe: probe, bridgeSources: bridgeSources}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	if h == nil || router == nil {
		return
	}
	router.HandleFunc("/api/v1/validation/targets", h.list).Methods(http.MethodGet)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	result := Discover(r.Context(), h.probe, h.bridgeSources...)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
