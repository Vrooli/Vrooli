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
			logStructuredError("landing_config_failed", map[string]interface{}{
				"variant": variant,
				"error":   err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to load landing config. Please try again.", ApiErrorTypeServerError)
			return
		}

		writeJSON(w, config)
	}
}

func handlePlans(service *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		overview, err := service.GetPricingOverview()
		if err != nil {
			logStructuredError("plans_load_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to load pricing plans. Please try again.", ApiErrorTypeServerError)
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
			logStructuredError("subscription_fetch_failed", map[string]interface{}{
				"user":  user,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve subscription status. Please try again.", ApiErrorTypeServerError)
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
			logStructuredError("credits_fetch_failed", map[string]interface{}{
				"user":  user,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve credit balance. Please try again.", ApiErrorTypeServerError)
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
			logStructuredError("entitlements_fetch_failed", map[string]interface{}{
				"user":  user,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve entitlements. Please try again.", ApiErrorTypeServerError)
			return
		}
		writeJSON(w, entitlements)
	}
}

func handleDownloads(authorizer *DownloadAuthorizer, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := strings.TrimSpace(r.URL.Query().Get("app"))
		if appKey == "" {
			writeJSONError(w, http.StatusBadRequest, "App key is required.", ApiErrorTypeValidation)
			return
		}
		platform := strings.TrimSpace(r.URL.Query().Get("platform"))
		if platform == "" {
			writeJSONError(w, http.StatusBadRequest, "Platform is required.", ApiErrorTypeValidation)
			return
		}

		user := resolveUserIdentity(r)

		asset, err := authorizer.Authorize(appKey, platform, user)
		if err != nil {
			logStructuredError("download_authorization_failed", map[string]interface{}{
				"app_key":  appKey,
				"platform": platform,
				"user":     user,
				"error":    err.Error(),
			})
			switch {
			case errors.Is(err, ErrDownloadNotFound):
				writeJSONError(w, http.StatusNotFound, "Download not found for this platform.", ApiErrorTypeNotFound)
			case errors.Is(err, ErrDownloadAppNotFound):
				writeJSONError(w, http.StatusNotFound, "Application not found.", ApiErrorTypeNotFound)
			case errors.Is(err, ErrDownloadRequiresActiveSubscription):
				writeJSONError(w, http.StatusForbidden, "An active subscription is required to download this content.", ApiErrorTypeForbidden)
			case errors.Is(err, ErrDownloadIdentityRequired):
				writeJSONError(w, http.StatusBadRequest, "Please provide your email to download.", ApiErrorTypeValidation)
			case errors.Is(err, ErrDownloadPlatformRequired):
				writeJSONError(w, http.StatusBadRequest, "Please select a platform.", ApiErrorTypeValidation)
			case errors.Is(err, ErrDownloadEntitlementsUnavailable):
				writeJSONError(w, http.StatusServiceUnavailable, "Unable to verify your entitlements. Please try again.", ApiErrorTypeServerError)
			default:
				writeJSONError(w, http.StatusInternalServerError, "Failed to authorize download. Please try again.", ApiErrorTypeServerError)
			}
			return
		}

		if asset != nil && strings.TrimSpace(asset.ArtifactSource) == "managed" && asset.ArtifactID != nil && *asset.ArtifactID > 0 {
			artifact, err := hosting.GetArtifact(r.Context(), plans.BundleKey(), *asset.ArtifactID)
			if err != nil {
				logStructuredError("artifact_fetch_failed", map[string]interface{}{
					"app_key":     appKey,
					"platform":    platform,
					"artifact_id": *asset.ArtifactID,
					"error":       err.Error(),
				})
				writeJSONError(w, http.StatusInternalServerError, "Failed to resolve download. Please try again.", ApiErrorTypeServerError)
				return
			}
			if artifact == nil {
				logStructuredError("artifact_not_found", map[string]interface{}{
					"app_key":     appKey,
					"platform":    platform,
					"artifact_id": *asset.ArtifactID,
				})
				writeJSONError(w, http.StatusNotFound, "Download artifact not found.", ApiErrorTypeNotFound)
				return
			}
			signedURL, err := hosting.PresignGetArtifact(r.Context(), plans.BundleKey(), *artifact)
			if err != nil {
				logStructuredError("presign_url_failed", map[string]interface{}{
					"app_key":     appKey,
					"platform":    platform,
					"artifact_id": *asset.ArtifactID,
					"error":       err.Error(),
				})
				writeJSONError(w, http.StatusInternalServerError, "Failed to generate download link. Please try again.", ApiErrorTypeServerError)
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
		if _, err := w.Write(data); err != nil {
			logStructuredError("write_json_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ApiErrorType constants for structured JSON error responses.
// These align with the frontend's ApiError class for consistent error handling.
const (
	ApiErrorTypeNetwork     = "network"
	ApiErrorTypeTimeout     = "timeout"
	ApiErrorTypeUnauthorized = "unauthorized"
	ApiErrorTypeForbidden   = "forbidden"
	ApiErrorTypeNotFound    = "not_found"
	ApiErrorTypeValidation  = "validation"
	ApiErrorTypeRateLimited = "rate_limited"
	ApiErrorTypeServerError = "server_error"
)

// ApiErrorResponse is a structured JSON error response that the frontend can parse.
// It aligns with the frontend's ApiError class for graceful degradation.
type ApiErrorResponse struct {
	Error     string `json:"error"`               // Human-readable error message
	ErrorType string `json:"error_type,omitempty"` // Machine-readable error type
	Retryable bool   `json:"retryable,omitempty"`  // Whether the client should offer retry
}

// writeJSONError writes a structured JSON error response with proper status code.
// The errorType should be one of the ApiErrorType constants.
// If errorType is empty, it will be inferred from the HTTP status code.
func writeJSONError(w http.ResponseWriter, status int, message string, errorType string) {
	// Infer error type from status if not provided
	if errorType == "" {
		errorType = inferErrorType(status)
	}

	// Determine if error is retryable
	retryable := isRetryableErrorType(errorType)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ApiErrorResponse{
		Error:     message,
		ErrorType: errorType,
		Retryable: retryable,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logStructuredError("write_json_error_failed", map[string]interface{}{
			"error":          err.Error(),
			"status":         status,
			"original_error": message,
		})
	}
}

// inferErrorType derives an error type from HTTP status code
func inferErrorType(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ApiErrorTypeValidation
	case http.StatusUnauthorized:
		return ApiErrorTypeUnauthorized
	case http.StatusForbidden:
		return ApiErrorTypeForbidden
	case http.StatusNotFound:
		return ApiErrorTypeNotFound
	case http.StatusTooManyRequests:
		return ApiErrorTypeRateLimited
	case http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ApiErrorTypeServerError
	default:
		if status >= 500 {
			return ApiErrorTypeServerError
		}
		return ApiErrorTypeValidation
	}
}

// isRetryableErrorType returns true if the error type typically warrants retry
func isRetryableErrorType(errorType string) bool {
	switch errorType {
	case ApiErrorTypeNetwork, ApiErrorTypeTimeout, ApiErrorTypeServerError, ApiErrorTypeRateLimited:
		return true
	default:
		return false
	}
}
