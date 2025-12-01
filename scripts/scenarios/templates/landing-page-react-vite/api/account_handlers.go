package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
		writeJSON(w, overview)
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
		writeJSON(w, subscription)
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
		writeJSON(w, credits)
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

func handleDownloads(authorizer *DownloadAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform := strings.TrimSpace(r.URL.Query().Get("platform"))
		if platform == "" {
			http.Error(w, "platform is required", http.StatusBadRequest)
			return
		}

		user := resolveUserIdentity(r)

		asset, err := authorizer.Authorize(platform, user)
		if err != nil {
			switch {
			case errors.Is(err, ErrDownloadNotFound):
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
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
