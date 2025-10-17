package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(os.Stdout)
	log.SetPrefix("[test] ")
	return func() {
		log.SetOutput(originalOutput)
		log.SetPrefix("")
	}
}

// TestDB manages test database connection
type TestDB struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an isolated test database environment
func setupTestDB(t *testing.T) *TestDB {
	// Store original db connection
	originalDB := db

	// Create test database connection
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "token_economy"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open test database connection: %v", err)
	}

	// Try to ping - if fails, skip test requiring DB
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test requiring database: %v", err)
	}

	// Set global db to test db
	db = testDB

	// Clean up test data
	cleanupTestData(t, testDB)

	return &TestDB{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(t, testDB)
			testDB.Close()
			db = originalDB
		},
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, testDB *sql.DB) {
	tables := []string{"transactions", "balances", "achievements", "wallets", "tokens", "exchange_rates"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE household_id = '00000000-0000-0000-0000-000000000099'", table)
		if _, err := testDB.Exec(query); err != nil {
			// Try without household_id for tables that don't have it
			query = fmt.Sprintf("DELETE FROM %s WHERE id LIKE 'test_%%'", table)
			if _, err := testDB.Exec(query); err != nil {
				t.Logf("Warning: Failed to clean table %s: %v", table, err)
			}
		}
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return httptest.NewRecorder()
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response contains an array
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	bodyStr := w.Body.String()
	if expectedErrorMessage != "" && !strings.Contains(bodyStr, expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, bodyStr)
	}
}

// createTestToken creates a test token entry in the database
func createTestToken(t *testing.T, testDB *sql.DB, householdID string) *Token {
	token := &Token{
		ID:             uuid.New().String(),
		HouseholdID:    householdID,
		Symbol:         "TEST",
		Name:           "Test Token",
		Type:           "fungible",
		TotalSupply:    1000.0,
		MaxSupply:      10000.0,
		Decimals:       18,
		CreatorScenario: "token-economy",
		Metadata:       map[string]interface{}{"test": true},
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	metadataJSON, _ := json.Marshal(token.Metadata)

	query := `INSERT INTO tokens (id, household_id, symbol, name, type, total_supply, max_supply,
			  decimals, creator_scenario, metadata, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`

	_, err := testDB.Exec(query, token.ID, token.HouseholdID, token.Symbol, token.Name, token.Type,
		token.TotalSupply, token.MaxSupply, token.Decimals, token.CreatorScenario, metadataJSON)

	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}

	return token
}

// createTestWallet creates a test wallet entry in the database
func createTestWallet(t *testing.T, testDB *sql.DB, householdID string) *Wallet {
	wallet := &Wallet{
		ID:          uuid.New().String(),
		HouseholdID: householdID,
		UserID:      "test_user_" + uuid.New().String()[:8],
		Address:     "0x" + uuid.New().String()[:32],
		Type:        "user",
		Metadata:    map[string]interface{}{"test": true},
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	metadataJSON, _ := json.Marshal(wallet.Metadata)

	query := `INSERT INTO wallets (id, household_id, user_id, address, type, metadata, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	_, err := testDB.Exec(query, wallet.ID, wallet.HouseholdID, wallet.UserID,
		wallet.Address, wallet.Type, metadataJSON)

	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}

	return wallet
}

// createTestBalance creates a test balance entry in the database
func createTestBalance(t *testing.T, testDB *sql.DB, walletID, tokenID string, amount float64) {
	query := `INSERT INTO balances (wallet_id, token_id, amount, locked_amount)
			  VALUES ($1, $2, $3, 0)
			  ON CONFLICT (wallet_id, token_id)
			  DO UPDATE SET amount = $3`

	_, err := testDB.Exec(query, walletID, tokenID, amount)
	if err != nil {
		t.Fatalf("Failed to create test balance: %v", err)
	}
}

// createTestTransaction creates a test transaction entry in the database
func createTestTransaction(t *testing.T, testDB *sql.DB, fromWallet, toWallet, tokenID string, amount float64) *Transaction {
	tx := &Transaction{
		ID:         uuid.New().String(),
		Hash:       uuid.New().String(),
		FromWallet: fromWallet,
		ToWallet:   toWallet,
		TokenID:    tokenID,
		Amount:     amount,
		Type:       "transfer",
		Status:     "confirmed",
		Metadata:   map[string]interface{}{"test": true},
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	metadataJSON, _ := json.Marshal(tx.Metadata)

	query := `INSERT INTO transactions (id, hash, from_wallet, to_wallet, token_id, amount, type, status, metadata, created_at, household_id)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), '00000000-0000-0000-0000-000000000099')`

	_, err := testDB.Exec(query, tx.ID, tx.Hash, tx.FromWallet, tx.ToWallet, tx.TokenID,
		tx.Amount, tx.Type, tx.Status, metadataJSON)

	if err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}

	return tx
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// TokenRequest creates a test token creation request
func (g *TestDataGenerator) TokenRequest(symbol, name, tokenType string) map[string]interface{} {
	return map[string]interface{}{
		"symbol":         symbol,
		"name":           name,
		"type":           tokenType,
		"initial_supply": 1000.0,
		"max_supply":     10000.0,
		"metadata":       map[string]interface{}{"test": true},
	}
}

// WalletRequest creates a test wallet creation request
func (g *TestDataGenerator) WalletRequest(walletType string) map[string]interface{} {
	return map[string]interface{}{
		"type":    walletType,
		"user_id": "test_user_" + uuid.New().String()[:8],
	}
}

// MintRequest creates a test mint request
func (g *TestDataGenerator) MintRequest(tokenID, toWallet string, amount float64) map[string]interface{} {
	return map[string]interface{}{
		"token_id":  tokenID,
		"to_wallet": toWallet,
		"amount":    amount,
		"metadata":  map[string]interface{}{"test": true},
	}
}

// TransferRequest creates a test transfer request
func (g *TestDataGenerator) TransferRequest(fromWallet, toWallet, tokenID string, amount float64) map[string]interface{} {
	return map[string]interface{}{
		"from_wallet": fromWallet,
		"to_wallet":   toWallet,
		"token_id":    tokenID,
		"amount":      amount,
		"memo":        "Test transfer",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
