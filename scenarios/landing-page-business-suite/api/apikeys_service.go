package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/securevalue"
)

// HTTPDoer is an interface for making HTTP requests, used for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// APIKeyStore is the context-aware persistence contract for encrypted provider
// credentials.
//
// seam: APIKeyStore keeps API-key persistence independent of a concrete pool
// and preserves request-scoped test isolation.
type APIKeyStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// APIKeyService manages encrypted API keys for AI providers.
type APIKeyService struct {
	db            APIKeyStore
	encryptionKey []byte // 32 bytes for AES-256
	httpClient    HTTPDoer
	dialects      *DialectHelper
}

// APIKey represents an AI provider API key (without the actual key value).
type APIKey struct {
	ID             string     `json:"id"`
	Provider       string     `json:"provider"`
	KeyHint        string     `json:"key_hint"`
	IsActive       bool       `json:"is_active"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// APIKeyCreateRequest is the request to create a new API key.
type APIKeyCreateRequest struct {
	Provider string `json:"provider"` // openrouter, openai, anthropic
	Key      string `json:"key"`
}

// NewAPIKeyService creates a new API key service with encryption.
func NewAPIKeyService(db APIKeyStore) (*APIKeyService, error) {
	return NewAPIKeyServiceWithHTTPClient(db, &http.Client{Timeout: 15 * time.Second})
}

// NewAPIKeyServiceWithHTTPClient creates a new API key service with a custom HTTP client.
// This is useful for testing with a mock HTTP client.
func NewAPIKeyServiceWithHTTPClient(db APIKeyStore, httpClient HTTPDoer) (*APIKeyService, error) {
	return NewAPIKeyServiceWithOptions(db, httpClient, "postgres")
}

// isProductionEnvironment checks if the service is running in production mode.
func isProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
	return env == "production" || env == "prod"
}

// NewAPIKeyServiceWithOptions creates a new API key service with custom options.
// Dialect can be "postgres" or "sqlite".
func NewAPIKeyServiceWithOptions(db APIKeyStore, httpClient HTTPDoer, dialect string) (*APIKeyService, error) {
	dialects := NewDialectHelper(dialect)

	// Get encryption key from environment
	keyStr := resolveSecret("LPBS_API_KEY_ENCRYPTION_KEY")
	if keyStr == "" {
		// In production, encryption is mandatory
		if isProductionEnvironment() {
			return nil, fmt.Errorf(
				"LPBS_API_KEY_ENCRYPTION_KEY is required in production.\n" +
					"Generate a key with: ./lpbs-api generate-encryption-key\n" +
					"Then set it in your environment or ~/.vrooli/secrets.json",
			)
		}
		// Development mode - allow unencrypted storage with prominent warning
		logStructured("apikeys_no_encryption_key_dev", map[string]interface{}{
			"level":    "warn",
			"message":  "LPBS_API_KEY_ENCRYPTION_KEY not set; API keys will be stored unencrypted",
			"security": true,
			"action":   "Set LPBS_API_KEY_ENCRYPTION_KEY before deploying to production",
		})
		return &APIKeyService{db: db, encryptionKey: nil, httpClient: httpClient, dialects: dialects}, nil
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (got %d)", len(key))
	}

	return &APIKeyService{db: db, encryptionKey: key, httpClient: httpClient, dialects: dialects}, nil
}

// encrypt encrypts plaintext using AES-256-GCM.
func (s *APIKeyService) encrypt(plaintext string) (string, error) {
	return securevalue.Encrypt(s.encryptionKey, plaintext)
}

// decrypt decrypts ciphertext using AES-256-GCM.
func (s *APIKeyService) decrypt(ciphertext string) (string, error) {
	return securevalue.Decrypt(s.encryptionKey, ciphertext)
}

// getKeyHint returns the last 4 characters of a key for display.
func getKeyHint(key string) string {
	key = strings.TrimSpace(key)
	if len(key) < 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// Store stores a new API key for a provider.
func (s *APIKeyService) Store(ctx context.Context, provider, key string) (*APIKey, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	key = strings.TrimSpace(key)

	if provider == "" || key == "" {
		return nil, fmt.Errorf("provider and key are required")
	}

	// Validate provider
	validProviders := map[string]bool{"openrouter": true, "openai": true, "anthropic": true}
	if !validProviders[provider] {
		return nil, fmt.Errorf("invalid provider: %s (must be openrouter, openai, or anthropic)", provider)
	}

	// Encrypt the key
	encryptedKey, err := s.encrypt(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt key: %w", err)
	}

	keyHint := getKeyHint(key)

	// Upsert the key (SQLite uses different syntax for upsert)
	var query string
	if s.dialects.IsSQLite() {
		query = `
			INSERT INTO api_keys (provider, encrypted_key, key_hint, is_active, updated_at)
			VALUES (?, ?, ?, 1, datetime('now'))
			ON CONFLICT (provider)
			DO UPDATE SET encrypted_key = excluded.encrypted_key,
				key_hint = excluded.key_hint,
				is_active = 1,
				updated_at = datetime('now')
			RETURNING id, provider, key_hint, is_active, last_verified_at, created_at, updated_at
		`
	} else {
		query = `
			INSERT INTO api_keys (provider, encrypted_key, key_hint, is_active, updated_at)
			VALUES ($1, $2, $3, true, NOW())
			ON CONFLICT (provider)
			DO UPDATE SET encrypted_key = EXCLUDED.encrypted_key,
				key_hint = EXCLUDED.key_hint,
				is_active = true,
				updated_at = NOW()
			RETURNING id, provider, key_hint, is_active, last_verified_at, created_at, updated_at
		`
	}

	var apiKey APIKey
	var lastVerified sql.NullTime
	err = s.db.QueryRowContext(ctx, query, provider, encryptedKey, keyHint).Scan(
		&apiKey.ID, &apiKey.Provider, &apiKey.KeyHint, &apiKey.IsActive,
		&lastVerified, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("store api key: %w", err)
	}

	if lastVerified.Valid {
		apiKey.LastVerifiedAt = &lastVerified.Time
	}

	logStructured("api_key_stored", map[string]interface{}{
		"level":    "info",
		"provider": provider,
	})

	return &apiKey, nil
}

// Get retrieves the decrypted API key for a provider.
func (s *APIKeyService) Get(ctx context.Context, provider string) (string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))

	var query string
	if s.dialects.IsSQLite() {
		query = `SELECT encrypted_key, is_active FROM api_keys WHERE provider = ?`
	} else {
		query = `SELECT encrypted_key, is_active FROM api_keys WHERE provider = $1`
	}

	var encryptedKey string
	var isActive bool
	err := s.db.QueryRowContext(ctx, query, provider).Scan(&encryptedKey, &isActive)

	if err == sql.ErrNoRows {
		return "", nil // No key configured
	}
	if err != nil {
		return "", fmt.Errorf("get api key: %w", err)
	}

	if !isActive {
		return "", nil // Key is disabled
	}

	return s.decrypt(encryptedKey)
}

// List returns all API keys (without decrypted values).
func (s *APIKeyService) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, provider, key_hint, is_active, last_verified_at, created_at, updated_at
		FROM api_keys
		ORDER BY provider
	`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		var lastVerified sql.NullTime
		if err := rows.Scan(&key.ID, &key.Provider, &key.KeyHint, &key.IsActive,
			&lastVerified, &key.CreatedAt, &key.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		if lastVerified.Valid {
			key.LastVerifiedAt = &lastVerified.Time
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// Delete removes an API key for a provider.
func (s *APIKeyService) Delete(ctx context.Context, provider string) error {
	provider = strings.TrimSpace(strings.ToLower(provider))

	var query string
	if s.dialects.IsSQLite() {
		query = `DELETE FROM api_keys WHERE provider = ?`
	} else {
		query = `DELETE FROM api_keys WHERE provider = $1`
	}

	result, err := s.db.ExecContext(ctx, query, provider)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("api key not found: %s", provider)
	}

	logStructured("api_key_deleted", map[string]interface{}{
		"level":    "info",
		"provider": provider,
	})

	return nil
}

// SetActive enables or disables an API key.
func (s *APIKeyService) SetActive(ctx context.Context, provider string, active bool) error {
	provider = strings.TrimSpace(strings.ToLower(provider))

	var query string
	if s.dialects.IsSQLite() {
		query = `UPDATE api_keys SET is_active = ?, updated_at = datetime('now') WHERE provider = ?`
	} else {
		query = `UPDATE api_keys SET is_active = $1, updated_at = NOW() WHERE provider = $2`
	}

	result, err := s.db.ExecContext(ctx, query, active, provider)
	if err != nil {
		return fmt.Errorf("set api key active: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("api key not found: %s", provider)
	}

	return nil
}

// Test validates an API key by making a simple API call to the provider.
func (s *APIKeyService) Test(ctx context.Context, provider string) (bool, string, error) {
	key, err := s.Get(ctx, provider)
	if err != nil {
		return false, "", err
	}
	if key == "" {
		return false, "No API key configured for this provider", nil
	}

	var success bool
	var message string

	switch provider {
	case "openrouter":
		success, message = s.testOpenRouter(ctx, key)
	case "openai":
		success, message = s.testOpenAI(ctx, key)
	case "anthropic":
		success, message = s.testAnthropic(ctx, key)
	default:
		return false, fmt.Sprintf("Unknown provider: %s", provider), nil
	}

	// Update last_verified_at if successful
	if success {
		var updateQuery string
		if s.dialects.IsSQLite() {
			updateQuery = `UPDATE api_keys SET last_verified_at = datetime('now'), updated_at = datetime('now') WHERE provider = ?`
		} else {
			updateQuery = `UPDATE api_keys SET last_verified_at = NOW(), updated_at = NOW() WHERE provider = $1`
		}
		_, updateErr := s.db.ExecContext(ctx, updateQuery, provider)
		logOnError(updateErr, "update_api_key_last_verified", map[string]interface{}{
			"provider": provider,
		})
	}

	return success, message, nil
}

// testOpenRouter tests an OpenRouter API key.
func (s *APIKeyService) testOpenRouter(ctx context.Context, key string) (bool, string) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/auth/key", nil)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, "API key is valid"
	}
	if resp.StatusCode == 401 {
		return false, "Invalid API key"
	}
	return false, fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
}

// testOpenAI tests an OpenAI API key.
func (s *APIKeyService) testOpenAI(ctx context.Context, key string) (bool, string) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, "API key is valid"
	}
	if resp.StatusCode == 401 {
		return false, "Invalid API key"
	}
	return false, fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
}

// testAnthropic tests an Anthropic API key.
func (s *APIKeyService) testAnthropic(ctx context.Context, key string) (bool, string) {
	// Anthropic requires a POST to /v1/messages with a minimal request
	body := `{"model":"claude-3-haiku-20240307","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, "API key is valid"
	}
	if resp.StatusCode == 401 {
		return false, "Invalid API key"
	}
	// 400 or 429 indicates the key works but there might be other issues
	if resp.StatusCode == 400 || resp.StatusCode == 429 {
		return true, "API key is valid (rate limited or validation error)"
	}
	return false, fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
}

// GenerateEncryptionKey generates a new 32-byte encryption key and prints it.
// This is a helper for setup.
func GenerateEncryptionKey() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

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
