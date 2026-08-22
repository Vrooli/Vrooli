package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
)

type shelfStore interface {
	CreateCheckShelf(context.Context, persistence.CheckShelf) error
	ListCheckShelves(context.Context, bool) ([]persistence.CheckShelf, error)
}

func (h *Handlers) ListCheckShelves(w http.ResponseWriter, r *http.Request) {
	store, ok := h.store.(shelfStore)
	if !ok {
		http.Error(w, "check shelf storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	includeExpired := r.URL.Query().Get("includeExpired") == "true"
	shelves, err := store.ListCheckShelves(r.Context(), includeExpired)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("shelves", "list check shelves", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"shelves": shelves})
}

func (h *Handlers) ShelveCheck(w http.ResponseWriter, r *http.Request) {
	store, ok := h.store.(shelfStore)
	if !ok {
		http.Error(w, "check shelf storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	checkID := strings.TrimSpace(mux.Vars(r)["checkId"])
	if checkID == "" {
		http.Error(w, "check id is required", http.StatusBadRequest)
		return
	}
	var request struct {
		Reason string `json:"reason"`
		Expiry string `json:"expiry"`
		SetBy  string `json:"setBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.Expiry) == "" {
		http.Error(w, "reason and expiry are required; permanent shelves are not allowed", http.StatusBadRequest)
		return
	}
	expiresAt, err := parseShelfExpiry(request.Expiry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setBy := strings.TrimSpace(request.SetBy)
	if setBy == "" {
		setBy = "operator"
	}
	if err := store.CreateCheckShelf(r.Context(), persistence.CheckShelf{CheckID: checkID, Reason: strings.TrimSpace(request.Reason), ExpiresAt: expiresAt, SetBy: setBy}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"checkId": checkID, "expiresAt": expiresAt, "shelved": true})
}

func (h *Handlers) UnshelveCheck(w http.ResponseWriter, r *http.Request) {
	store, ok := h.store.(interface {
		DeleteCheckShelf(context.Context, string) error
	})
	if !ok {
		http.Error(w, "check shelf storage is unavailable", http.StatusServiceUnavailable)
		return
	}
	checkID := strings.TrimSpace(mux.Vars(r)["checkId"])
	if err := store.DeleteCheckShelf(r.Context(), checkID); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("shelves", "unshelve check", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"checkId": checkID, "shelved": false})
}

func parseShelfExpiry(raw string) (time.Time, error) {
	if duration, err := time.ParseDuration(strings.TrimSpace(raw)); err == nil {
		if duration <= 0 {
			return time.Time{}, fmt.Errorf("expiry must be in the future")
		}
		return time.Now().UTC().Add(duration), nil
	}
	value, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	if err != nil || !value.After(time.Now().UTC()) {
		return time.Time{}, fmt.Errorf("expiry must be a positive duration or future RFC3339 timestamp")
	}
	return value.UTC(), nil
}
