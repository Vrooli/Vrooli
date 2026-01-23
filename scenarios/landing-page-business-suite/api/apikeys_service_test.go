package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// mockHTTPDoer is a mock HTTP client for testing.
type mockHTTPDoer struct {
	statusCode int
	body       string
	err        error
	requests   []*http.Request
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
func createTestAPIKeyService(t *testing.T, httpClient HTTPDoer) (*APIKeyService, *sql.DB) {
	t.Helper()

	db := createTestAPIKeysDB(t)
	if httpClient == nil {
		httpClient = &mockHTTPDoer{statusCode: 200}
	}

	// Service without encryption key (development mode) with SQLite dialect
	svc := &APIKeyService{
		db:            db,
		encryptionKey: nil,
		httpClient:    httpClient,
		dialects:      NewDialectHelper("sqlite"),
	}

	return svc, db
}

// createTestAPIKeyServiceWithEncryption creates an API key service with encryption enabled.
func createTestAPIKeyServiceWithEncryption(t *testing.T, httpClient HTTPDoer) (*APIKeyService, *sql.DB) {
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

	svc := &APIKeyService{
		db:            db,
		encryptionKey: encryptionKey,
		httpClient:    httpClient,
		dialects:      NewDialectHelper("sqlite"),
	}

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
	plainKey := "sk-or-test-key-12345678"

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

func TestAPIKeyService_Store_InvalidProvider_ReturnsError(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Store(ctx, "invalid-provider", "some-key")
	if err == nil {
		t.Error("Expected error for invalid provider, got nil")
	}
}

func TestAPIKeyService_Store_EmptyKey_ReturnsError(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Store(ctx, "openrouter", "")
	if err == nil {
		t.Error("Expected error for empty key, got nil")
	}
}

// ============================================================================
// Get Tests
// ============================================================================

func TestAPIKeyService_Get_ReturnsDecryptedKey(t *testing.T) {
	svc, db := createTestAPIKeyServiceWithEncryption(t, nil)
	defer db.Close()

	ctx := context.Background()
	plainKey := "sk-or-test-key-12345678"

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

func TestAPIKeyService_Get_NonExistentProvider_ReturnsEmpty(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	key, err := svc.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if key != "" {
		t.Errorf("Expected empty string for nonexistent provider, got '%s'", key)
	}
}

func TestAPIKeyService_Get_InactiveKey_ReturnsEmpty(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store and deactivate a key
	_, err := svc.Store(ctx, "openrouter", "some-key")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}
	err = svc.SetActive(ctx, "openrouter", false)
	if err != nil {
		t.Fatalf("SetActive() returned error: %v", err)
	}

	// Try to get the inactive key
	key, err := svc.Get(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if key != "" {
		t.Errorf("Expected empty string for inactive key, got '%s'", key)
	}
}

// ============================================================================
// List Tests
// ============================================================================

func TestAPIKeyService_List_ReturnsAllKeys(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store multiple keys
	_, _ = svc.Store(ctx, "openrouter", "sk-or-test")
	_, _ = svc.Store(ctx, "openai", "sk-openai-test")
	_, _ = svc.Store(ctx, "anthropic", "sk-ant-test")

	// List all keys
	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify keys don't contain actual values
	for _, key := range keys {
		if !strings.HasPrefix(key.KeyHint, "****") {
			t.Errorf("Key hint doesn't start with ****: %s", key.KeyHint)
		}
	}
}

func TestAPIKeyService_List_EmptyTable_ReturnsEmptyOrNil(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	// In Go, returning nil for an empty slice is valid and common
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestAPIKeyService_Delete_RemovesKey(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	_, err := svc.Store(ctx, "openrouter", "some-key")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Delete the key
	err = svc.Delete(ctx, "openrouter")
	if err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// Verify it's gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE provider = ?", "openrouter").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count keys: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 keys after delete, got %d", count)
	}
}

func TestAPIKeyService_Delete_NonExistent_ReturnsError(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	err := svc.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for deleting nonexistent key, got nil")
	}
}

// ============================================================================
// SetActive Tests
// ============================================================================

func TestAPIKeyService_SetActive_TogglesStatus(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	// Store a key
	_, err := svc.Store(ctx, "openrouter", "some-key")
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}

	// Deactivate
	err = svc.SetActive(ctx, "openrouter", false)
	if err != nil {
		t.Fatalf("SetActive(false) returned error: %v", err)
	}

	// Verify deactivated
	var isActive bool
	err = db.QueryRow("SELECT is_active FROM api_keys WHERE provider = ?", "openrouter").Scan(&isActive)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if isActive {
		t.Error("Expected IsActive to be false after deactivation")
	}

	// Reactivate
	err = svc.SetActive(ctx, "openrouter", true)
	if err != nil {
		t.Fatalf("SetActive(true) returned error: %v", err)
	}

	// Verify reactivated
	err = db.QueryRow("SELECT is_active FROM api_keys WHERE provider = ?", "openrouter").Scan(&isActive)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if !isActive {
		t.Error("Expected IsActive to be true after reactivation")
	}
}

func TestAPIKeyService_SetActive_NonExistent_ReturnsError(t *testing.T) {
	svc, db := createTestAPIKeyService(t, nil)
	defer db.Close()

	ctx := context.Background()

	err := svc.SetActive(ctx, "nonexistent", true)
	if err == nil {
		t.Error("Expected error for nonexistent key, got nil")
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
		t.Errorf("Expected 1 request, got %d", len(mockHTTP.requests))
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
	// Save current environment
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldKey := os.Getenv("LPBS_API_KEY_ENCRYPTION_KEY")
	defer func() {
		os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		if oldKey != "" {
			os.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", oldKey)
		} else {
			os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")
		}
	}()

	// Set production mode without encryption key
	os.Setenv("LPBS_ENVIRONMENT", "production")
	os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")

	db := createTestAPIKeysDB(t)
	defer db.Close()

	// Attempting to create service without encryption key in production should fail
	_, err := NewAPIKeyServiceWithOptions(db, nil, "sqlite")
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

func TestAPIKeyService_ProductionMode_WithEncryptionKey_Succeeds(t *testing.T) {
	// Save current environment
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldKey := os.Getenv("LPBS_API_KEY_ENCRYPTION_KEY")
	defer func() {
		os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		if oldKey != "" {
			os.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", oldKey)
		} else {
			os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")
		}
	}()

	// Set production mode with valid encryption key
	os.Setenv("LPBS_ENVIRONMENT", "production")
	// Generate a valid 32-byte key encoded as base64
	testKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	os.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", testKey)

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

func TestAPIKeyService_DevelopmentMode_WithoutEncryptionKey_Succeeds(t *testing.T) {
	// Save current environment
	oldEnv := os.Getenv("LPBS_ENVIRONMENT")
	oldKey := os.Getenv("LPBS_API_KEY_ENCRYPTION_KEY")
	defer func() {
		if oldEnv != "" {
			os.Setenv("LPBS_ENVIRONMENT", oldEnv)
		} else {
			os.Unsetenv("LPBS_ENVIRONMENT")
		}
		if oldKey != "" {
			os.Setenv("LPBS_API_KEY_ENCRYPTION_KEY", oldKey)
		} else {
			os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")
		}
	}()

	// Set development mode (or empty, which defaults to dev)
	os.Setenv("LPBS_ENVIRONMENT", "development")
	os.Unsetenv("LPBS_API_KEY_ENCRYPTION_KEY")

	db := createTestAPIKeysDB(t)
	defer db.Close()

	// Should succeed in development without encryption key (with warning)
	svc, err := NewAPIKeyServiceWithOptions(db, nil, "sqlite")
	if err != nil {
		t.Fatalf("Expected success in development without encryption key, got error: %v", err)
	}
	if svc == nil {
		t.Error("Expected non-nil service")
	}
}
