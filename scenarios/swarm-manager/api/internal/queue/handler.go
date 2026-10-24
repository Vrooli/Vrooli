// Package queue provides a filesystem-backed queue for local operations.
//
// Queue items are stored in scenario runtime state via api-core/storage by default.
//
// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/INVARIANTS.md
package queue

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/storage"

	"github.com/gorilla/mux"
)

// Item represents a queued operation.
type Item struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Created string          `json:"created"`
}

// CreateRequest accepts a new queue item.
type CreateRequest struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ListResponse wraps queue listings.
type ListResponse struct {
	Items []Item `json:"items"`
}

// ItemResponse wraps queue item responses.
type ItemResponse struct {
	Item Item `json:"item"`
}

// Store persists queue items.
type Store struct {
	path string
}

// NewStore creates a queue store. If path is empty, uses the scenario default.
func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		if resolved, err := runtimepaths.StatePath("queue.json"); err == nil {
			path = resolved
		}
	}
	return &Store{path: path}
}

// Load returns all queue items.
func (s *Store) Load() ([]Item, error) {
	var items []Item
	exists, err := storage.ReadJSON(s.path, &items)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []Item{}, nil
	}
	return normalizeItems(items), nil
}

// Save persists the queue list.
func (s *Store) Save(items []Item) error {
	return storage.WriteJSONAtomic(s.path, items)
}

// EventLogger records queue state-change events for analytics.
type EventLogger interface {
	EmitQueued(backlogKind, backlogName string, position int)
	EmitDequeued(backlogKind, backlogName, reason string)
}

// Handler exposes HTTP endpoints for queue operations.
type Handler struct {
	store       *Store
	eventLogger EventLogger
}

// NewHandler creates a new queue handler.
func NewHandler(path string) *Handler {
	return &Handler{store: NewStore(path)}
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (h *Handler) SetEventLogger(l EventLogger) {
	h.eventLogger = l
}

// RegisterRoutes registers queue endpoints.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/queue", h.List).Methods("GET")
	r.HandleFunc("/api/v1/queue", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/queue/{id}", h.Delete).Methods("DELETE")
}

// List returns all queue items.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	items, err := h.store.Load()
	if err != nil {
		apierr.MapError(w, "[queue] list", apierr.Internal("failed to load queue"))
		return
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Created < items[j].Created
	})
	if err := httputil.JSON(w, ListResponse{Items: items}); err != nil {
		apierr.MapError(w, "[queue] list", apierr.Internal("failed to encode response"))
		return
	}
}

// Create adds a new queue item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[queue] create", apierr.BadRequest("invalid request body"))
		return
	}
	if strings.TrimSpace(req.Kind) == "" {
		apierr.MapError(w, "[queue] create", apierr.BadRequest("kind is required"))
		return
	}

	items, err := h.store.Load()
	if err != nil {
		apierr.MapError(w, "[queue] create", apierr.Internal("failed to load queue"))
		return
	}

	item := Item{
		ID:      idgen.Generate(),
		Kind:    strings.TrimSpace(req.Kind),
		Payload: normalizePayload(req.Payload),
		Created: time.Now().UTC().Format(time.RFC3339),
	}
	items = append(items, item)

	if err := h.store.Save(items); err != nil {
		apierr.MapError(w, "[queue] create", apierr.Internal("failed to persist queue"))
		return
	}

	slog.Info("queue item added", "id", item.ID, "kind", item.Kind)
	if h.eventLogger != nil {
		h.eventLogger.EmitQueued(item.Kind, item.ID, len(items))
	}
	if err := httputil.JSONWithStatus(w, http.StatusCreated, ItemResponse{Item: item}); err != nil {
		apierr.MapError(w, "[queue] create", apierr.Internal("failed to encode response"))
		return
	}
}

// Delete removes a queue item by ID (idempotent).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		apierr.MapError(w, "[queue] delete", apierr.BadRequest("id is required"))
		return
	}

	items, err := h.store.Load()
	if err != nil {
		apierr.MapError(w, "[queue] delete", apierr.Internal("failed to load queue"))
		return
	}

	updated := make([]Item, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			updated = append(updated, item)
		}
	}

	if err := h.store.Save(updated); err != nil {
		apierr.MapError(w, "[queue] delete", apierr.Internal("failed to persist queue"))
		return
	}

	slog.Info("queue item removed", "id", id)
	if h.eventLogger != nil {
		h.eventLogger.EmitDequeued("", id, "removed")
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeItems(items []Item) []Item {
	for i := range items {
		items[i].Kind = strings.TrimSpace(items[i].Kind)
		if items[i].Created == "" {
			items[i].Created = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return items
}

func normalizePayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 || string(payload) == "null" {
		return nil
	}
	return payload
}

// generateID removed in favor of idgen.Generate()
