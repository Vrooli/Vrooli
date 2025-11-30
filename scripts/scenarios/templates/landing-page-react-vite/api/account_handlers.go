package main

import (
	"encoding/json"
	"net/http"
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

func handleDownloads(downloadService *DownloadService, accountService *AccountService, planService *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform := r.URL.Query().Get("platform")
		if platform == "" {
			http.Error(w, "platform is required", http.StatusBadRequest)
			return
		}

		user := resolveUserIdentity(r)

		asset, err := downloadService.GetAsset(planService.BundleKey(), platform)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if asset.RequiresEntitlement {
			entitlements, err := accountService.GetEntitlements(user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if entitlements.Status != "active" && entitlements.Status != "trialing" {
				http.Error(w, "active subscription required for downloads", http.StatusForbidden)
				return
			}
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
