// DOC: docs/reference/api-endpoints.md#graph
package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// graphIndexProvider gives access to the graph index.
type graphIndexProvider interface {
	Get(ctx context.Context) (*GraphIndex, error)
	Regenerate(ctx context.Context) error
}

type graphHealthConfigStore interface {
	Get(ctx context.Context) (HealthConfig, error)
	Put(ctx context.Context, cfg HealthConfig) error
}

type operatingMapProvider interface {
	Get(context.Context) (OperatingMap, error)
}

// Handlers provides HTTP handlers for graph operations.
type Handlers struct {
	indexStore        graphIndexProvider
	healthConfigStore graphHealthConfigStore
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

// NewHandlers creates new graph handlers.
func NewHandlers(indexStore graphIndexProvider, configStore ...graphHealthConfigStore) *Handlers {
	h := &Handlers{indexStore: indexStore}
	if len(configStore) > 0 {
		h.healthConfigStore = configStore[0]
	}
	return h
}

// GetGraph handles GET /api/v1/graph - returns the full graph.
func (h *Handlers) GetGraph(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(idx)
}

// Regenerate handles POST /api/v1/graph/regenerate - forces rebuild.
func (h *Handlers) Regenerate(w http.ResponseWriter, r *http.Request) {
	if err := h.indexStore.Regenerate(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(idx)
}

// GetOrphans handles GET /api/v1/graph/orphans - returns orphaned skills.
func (h *Handlers) GetOrphans(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orphans := OrphanedSkills(idx.Graph)
	if orphans == nil {
		orphans = []Node{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(orphans)
}

// GetSkillless handles GET /api/v1/graph/skillless - returns agents without skills.
func (h *Handlers) GetSkillless(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	agents := SkilllessAgents(idx.Graph)
	if agents == nil {
		agents = []Node{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

// GetEmptyTeams handles GET /api/v1/graph/empty-teams - returns teams with no members.
func (h *Handlers) GetEmptyTeams(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	teams := EmptyTeams(idx.Graph)
	if teams == nil {
		teams = []Node{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(teams)
}

// GetUnaffiliated handles GET /api/v1/graph/unaffiliated - returns agents in no teams.
func (h *Handlers) GetUnaffiliated(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	agents := UnaffiliatedAgents(idx.Graph)
	if agents == nil {
		agents = []Node{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(agents)
}

// GetPopular handles GET /api/v1/graph/popular - returns most referenced nodes.
func (h *Handlers) GetPopular(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	popular := Popular(idx.Graph, limit)
	if popular == nil {
		popular = []Node{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(popular)
}

// GetCycles handles GET /api/v1/graph/cycles - returns circular references.
func (h *Handlers) GetCycles(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cycles := DetectCircularRefs(idx.Graph)
	if cycles == nil {
		cycles = [][]string{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cycles)
}

// nodeDetail is the response shape for GetNode.
type nodeDetail struct {
	Node          Node         `json:"node"`
	AdjacentEdges []Edge       `json:"adjacentEdges"`
	HealthScore   *HealthScore `json:"healthScore,omitempty"`
}

// GetNode handles GET /api/v1/graph/nodes/{id} - returns a node with adjacent edges and health.
func (h *Handlers) GetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find the node
	var found *Node
	for i := range idx.Graph.Nodes {
		if idx.Graph.Nodes[i].ID == id {
			found = &idx.Graph.Nodes[i]
			break
		}
	}
	if found == nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	// Collect adjacent edges
	var edges []Edge
	for _, e := range idx.Graph.Edges {
		if e.From == id || e.To == id {
			edges = append(edges, e)
		}
	}
	if edges == nil {
		edges = []Edge{}
	}

	// Find health score
	var hs *HealthScore
	for i := range idx.Graph.HealthScores {
		if idx.Graph.HealthScores[i].NodeID == id {
			hs = &idx.Graph.HealthScores[i]
			break
		}
	}

	detail := nodeDetail{
		Node:          *found,
		AdjacentEdges: edges,
		HealthScore:   hs,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

// GetNodeEdges handles GET /api/v1/graph/nodes/{id}/edges - returns edges for a node.
func (h *Handlers) GetNodeEdges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var edges []Edge
	for _, e := range idx.Graph.Edges {
		if e.From == id || e.To == id {
			edges = append(edges, e)
		}
	}
	if edges == nil {
		edges = []Edge{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(edges)
}

// GetHealthScores handles GET /api/v1/graph/health - returns health scores.
func (h *Handlers) GetHealthScores(w http.ResponseWriter, r *http.Request) {
	idx, err := h.indexStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scores := idx.Graph.HealthScores
	if scores == nil {
		scores = []HealthScore{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(scores)
}

// GetHealthConfig handles GET /api/v1/graph/health-config.
func (h *Handlers) GetHealthConfig(w http.ResponseWriter, r *http.Request) {
	if h.healthConfigStore == nil {
		http.Error(w, "graph health config store not configured", http.StatusServiceUnavailable)
		return
	}

	cfg, err := h.healthConfigStore.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// PutHealthConfig handles PUT /api/v1/graph/health-config.
func (h *Handlers) PutHealthConfig(w http.ResponseWriter, r *http.Request) {
	if h.healthConfigStore == nil {
		http.Error(w, "graph health config store not configured", http.StatusServiceUnavailable)
		return
	}

	var cfg HealthConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg = withHealthConfigDefaults(cfg)
	if err := ValidateHealthConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.healthConfigStore.Put(r.Context(), cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.indexStore.Regenerate(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}
