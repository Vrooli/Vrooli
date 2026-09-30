package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/delivery"
)

type Request struct {
	AppKey        string `json:"app_key"`
	RemoteProfile string `json:"remote_profile,omitempty"`
	Channel       string `json:"channel,omitempty"`
}

type Gate struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

type Response struct {
	Ready bool   `json:"ready"`
	Gates []Gate `json:"gates"`
	Error string `json:"error,omitempty"`
}

type StorageService interface {
	GetSettings(context.Context, string) (*delivery.StorageSettings, error)
}

type StorageTester interface {
	TestConnection(context.Context, string) error
}

type CatalogService interface {
	GetApp(string, string) (*delivery.App, error)
}

// RemoteProfileService is the local proxy surface for a stored remote LPBS
// deployment. Readiness must exercise the remote target and not just assert a
// profile row exists: a registered profile with an inactive session, an
// unreachable bucket, or a missing app is not ready to accept a release.
type RemoteProfileService interface {
	List(context.Context) ([]administration.RemoteProfile, error)
	Test(context.Context, int64) (*administration.RemoteProfile, error)
	Proxy(context.Context, int64, administration.RemoteProfileProxyRequest) (*administration.RemoteProxyResponse, error)
}

type Dependencies struct {
	Storage        StorageService
	TestStorage    StorageTester
	Catalog        CatalogService
	RemoteProfiles RemoteProfileService
	// StripeReadiness is intentionally injected at the composition root. The
	// deployment package must report commerce readiness without owning Stripe
	// credentials or provider clients.
	StripeReadiness func(context.Context) Gate
	BundleKey       func() string
	WriteError      func(http.ResponseWriter, int, string, string)
}

func Readiness(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err.Error() != "EOF" {
			deps.WriteError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err), "validation")
			return
		}
		response := CheckReadiness(r.Context(), deps, request)
		w.Header().Set("Content-Type", "application/json")
		if !response.Ready {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}

// CheckReadiness contains the transport-neutral deployment validation. Both
// the legacy REST compatibility endpoint and Connect use this exact workflow.
func CheckReadiness(ctx context.Context, deps Dependencies, request Request) Response {
	bundleKey := deps.BundleKey()
	gates := []Gate{storageGate(ctx, deps.Storage, deps.TestStorage, bundleKey)}
	if deps.StripeReadiness != nil {
		gates = append(gates, deps.StripeReadiness(ctx))
	}
	if strings.TrimSpace(request.AppKey) != "" {
		gates = append(gates, appGate(deps.Catalog, bundleKey, request.AppKey))
	}
	if strings.TrimSpace(request.RemoteProfile) != "" {
		profileGate, profileID := remoteProfileGate(ctx, deps.RemoteProfiles, request.RemoteProfile)
		gates = append(gates, profileGate)
		// The remote target, not the local proxy host, owns the bucket and the
		// app registry that a remote release writes to. Prove both, and only
		// when the session is active so the proxy call cannot masquerade as a
		// missing-permission failure.
		gates = append(gates, remoteStorageGate(ctx, deps.RemoteProfiles, profileID, profileGate.Ready))
		if strings.TrimSpace(request.AppKey) != "" {
			gates = append(gates, remoteAppGate(ctx, deps.RemoteProfiles, profileID, request.AppKey, profileGate.Ready))
		}
	}
	response := Response{Ready: true, Gates: gates}
	for _, gate := range gates {
		if !gate.Ready {
			response.Ready = false
			response.Error = firstUnreadyMessage(gates)
			break
		}
	}
	return response
}

func storageGate(ctx context.Context, storage StorageService, tester StorageTester, bundleKey string) Gate {
	gate := Gate{Name: "download_storage"}
	settings, err := storage.GetSettings(ctx, bundleKey)
	if err != nil {
		gate.Message = fmt.Sprintf("read storage settings: %v", err)
		return gate
	}
	if settings == nil || strings.TrimSpace(settings.Bucket) == "" {
		gate.Message = "S3 download storage is not configured"
		return gate
	}
	if tester != nil {
		probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := tester.TestConnection(probeCtx, bundleKey); err != nil {
			gate.Message = fmt.Sprintf("distribution object operations failed: %v", err)
			return gate
		}
	}
	gate.Ready = true
	return gate
}

func appGate(catalog CatalogService, bundleKey, appKey string) Gate {
	gate := Gate{Name: "app_registered"}
	app, err := catalog.GetApp(bundleKey, appKey)
	if err != nil {
		gate.Message = fmt.Sprintf("lookup app %q: %v", appKey, err)
		return gate
	}
	if app == nil {
		gate.Message = fmt.Sprintf("app %q is not registered in download_apps", appKey)
		return gate
	}
	gate.Ready = true
	return gate
}

// remoteProfileGate resolves the stored remote profile by tag and proves its
// session is active. It returns the numeric profile id for the downstream
// remote probes; a zero id means the profile could not be resolved.
func remoteProfileGate(ctx context.Context, profiles RemoteProfileService, tag string) (Gate, int64) {
	gate := Gate{Name: "remote_profile_session"}
	if profiles == nil {
		gate.Message = "remote-profile service is unavailable"
		return gate, 0
	}
	registered, err := profiles.List(ctx)
	if err != nil {
		gate.Message = fmt.Sprintf("list remote profiles: %v", err)
		return gate, 0
	}
	for _, profile := range registered {
		if profile.Tag != tag {
			continue
		}
		if _, err := profiles.Test(ctx, profile.ID); err != nil {
			gate.Message = fmt.Sprintf("remote profile %q session is not active: %v", tag, err)
			return gate, profile.ID
		}
		gate.Ready = true
		return gate, profile.ID
	}
	gate.Message = fmt.Sprintf("remote profile %q is not registered", tag)
	return gate, 0
}

// remoteStorageGate proves the remote target's download bucket accepts object
// write, read, and delete. The local readiness probe cannot speak for the
// remote bucket that an actual release writes to.
func remoteStorageGate(ctx context.Context, profiles RemoteProfileService, profileID int64, sessionReady bool) Gate {
	gate := Gate{Name: "remote_download_storage"}
	if profiles == nil {
		gate.Message = "remote-profile service is unavailable"
		return gate
	}
	if !sessionReady || profileID == 0 {
		gate.Message = "remote profile session is not active"
		return gate
	}
	response, err := profiles.Proxy(ctx, profileID, administration.RemoteProfileProxyRequest{
		Method: http.MethodPost,
		Path:   "/admin/download-storage/test",
	})
	if err != nil {
		gate.Message = fmt.Sprintf("remote storage test failed: %v", err)
		return gate
	}
	if response == nil {
		gate.Message = "remote storage test returned no response"
		return gate
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		gate.Message = fmt.Sprintf("remote storage test failed: status %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
		return gate
	}
	gate.Ready = true
	return gate
}

// remoteAppGate proves the release app key already exists in the remote
// catalog. A release that names an unregistered app can stage bytes but can
// never promote them, so it is not ready.
func remoteAppGate(ctx context.Context, profiles RemoteProfileService, profileID int64, appKey string, sessionReady bool) Gate {
	gate := Gate{Name: "remote_app_key"}
	if profiles == nil {
		gate.Message = "remote-profile service is unavailable"
		return gate
	}
	if !sessionReady || profileID == 0 {
		gate.Message = "remote profile session is not active"
		return gate
	}
	response, err := profiles.Proxy(ctx, profileID, administration.RemoteProfileProxyRequest{
		Method: http.MethodGet,
		Path:   "/admin/download-apps",
	})
	if err != nil {
		gate.Message = fmt.Sprintf("list remote download apps: %v", err)
		return gate
	}
	if response == nil {
		gate.Message = "remote app lookup returned no response"
		return gate
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		gate.Message = fmt.Sprintf("list remote download apps failed: status %d: %s", response.StatusCode, strings.TrimSpace(string(response.Body)))
		return gate
	}
	var parsed struct {
		Apps []struct {
			AppKey string `json:"app_key"`
			Name   string `json:"name"`
		} `json:"apps"`
	}
	if err := json.Unmarshal(response.Body, &parsed); err != nil {
		gate.Message = fmt.Sprintf("parse remote download apps: %v", err)
		return gate
	}
	for _, app := range parsed.Apps {
		if strings.TrimSpace(app.AppKey) == appKey {
			gate.Ready = true
			return gate
		}
	}
	gate.Message = fmt.Sprintf("remote app key missing: %s", appKey)
	return gate
}

func firstUnreadyMessage(gates []Gate) string {
	for _, gate := range gates {
		if !gate.Ready {
			if gate.Message != "" {
				return fmt.Sprintf("%s: %s", gate.Name, gate.Message)
			}
			return gate.Name + ": not ready"
		}
	}
	return ""
}
