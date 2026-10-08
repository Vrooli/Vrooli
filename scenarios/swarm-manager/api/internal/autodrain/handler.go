package autodrain

import (
	"encoding/json"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler serves the auto-drain toggle over HTTP.
type Handler struct {
	store *Store
}

// NewHandler creates an auto-drain Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// RegisterRoutes registers GET/PUT /api/v1/execution/auto-drain.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/execution/auto-drain", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/execution/auto-drain", h.Put).Methods("PUT")
}

// Get returns the current toggle state.
func (h *Handler) Get(w http.ResponseWriter, _ *http.Request) {
	st, err := h.store.Load()
	if err != nil {
		apierr.MapError(w, "[auto-drain]", apierr.Internal("failed to read auto-drain state"))
		return
	}
	h.write(w, st)
}

// Put sets the toggle state from a {"enabled": bool} body.
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	var st State
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		apierr.MapError(w, "[auto-drain]", apierr.BadRequest("invalid request body"))
		return
	}
	if err := h.store.Save(st); err != nil {
		apierr.MapError(w, "[auto-drain]", apierr.Internal("failed to persist auto-drain state"))
		return
	}
	h.write(w, st)
}

func (h *Handler) write(w http.ResponseWriter, st State) {
	if err := httputil.JSON(w, st); err != nil {
		apierr.MapError(w, "[auto-drain]", apierr.Internal("failed to encode response"))
	}
}
