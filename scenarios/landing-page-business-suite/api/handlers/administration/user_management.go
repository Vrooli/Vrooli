package administration

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	admin "landing-page-business-suite-api/internal/administration"
)

type UserManagementService interface {
	List(context.Context, string, int, int) (admin.UsersListResponse, error)
	Get(context.Context, string) (*admin.UserAccountResponse, error)
	ListSessions(context.Context, string) ([]admin.UserSessionResponse, error)
	RevokeSession(context.Context, string, string) (bool, error)
	RevokeAllSessions(context.Context, string) (int64, error)
}
type UserManagementDependencies struct {
	Service    UserManagementService
	Path       func(*http.Request, string) (string, bool)
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string)
	Log        func(string, map[string]any)
	LogError   func(string, map[string]any)
}

func ListUsers(d UserManagementDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, err := d.Service.List(r.Context(), r.URL.Query().Get("search"), queryInt(r, "page"), queryInt(r, "per_page"))
		if err != nil {
			d.error(w, err, "Failed to query users", http.StatusInternalServerError)
			return
		}
		d.WriteJSON(w, value)
	}
}

func GetUser(d UserManagementDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.Path(r, "id")
		if !ok || id == "" {
			d.WriteError(w, http.StatusBadRequest, "User ID required")
			return
		}
		value, err := d.Service.Get(r.Context(), id)
		if err == sql.ErrNoRows {
			d.WriteError(w, http.StatusNotFound, "User not found")
			return
		}
		if err != nil {
			d.error(w, err, "Failed to get user", http.StatusInternalServerError)
			return
		}
		d.WriteJSON(w, value)
	}
}

func ListUserSessions(d UserManagementDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.Path(r, "id")
		if !ok || id == "" {
			d.WriteError(w, http.StatusBadRequest, "User ID required")
			return
		}
		value, err := d.Service.ListSessions(r.Context(), id)
		if err != nil {
			d.error(w, err, "Failed to query sessions", http.StatusInternalServerError)
			return
		}
		d.WriteJSON(w, value)
	}
}

func RevokeUserSession(d UserManagementDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, hasID := d.Path(r, "id")
		sid, hasSID := d.Path(r, "sid")
		if !hasID || !hasSID || id == "" || sid == "" {
			d.WriteError(w, http.StatusBadRequest, "User ID and session ID required")
			return
		}
		revoked, err := d.Service.RevokeSession(r.Context(), id, sid)
		if err != nil {
			d.error(w, err, "Failed to revoke session", http.StatusInternalServerError)
			return
		}
		if !revoked {
			d.WriteError(w, http.StatusNotFound, "Session not found")
			return
		}
		d.Log("admin_revoke_session", map[string]any{"level": "info", "user_id": id, "session_id": sid})
		d.WriteJSON(w, map[string]any{"success": true, "message": "Session revoked"})
	}
}

func RevokeAllUserSessions(d UserManagementDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.Path(r, "id")
		if !ok || id == "" {
			d.WriteError(w, http.StatusBadRequest, "User ID required")
			return
		}
		count, err := d.Service.RevokeAllSessions(r.Context(), id)
		if err != nil {
			d.error(w, err, "Failed to revoke sessions", http.StatusInternalServerError)
			return
		}
		d.Log("admin_revoke_all_sessions", map[string]any{"level": "info", "user_id": id, "sessions_revoked": count})
		d.WriteJSON(w, map[string]any{"success": true, "message": "All sessions revoked", "sessions_revoked": count})
	}
}

func (d UserManagementDependencies) error(w http.ResponseWriter, err error, message string, status int) {
	d.LogError("user_management_request_failed", map[string]any{"error": err.Error()})
	d.WriteError(w, status, message)
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
