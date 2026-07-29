package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/account"
)

type UserAccountResponse = account.UserAccountResponse
type SubscriptionInfo = account.SubscriptionInfo
type CreditInfo = account.CreditInfo
type UserSessionResponse = account.UserSessionResponse
type UsersListResponse = account.UsersListResponse

func handleAdminListUsers(service *UserManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := service.List(r.Context(), r.URL.Query().Get("search"), queryInt(r, "page"), queryInt(r, "per_page"))
		if err != nil {
			writeUserManagementError(w, err, "Failed to query users", http.StatusInternalServerError)
			return
		}
		writeUserManagementJSON(w, response)
	}
}

func handleAdminGetUser(service *UserManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["id"]
		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}
		user, err := service.Get(r.Context(), userID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if err != nil {
			writeUserManagementError(w, err, "Failed to get user", http.StatusInternalServerError)
			return
		}
		writeUserManagementJSON(w, user)
	}
}

func handleAdminGetUserSessions(service *UserManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["id"]
		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}
		sessions, err := service.ListSessions(r.Context(), userID)
		if err != nil {
			writeUserManagementError(w, err, "Failed to query sessions", http.StatusInternalServerError)
			return
		}
		writeUserManagementJSON(w, sessions)
	}
}

func handleAdminRevokeUserSession(service *UserManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID, sessionID := vars["id"], vars["sid"]
		if userID == "" || sessionID == "" {
			http.Error(w, "User ID and session ID required", http.StatusBadRequest)
			return
		}
		revoked, err := service.RevokeSession(r.Context(), userID, sessionID)
		if err != nil {
			writeUserManagementError(w, err, "Failed to revoke session", http.StatusInternalServerError)
			return
		}
		if !revoked {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		logStructured("admin_revoke_session", map[string]interface{}{"level": "info", "user_id": userID, "session_id": sessionID})
		writeUserManagementJSON(w, map[string]interface{}{"success": true, "message": "Session revoked"})
	}
}

func handleAdminRevokeAllUserSessions(service *UserManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["id"]
		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}
		count, err := service.RevokeAllSessions(r.Context(), userID)
		if err != nil {
			writeUserManagementError(w, err, "Failed to revoke sessions", http.StatusInternalServerError)
			return
		}
		logStructured("admin_revoke_all_sessions", map[string]interface{}{"level": "info", "user_id": userID, "sessions_revoked": count})
		writeUserManagementJSON(w, map[string]interface{}{"success": true, "message": "All sessions revoked", "sessions_revoked": count})
	}
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}

func writeUserManagementJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logStructuredError("user_management_response_encode_failed", map[string]interface{}{"error": err.Error()})
	}
}

func writeUserManagementError(w http.ResponseWriter, err error, message string, status int) {
	logStructuredError("user_management_request_failed", map[string]interface{}{"error": err.Error()})
	http.Error(w, message, status)
}
