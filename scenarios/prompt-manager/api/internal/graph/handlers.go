// DOC: docs/reference/api-endpoints.md#graph
package graph

import (
	"context"
	"encoding/json"
	"net/http"
)

// graphIndexProvider gives access to the graph index.
type graphIndexProvider interface {
	Get(ctx context.Context) (*GraphIndex, error)
	Regenerate(ctx context.Context) error
}

type operatingMapProvider interface {
	Get(context.Context) (OperatingMap, error)
}

// Handlers serves the operating-map read that the memberflow Connect mount
// delegates to. The graph read surface itself is served by ConnectHandler.
type Handlers struct {
	operatingMapStore operatingMapProvider
}

// SetOperatingMapStore installs the shared swarm-map assembler.
func (h *Handlers) SetOperatingMapStore(store operatingMapProvider) { h.operatingMapStore = store }

// GetOperatingMap handles GET /operating-models/map.
func (h *Handlers) GetOperatingMap(w http.ResponseWriter, r *http.Request) {
	if h.operatingMapStore == nil {
		http.Error(w, "operating map unavailable", http.StatusServiceUnavailable)
		return
	}
	result, err := h.operatingMapStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// NewHandlers creates the graph handlers.
func NewHandlers() *Handlers { return &Handlers{} }
