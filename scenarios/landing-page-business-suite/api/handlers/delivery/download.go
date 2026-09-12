// Package delivery owns entitlement-gated delivery HTTP transport.
package delivery

import (
	"context"
	"net/http"
	"strings"
)

type Authorization struct {
	Payload    any
	Managed    bool
	ArtifactID int64
	SetURL     func(string)
}

type ErrorKind string

const (
	ErrorNotFound                ErrorKind = "not_found"
	ErrorAppNotFound             ErrorKind = "app_not_found"
	ErrorSubscriptionRequired    ErrorKind = "subscription_required"
	ErrorIdentityRequired        ErrorKind = "identity_required"
	ErrorPlatformRequired        ErrorKind = "platform_required"
	ErrorEntitlementsUnavailable ErrorKind = "entitlements_unavailable"
)

type Dependencies struct {
	UserEmail      func(context.Context) string
	Authorize      func(context.Context, string, string, string) (Authorization, error)
	ClassifyError  func(error) ErrorKind
	ResolveManaged func(context.Context, int64) (string, bool, error)
	ManagedError   func(error) (string, string)
	WriteJSON      func(http.ResponseWriter, any)
	WriteError     func(http.ResponseWriter, int, string, string)
	Log            func(string, map[string]any)
}

func Authorize(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := strings.TrimSpace(r.URL.Query().Get("app"))
		if appKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "App key is required.", "validation")
			return
		}
		platform := strings.TrimSpace(r.URL.Query().Get("platform"))
		if platform == "" {
			deps.WriteError(w, http.StatusBadRequest, "Platform is required.", "validation")
			return
		}
		user := deps.UserEmail(r.Context())
		if user == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Authentication required", "unauthorized")
			return
		}

		asset, err := deps.Authorize(r.Context(), appKey, platform, user)
		if err != nil {
			deps.Log("download_authorization_failed", map[string]any{"app_key": appKey, "platform": platform, "user": user, "error": err.Error()})
			switch deps.ClassifyError(err) {
			case ErrorNotFound:
				deps.WriteError(w, http.StatusNotFound, "Download not found for this platform.", "not_found")
			case ErrorAppNotFound:
				deps.WriteError(w, http.StatusNotFound, "Application not found.", "not_found")
			case ErrorSubscriptionRequired:
				deps.WriteError(w, http.StatusForbidden, "An active subscription is required to download this content.", "forbidden")
			case ErrorIdentityRequired:
				deps.WriteError(w, http.StatusBadRequest, "Please provide your email to download.", "validation")
			case ErrorPlatformRequired:
				deps.WriteError(w, http.StatusBadRequest, "Please select a platform.", "validation")
			case ErrorEntitlementsUnavailable:
				deps.WriteError(w, http.StatusServiceUnavailable, "Unable to verify your entitlements. Please try again.", "server_error")
			default:
				deps.WriteError(w, http.StatusInternalServerError, "Failed to authorize download. Please try again.", "server_error")
			}
			return
		}
		if asset.Managed && asset.ArtifactID > 0 {
			url, found, err := deps.ResolveManaged(r.Context(), asset.ArtifactID)
			if err != nil {
				event, message := deps.ManagedError(err)
				deps.Log(event, map[string]any{"app_key": appKey, "platform": platform, "artifact_id": asset.ArtifactID, "error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, message, "server_error")
				return
			}
			if !found {
				deps.Log("artifact_not_found", map[string]any{"app_key": appKey, "platform": platform, "artifact_id": asset.ArtifactID})
				deps.WriteError(w, http.StatusNotFound, "Download artifact not found.", "not_found")
				return
			}
			asset.SetURL(url)
		}
		deps.WriteJSON(w, asset.Payload)
	}
}
