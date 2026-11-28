package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/securevalue"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	_ "github.com/mattn/go-sqlite3"
)

// mockHTTPDoer is a mock HTTP client for testing.
type mockHTTPDoer struct {
	statusCode int
	body       string
	err        error
	requests   []*http.Request
}

func apiKeyHandlerDependencies(service *administration.APIKeyService) adminhttp.APIKeyDependencies {
	return adminhttp.APIKeyDependencies{
		Service:    service,
		WriteError: writeJSONError,
		LogError:   logx.Error,
	}
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.body)),
	}, nil
}

// createTestAPIKeysDB creates an in-memory SQLite database for testing.
func createTestAPIKeysDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use SQLite with _loc=auto to enable custom functions via connection string
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_functions=true")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Register a NOW() function for SQLite (equivalent to PostgreSQL's NOW())
	// This creates a custom SQL function that returns the current timestamp
	_, err = db.Exec(`SELECT datetime('now')`) // Verify connection works
	if err != nil {
		t.Fatalf("Failed to verify SQLite connection: %v", err)
	}

	// Create required tables with SQLite-compatible schema
	schema := `
		CREATE TABLE api_keys (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			provider TEXT NOT NULL UNIQUE,
			encrypted_key TEXT NOT NULL,
			key_hint TEXT,
			is_active INTEGER DEFAULT 1,
			last_verified_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// createTestAPIKeyService creates an API key service for testing without encryption.
func createTestAPIKeyService(t *testing.T, httpClient administration.APIKeyHTTPDoer) (*administration.APIKeyService, *sql.DB) {
	t.Helper()

	db := createTestAPIKeysDB(t)
	if httpClient == nil {
		httpClient = &mockHTTPDoer{statusCode: 200}
	}

	// Service without encryption key (development mode) with SQLite dialect.
	// Construct through the Account domain rather than reaching into its fields.
	svc := newAPIKeyServiceForTest(db, httpClient, "sqlite", nil)

	return svc, db
}

// createTestAPIKeyServiceWithEncryption creates an API key service with encryption enabled.
func createTestAPIKeyServiceWithEncryption(t *testing.T, httpClient administration.APIKeyHTTPDoer) (*administration.APIKeyService, *sql.DB) {
	t.Helper()

	db := createTestAPIKeysDB(t)
	if httpClient == nil {
		httpClient = &mockHTTPDoer{statusCode: 200}
	}

	// Generate a 32-byte encryption key
	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}

	svc := newAPIKeyServiceForTest(db, httpClient, "sqlite", encryptionKey)

	return svc, db
}

// ============================================================================
// Store Tests
// ============================================================================

func TestAPIKeyService_Store_StoresKey(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	apiKey, err := svc.Store(ctx, "openrouter", "sk-or-test-key-12345678")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Verify the result
	if apiKey.Provider != "openrouter" {
		t.Errorf("Expected provider 'openrouter', got '%s'", apiKey.Provider)
	}
	if apiKey.KeyHint != "****5678" {
		t.Errorf("Expected key hint '****5678', got '%s'", apiKey.KeyHint)
	}
	if !apiKey.IsActive {
		t.Error("Expected IsActive to be true")
	}

	// Verify database storage
	var provider, keyHint string
	var isActive bool
	err = db.QueryRow("SELECT provider, key_hint, is_active FROM api_keys WHERE provider = ?", "openrouter").Scan(&provider, &keyHint, &isActive)
	if err != nil {
		t.Fatalf("Failed to query stored key: %v", err)
	}
	if provider != "openrouter" || keyHint != "****5678" || !isActive {
		t.Errorf("Database values don't match: provider=%s, key_hint=%s, is_active=%v", provider, keyHint, isActive)
	}
}

func TestAPIKeyService_Store_EncryptsKey(t *testing.T) {
	svc, db := createTestAPIKeyServiceWithEncryption(t, nil)
	defer db.Close()

	ctx := context.Background()
	plainKey := "test-openrouter-key-12345678"

	// Store a key
	_, err := svc.Store(ctx, "openrouter", plainKey)
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Verify the stored key is encrypted (not plaintext)
	var encryptedKey string
	err = db.QueryRow("SELECT encrypted_key FROM api_keys WHERE provider = ?", "openrouter").Scan(&encryptedKey)
	if err != nil {
		t.Fatalf("Failed to query stored key: %v", err)
	}

	// The encrypted key should be base64 encoded and different from plaintext
	if encryptedKey == plainKey {
		t.Error("Key was stored as plaintext instead of encrypted")
	}

	// It should be valid base64
	_, err = base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		t.Errorf("Encrypted key is not valid base64: %v", err)
	}
}

func TestAPIKeyService_Store_GeneratesKeyHint(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	testCases := []struct {
		key          string
		expectedHint string
	}{
		{"sk-or-test-key-abcd", "****abcd"},
		{"xyz", "****"},
		{"ab", "****"},
		{"test-key-wxyz1234", "****1234"},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			// Delete any existing key first
			_, _ = db.Exec("DELETE FROM api_keys WHERE provider = ?", "openrouter")

			apiKey, err := svc.Store(ctx, "openrouter", tc.key)
			if err != nil {
				t.Fatalf("Store() returned error: %v", err)
			}
			if apiKey.KeyHint != tc.expectedHint {
				t.Errorf("Expected hint '%s', got '%s'", tc.expectedHint, apiKey.KeyHint)
			}
		})
	}
}

func TestAPIKeyService_Store_DuplicateProvider_Updates(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store initial key
	_, err := svc.Store(ctx, "openrouter", "sk-or-first-key-1234")
	if err != nil {
		t.Fatalf("First Store() returned error: %v", err)
	}

	// Store updated key for same provider
	apiKey, err := svc.Store(ctx, "openrouter", "sk-or-second-key-5678")
	if err != nil {
		t.Fatalf("Second Store() returned error: %v", err)
	}

	// Verify the key hint was updated
	if apiKey.KeyHint != "****5678" {
		t.Errorf("Expected updated hint '****5678', got '%s'", apiKey.KeyHint)
	}

	// Verify only one row exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE provider = ?", "openrouter").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count keys: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 key, got %d", count)
	}
}

func TestAPIKeyService_Store_RejectsInvalidInputs(t *testing.T) {
	cases := []struct{ name, provider, key, wantError string }{
		{name: "unknown provider", provider: "invalid-provider", key: "some-key", wantError: "invalid provider"},
		{name: "empty provider", provider: "", key: "some-key", wantError: "provider and key are required"},
		{name: "whitespace provider", provider: " \t ", key: "some-key", wantError: "provider and key are required"},
		{name: "empty key", provider: "openrouter", key: "", wantError: "provider and key are required"},
		{name: "whitespace key", provider: "openrouter", key: "  \t ", wantError: "provider and key are required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			_, err := svc.Store(context.Background(), tc.provider, tc.key)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("Store(%q, %q) error = %v, want text %q", tc.provider, tc.key, err, tc.wantError)
			}
		})
	}
}

// ============================================================================
// Get Tests
// ============================================================================

func TestAPIKeyService_Get_ReturnsDecryptedKey(t *testing.T) {
	svc, db := createTestAPIKeyServiceWithEncryption(t, nil)
	defer db.Close()

	ctx := context.Background()
	plainKey := "test-openrouter-key-12345678"

	// Store a key
	_, err := svc.Store(ctx, "openrouter", plainKey)
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Get the key
	retrievedKey, err := svc.Get(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	// Verify it's the original plaintext key
	if retrievedKey != plainKey {
		t.Errorf("Expected '%s', got '%s'", plainKey, retrievedKey)
	}
}

func TestAPIKeyService_Get_ReturnsEmptyForUnavailableKeys(t *testing.T) {
	cases := []struct {
		name, provider, storeKey string
		deactivate               bool
	}{
		{name: "provider does not exist", provider: "nonexistent"},
		{name: "key is inactive", provider: "openrouter", storeKey: "some-key", deactivate: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			if tc.storeKey != "" {
				if _, err := svc.Store(context.Background(), tc.provider, tc.storeKey); err != nil {
					t.Fatalf("store key: %v", err)
				}
				if tc.deactivate {
					if err := svc.SetActive(context.Background(), tc.provider, false); err != nil {
						t.Fatalf("deactivate key: %v", err)
					}
				}
			}
			got, err := svc.Get(context.Background(), tc.provider)
			if err != nil || got != "" {
				t.Fatalf("Get(%q) = %q, %v; want empty key and no error", tc.provider, got, err)
			}
		})
	}
}

// ============================================================================
// List Tests
// ============================================================================

func TestAPIKeyService_List_ReturnsStoredPublicMetadata(t *testing.T) {
	cases := []struct {
		name string
		keys []struct{ provider, value, hint string }
	}{
		{name: "empty table"},
		{name: "all providers", keys: []struct{ provider, value, hint string }{
			{provider: "openrouter", value: "sk-or-test", hint: "****test"},
			{provider: "openai", value: "sk-openai-test", hint: "****test"},
			{provider: "anthropic", value: "sk-ant-test", hint: "****test"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			expected := make(map[string]string, len(tc.keys))
			for _, key := range tc.keys {
				if _, err := svc.Store(context.Background(), key.provider, key.value); err != nil {
					t.Fatalf("store %s: %v", key.provider, err)
				}
				expected[key.provider] = key.hint
			}
			got, err := svc.List(context.Background())
			if err != nil {
				t.Fatalf("List() error: %v", err)
			}
			if len(got) != len(expected) {
				t.Fatalf("List() returned %d keys, want %d", len(got), len(expected))
			}
			for _, key := range got {
				wantHint, exists := expected[key.Provider]
				if !exists || key.KeyHint != wantHint || !strings.HasPrefix(key.KeyHint, "****") {
					t.Errorf("listed key = %+v; want masked metadata for a requested provider", key)
				}
			}
		})
	}
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestAPIKeyService_Delete_ExistingOrMissing(t *testing.T) {
	cases := []struct{ name, provider, storeKey, wantError string }{
		{name: "existing key", provider: "openrouter", storeKey: "some-key"},
		{name: "missing key", provider: "nonexistent", wantError: "api key not found: nonexistent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			if tc.storeKey != "" {
				if _, err := svc.Store(context.Background(), tc.provider, tc.storeKey); err != nil {
					t.Fatalf("store key: %v", err)
				}
			}
			err := svc.Delete(context.Background(), tc.provider)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Delete() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Delete() error: %v", err)
			}
			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE provider = ?", tc.provider).Scan(&count); err != nil {
				t.Fatalf("count remaining keys: %v", err)
			}
			if count != 0 {
				t.Fatalf("remaining keys = %d, want 0", count)
			}
		})
	}
}

// ============================================================================
// SetActive Tests
// ============================================================================

func TestAPIKeyService_SetActive_ExistingOrMissing(t *testing.T) {
	cases := []struct {
		name, provider, storeKey string
		wantError                bool
	}{
		{name: "existing key", provider: "openrouter", storeKey: "some-key"},
		{name: "missing key", provider: "nonexistent", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			if tc.storeKey != "" {
				if _, err := svc.Store(context.Background(), tc.provider, tc.storeKey); err != nil {
					t.Fatalf("store key: %v", err)
				}
			}
			err := svc.SetActive(context.Background(), tc.provider, false)
			if tc.wantError {
				if err == nil {
					t.Fatal("SetActive() succeeded for missing provider")
				}
				return
			}
			if err != nil {
				t.Fatalf("SetActive(false): %v", err)
			}
			var active bool
			if err := db.QueryRow("SELECT is_active FROM api_keys WHERE provider = ?", tc.provider).Scan(&active); err != nil {
				t.Fatalf("read inactive state: %v", err)
			}
			if active {
				t.Fatal("key remained active after deactivation")
			}
			if err := svc.SetActive(context.Background(), tc.provider, true); err != nil {
				t.Fatalf("SetActive(true): %v", err)
			}
			if err := db.QueryRow("SELECT is_active FROM api_keys WHERE provider = ?", tc.provider).Scan(&active); err != nil {
				t.Fatalf("read active state: %v", err)
			}
			if !active {
				t.Fatal("key remained inactive after reactivation")
			}
		})
	}
}

// ============================================================================
// Test API Key Tests
// ============================================================================

func TestAPIKeyService_Test_OpenRouter_Success(t *testing.T) {
	mockHTTP := &mockHTTPDoer{statusCode: 200}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	_, err := svc.Store(ctx, "openrouter", "sk-or-test")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Test the key
	success, message, err := svc.Test(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Test() returned error: %v", err)
	}

	if !success {
		t.Errorf("Expected success=true, got false with message: %s", message)
	}

	// Verify the request was made
	if len(mockHTTP.requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(mockHTTP.requests))
	}
	req := mockHTTP.requests[0]
	if req.Method != http.MethodGet {
		t.Errorf("Expected GET probe, got '%s'", req.Method)
	}
	if req.URL.String() != "https://openrouter.ai/api/v1/auth/key" {
		t.Errorf("Expected OpenRouter auth URL, got '%s'", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer sk-or-test" {
		t.Errorf("Expected OpenRouter bearer token, got '%s'", req.Header.Get("Authorization"))
	}
}

func TestAPIKeyService_Test_OpenRouter_InvalidKey(t *testing.T) {
	mockHTTP := &mockHTTPDoer{statusCode: 401}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	_, err := svc.Store(ctx, "openrouter", "invalid-key")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Test the key
	success, message, err := svc.Test(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Test() returned error: %v", err)
	}

	if success {
		t.Error("Expected success=false for 401 response")
	}
	if !strings.Contains(message, "Invalid") {
		t.Errorf("Expected message to contain 'Invalid', got: %s", message)
	}
}

func TestAPIKeyService_Test_OpenRouter_UnexpectedStatus(t *testing.T) {
	mockHTTP := &mockHTTPDoer{statusCode: http.StatusServiceUnavailable}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()
	if _, err := svc.Store(context.Background(), "openrouter", "sk-or-synthetic"); err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	success, message, err := svc.Test(context.Background(), "openrouter")
	if err != nil || success || message != "Unexpected status: 503" {
		t.Fatalf("Test() = (%v, %q, %v), want (false, unexpected status, nil)", success, message, err)
	}
}

func TestAPIKeyService_Test_OpenRouter_NetworkErrorDoesNotExposeKey(t *testing.T) {
	mockHTTP := &mockHTTPDoer{err: fmt.Errorf("synthetic network down")}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()
	const syntheticKey = "sk-or-synthetic-secret"
	if _, err := svc.Store(context.Background(), "openrouter", syntheticKey); err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	success, message, err := svc.Test(context.Background(), "openrouter")
	if err != nil || success || !strings.Contains(message, "Connection error: synthetic network down") {
		t.Fatalf("Test() = (%v, %q, %v), want connection error, nil", success, message, err)
	}
	if strings.Contains(message, syntheticKey) {
		t.Fatalf("Test() exposed the synthetic API key: %q", message)
	}
}

func TestAPIKeyService_Test_NoKeyConfigured_ReturnsMessage(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Test without storing a key
	success, message, err := svc.Test(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Test() returned error: %v", err)
	}

	if success {
		t.Error("Expected success=false when no key configured")
	}
	if !strings.Contains(message, "No API key") {
		t.Errorf("Expected message about no key configured, got: %s", message)
	}
}

func TestAPIKeyService_Test_UpdatesLastVerified_OnSuccess(t *testing.T) {
	mockHTTP := &mockHTTPDoer{statusCode: 200}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	_, err := svc.Store(ctx, "openrouter", "sk-or-test")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Test the key
	success, _, err := svc.Test(ctx, "openrouter")
	if err != nil || !success {
		t.Fatalf("Test() failed: success=%v, err=%v", success, err)
	}

	// Verify last_verified_at was updated
	var lastVerified sql.NullTime
	err = db.QueryRow("SELECT last_verified_at FROM api_keys WHERE provider = ?", "openrouter").Scan(&lastVerified)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if !lastVerified.Valid {
		t.Error("Expected last_verified_at to be set after successful test")
	}
	// Verify it's recent (within last minute)
	if time.Since(lastVerified.Time) > time.Minute {
		t.Error("last_verified_at is not recent")
	}
}

// ============================================================================
// No Encryption Key Tests
// ============================================================================

func TestAPIKeyService_NoEncryptionKey_StoresPlaintext(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()
	plainKey := "sk-or-test-plaintext"

	// Store a key without encryption
	_, err := svc.Store(ctx, "openrouter", plainKey)
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Verify the key is stored as-is (no encryption in dev mode)
	var storedKey string
	err = db.QueryRow("SELECT encrypted_key FROM api_keys WHERE provider = ?", "openrouter").Scan(&storedKey)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// In dev mode (no encryption key), the key should be stored as plaintext
	if storedKey != plainKey {
		t.Errorf("Expected plaintext key '%s', got '%s'", plainKey, storedKey)
	}
}

// ============================================================================
// Production Environment Tests
// ============================================================================

func TestAPIKeyService_ProductionMode_RequiresEncryptionKey(t *testing.T) {
	// The package test harness enables fallback credentials for ordinary unit
	// tests; this test explicitly injects an unconfigured authority result and
	// verifies the production refusal path without depending on host state.
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "")

	db := createTestAPIKeysDB(t)
	defer db.Close()

	// Attempting to create the service without an authority value in
	// production should fail.
	_, err := administration.NewAPIKeyServiceWithCredentialResolver(
		db, nil, "sqlite",
		func(string) (string, error) {
			return "", fmt.Errorf("%w: api-key-encryption-key", credentialauthority.ErrUnconfigured)
		},
		func() bool { return true }, nil, nil,
	)
	if err == nil {
		t.Error("Expected error when creating API key service in production without encryption key")
	}

	// Verify the error message is helpful
	if !strings.Contains(err.Error(), "LPBS_API_KEY_ENCRYPTION_KEY") {
		t.Errorf("Error message should mention LPBS_API_KEY_ENCRYPTION_KEY, got: %v", err)
	}
	if !strings.Contains(err.Error(), "production") {
		t.Errorf("Error message should mention production, got: %v", err)
	}
}

func TestAPIKeyService_ControlPlaneProductionOverridesLocalDevelopment(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "production")
	t.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", "")
	// Keep credential resolution hermetic; production must still refuse the
	// unconfigured result when the unit harness normally permits fallback.
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "1")

	db := createTestAPIKeysDB(t)
	defer db.Close()
	_, err := NewAPIKeyServiceWithOptions(db, nil, "sqlite")
	if err == nil || !strings.Contains(err.Error(), "required in production") {
		t.Fatalf("NewAPIKeyServiceWithOptions() error = %v, want production encryption refusal", err)
	}
}

func TestAPIKeyService_ProductionMode_WithEncryptionKey_Succeeds(t *testing.T) {
	// Isolate both environment sources so the control-plane setting cannot leak in.
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	testKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", testKey)

	db := createTestAPIKeysDB(t)
	defer db.Close()

	// Should succeed with encryption key
	svc, err := NewAPIKeyServiceWithOptions(db, nil, "sqlite")
	if err != nil {
		t.Fatalf("Expected success with encryption key in production, got error: %v", err)
	}
	if svc == nil {
		t.Error("Expected non-nil service")
	}
}

func TestAPIKeyService_MigrateEncryptionResealsLegacyRows(t *testing.T) {
	db := createTestAPIKeysDB(t)
	defer db.Close()
	legacyKey := make([]byte, 32)
	for i := range legacyKey {
		legacyKey[i] = byte(i + 1)
	}
	legacyCiphertext, err := securevalue.Encrypt(legacyKey, "legacy-api-key")
	if err != nil {
		t.Fatalf("encrypt legacy key: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO api_keys (provider, encrypted_key, key_hint) VALUES ('openrouter', ?, '****key')`, legacyCiphertext); err != nil {
		t.Fatalf("insert legacy key: %v", err)
	}
	ring := securevalue.Ring{Active: 1, Keys: []securevalue.VersionedKey{{Version: 1, Key: base64.StdEncoding.EncodeToString(legacyKey)}}}
	ring, err = ring.Rotate()
	if err != nil {
		t.Fatalf("rotate ring: %v", err)
	}
	serialized, err := ring.Marshal()
	if err != nil {
		t.Fatalf("marshal ring: %v", err)
	}
	svc, err := administration.NewAPIKeyServiceWithRuntime(db, nil, "sqlite", func(string) string { return serialized }, func() bool { return false }, nil, nil)
	if err != nil {
		t.Fatalf("construct ring service: %v", err)
	}
	count, err := svc.MigrateEncryption(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("migration count=%d err=%v, want 1", count, err)
	}
	got, err := svc.Get(context.Background(), "openrouter")
	if err != nil || got != "legacy-api-key" {
		t.Fatalf("migrated key=%q err=%v", got, err)
	}
	var state string
	if err := db.QueryRow(`SELECT encryption_state FROM api_keys WHERE provider = 'openrouter'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "v2" {
		t.Fatalf("encryption state=%q, want v2", state)
	}
}

func TestAPIKeyService_PropagatesCredentialProviderFailure(t *testing.T) {
	db := createTestAPIKeysDB(t)
	defer db.Close()
	providerErr := fmt.Errorf("provider unavailable: %w", credentialauthority.ErrProviderUnavailable)

	_, err := administration.NewAPIKeyServiceWithCredentialResolver(
		db, nil, "sqlite",
		func(string) (string, error) { return "", providerErr },
		func() bool { return false }, nil, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "resolve API key encryption credential") {
		t.Fatalf("expected provider failure to propagate, got %v", err)
	}
}

func TestAPIKeyService_DevelopmentMode_WithoutEncryptionKey_Succeeds(t *testing.T) {
	// Keep this test independent of the host authority store.
	t.Setenv("LPBS_TEST_CREDENTIAL_FALLBACK", "")

	db := createTestAPIKeysDB(t)
	defer db.Close()

	// Should succeed in development without an encryption key.
	svc, err := administration.NewAPIKeyServiceWithCredentialResolver(
		db, nil, "sqlite",
		func(string) (string, error) {
			return "", fmt.Errorf("%w: api-key-encryption-key", credentialauthority.ErrUnconfigured)
		},
		func() bool { return false }, nil, nil,
	)
	if err != nil {
		t.Fatalf("Expected success in development without encryption key, got error: %v", err)
	}
	if svc == nil {
		t.Error("Expected non-nil service")
	}
}

// ============================================================================
// Handler Tests
// ============================================================================

func TestHandleListAPIKeys_Empty(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	handler := adminhttp.ListAPIKeys(apiKeyHandlerDependencies(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	keys, ok := resp["keys"].([]interface{})
	if !ok {
		t.Fatal("Expected 'keys' to be an array")
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

func TestHandleListAPIKeys_MultipleKeys(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()
	_, _ = svc.Store(ctx, "openrouter", "sk-or-test")
	_, _ = svc.Store(ctx, "openai", "sk-openai-test")
	_, _ = svc.Store(ctx, "anthropic", "sk-ant-test")

	handler := adminhttp.ListAPIKeys(apiKeyHandlerDependencies(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	keys, ok := resp["keys"].([]interface{})
	if !ok {
		t.Fatal("Expected 'keys' to be an array")
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
}

func TestHandleCreateAPIKey_Success(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	handler := adminhttp.CreateAPIKey(apiKeyHandlerDependencies(svc))

	body := `{"provider": "openai", "key": "test-openai-key-12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp administration.APIKey
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", resp.Provider)
	}
	if resp.KeyHint != "****5678" {
		t.Errorf("Expected key hint '****5678', got '%s'", resp.KeyHint)
	}
}

func TestHandleCreateAPIKey_RejectsSharedOpenRouterCredential(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	handler := adminhttp.CreateAPIKey(apiKeyHandlerDependencies(svc))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(`{"provider":"openrouter","key":"sk-or-managed"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusConflict, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "shared credential authority") {
		t.Fatalf("response = %q, want shared-authority guidance", w.Body.String())
	}
}

func TestHandleCreateAPIKey_RejectsInvalidRequests(t *testing.T) {
	cases := []struct{ name, body, wantError string }{
		{name: "unknown provider", body: `{"provider":"invalid-provider","key":"some-key"}`, wantError: "invalid provider"},
		{name: "empty key", body: `{"provider":"openai","key":""}`, wantError: "provider and key are required"},
		{name: "malformed JSON", body: "{invalid", wantError: "Invalid request body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			handler := adminhttp.CreateAPIKey(apiKeyHandlerDependencies(svc))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler(w, req)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), tc.wantError) {
				t.Fatalf("response = %d %q, want HTTP 400 containing %q", w.Code, w.Body.String(), tc.wantError)
			}
		})
	}
}

func TestHandleDeleteAPIKey_Success(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()
	_, _ = svc.Store(ctx, "openrouter", "sk-or-test")

	handler := adminhttp.DeleteAPIKey(apiKeyHandlerDependencies(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/api-keys?provider=openrouter", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify key is deleted
	key, _ := svc.Get(ctx, "openrouter")
	if key != "" {
		t.Error("Expected key to be deleted")
	}
}

func TestHandleDeleteAPIKey_RejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name, path, wantError string
		wantStatus            int
	}{
		{name: "missing provider", path: "/api/v1/api-keys", wantStatus: http.StatusBadRequest, wantError: "Provider is required"},
		{name: "unknown provider", path: "/api/v1/api-keys?provider=nonexistent", wantStatus: http.StatusNotFound, wantError: "api key not found: nonexistent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, nil)
			defer db.Close()
			handler := adminhttp.DeleteAPIKey(apiKeyHandlerDependencies(svc))
			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			w := httptest.NewRecorder()
			handler(w, req)
			if w.Code != tc.wantStatus || !strings.Contains(w.Body.String(), tc.wantError) {
				t.Fatalf("response = %d %q, want HTTP %d containing %q", w.Code, w.Body.String(), tc.wantStatus, tc.wantError)
			}
		})
	}
}

func TestHandleTestAPIKey_ReportsProviderResult(t *testing.T) {
	cases := []struct {
		name, key, wantMessage string
		upstreamStatus         int
		wantSuccess            bool
	}{
		{name: "valid key", key: "sk-or-test", upstreamStatus: http.StatusOK, wantSuccess: true, wantMessage: "valid"},
		{name: "invalid key", key: "invalid-key", upstreamStatus: http.StatusUnauthorized, wantMessage: "Invalid"},
		{name: "no key configured", wantMessage: "No API key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockHTTP := &mockHTTPDoer{statusCode: tc.upstreamStatus}
			svc, db := createTestAPIKeyService(t, mockHTTP)
			defer db.Close()
			if tc.key != "" {
				if _, err := svc.Store(context.Background(), "openrouter", tc.key); err != nil {
					t.Fatalf("store test key: %v", err)
				}
			}
			handler := adminhttp.TestAPIKey(apiKeyHandlerDependencies(svc))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/test?provider=openrouter", nil)
			w := httptest.NewRecorder()
			handler(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
			}
			var response struct {
				Success  bool   `json:"success"`
				Message  string `json:"message"`
				Provider string `json:"provider"`
			}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Success != tc.wantSuccess || response.Provider != "openrouter" || !strings.Contains(response.Message, tc.wantMessage) {
				t.Fatalf("response = %+v, want success=%t provider=openrouter message containing %q", response, tc.wantSuccess, tc.wantMessage)
			}
			wantRequests := 1
			if tc.key == "" {
				wantRequests = 0
			}
			if len(mockHTTP.requests) != wantRequests {
				t.Fatalf("upstream requests = %d, want %d", len(mockHTTP.requests), wantRequests)
			}
		})
	}
}

func TestHandleTestAPIKey_MissingProvider(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	handler := adminhttp.TestAPIKey(apiKeyHandlerDependencies(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing provider, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleToggleAPIKey_Success(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()
	_, _ = svc.Store(ctx, "openrouter", "sk-or-test")

	handler := adminhttp.ToggleAPIKey(apiKeyHandlerDependencies(svc))

	// Toggle off
	body := `{"provider": "openrouter", "active": false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/api-keys/toggle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNoContent, w.Code, w.Body.String())
	}

	// Verify key is inactive (Get returns empty for inactive keys)
	key, _ := svc.Get(ctx, "openrouter")
	if key != "" {
		t.Error("Expected inactive key to return empty from Get")
	}
}

func TestHandleToggleAPIKey_InvalidJSON(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	handler := adminhttp.ToggleAPIKey(apiKeyHandlerDependencies(svc))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/api-keys/toggle", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

// ============================================================================
// Provider Test Function Tests
// ============================================================================

func TestTestOpenAI_ProviderResponses(t *testing.T) {
	cases := []struct {
		name, key, wantMessage string
		statusCode             int
		requestError           error
		wantSuccess            bool
	}{
		{name: "success", key: "sk-openai-test", statusCode: http.StatusOK, wantSuccess: true, wantMessage: "valid"},
		{name: "invalid key", key: "invalid-key", statusCode: http.StatusUnauthorized, wantMessage: "Invalid"},
		{name: "connection error", key: "sk-test", requestError: fmt.Errorf("connection refused"), wantMessage: "Connection error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockHTTP := &mockHTTPDoer{statusCode: tc.statusCode, err: tc.requestError}
			svc, db := createTestAPIKeyService(t, mockHTTP)
			defer db.Close()
			success, message := svc.TestOpenAIProvider(context.Background(), tc.key)
			if success != tc.wantSuccess || !strings.Contains(message, tc.wantMessage) {
				t.Fatalf("TestOpenAIProvider() = %t, %q; want success=%t and message containing %q", success, message, tc.wantSuccess, tc.wantMessage)
			}
			if len(mockHTTP.requests) != 1 {
				t.Fatalf("requests = %d, want 1", len(mockHTTP.requests))
			}
			req := mockHTTP.requests[0]
			if req.URL.String() != "https://api.openai.com/v1/models" || req.Header.Get("Authorization") != "Bearer "+tc.key {
				t.Fatalf("request URL/auth = %q/%q, want OpenAI models and caller key", req.URL, req.Header.Get("Authorization"))
			}
		})
	}
}

func TestTestAnthropic_Success(t *testing.T) {
	mockHTTP := &mockHTTPDoer{statusCode: 200}
	svc, db := createTestAPIKeyService(t, mockHTTP)
	defer db.Close()

	ctx := context.Background()
	success, message := svc.TestAnthropicProvider(ctx, "sk-ant-test")

	if !success {
		t.Errorf("Expected success=true, got false with message: %s", message)
	}

	// Verify request was made correctly
	if len(mockHTTP.requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(mockHTTP.requests))
	}
	req := mockHTTP.requests[0]
	if req.URL.String() != "https://api.anthropic.com/v1/messages" {
		t.Errorf("Expected URL 'https://api.anthropic.com/v1/messages', got '%s'", req.URL.String())
	}
	if req.Header.Get("x-api-key") != "sk-ant-test" {
		t.Errorf("Expected x-api-key header 'sk-ant-test', got '%s'", req.Header.Get("x-api-key"))
	}
	if req.Header.Get("anthropic-version") != "2023-06-01" {
		t.Errorf("Expected anthropic-version header '2023-06-01', got '%s'", req.Header.Get("anthropic-version"))
	}
}

func TestTestAnthropic_AcceptsKeyDespiteValidationStatuses(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
	}{
		{name: "rate limited", statusCode: http.StatusTooManyRequests},
		{name: "validation error", statusCode: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := createTestAPIKeyService(t, &mockHTTPDoer{statusCode: tc.statusCode})
			defer db.Close()
			success, message := svc.TestAnthropicProvider(context.Background(), "sk-ant-test")
			if !success || message != "API key is valid (rate limited or validation error)" {
				t.Fatalf("TestAnthropicProvider() = %t, %q; want valid-key result for status %d", success, message, tc.statusCode)
			}
		})
	}
}

// ============================================================================
// GenerateEncryptionKey Tests
// ============================================================================

func TestGenerateEncryptionKey_Unique(t *testing.T) {
	key1 := GenerateEncryptionKey()
	key2 := GenerateEncryptionKey()

	if key1 == key2 {
		t.Error("Expected different keys on each call, got identical keys")
	}
}

func TestGenerateEncryptionKey_ValidBase64(t *testing.T) {
	key := GenerateEncryptionKey()

	// Should be valid base64
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("Generated key is not valid base64: %v", err)
	}

	// Should decode to exactly 32 bytes
	if len(decoded) != 32 {
		t.Errorf("Expected decoded key to be 32 bytes, got %d", len(decoded))
	}
}
