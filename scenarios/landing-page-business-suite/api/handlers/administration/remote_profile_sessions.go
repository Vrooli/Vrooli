package administration

import (
	"context"
	"net/http"
	"strings"

	admin "landing-page-business-suite-api/internal/administration"
)

type RemoteProfileSessionService interface {
	List(context.Context, string) ([]admin.IncomingRemoteProfileSession, error)
	Revoke(context.Context, string) (bool, error)
}

type RemoteProfileSessionDependencies struct {
	Service     RemoteProfileSessionService
	Path        func(*http.Request, string) (string, bool)
	WriteData   func(http.ResponseWriter, any)
	WriteSimple func(http.ResponseWriter)
	WriteError  func(http.ResponseWriter, int, string, string)
}

func ListIncomingRemoteProfileSessions(deps RemoteProfileSessionDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := deps.Service.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("connector_id")))
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to list incoming remote sessions", "server_error")
			return
		}
		deps.WriteData(w, map[string]any{"sessions": sessions})
	}
}

func RevokeIncomingRemoteProfileSession(deps RemoteProfileSessionDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := deps.Path(r, "session_id")
		id = strings.TrimSpace(id)
		if id == "" {
			deps.WriteError(w, http.StatusBadRequest, "Session ID is required", "validation")
			return
		}
		revoked, err := deps.Service.Revoke(r.Context(), id)
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to revoke incoming remote session", "server_error")
			return
		}
		if !revoked {
			deps.WriteError(w, http.StatusNotFound, "Incoming remote session not found", "not_found")
			return
		}
		deps.WriteSimple(w)
	}
}
