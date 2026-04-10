package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type IncomingRemoteProfileSessionResponse struct {
	SessionID    string    `json:"session_id"`
	AdminEmail   string    `json:"admin_email"`
	ConnectorID  string    `json:"connector_id"`
	ProfileTag   string    `json:"profile_tag,omitempty"`
	Origin       string    `json:"origin,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	ExpiresAt    time.Time `json:"expires_at"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
}

func handleAdminListIncomingRemoteProfileSessions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connectorFilter := strings.TrimSpace(r.URL.Query().Get("connector_id"))
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, admin_email, created_at, last_activity, expires_at, ip_address, user_agent
			FROM admin_sessions
			ORDER BY last_activity DESC
		`)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to list incoming remote sessions", ApiErrorTypeServerError)
			return
		}
		defer rows.Close()

		sessions := []IncomingRemoteProfileSessionResponse{}
		for rows.Next() {
			var (
				sessionID    string
				adminEmail   string
				createdAt    time.Time
				lastActivity time.Time
				expiresAt    time.Time
				ipAddress    sql.NullString
				userAgent    sql.NullString
			)
			if err := rows.Scan(&sessionID, &adminEmail, &createdAt, &lastActivity, &expiresAt, &ipAddress, &userAgent); err != nil {
				continue
			}
			meta, ok := parseRemoteProfileSessionUserAgent(userAgent.String)
			if !ok {
				continue
			}
			if connectorFilter != "" && connectorFilter != meta.ConnectorID {
				continue
			}
			sessions = append(sessions, IncomingRemoteProfileSessionResponse{
				SessionID:    sessionID,
				AdminEmail:   adminEmail,
				ConnectorID:  meta.ConnectorID,
				ProfileTag:   meta.ProfileTag,
				Origin:       meta.Origin,
				CreatedAt:    createdAt,
				LastActivity: lastActivity,
				ExpiresAt:    expiresAt,
				IPAddress:    NullStringValue(ipAddress),
				UserAgent:    NullStringValue(userAgent),
			})
		}

		writeJSONSuccessData(w, map[string]interface{}{"sessions": sessions})
	}
}

func handleAdminRevokeIncomingRemoteProfileSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := strings.TrimSpace(mux.Vars(r)["session_id"])
		if sessionID == "" {
			writeJSONError(w, http.StatusBadRequest, "Session ID is required", ApiErrorTypeValidation)
			return
		}

		res, err := db.ExecContext(r.Context(), `
			DELETE FROM admin_sessions
			WHERE id = $1
			  AND user_agent LIKE $2
		`, sessionID, remoteProfileSessionAgentPrefix+"%")
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to revoke incoming remote session", ApiErrorTypeServerError)
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			writeJSONError(w, http.StatusNotFound, "Incoming remote session not found", ApiErrorTypeNotFound)
			return
		}
		writeJSONSuccessSimple(w)
	}
}
