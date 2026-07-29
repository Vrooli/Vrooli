package main

import (
	"context"
	"errors"
	"net/http"

	"landing-page-business-suite-api/internal/administration"
)

type adminEmailResolver func(*http.Request) (string, bool)

type RemoteProfileManager interface {
	List(ctx context.Context) ([]administration.RemoteProfile, error)
	Create(ctx context.Context, req administration.RemoteProfileCreateRequest, createdByEmail string) (*administration.RemoteProfile, error)
	Update(ctx context.Context, id int64, req administration.RemoteProfileUpdateRequest) (*administration.RemoteProfile, error)
	Delete(ctx context.Context, id int64) error
	Login(ctx context.Context, id int64, email string, password string) (*administration.RemoteProfile, error)
	Logout(ctx context.Context, id int64) (*administration.RemoteProfile, error)
	Test(ctx context.Context, id int64) (*administration.RemoteProfile, error)
	SessionLinks(ctx context.Context, id int64) (*administration.RemoteProfileSessionLinks, error)
	RevokeRemoteSessions(ctx context.Context, id int64) (*administration.RemoteProfileSessionLinks, error)
	Proxy(ctx context.Context, id int64, req administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error)
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
			profiles = []administration.RemoteProfile{}
		}
		writeJSONSuccessData(w, map[string]interface{}{"profiles": profiles})
	}
}

func handleAdminCreateRemoteProfile(svc RemoteProfileManager, resolveEmail adminEmailResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req administration.RemoteProfileCreateRequest
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
		var req administration.RemoteProfileUpdateRequest
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
	return handleRemoteProfileIDOperation(
		func(ctx context.Context, id int64) (interface{}, error) {
			return nil, svc.Delete(ctx, id)
		},
		"delete_remote_profile_failed",
		"Failed to delete remote profile",
		func(w http.ResponseWriter, _ interface{}) { writeJSONSuccessSimple(w) },
	)
}

func handleAdminRemoteProfileLogin(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		var req administration.RemoteProfileLoginRequest
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
	return handleRemoteProfileIDOperation(
		func(ctx context.Context, id int64) (interface{}, error) { return svc.Logout(ctx, id) },
		"remote_profile_logout_failed",
		"Remote logout failed",
		writeJSONSuccessData,
	)
}

func handleAdminRemoteProfileTest(svc RemoteProfileManager) http.HandlerFunc {
	return handleRemoteProfileIDOperation(
		func(ctx context.Context, id int64) (interface{}, error) { return svc.Test(ctx, id) },
		"remote_profile_test_failed",
		"Remote profile test failed",
		writeJSONSuccessData,
	)
}

func handleAdminRemoteProfileSessionLinks(svc RemoteProfileManager) http.HandlerFunc {
	return handleRemoteProfileIDOperation(
		func(ctx context.Context, id int64) (interface{}, error) { return svc.SessionLinks(ctx, id) },
		"remote_profile_session_links_failed",
		"Remote profile session inspection failed",
		writeJSONSuccessData,
	)
}

func handleAdminRemoteProfileRemoteRevoke(svc RemoteProfileManager) http.HandlerFunc {
	return handleRemoteProfileIDOperation(
		func(ctx context.Context, id int64) (interface{}, error) { return svc.RevokeRemoteSessions(ctx, id) },
		"remote_profile_remote_revoke_failed",
		"Remote session revoke failed",
		writeJSONSuccessData,
	)
}

type (
	remoteProfileIDOperation   func(context.Context, int64) (interface{}, error)
	remoteProfileSuccessWriter func(http.ResponseWriter, interface{})
)

// handleRemoteProfileIDOperation keeps every ID-only remote-profile endpoint
// consistent: path parsing, domain-error mapping, structured logging, and the
// server-error response cannot drift between session-management operations.
func handleRemoteProfileIDOperation(
	operation remoteProfileIDOperation,
	failureEvent string,
	failureMessage string,
	writeSuccess remoteProfileSuccessWriter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		result, err := operation(r.Context(), id)
		if err != nil {
			if writeRemoteProfileError(w, err) {
				return
			}
			logStructuredError(failureEvent, map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
			writeJSONError(w, http.StatusInternalServerError, failureMessage, ApiErrorTypeServerError)
			return
		}
		writeSuccess(w, result)
	}
}

func handleAdminRemoteProfileProxy(svc RemoteProfileManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, administration.RemoteProfileProxyBodyLimit)
		var req administration.RemoteProfileProxyRequest
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
	var remoteErr *administration.RemoteProfileError
	if errors.As(err, &remoteErr) {
		writeJSONError(w, remoteErr.Status, remoteErr.Message, remoteErr.ErrorType)
		return true
	}
	if errors.Is(err, administration.ErrRemoteProfileNotFound) {
		writeJSONError(w, http.StatusNotFound, "Remote profile not found", ApiErrorTypeNotFound)
		return true
	}
	if errors.Is(err, administration.ErrRemoteProfileTagExists) {
		writeJSONError(w, http.StatusConflict, "Remote profile tag already exists", ApiErrorTypeValidation)
		return true
	}
	if errors.Is(err, administration.ErrRemoteProfileSessionMissing) {
		writeJSONError(w, http.StatusConflict, "Remote profile is not logged in", ApiErrorTypeValidation)
		return true
	}
	if errors.Is(err, administration.ErrRemoteProfileDisallowedPath) {
		writeJSONError(w, http.StatusForbidden, "Remote proxy path is not allowed", ApiErrorTypeForbidden)
		return true
	}
	if errors.Is(err, administration.ErrRemoteProfileInvalid) {
		writeJSONError(w, http.StatusBadRequest, "Invalid remote profile data", ApiErrorTypeValidation)
		return true
	}
	return false
}
