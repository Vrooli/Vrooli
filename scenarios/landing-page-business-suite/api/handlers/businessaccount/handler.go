// Package businessaccount exposes the authenticated LPBS account-selection
// boundary used by desktop linking and the browser consent flow.
package businessaccount

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	domain "landing-page-business-suite-api/internal/businessaccount"

	"github.com/gorilla/mux"
)

type Dependencies struct {
	Repository domain.Repository
	UserID     func(context.Context) string
	UserEmail  func(context.Context) string
}

type createRequest struct {
	DisplayName string `json:"display_name"`
}

func RegisterRoutes(router *mux.Router, deps Dependencies, requireUserAuth func(http.HandlerFunc) http.HandlerFunc) {
	router.HandleFunc("/api/v1/business-accounts", requireUserAuth(list(deps))).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/business-accounts", requireUserAuth(create(deps))).Methods(http.MethodPost)
}

func list(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, email, ok := currentIdentity(deps, r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "authenticated LPBS user required")
			return
		}
		accounts, err := deps.Repository.ListForUser(r.Context(), userID, email)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "business accounts unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"accounts": accounts})
	}
}

func create(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, email, ok := currentIdentity(deps, r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "authenticated LPBS user required")
			return
		}
		var request createRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid business account request")
			return
		}
		account, err := deps.Repository.CreateForUser(r.Context(), userID, email, request.DisplayName)
		if err != nil {
			if errors.Is(err, domain.ErrInvalid) {
				writeError(w, http.StatusBadRequest, "business account name is required")
				return
			}
			writeError(w, http.StatusServiceUnavailable, "business account unavailable")
			return
		}
		writeJSON(w, http.StatusCreated, account)
	}
}

func currentIdentity(deps Dependencies, ctx context.Context) (string, string, bool) {
	if deps.Repository == nil || deps.UserID == nil || deps.UserEmail == nil {
		return "", "", false
	}
	userID := strings.TrimSpace(deps.UserID(ctx))
	email := strings.TrimSpace(deps.UserEmail(ctx))
	return userID, email, userID != "" && email != ""
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
