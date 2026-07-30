// Package administration owns HTTP transport for administrator capabilities.
package administration

import (
	"context"
	"errors"
	"net/http"

	admin "landing-page-business-suite-api/internal/administration"
)

// RemoteProfileManager is the application boundary consumed by remote-profile
// HTTP handlers. Keeping it here makes the transport independently testable.
type RemoteProfileManager interface {
	List(context.Context) ([]admin.RemoteProfile, error)
	Create(context.Context, admin.RemoteProfileCreateRequest, string) (*admin.RemoteProfile, error)
	Update(context.Context, int64, admin.RemoteProfileUpdateRequest) (*admin.RemoteProfile, error)
	Delete(context.Context, int64) error
	Login(context.Context, int64, string, string) (*admin.RemoteProfile, error)
	Logout(context.Context, int64) (*admin.RemoteProfile, error)
	Test(context.Context, int64) (*admin.RemoteProfile, error)
	SessionLinks(context.Context, int64) (*admin.RemoteProfileSessionLinks, error)
	RevokeRemoteSessions(context.Context, int64) (*admin.RemoteProfileSessionLinks, error)
	Proxy(context.Context, int64, admin.RemoteProfileProxyRequest) (*admin.RemoteProxyResponse, error)
}

// RemoteProfileDependencies makes framework and presentation concerns explicit
// at the composition edge.
type RemoteProfileDependencies struct {
	Service       RemoteProfileManager
	ResolveEmail  func(*http.Request) (string, bool)
	DecodeJSON    func(http.ResponseWriter, *http.Request, any) bool
	PathInt64     func(http.ResponseWriter, *http.Request, string) (int64, bool)
	ValidateEmail func(http.ResponseWriter, string) (string, bool)
	WriteData     func(http.ResponseWriter, any)
	WriteSimple   func(http.ResponseWriter)
	WriteError    func(http.ResponseWriter, int, string, string)
	LogError      func(string, map[string]any)
}

func ListRemoteProfiles(deps RemoteProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profiles, err := deps.Service.List(r.Context())
		if err != nil {
			deps.LogError("list_remote_profiles_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to list remote profiles", "server_error")
			return
		}
		if profiles == nil {
			profiles = []admin.RemoteProfile{}
		}
		deps.WriteData(w, map[string]any{"profiles": profiles})
	}
}

func CreateRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request admin.RemoteProfileCreateRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		email := ""
		if deps.ResolveEmail != nil {
			email, _ = deps.ResolveEmail(r)
		}
		profile, err := deps.Service.Create(r.Context(), request, email)
		if writeRemoteProfileError(w, err, deps) {
			return
		}
		if err != nil {
			deps.LogError("create_remote_profile_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to create remote profile", "server_error")
			return
		}
		deps.WriteData(w, profile)
	}
}

func UpdateRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt64(w, r, "id")
		if !ok {
			return
		}
		var request admin.RemoteProfileUpdateRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		profile, err := deps.Service.Update(r.Context(), id, request)
		if writeRemoteProfileError(w, err, deps) {
			return
		}
		if err != nil {
			deps.LogError("update_remote_profile_failed", map[string]any{"error": err.Error(), "id": id})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to update remote profile", "server_error")
			return
		}
		deps.WriteData(w, profile)
	}
}

func DeleteRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return idOperation(deps, func(ctx context.Context, id int64) (any, error) { return nil, deps.Service.Delete(ctx, id) }, "delete_remote_profile_failed", "Failed to delete remote profile", func(w http.ResponseWriter, _ any) { deps.WriteSimple(w) })
}

func LoginRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt64(w, r, "id")
		if !ok {
			return
		}
		var request admin.RemoteProfileLoginRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		email, ok := deps.ValidateEmail(w, request.Email)
		if !ok {
			return
		}
		profile, err := deps.Service.Login(r.Context(), id, email, request.Password)
		if writeRemoteProfileError(w, err, deps) {
			return
		}
		if err != nil {
			deps.LogError("remote_profile_login_failed", map[string]any{"error": err.Error(), "id": id})
			deps.WriteError(w, http.StatusInternalServerError, "Remote login failed", "server_error")
			return
		}
		deps.WriteData(w, profile)
	}
}

func LogoutRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return idOperation(deps, func(ctx context.Context, id int64) (any, error) { return deps.Service.Logout(ctx, id) }, "remote_profile_logout_failed", "Remote logout failed", deps.WriteData)
}

func TestRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return idOperation(deps, func(ctx context.Context, id int64) (any, error) { return deps.Service.Test(ctx, id) }, "remote_profile_test_failed", "Remote profile test failed", deps.WriteData)
}

func RemoteProfileSessionLinks(deps RemoteProfileDependencies) http.HandlerFunc {
	return idOperation(deps, func(ctx context.Context, id int64) (any, error) { return deps.Service.SessionLinks(ctx, id) }, "remote_profile_session_links_failed", "Remote profile session inspection failed", deps.WriteData)
}

func RevokeRemoteProfileSessions(deps RemoteProfileDependencies) http.HandlerFunc {
	return idOperation(deps, func(ctx context.Context, id int64) (any, error) { return deps.Service.RevokeRemoteSessions(ctx, id) }, "remote_profile_remote_revoke_failed", "Remote session revoke failed", deps.WriteData)
}

func ProxyRemoteProfile(deps RemoteProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt64(w, r, "id")
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, admin.RemoteProfileProxyBodyLimit)
		var request admin.RemoteProfileProxyRequest
		if !deps.DecodeJSON(w, r, &request) {
			return
		}
		result, err := deps.Service.Proxy(r.Context(), id, request)
		if writeRemoteProfileError(w, err, deps) {
			return
		}
		if err != nil {
			deps.LogError("remote_profile_proxy_failed", map[string]any{"error": err.Error(), "id": id})
			deps.WriteError(w, http.StatusInternalServerError, "Remote proxy failed", "server_error")
			return
		}
		contentType := result.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(result.StatusCode)
		if len(result.Body) > 0 {
			if _, err := w.Write(result.Body); err != nil {
				deps.LogError("remote_proxy_response_write_failed", map[string]any{"error": err.Error(), "id": id})
			}
		}
	}
}

type idOperationFunc func(context.Context, int64) (any, error)

func idOperation(deps RemoteProfileDependencies, operation idOperationFunc, failureEvent, failureMessage string, writeSuccess func(http.ResponseWriter, any)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := deps.PathInt64(w, r, "id")
		if !ok {
			return
		}
		result, err := operation(r.Context(), id)
		if writeRemoteProfileError(w, err, deps) {
			return
		}
		if err != nil {
			deps.LogError(failureEvent, map[string]any{"error": err.Error(), "id": id})
			deps.WriteError(w, http.StatusInternalServerError, failureMessage, "server_error")
			return
		}
		writeSuccess(w, result)
	}
}

func writeRemoteProfileError(w http.ResponseWriter, err error, deps RemoteProfileDependencies) bool {
	if err == nil {
		return false
	}
	var remoteErr *admin.RemoteProfileError
	if errors.As(err, &remoteErr) {
		deps.WriteError(w, remoteErr.Status, remoteErr.Message, remoteErr.ErrorType)
		return true
	}
	for _, candidate := range []struct {
		err           error
		status        int
		message, kind string
	}{
		{admin.ErrRemoteProfileNotFound, http.StatusNotFound, "Remote profile not found", "not_found"},
		{admin.ErrRemoteProfileTagExists, http.StatusConflict, "Remote profile tag already exists", "validation"},
		{admin.ErrRemoteProfileSessionMissing, http.StatusConflict, "Remote profile is not logged in", "validation"},
		{admin.ErrRemoteProfileDisallowedPath, http.StatusForbidden, "Remote proxy path is not allowed", "forbidden"},
		{admin.ErrRemoteProfileInvalid, http.StatusBadRequest, "Invalid remote profile data", "validation"},
	} {
		if errors.Is(err, candidate.err) {
			deps.WriteError(w, candidate.status, candidate.message, candidate.kind)
			return true
		}
	}
	return false
}
