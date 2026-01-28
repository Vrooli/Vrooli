// Package queue provides a filesystem-backed queue for local operations.
//
// Queue items are stored at scenarios/swarm-manager/.vrooli/queue.json by default.
package queue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"
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
		path = filepath.Join("scenarios", "swarm-manager", ".vrooli", "queue.json")
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

// Handler exposes HTTP endpoints for queue operations.
type Handler struct {
	store *Store
}

// NewHandler creates a new queue handler.
func NewHandler(path string) *Handler {
	return &Handler{store: NewStore(path)}
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
		httputil.InternalError(w, "[queue] list", "failed to load queue")
		return
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Created < items[j].Created
	})
	if err := httputil.JSON(w, ListResponse{Items: items}); err != nil {
		httputil.InternalError(w, "[queue] list", "failed to encode response")
		return
	}
}

// Create adds a new queue item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.BadRequest(w, "[queue] create", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Kind) == "" {
		httputil.BadRequest(w, "[queue] create", "kind is required")
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[queue] create", "failed to load queue")
		return
	}

	item := Item{
		ID:      generateID(),
		Kind:    strings.TrimSpace(req.Kind),
		Payload: normalizePayload(req.Payload),
		Created: time.Now().UTC().Format(time.RFC3339),
	}
	items = append(items, item)

	if err := h.store.Save(items); err != nil {
		httputil.InternalError(w, "[queue] create", "failed to persist queue")
		return
	}

	log.Printf("[queue] added: id=%s kind=%s", item.ID, item.Kind)
	if err := httputil.JSONWithStatus(w, http.StatusCreated, ItemResponse{Item: item}); err != nil {
		httputil.InternalError(w, "[queue] create", "failed to encode response")
		return
	}
}

// Delete removes a queue item by ID (idempotent).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		httputil.BadRequest(w, "[queue] delete", "id is required")
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[queue] delete", "failed to load queue")
		return
	}

	updated := make([]Item, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			updated = append(updated, item)
		}
	}

	if err := h.store.Save(updated); err != nil {
		httputil.InternalError(w, "[queue] delete", "failed to persist queue")
		return
	}

	log.Printf("[queue] removed: id=%s", id)
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

func generateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID if entropy fails (should be rare).
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(bytes)
}
