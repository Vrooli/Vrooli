package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/account"
	"landing-page-business-suite-api/internal/envx"
)

type HTTPDoer = account.HTTPDoer
type APIKeyStore = account.APIKeyStore
type APIKeyService = account.APIKeyService
type APIKey = account.APIKey
type APIKeyCreateRequest = account.APIKeyCreateRequest

func isProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
	return env == "production" || env == "prod"
}

func NewAPIKeyService(db APIKeyStore) (*APIKeyService, error) {
	return NewAPIKeyServiceWithHTTPClient(db, &http.Client{Timeout: 15 * time.Second})
}

func NewAPIKeyServiceWithHTTPClient(db APIKeyStore, client HTTPDoer) (*APIKeyService, error) {
	return NewAPIKeyServiceWithOptions(db, client, "postgres")
}

func NewAPIKeyServiceWithOptions(db APIKeyStore, client HTTPDoer, dialect string) (*APIKeyService, error) {
	return account.NewAPIKeyServiceWithRuntime(db, client, dialect, resolveSecret, isProductionEnvironment, logStructured, logStructuredError)
}

func newAPIKeyServiceForTest(db APIKeyStore, client HTTPDoer, dialect string, key []byte) *APIKeyService {
	return account.NewAPIKeyServiceForTest(db, client, dialect, key, nil, nil)
}

func GenerateEncryptionKey() string { return account.GenerateEncryptionKey() }

// API Handlers

func handleListAPIKeys(svc *APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("list_api_keys_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusInternalServerError, "Failed to list API keys", ApiErrorTypeServerError)
			return
		}

		if keys == nil {
			keys = []APIKey{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": keys,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleCreateAPIKey(svc *APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req APIKeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		key, err := svc.Store(r.Context(), req.Provider, req.Key)
		if err != nil {
			logStructuredError("create_api_key_failed", map[string]interface{}{
				"error":    err.Error(),
				"provider": req.Provider,
			})
			writeJSONError(w, http.StatusBadRequest, err.Error(), ApiErrorTypeValidation)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(key); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleDeleteAPIKey(svc *APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		if provider == "" {
			writeJSONError(w, http.StatusBadRequest, "Provider is required", ApiErrorTypeValidation)
			return
		}

		if err := svc.Delete(r.Context(), provider); err != nil {
			logStructuredError("delete_api_key_failed", map[string]interface{}{
				"error":    err.Error(),
				"provider": provider,
			})
			writeJSONError(w, http.StatusNotFound, err.Error(), ApiErrorTypeNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleTestAPIKey(svc *APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		if provider == "" {
			writeJSONError(w, http.StatusBadRequest, "Provider is required", ApiErrorTypeValidation)
			return
		}

		success, message, err := svc.Test(r.Context(), provider)
		if err != nil {
			logStructuredError("test_api_key_failed", map[string]interface{}{
				"error":    err.Error(),
				"provider": provider,
			})
			writeJSONError(w, http.StatusInternalServerError, err.Error(), ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  success,
			"message":  message,
			"provider": provider,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func handleToggleAPIKey(svc *APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string `json:"provider"`
			Active   bool   `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if err := svc.SetActive(r.Context(), req.Provider, req.Active); err != nil {
			logStructuredError("toggle_api_key_failed", map[string]interface{}{
				"error":    err.Error(),
				"provider": req.Provider,
			})
			writeJSONError(w, http.StatusNotFound, err.Error(), ApiErrorTypeNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// init registers the key generation command for setup
func init() {
	if len(os.Args) > 1 && os.Args[1] == "generate-encryption-key" {
		_, _ = os.Stdout.WriteString("New encryption key (add to LPBS_API_KEY_ENCRYPTION_KEY):\n")
		_, _ = os.Stdout.WriteString(GenerateEncryptionKey() + "\n")
		os.Exit(0)
	}
}

// Compile-time interface check for APIKeyServicer
var _ APIKeyServicer = (*APIKeyService)(nil)
