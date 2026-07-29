package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/delivery"
)

// DeployReadinessRequest is the payload from deployment-manager (or another
// inter-scenario caller) asking LPBS to confirm it can accept a new release
// for the given app/profile/channel triple.
type DeployReadinessRequest struct {
	AppKey        string `json:"app_key"`
	RemoteProfile string `json:"remote_profile,omitempty"`
	Channel       string `json:"channel,omitempty"`
}

// DeployReadinessGate reports the result of a single gate check.
type DeployReadinessGate struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// DeployReadinessResponse is the structured outcome of the readiness check.
type DeployReadinessResponse struct {
	Ready bool                  `json:"ready"`
	Gates []DeployReadinessGate `json:"gates"`
	Error string                `json:"error,omitempty"`
}

// handleDeployReadiness runs the server-side equivalent of the CLI
// `deploy-readiness` checks: storage configured, app exists, and (when a
// remote-profile tag is supplied) the profile is registered.
func handleDeployReadiness(downloadHosting *delivery.Service, downloadService *DownloadService, remoteProfiles *administration.RemoteProfileService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DeployReadinessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err), ApiErrorTypeValidation)
			return
		}
		bundleKey := planService.BundleKey()
		ctx := r.Context()

		gates := make([]DeployReadinessGate, 0, 3)

		gates = append(gates, checkStorageGate(ctx, downloadHosting, bundleKey))

		if strings.TrimSpace(req.AppKey) != "" {
			gates = append(gates, checkAppGate(downloadService, bundleKey, req.AppKey))
		}

		if strings.TrimSpace(req.RemoteProfile) != "" {
			gates = append(gates, checkRemoteProfileGate(ctx, remoteProfiles, req.RemoteProfile))
		}

		ready := true
		for _, g := range gates {
			if !g.Ready {
				ready = false
				break
			}
		}

		resp := DeployReadinessResponse{Ready: ready, Gates: gates}
		w.Header().Set("Content-Type", "application/json")
		if !ready {
			resp.Error = firstUnreadyMessage(gates)
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func checkStorageGate(ctx context.Context, downloadHosting *delivery.Service, bundleKey string) DeployReadinessGate {
	gate := DeployReadinessGate{Name: "download_storage"}
	settings, err := downloadHosting.GetSettings(ctx, bundleKey)
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

func checkAppGate(downloadService *DownloadService, bundleKey, appKey string) DeployReadinessGate {
	gate := DeployReadinessGate{Name: "app_registered"}
	app, err := downloadService.GetApp(bundleKey, appKey)
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

func checkRemoteProfileGate(ctx context.Context, remoteProfiles *administration.RemoteProfileService, tag string) DeployReadinessGate {
	gate := DeployReadinessGate{Name: "remote_profile_registered"}
	profiles, err := remoteProfiles.List(ctx)
	if err != nil {
		gate.Message = fmt.Sprintf("list remote profiles: %v", err)
		return gate
	}
	for _, p := range profiles {
		if p.Tag == tag {
			gate.Ready = true
			return gate
		}
	}
	gate.Message = fmt.Sprintf("remote profile %q is not registered", tag)
	return gate
}

func firstUnreadyMessage(gates []DeployReadinessGate) string {
	for _, g := range gates {
		if !g.Ready {
			if g.Message != "" {
				return fmt.Sprintf("%s: %s", g.Name, g.Message)
			}
			return g.Name + ": not ready"
		}
	}
	return ""
}
