package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func handleGetTierLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tierID := vars["tier"]

		if tierID == "" {
			limits, err := svc.GetAllTierLimits(r.Context())
			if err != nil {
				logStructuredError("get_all_tier_limits_failed", map[string]interface{}{"error": err.Error()})
				writeJSONError(w, http.StatusInternalServerError, "Failed to get tier limits", ApiErrorTypeServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"limits": limits}); err != nil {
				logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
			}
			return
		}

		limits, err := svc.GetTierLimits(r.Context(), tierID)
		if err != nil {
			logStructuredError("get_tier_limits_failed", map[string]interface{}{"error": err.Error(), "tier_id": tierID})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get tier limits", ApiErrorTypeServerError)
			return
		}
		if limits == nil {
			limits = []TierLimit{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"tier_id": tierID, "limits": limits}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleUpdateTierLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tierID := mux.Vars(r)["tier"]
		if tierID == "" {
			writeJSONError(w, http.StatusBadRequest, "Tier ID is required", ApiErrorTypeValidation)
			return
		}

		var req struct {
			LimitKey     string          `json:"limit_key"`
			AppBundleKey *string         `json:"app_bundle_key"`
			Update       TierLimitUpdate `json:"update"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}
		if req.LimitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "limit_key is required", ApiErrorTypeValidation)
			return
		}

		limit, err := svc.UpdateLimit(r.Context(), tierID, req.LimitKey, req.AppBundleKey, req.Update)
		if err != nil {
			logStructuredError("update_tier_limit_failed", map[string]interface{}{"error": err.Error(), "tier_id": tierID, "limit_key": req.LimitKey})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(limit); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleGetAppLimits(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := mux.Vars(r)["app"]
		if appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "App key is required", ApiErrorTypeValidation)
			return
		}

		limits, err := svc.GetAppLimits(r.Context(), appKey)
		if err != nil {
			logStructuredError("get_app_limits_failed", map[string]interface{}{"error": err.Error(), "app_key": appKey})
			writeJSONError(w, http.StatusInternalServerError, "Failed to get app limits", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"app_bundle_key": appKey, "limits": limits}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleCreateTierLimit(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit TierLimit
		if err := json.NewDecoder(r.Body).Decode(&limit); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		result, err := svc.CreateLimit(r.Context(), limit)
		if err != nil {
			logStructuredError("create_tier_limit_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleDeleteTierLimit(svc *LimitsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TierID       string  `json:"tier_id"`
			LimitKey     string  `json:"limit_key"`
			AppBundleKey *string `json:"app_bundle_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}
		if req.TierID == "" || req.LimitKey == "" {
			writeJSONError(w, http.StatusBadRequest, "tier_id and limit_key are required", ApiErrorTypeValidation)
			return
		}

		if err := svc.DeleteLimit(r.Context(), req.TierID, req.LimitKey, req.AppBundleKey); err != nil {
			logStructuredError("delete_tier_limit_failed", map[string]interface{}{"error": err.Error(), "tier_id": req.TierID, "limit_key": req.LimitKey})
			writeJSONError(w, http.StatusNotFound, err.Error(), ApiErrorTypeNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
