// Package privacyconsent exposes the human consent lifecycle over HTTP. It
// derives the owner exclusively from API Core's verified request principal.
package privacyconsent

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"

	store "ai-gateway/internal/privacyconsent"
)

type Handler struct{ authority *store.Store }

func New(authority *store.Store) http.Handler {
	h := &Handler{authority: authority}
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ai/privacy-consent", h.list).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/ai/privacy-consent", h.grant).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/ai/privacy-consent/verify", h.verify).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/ai/privacy-consent/{id}", h.revoke).Methods(http.MethodDelete)
	return router
}

// verify is a bounded online check for services that must gate an external
// dispatch. The receipt is accepted only in the request body, never echoed.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	principal, ok := humanPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "verified human identity required")
		return
	}
	var input struct {
		Source  string `json:"source"`
		Receipt string `json:"receipt"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || h.authority == nil {
		writeError(w, http.StatusBadRequest, "invalid consent verification request")
		return
	}
	if err := h.authority.Verify(identity.WithPrincipal(r.Context(), principal), input.Receipt, input.Source); err != nil {
		writeError(w, http.StatusForbidden, "consent is not valid for this source")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"valid": true})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	principal, ok := humanPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "verified human identity required")
		return
	}
	grants, err := h.authority.List(r.Context(), principal.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "consent status is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants})
}

func (h *Handler) grant(w http.ResponseWriter, r *http.Request) {
	principal, ok := humanPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "verified human identity required")
		return
	}
	var input struct {
		Source       string `json:"source"`
		LifetimeDays int    `json:"lifetime_days"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid consent request")
		return
	}
	if input.LifetimeDays == 0 {
		input.LifetimeDays = 30
	}
	receipt, grant, err := h.authority.Grant(r.Context(), principal.Subject, input.Source, time.Duration(input.LifetimeDays)*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusBadRequest, "consent source or lifetime is invalid")
		return
	}
	// The bearer receipt is returned once. Status/list responses never include it.
	writeJSON(w, http.StatusCreated, map[string]any{"grant": grant, "receipt": receipt})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	principal, ok := humanPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "verified human identity required")
		return
	}
	err := h.authority.Revoke(r.Context(), principal.Subject, strings.TrimSpace(mux.Vars(r)["id"]))
	if errors.Is(err, store.ErrInvalidGrant) {
		writeError(w, http.StatusNotFound, "consent grant was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "consent grant was not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func humanPrincipal(r *http.Request) (identity.Principal, bool) {
	principal, ok := identity.PrincipalFromContext(r.Context())
	return principal, ok && principal.IsHuman() && strings.TrimSpace(principal.Subject) != ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
