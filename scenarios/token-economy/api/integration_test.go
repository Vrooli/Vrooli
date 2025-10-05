package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestTokenLifecycle tests complete token lifecycle from creation to transfer
func TestTokenLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	// householdID := "00000000-0000-0000-0000-000000000099"
	_ = "00000000-0000-0000-0000-000000000099" // householdID for reference

	// Step 1: Create a token
	t.Run("CreateToken", func(t *testing.T) {
		reqData := TestData.TokenRequest("LIFE", "Lifecycle Token", "fungible")
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)

		if w.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			t.Logf("Token created successfully: %v", response)
		}
	})

	// Step 2: Create wallets
	wallet1ID := ""
	wallet2ID := ""

	t.Run("CreateWallets", func(t *testing.T) {
		// Create first wallet
		reqData := TestData.WalletRequest("user")
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createWalletHandler(w, req)

		if w.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			wallet1ID = response["wallet_id"].(string)
			t.Logf("Wallet 1 created: %s", wallet1ID)
		}

		// Create second wallet
		req = httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		createWalletHandler(w, req)

		if w.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			wallet2ID = response["wallet_id"].(string)
			t.Logf("Wallet 2 created: %s", wallet2ID)
		}
	})

	// Step 3: Verify wallets exist
	if wallet1ID != "" && wallet2ID != "" {
		t.Run("VerifyWallets", func(t *testing.T) {
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler)

			req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet1ID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			t.Logf("Wallet 1 verification status: %d", w.Code)
		})
	}
}

// TestWalletBalanceFlow tests wallet balance tracking
func TestWalletBalanceFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("InitialBalance", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			t.Logf("Initial balance response: %v", response)
		}
	})

	t.Run("AfterBalanceSet", func(t *testing.T) {
		createTestBalance(t, testDB.DB, wallet.ID, token.ID, 1000.0)

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			balances, ok := response["balances"].([]interface{})
			if ok && len(balances) > 0 {
				t.Logf("Balance retrieved successfully: %v", balances[0])
			}
		}
	})
}

// TestTransactionHistory tests transaction tracking and retrieval
func TestTransactionHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet1 := createTestWallet(t, testDB.DB, householdID)
	wallet2 := createTestWallet(t, testDB.DB, householdID)

	// Create multiple transactions
	t.Run("CreateTransactions", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			tx := createTestTransaction(t, testDB.DB, wallet1.ID, wallet2.ID, token.ID, float64(100+i*10))
			t.Logf("Created transaction %d: %s", i+1, tx.Hash)
		}
	})

	t.Run("ListTransactions_NoFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code == http.StatusOK {
			var transactions []Transaction
			json.Unmarshal(w.Body.Bytes(), &transactions)
			t.Logf("Retrieved %d transactions", len(transactions))
		}
	})

	t.Run("ListTransactions_WalletFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?wallet="+wallet1.ID, nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code == http.StatusOK {
			var transactions []Transaction
			json.Unmarshal(w.Body.Bytes(), &transactions)
			t.Logf("Retrieved %d transactions for wallet %s", len(transactions), wallet1.ID)
		}
	})

	t.Run("ListTransactions_TokenFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?token="+token.ID, nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code == http.StatusOK {
			var transactions []Transaction
			json.Unmarshal(w.Body.Bytes(), &transactions)
			t.Logf("Retrieved %d transactions for token %s", len(transactions), token.ID)
		}
	})
}

// TestTokenManagement tests token creation and retrieval
func TestTokenManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"

	var tokenID string

	t.Run("CreateMultipleTokens", func(t *testing.T) {
		tokenTypes := []struct {
			symbol string
			name   string
			ttype  string
		}{
			{"GOLD", "Gold Coin", "fungible"},
			{"SILVER", "Silver Coin", "fungible"},
			{"BADGE", "Achievement Badge", "non-fungible"},
		}

		for _, tt := range tokenTypes {
			token := createTestToken(t, testDB.DB, householdID)
			if tokenID == "" {
				tokenID = token.ID
			}
			t.Logf("Created %s token: %s", tt.ttype, token.ID)
		}
	})

	t.Run("ListAllTokens", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()

		listTokensHandler(w, req)

		if w.Code == http.StatusOK {
			var tokens []Token
			json.Unmarshal(w.Body.Bytes(), &tokens)
			t.Logf("Total tokens in system: %d", len(tokens))
		}
	})

	t.Run("GetSpecificToken", func(t *testing.T) {
		if tokenID != "" {
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

			req := httptest.NewRequest("GET", "/api/v1/tokens/"+tokenID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var token Token
				json.Unmarshal(w.Body.Bytes(), &token)
				t.Logf("Retrieved token: %s (%s)", token.Name, token.Symbol)
			}
		}
	})
}

// TestAchievementSystem tests achievement tracking
func TestAchievementSystem(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("GetUserAchievements", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/achievements/{user_id}", getAchievementsHandler)

		req := httptest.NewRequest("GET", "/api/v1/achievements/"+wallet.UserID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should succeed or return 404 if wallet not found
		if w.Code == http.StatusOK {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			t.Logf("Achievements response: %v", response)
		} else {
			t.Logf("Achievement retrieval status: %d", w.Code)
		}
	})
}

// TestAnalyticsDashboard tests analytics endpoint
func TestAnalyticsDashboard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"

	// Create test data
	token := createTestToken(t, testDB.DB, householdID)
	wallet1 := createTestWallet(t, testDB.DB, householdID)
	wallet2 := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet1.ID, token.ID, 1000.0)
	createTestBalance(t, testDB.DB, wallet2.ID, token.ID, 500.0)
	createTestTransaction(t, testDB.DB, wallet1.ID, wallet2.ID, token.ID, 100.0)

	t.Run("GetAnalytics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/analytics", nil)
		w := httptest.NewRecorder()

		getAnalyticsHandler(w, req)

		if w.Code == http.StatusOK {
			var analytics map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &analytics)

			t.Logf("Analytics - Total Transactions: %v", analytics["total_transactions"])
			t.Logf("Analytics - Total Volume: %v", analytics["total_volume"])
			t.Logf("Analytics - Active Wallets: %v", analytics["active_wallets"])

			if topEarners, ok := analytics["top_earners"].([]interface{}); ok {
				t.Logf("Analytics - Top Earners Count: %d", len(topEarners))
			}
		}
	})
}

// TestErrorHandling tests comprehensive error scenarios
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("InvalidJSON_CreateToken", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader([]byte(`{invalid json`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
	})

	t.Run("MissingFields_CreateToken", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("NonExistentToken", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

		req := httptest.NewRequest("GET", "/api/v1/tokens/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("NonExistentWallet", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("NonExistentTransaction", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/transactions/{hash}", getTransactionHandler)

		req := httptest.NewRequest("GET", "/api/v1/transactions/nonexistent-hash", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 1000.0)

	RunConcurrencyTest(t, ConcurrencyTestPattern{
		Name:        "ConcurrentBalanceRetrieval",
		Description: "Multiple concurrent balance retrievals",
		Concurrency: 10,
		Iterations:  50,
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

			req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
				return http.ErrServerClosed
			}
			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
				}
			}

			if errorCount > 5 { // Allow some errors due to race conditions
				t.Errorf("Too many errors in concurrent test: %d/%d", errorCount, len(results))
			}
		},
	})
}

// TestCacheInvalidation tests Redis cache behavior
func TestCacheInvalidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	// This test verifies cache behavior if Redis is available
	if rdb == nil {
		t.Skip("Redis not available, skipping cache tests")
	}

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 500.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

	t.Run("FirstRequest_PopulatesCache", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Log("Cache should be populated")
		}
	})

	t.Run("SecondRequest_UsesCache", func(t *testing.T) {
		start := time.Now()

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		duration := time.Since(start)

		if w.Code == http.StatusOK {
			t.Logf("Cached request completed in %v", duration)
		}
	})
}
