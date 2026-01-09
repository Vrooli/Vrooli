package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func handleLandingConfig(service *LandingConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		variant := r.URL.Query().Get("variant")
		config, err := service.GetLandingConfig(r.Context(), variant)
		if err != nil {
			http.Error(w, "Failed to load landing config", http.StatusInternalServerError)
			return
		}

		writeJSON(w, config)
	}
}

func handlePlans(service *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overview, err := service.GetPricingOverview()
		if err != nil {
			http.Error(w, "Failed to load plans", http.StatusInternalServerError)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.GetPricingResponse{Pricing: overview})
	}
}

func handleMeSubscription(accountService *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := resolveUserIdentity(r)
		subscription, err := accountService.GetSubscription(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, &landing_page_react_vite_v1.VerifySubscriptionResponse{Status: subscription})
	}
}

func handleMeCredits(accountService *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := resolveUserIdentity(r)
		credits, err := accountService.GetCredits(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		balance := map[string]interface{}{}
		if credits.Balance != nil {
			if data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(credits.Balance); err == nil {
				_ = json.Unmarshal(data, &balance)
			}
		}

		writeJSON(w, map[string]interface{}{
			"balance":                    balance,
			"display_credits_label":      credits.DisplayCreditsLabel,
			"display_credits_multiplier": credits.DisplayCreditsMultiplier,
		})
	}
}

func handleEntitlements(accountService *AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := resolveUserIdentity(r)
		entitlements, err := accountService.GetEntitlements(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, entitlements)
	}
}

func handleDownloads(authorizer *DownloadAuthorizer, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := strings.TrimSpace(r.URL.Query().Get("app"))
		if appKey == "" {
			http.Error(w, "app is required", http.StatusBadRequest)
			return
		}
		platform := strings.TrimSpace(r.URL.Query().Get("platform"))
		if platform == "" {
			http.Error(w, "platform is required", http.StatusBadRequest)
			return
		}

		user := resolveUserIdentity(r)

		asset, err := authorizer.Authorize(appKey, platform, user)
		if err != nil {
			switch {
			case errors.Is(err, ErrDownloadNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, ErrDownloadAppNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, ErrDownloadRequiresActiveSubscription):
				http.Error(w, err.Error(), http.StatusForbidden)
			case errors.Is(err, ErrDownloadIdentityRequired):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, ErrDownloadPlatformRequired):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, ErrDownloadEntitlementsUnavailable):
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if asset != nil && strings.TrimSpace(asset.ArtifactSource) == "managed" && asset.ArtifactID != nil && *asset.ArtifactID > 0 {
			artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), *asset.ArtifactID)
			if err != nil {
				http.Error(w, "failed to resolve managed artifact", http.StatusInternalServerError)
				return
			}
			if artifact == nil {
				http.Error(w, "download artifact not found", http.StatusNotFound)
				return
			}
			signedURL, err := hosting.PresignGetArtifact(r.Context(), plans.BundleKey(), *artifact)
			if err != nil {
				http.Error(w, "failed to generate download url", http.StatusInternalServerError)
				return
			}
			asset.ArtifactURL = signedURL
		}

		writeJSON(w, asset)
	}
}

func resolveUserIdentity(r *http.Request) string {
	if user := r.Header.Get("X-User-Email"); user != "" {
		return user
	}
	if user := r.URL.Query().Get("user"); user != "" {
		return user
	}
	return ""
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if msg, ok := payload.(proto.Message); ok {
		data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
