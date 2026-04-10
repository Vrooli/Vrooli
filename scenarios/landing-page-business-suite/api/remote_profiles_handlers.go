package main

import (
	"context"
	"errors"
	"net/http"
)

type adminEmailResolver func(*http.Request) (string, bool)

type RemoteProfileManager interface {
	List(ctx context.Context) ([]RemoteProfile, error)
	Create(ctx context.Context, req RemoteProfileCreateRequest, createdByEmail string) (*RemoteProfile, error)
	Update(ctx context.Context, id int64, req RemoteProfileUpdateRequest) (*RemoteProfile, error)
	Delete(ctx context.Context, id int64) error
	Login(ctx context.Context, id int64, email string, password string) (*RemoteProfile, error)
	Logout(ctx context.Context, id int64) (*RemoteProfile, error)
	Test(ctx context.Context, id int64) (*RemoteProfile, error)
	SessionLinks(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error)
	RevokeRemoteSessions(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error)
	Proxy(ctx context.Context, id int64, req RemoteProfileProxyRequest) (*RemoteProxyResponse, error)
}

func handleAdminListRemoteProfiles(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profiles, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("list_remote_profiles_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to list remote profiles", ApiErrorTypeServerError)
			return
		}
		if profiles == nil {
			profiles = []RemoteProfile{}
		}
		writeJSONSuccessData(w, map[string]interface{}{"profiles": profiles})
	}
}

func handleAdminCreateRemoteProfile(svc RemoteProfileManager, resolveEmail adminEmailResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RemoteProfileCreateRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}
		email := ""
		if resolveEmail != nil {
			email, _ = resolveEmail(r)
		}
		profile, err := svc.Create(r.Context(), req, email)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("create_remote_profile_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to create remote profile", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, profile)
	}
}

func handleAdminUpdateRemoteProfile(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		var req RemoteProfileUpdateRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}
		profile, err := svc.Update(r.Context(), id, req)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("update_remote_profile_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to update remote profile", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, profile)
	}
}

func handleAdminDeleteRemoteProfile(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		if err := svc.Delete(r.Context(), id); err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("delete_remote_profile_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete remote profile", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessSimple(w)
	}
}

func handleAdminRemoteProfileLogin(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		var req RemoteProfileLoginRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}
		email, ok := ValidateEmailForHandler(w, req.Email)
		if !ok {
			return
		}
		profile, err := svc.Login(r.Context(), id, email, req.Password)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_login_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote login failed", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, profile)
	}
}

func handleAdminRemoteProfileLogout(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		profile, err := svc.Logout(r.Context(), id)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_logout_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote logout failed", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, profile)
	}
}

func handleAdminRemoteProfileTest(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		profile, err := svc.Test(r.Context(), id)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_test_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote profile test failed", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, profile)
	}
}

func handleAdminRemoteProfileSessionLinks(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		links, err := svc.SessionLinks(r.Context(), id)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_session_links_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote profile session inspection failed", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, links)
	}
}

func handleAdminRemoteProfileRemoteRevoke(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		links, err := svc.RevokeRemoteSessions(r.Context(), id)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_remote_revoke_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote session revoke failed", ApiErrorTypeServerError)
			return
		}
		writeJSONSuccessData(w, links)
	}
}

func handleAdminRemoteProfileProxy(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, remoteProfileProxyBodyLimit)
		var req RemoteProfileProxyRequest
		if !decodeJSONBody(w, r, &req) {
			return
		}
		result, err := svc.Proxy(r.Context(), id, req)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError("remote_profile_proxy_failed", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, "Remote proxy failed", ApiErrorTypeServerError)
			return
		}

		if result.ContentType != "" {
			w.Header().Set("Content-Type", result.ContentType)
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(result.StatusCode)
		if len(result.Body) > 0 {
			if _, err := w.Write(result.Body); err != nil {
				logStructuredError("remote_proxy_response_write_failed", map[string]interface{}{
					"error": err.Error(),
					"id":    id,
				})
			}
		}
	}
}

func writeRemoteProfileError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var remoteErr *RemoteProfileError
	if errors.As(err, &remoteErr) {
		writeJSONError(w, remoteErr.Status, remoteErr.Message, remoteErr.ErrorType)
		return true
	}
	if errors.Is(err, ErrRemoteProfileNotFound) {
		writeJSONError(w, http.StatusNotFound, "Remote profile not found", ApiErrorTypeNotFound)
		return true
	}
	if errors.Is(err, ErrRemoteProfileTagExists) {
		writeJSONError(w, http.StatusConflict, "Remote profile tag already exists", ApiErrorTypeValidation)
		return true
	}
	if errors.Is(err, ErrRemoteProfileSessionMissing) {
		writeJSONError(w, http.StatusConflict, "Remote profile is not logged in", ApiErrorTypeValidation)
		return true
	}
	if errors.Is(err, ErrRemoteProfileDisallowedPath) {
		writeJSONError(w, http.StatusForbidden, "Remote proxy path is not allowed", ApiErrorTypeForbidden)
		return true
	}
	if errors.Is(err, ErrRemoteProfileInvalid) {
		writeJSONError(w, http.StatusBadRequest, "Invalid remote profile data", ApiErrorTypeValidation)
		return true
	}
	return false
}
