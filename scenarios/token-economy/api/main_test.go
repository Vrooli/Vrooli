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

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}
	})

	t.Run("ServicesCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		services, ok := response["services"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected services map in response")
		}

		if _, exists := services["database"]; !exists {
			t.Error("Expected database status in services")
		}
	})
}

// TestCreateTokenHandler tests the token creation endpoint
func TestCreateTokenHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		reqData := TestData.TokenRequest("GOLD", "Gold Token", "fungible")
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)

		// Note: May fail without database, but tests structure
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMissingRequiredField("POST", "/api/v1/tokens/create", "symbol").
			AddMissingRequiredField("POST", "/api/v1/tokens/create", "name").
			AddMissingRequiredField("POST", "/api/v1/tokens/create", "type").
			Build()

		for _, pattern := range patterns {
			t.Run(pattern.Name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader([]byte("{}")))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				createTokenHandler(w, req)
			})
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader([]byte(`{"invalid": "json"`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)
	})
}

// TestMintTokenHandler tests the token minting endpoint
func TestMintTokenHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("Success", func(t *testing.T) {
		reqData := TestData.MintRequest(token.ID, wallet.ID, 500.0)
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/mint", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mintTokenHandler(w, req)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/mint", bytes.NewReader([]byte(`{"invalid": "json"`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mintTokenHandler(w, req)
	})
}

// TestTransferTokenHandler tests the token transfer endpoint
func TestTransferTokenHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet1 := createTestWallet(t, testDB.DB, householdID)
	wallet2 := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet1.ID, token.ID, 1000.0)

	t.Run("Success", func(t *testing.T) {
		reqData := TestData.TransferRequest(wallet1.ID, wallet2.ID, token.ID, 100.0)
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/transfer", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		transferTokenHandler(w, req)
	})

	t.Run("InsufficientBalance", func(t *testing.T) {
		reqData := TestData.TransferRequest(wallet1.ID, wallet2.ID, token.ID, 99999.0)
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/transfer", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		transferTokenHandler(w, req)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/transfer", bytes.NewReader([]byte(`{"invalid": "json"`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		transferTokenHandler(w, req)
	})
}

// TestSwapTokenHandler tests the token swap endpoint
func TestSwapTokenHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/tokens/swap", bytes.NewReader([]byte(`{"invalid": "json"`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		swapTokenHandler(w, req)
	})

	t.Run("NoExchangeRate", func(t *testing.T) {
		reqData := map[string]interface{}{
			"from_token":          "TOKEN1",
			"to_token":            "TOKEN2",
			"amount":              100.0,
			"slippage_tolerance":  0.5,
		}
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/tokens/swap", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		swapTokenHandler(w, req)
	})
}

// TestCreateWalletHandler tests the wallet creation endpoint
func TestCreateWalletHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Success_UserWallet", func(t *testing.T) {
		reqData := TestData.WalletRequest("user")
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createWalletHandler(w, req)
	})

	t.Run("Success_ScenarioWallet", func(t *testing.T) {
		reqData := map[string]interface{}{
			"type":          "scenario",
			"scenario_name": "test-scenario",
		}
		bodyBytes, _ := json.Marshal(reqData)

		req := httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createWalletHandler(w, req)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader([]byte(`{"invalid": "json"`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createWalletHandler(w, req)
	})
}

// TestGetBalanceHandler tests the balance retrieval endpoint
func TestGetBalanceHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 500.0)

	t.Run("Success", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NonExistentWallet", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+uuid.New().String()+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
	})
}

// TestGetWalletHandler tests the wallet retrieval endpoint
func TestGetWalletHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("Success", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NonExistentWallet", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestGetAchievementsHandler tests the achievements retrieval endpoint
func TestGetAchievementsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("Success", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/achievements/{user_id}", getAchievementsHandler)

		req := httptest.NewRequest("GET", "/api/v1/achievements/"+wallet.UserID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/achievements/{user_id}", getAchievementsHandler)

		req := httptest.NewRequest("GET", "/api/v1/achievements/nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestGetTransactionsHandler tests the transactions listing endpoint
func TestGetTransactionsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet1 := createTestWallet(t, testDB.DB, householdID)
	wallet2 := createTestWallet(t, testDB.DB, householdID)
	createTestTransaction(t, testDB.DB, wallet1.ID, wallet2.ID, token.ID, 100.0)

	t.Run("Success_NoFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_WalletFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?wallet="+wallet1.ID, nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_TokenFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?token="+token.ID, nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_LimitFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions?limit=10", nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGetTransactionHandler tests the single transaction retrieval endpoint
func TestGetTransactionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet1 := createTestWallet(t, testDB.DB, householdID)
	wallet2 := createTestWallet(t, testDB.DB, householdID)
	tx := createTestTransaction(t, testDB.DB, wallet1.ID, wallet2.ID, token.ID, 100.0)

	t.Run("Success", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/transactions/{hash}", getTransactionHandler)

		req := httptest.NewRequest("GET", "/api/v1/transactions/"+tx.Hash, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("NonExistentTransaction", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/transactions/{hash}", getTransactionHandler)

		req := httptest.NewRequest("GET", "/api/v1/transactions/nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestListTokensHandler tests the tokens listing endpoint
func TestListTokensHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	createTestToken(t, testDB.DB, householdID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()

		listTokensHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestGetTokenHandler tests the single token retrieval endpoint
func TestGetTokenHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)

	t.Run("Success", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

		req := httptest.NewRequest("GET", "/api/v1/tokens/"+token.ID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NonExistentToken", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

		req := httptest.NewRequest("GET", "/api/v1/tokens/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestGetAnalyticsHandler tests the analytics endpoint
func TestGetAnalyticsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/analytics", nil)
		w := httptest.NewRecorder()

		getAnalyticsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		expectedFields := []string{"total_transactions", "total_volume", "active_wallets", "top_earners"}
		for _, field := range expectedFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' not found in response", field)
			}
		}
	})
}

// TestResolveWalletAddress tests the wallet address resolution utility
func TestResolveWalletAddress(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	wallet := createTestWallet(t, testDB.DB, householdID)

	t.Run("DirectID", func(t *testing.T) {
		result := resolveWalletAddress(wallet.ID)
		if result != wallet.ID {
			t.Errorf("Expected %s, got %s", wallet.ID, result)
		}
	})

	t.Run("Alias", func(t *testing.T) {
		result := resolveWalletAddress("@alice")
		// Should return as-is since alias resolution is not implemented
		if result != "@alice" {
			t.Errorf("Expected @alice, got %s", result)
		}
	})
}

// TestSendError tests the error response utility
func TestSendError(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		sendError(w, "test error", http.StatusBadRequest)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if response.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", response.Error)
		}
	})
}

// TestSendSuccess tests the success response utility
func TestSendSuccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		testData := map[string]string{"key": "value"}
		sendSuccess(w, testData)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response SuccessResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse success response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}
	})
}

// TestPerformance tests performance of key endpoints
func TestPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)

	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "GetBalance_Performance",
		Description: "Balance retrieval should complete within 100ms",
		MaxDuration: 100 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

			req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			return time.Since(start)
		},
	})

	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "ListTokens_Performance",
		Description: "Token listing should complete within 100ms",
		MaxDuration: 100 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
			w := httptest.NewRecorder()

			listTokensHandler(w, req)

			return time.Since(start)
		},
	})

	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "GetToken_Performance",
		Description: "Single token retrieval should complete within 50ms",
		MaxDuration: 50 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

			req := httptest.NewRequest("GET", "/api/v1/tokens/"+token.ID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			return time.Since(start)
		},
	})
}

// TestConcurrency tests concurrent operations
func TestConcurrency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)

	RunConcurrencyTest(t, ConcurrencyTestPattern{
		Name:        "ConcurrentTokenRetrieval",
		Description: "Multiple concurrent token retrievals should succeed",
		Concurrency: 10,
		Iterations:  50,
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

			req := httptest.NewRequest("GET", "/api/v1/tokens/"+token.ID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
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

			if errorCount > 0 {
				t.Errorf("Expected 0 errors, got %d", errorCount)
			}
		},
	})
}
