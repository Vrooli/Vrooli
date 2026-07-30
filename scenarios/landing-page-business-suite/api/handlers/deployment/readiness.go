package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

type CatalogService interface {
	GetApp(string, string) (*delivery.App, error)
}

type RemoteProfileService interface {
	List(context.Context) ([]administration.RemoteProfile, error)
}

type Dependencies struct {
	Storage        StorageService
	Catalog        CatalogService
	RemoteProfiles RemoteProfileService
	BundleKey      func() string
	WriteError     func(http.ResponseWriter, int, string, string)
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
	gates := []Gate{storageGate(ctx, deps.Storage, bundleKey)}
	if strings.TrimSpace(request.AppKey) != "" {
		gates = append(gates, appGate(deps.Catalog, bundleKey, request.AppKey))
	}
	if strings.TrimSpace(request.RemoteProfile) != "" {
		gates = append(gates, remoteProfileGate(ctx, deps.RemoteProfiles, request.RemoteProfile))
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

func storageGate(ctx context.Context, storage StorageService, bundleKey string) Gate {
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

func remoteProfileGate(ctx context.Context, profiles RemoteProfileService, tag string) Gate {
	gate := Gate{Name: "remote_profile_registered"}
	if profiles == nil {
		gate.Message = "remote-profile service is unavailable"
		return gate
	}
	registered, err := profiles.List(ctx)
	if err != nil {
		gate.Message = fmt.Sprintf("list remote profiles: %v", err)
		return gate
	}
	for _, profile := range registered {
		if profile.Tag == tag {
			gate.Ready = true
			return gate
		}
	}
	gate.Message = fmt.Sprintf("remote profile %q is not registered", tag)
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
