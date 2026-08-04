package administration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/securevalue"
)

// HTTPDoer is an interface for making HTTP requests, used for testing.
type APIKeyHTTPDoer interface {
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
	httpClient    APIKeyHTTPDoer
	dialect       string
	logEvent      func(string, map[string]interface{})
	logError      func(string, map[string]interface{})
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

// NewAPIKeyServiceWithRuntime wires application-owned secret, environment,
// and logging behavior at the composition boundary.
func NewAPIKeyServiceWithRuntime(db APIKeyStore, httpClient APIKeyHTTPDoer, dialect string, resolveSecret func(string) string, isProduction func() bool, logEvent func(string, map[string]interface{}), logError func(string, map[string]interface{})) (*APIKeyService, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if dialect == "" {
		dialect = "postgres"
	}
	if resolveSecret == nil {
		return nil, fmt.Errorf("API key secret resolver is required")
	}
	if isProduction == nil {
		return nil, fmt.Errorf("API key production policy is required")
	}
	if err := ensureAPIKeyEncryptionState(db, dialect); err != nil {
		return nil, fmt.Errorf("ensure api key encryption state: %w", err)
	}
	keyStr := resolveSecret("LPBS_API_KEY_ENCRYPTION_KEY")
	if keyStr == "" {
		if isProduction() {
			return nil, fmt.Errorf("LPBS_API_KEY_ENCRYPTION_KEY is required in production; provision it with `vrooli credentials provision`")
		}
		if logEvent != nil {
			logEvent("apikeys_no_encryption_key_dev", map[string]interface{}{"level": "warn", "message": "LPBS_API_KEY_ENCRYPTION_KEY not set; API keys will be stored unencrypted", "security": true, "action": "Set LPBS_API_KEY_ENCRYPTION_KEY before deploying to production"})
		}
		return &APIKeyService{db: db, httpClient: httpClient, dialect: dialect, logEvent: logEvent, logError: logError}, nil
	}
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (got %d)", len(key))
	}
	return NewAPIKeyServiceForTest(db, httpClient, dialect, key, logEvent, logError), nil
}

// NewAPIKeyServiceForTest constructs the domain service with an explicit key.
// It is intentionally exported only within this repository's internal package.
func NewAPIKeyServiceForTest(db APIKeyStore, httpClient APIKeyHTTPDoer, dialect string, encryptionKey []byte, logEvent func(string, map[string]interface{}), logError func(string, map[string]interface{})) *APIKeyService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if dialect == "" {
		dialect = "postgres"
	}
	_ = ensureAPIKeyEncryptionState(db, dialect)
	return &APIKeyService{db: db, encryptionKey: encryptionKey, httpClient: httpClient, dialect: dialect, logEvent: logEvent, logError: logError}
}

func ensureAPIKeyEncryptionState(db APIKeyStore, dialect string) error {
	if dialect == "sqlite" {
		_, err := db.ExecContext(context.Background(), `ALTER TABLE api_keys ADD COLUMN encryption_state TEXT NOT NULL DEFAULT 'unknown'`)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
		return nil
	}
	_, err := db.ExecContext(context.Background(), `ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS encryption_state TEXT NOT NULL DEFAULT 'unknown'`)
	return err
}

func (s *APIKeyService) isSQLite() bool { return s.dialect == "sqlite" }

func (s *APIKeyService) log(event string, fields map[string]interface{}) {
	if s.logEvent != nil {
		s.logEvent(event, fields)
	}
}

func (s *APIKeyService) logFailure(event string, fields map[string]interface{}) {
	if s.logError != nil {
		s.logError(event, fields)
	}
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
	if s.isSQLite() {
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
	state := "sealed"
	if s.encryptionKey == nil {
		state = "unsealed"
	}
	if _, err := s.db.ExecContext(ctx, s.encryptionStateUpdateQuery(), state, provider); err != nil {
		return nil, fmt.Errorf("record api key encryption state: %w", err)
	}

	if lastVerified.Valid {
		apiKey.LastVerifiedAt = &lastVerified.Time
	}

	s.log("api_key_stored", map[string]interface{}{
		"level":    "info",
		"provider": provider,
	})

	return &apiKey, nil
}

func (s *APIKeyService) encryptionStateUpdateQuery() string {
	if s.isSQLite() {
		return `UPDATE api_keys SET encryption_state = ? WHERE provider = ?`
	}
	return `UPDATE api_keys SET encryption_state = $1 WHERE provider = $2`
}

// Get retrieves the decrypted API key for a provider.
func (s *APIKeyService) Get(ctx context.Context, provider string) (string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))

	var query string
	if s.isSQLite() {
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
	if s.isSQLite() {
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

	s.log("api_key_deleted", map[string]interface{}{
		"level":    "info",
		"provider": provider,
	})

	return nil
}

// SetActive enables or disables an API key.
func (s *APIKeyService) SetActive(ctx context.Context, provider string, active bool) error {
	provider = strings.TrimSpace(strings.ToLower(provider))

	var query string
	if s.isSQLite() {
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
		if s.isSQLite() {
			updateQuery = `UPDATE api_keys SET last_verified_at = datetime('now'), updated_at = datetime('now') WHERE provider = ?`
		} else {
			updateQuery = `UPDATE api_keys SET last_verified_at = NOW(), updated_at = NOW() WHERE provider = $1`
		}
		_, updateErr := s.db.ExecContext(ctx, updateQuery, provider)
		if updateErr != nil {
			s.logFailure("update_api_key_last_verified", map[string]interface{}{"error": updateErr.Error(), "provider": provider})
		}
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

// TestOpenAIProvider exposes the provider probe to in-repository contract
// tests without exposing the service's persistence or encryption internals.
func (s *APIKeyService) TestOpenAIProvider(ctx context.Context, key string) (bool, string) {
	return s.testOpenAI(ctx, key)
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

// TestAnthropicProvider exposes the provider probe to in-repository contract
// tests without exposing the service's persistence or encryption internals.
func (s *APIKeyService) TestAnthropicProvider(ctx context.Context, key string) (bool, string) {
	return s.testAnthropic(ctx, key)
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
