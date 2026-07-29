package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	downloadhttp "landing-page-business-suite-api/handlers/download"
)

type managedDownloadResolutionError struct {
	stage string
	err   error
}

func (e *managedDownloadResolutionError) Error() string { return e.err.Error() }
func (e *managedDownloadResolutionError) Unwrap() error { return e.err }

func handleDownloads(authorizer *DownloadAuthorizer, hosting *DownloadHostingService, plans *PlanService) http.HandlerFunc {
	return downloadhttp.Authorize(downloadhttp.Dependencies{
		UserEmail: getUserEmail,
		Authorize: func(ctx context.Context, appKey, platform, user string) (downloadhttp.Authorization, error) {
			asset, err := authorizer.Authorize(ctx, appKey, platform, user)
			if err != nil || asset == nil {
				return downloadhttp.Authorization{}, err
			}
			artifactID := int64(0)
			if asset.ArtifactID != nil {
				artifactID = *asset.ArtifactID
			}
			return downloadhttp.Authorization{Payload: asset, Managed: asset.ArtifactSource == "managed", ArtifactID: artifactID, SetURL: func(url string) { asset.ArtifactURL = url }}, nil
		},
		ClassifyError: func(err error) downloadhttp.ErrorKind {
			switch {
			case errors.Is(err, ErrDownloadNotFound):
				return downloadhttp.ErrorNotFound
			case errors.Is(err, ErrDownloadAppNotFound):
				return downloadhttp.ErrorAppNotFound
			case errors.Is(err, ErrDownloadRequiresActiveSubscription):
				return downloadhttp.ErrorSubscriptionRequired
			case errors.Is(err, ErrDownloadIdentityRequired):
				return downloadhttp.ErrorIdentityRequired
			case errors.Is(err, ErrDownloadPlatformRequired):
				return downloadhttp.ErrorPlatformRequired
			case errors.Is(err, ErrDownloadEntitlementsUnavailable):
				return downloadhttp.ErrorEntitlementsUnavailable
			}
			return ""
		},
		ResolveManaged: func(ctx context.Context, id int64) (string, bool, error) {
			artifact, err := hosting.GetArtifact(ctx, plans.BundleKey(), id)
			if err != nil {
				return "", false, &managedDownloadResolutionError{stage: "fetch", err: err}
			}
			if artifact == nil {
				return "", false, nil
			}
			url, err := hosting.PresignGetArtifact(ctx, plans.BundleKey(), *artifact)
			if err != nil {
				return "", true, &managedDownloadResolutionError{stage: "presign", err: err}
			}
			return url, true, nil
		},
		ManagedError: func(err error) (string, string) {
			var resolution *managedDownloadResolutionError
			if errors.As(err, &resolution) && resolution.stage == "fetch" {
				return "artifact_fetch_failed", "Failed to resolve download. Please try again."
			}
			return "presign_url_failed", "Failed to generate download link. Please try again."
		},
		WriteJSON:  writeJSON,
		WriteError: writeJSONError,
		Log:        logStructuredError,
	})
}

// NOTE: resolveUserIdentity has been removed as part of the user authentication implementation.
// User identity is now derived from JWT claims via getUserEmail(r.Context()).
// See user_auth_middleware.go for the authentication middleware.

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if msg, ok := payload.(proto.Message); ok {
		data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
			return
		}
		// #nosec G705 -- this is an application/json response with nosniff, not an HTML sink;
		// protojson output is required to preserve the generated API contract.
		if _, err := w.Write(data); err != nil {
			logStructuredError("write_json_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to encode response", ApiErrorTypeServerError)
	}
}

// ApiErrorType constants for structured JSON error responses.
// These align with the frontend's ApiError class for consistent error handling.
const (
	ApiErrorTypeNetwork      = "network"
	ApiErrorTypeTimeout      = "timeout"
	ApiErrorTypeUnauthorized = "unauthorized"
	ApiErrorTypeForbidden    = "forbidden"
	ApiErrorTypeNotFound     = "not_found"
	ApiErrorTypeValidation   = "validation"
	ApiErrorTypeRateLimited  = "rate_limited"
	ApiErrorTypeServerError  = "server_error"
)

// ApiErrorResponse is a structured JSON error response that the frontend can parse.
// It aligns with the frontend's ApiError class for graceful degradation.
type ApiErrorResponse struct {
	Error     string `json:"error"`                // Human-readable error message
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
