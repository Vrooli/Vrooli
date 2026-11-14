package billing

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

type LimitsDependencies struct {
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
}

func GetTierLimits(svc *commerce.LimitsService, deps LimitsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tierID := vars["tier"]

		if tierID == "" {
			limits, err := svc.GetAllTierLimits(r.Context())
			if err != nil {
				deps.Log("get_all_tier_limits_failed", map[string]any{"error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, "Failed to get tier limits", "server_error")
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"limits": limits}); err != nil {
				deps.Log("encode_response_failed", map[string]any{"error": err.Error()})
			}
			return
		}

		limits, err := svc.GetTierLimits(r.Context(), tierID)
		if err != nil {
			deps.Log("get_tier_limits_failed", map[string]any{"error": err.Error(), "tier_id": tierID})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get tier limits", "server_error")
			return
		}
		if limits == nil {
			limits = []commerce.TierLimit{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"tier_id": tierID, "limits": limits}); err != nil {
			deps.Log("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func UpdateTierLimits(svc *commerce.LimitsService, deps LimitsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tierID := mux.Vars(r)["tier"]
		if tierID == "" {
			deps.WriteError(w, http.StatusBadRequest, "Tier ID is required", "validation")
			return
		}

		var req struct {
			LimitKey     string                   `json:"limit_key"`
			AppBundleKey *string                  `json:"app_bundle_key"`
			Update       commerce.TierLimitUpdate `json:"update"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if req.LimitKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "limit_key is required", "validation")
			return
		}

		limit, err := svc.UpdateLimit(r.Context(), tierID, req.LimitKey, req.AppBundleKey, req.Update)
		if err != nil {
			deps.Log("update_tier_limit_failed", map[string]any{"error": err.Error(), "tier_id": tierID, "limit_key": req.LimitKey})
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(limit); err != nil {
			deps.Log("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func GetAppLimits(svc *commerce.LimitsService, deps LimitsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := mux.Vars(r)["app"]
		if appKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "App key is required", "validation")
			return
		}

		limits, err := svc.GetAppLimits(r.Context(), appKey)
		if err != nil {
			deps.Log("get_app_limits_failed", map[string]any{"error": err.Error(), "app_key": appKey})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to get app limits", "server_error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"app_bundle_key": appKey, "limits": limits}); err != nil {
			deps.Log("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func CreateTierLimit(svc *commerce.LimitsService, deps LimitsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit commerce.TierLimit
		if err := json.NewDecoder(r.Body).Decode(&limit); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}

		result, err := svc.CreateLimit(r.Context(), limit)
		if err != nil {
			deps.Log("create_tier_limit_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			deps.Log("encode_response_failed", map[string]any{"error": err.Error()})
		}
	}
}

func DeleteTierLimit(svc *commerce.LimitsService, deps LimitsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TierID       string  `json:"tier_id"`
			LimitKey     string  `json:"limit_key"`
			AppBundleKey *string `json:"app_bundle_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		if req.TierID == "" || req.LimitKey == "" {
			deps.WriteError(w, http.StatusBadRequest, "tier_id and limit_key are required", "validation")
			return
		}

		if err := svc.DeleteLimit(r.Context(), req.TierID, req.LimitKey, req.AppBundleKey); err != nil {
			deps.Log("delete_tier_limit_failed", map[string]any{"error": err.Error(), "tier_id": req.TierID, "limit_key": req.LimitKey})
			deps.WriteError(w, http.StatusNotFound, err.Error(), "not_found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
