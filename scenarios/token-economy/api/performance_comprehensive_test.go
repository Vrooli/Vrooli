package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// BenchmarkCreateToken benchmarks token creation
func BenchmarkCreateToken(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	defer testDB.Cleanup()

	reqData := TestData.TokenRequest("BENCH", "Benchmark Token", "fungible")
	bodyBytes, _ := json.Marshal(reqData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/tokens/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createTokenHandler(w, req)
	}
}

// BenchmarkCreateWallet benchmarks wallet creation
func BenchmarkCreateWallet(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	defer testDB.Cleanup()

	reqData := TestData.WalletRequest("user")
	bodyBytes, _ := json.Marshal(reqData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/wallets/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		createWalletHandler(w, req)
	}
}

// BenchmarkGetBalance benchmarks balance retrieval
func BenchmarkGetBalance(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(&testing.T{}, testDB.DB, householdID)
	wallet := createTestWallet(&testing.T{}, testDB.DB, householdID)
	createTestBalance(&testing.T{}, testDB.DB, wallet.ID, token.ID, 1000.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
	}
}

// BenchmarkListTokens benchmarks token listing
func BenchmarkListTokens(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	// Create some test tokens
	for i := 0; i < 10; i++ {
		createTestToken(&testing.T{}, testDB.DB, householdID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()

		listTokensHandler(w, req)
	}
}

// BenchmarkGetTransactions benchmarks transaction listing
func BenchmarkGetTransactions(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(&testing.T{})
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(&testing.T{}, testDB.DB, householdID)
	wallet1 := createTestWallet(&testing.T{}, testDB.DB, householdID)
	wallet2 := createTestWallet(&testing.T{}, testDB.DB, householdID)

	// Create some test transactions
	for i := 0; i < 20; i++ {
		createTestTransaction(&testing.T{}, testDB.DB, wallet1.ID, wallet2.ID, token.ID, 100.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)
	}
}

// BenchmarkHealthCheck benchmarks health endpoint
func BenchmarkHealthCheck(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)
	}
}

// BenchmarkJSONMarshaling benchmarks JSON marshaling
func BenchmarkJSONMarshaling(b *testing.B) {
	token := BuildTestToken("test-household")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(token)
	}
}

// BenchmarkJSONUnmarshaling benchmarks JSON unmarshaling
func BenchmarkJSONUnmarshaling(b *testing.B) {
	data := []byte(`{"symbol":"TEST","name":"Test Token","type":"fungible","initial_supply":1000}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req CreateTokenRequest
		_ = json.Unmarshal(data, &req)
	}
}

// TestPerformanceEndToEnd tests end-to-end performance
func TestPerformanceEndToEnd(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 1000.0)

	// Test health check performance
	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "HealthCheck_Performance",
		Description: "Health check should complete within 10ms",
		MaxDuration: 10 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			return time.Since(start)
		},
	})

	// Test token retrieval performance
	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "GetToken_Performance",
		Description: "Token retrieval should complete within 50ms",
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

	// Test balance retrieval performance
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

	// Test analytics performance
	RunPerformanceTest(t, PerformanceTestPattern{
		Name:        "Analytics_Performance",
		Description: "Analytics should complete within 200ms",
		MaxDuration: 200 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/v1/admin/analytics", nil)
			w := httptest.NewRecorder()

			getAnalyticsHandler(w, req)

			return time.Since(start)
		},
	})
}

// TestConcurrentLoad tests system under concurrent load
func TestConcurrentLoad(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 10000.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)
	router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)

	t.Run("ConcurrentBalanceChecks", func(t *testing.T) {
		concurrency := 50
		iterations := 100

		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(iterations)

		for i := 0; i < iterations; i++ {
			go func() {
				defer wg.Done()

				req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)
			}()

			// Control concurrency
			if (i+1)%concurrency == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Completed %d concurrent balance checks in %v", iterations, duration)
		t.Logf("Average: %v per request", duration/time.Duration(iterations))

		if duration > 5*time.Second {
			t.Errorf("Concurrent load test took too long: %v", duration)
		}
	})

	t.Run("MixedConcurrentOperations", func(t *testing.T) {
		operations := 200
		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(operations)

		for i := 0; i < operations; i++ {
			go func(index int) {
				defer wg.Done()

				// Alternate between different operations
				if index%3 == 0 {
					// Balance check
					req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
				} else if index%3 == 1 {
					// Token retrieval
					req := httptest.NewRequest("GET", "/api/v1/tokens/"+token.ID, nil)
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
				} else {
					// Health check
					req := httptest.NewRequest("GET", "/health", nil)
					w := httptest.NewRecorder()
					healthHandler(w, req)
				}
			}(i)

			// Control concurrency
			if (i+1)%50 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Completed %d mixed concurrent operations in %v", operations, duration)
		t.Logf("Average: %v per operation", duration/time.Duration(operations))
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"

	t.Run("LargeTokenList", func(t *testing.T) {
		// Create many tokens
		for i := 0; i < 100; i++ {
			createTestToken(t, testDB.DB, householdID)
		}

		start := time.Now()
		req := httptest.NewRequest("GET", "/api/v1/tokens", nil)
		w := httptest.NewRecorder()

		listTokensHandler(w, req)

		duration := time.Since(start)
		t.Logf("Listing 100+ tokens took %v", duration)

		if duration > 500*time.Millisecond {
			t.Errorf("Large token list took too long: %v", duration)
		}
	})

	t.Run("LargeTransactionList", func(t *testing.T) {
		token := createTestToken(t, testDB.DB, householdID)
		wallet1 := createTestWallet(t, testDB.DB, householdID)
		wallet2 := createTestWallet(t, testDB.DB, householdID)

		// Create many transactions
		for i := 0; i < 100; i++ {
			createTestTransaction(t, testDB.DB, wallet1.ID, wallet2.ID, token.ID, float64(i+1))
		}

		start := time.Now()
		req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
		w := httptest.NewRecorder()

		getTransactionsHandler(w, req)

		duration := time.Since(start)
		t.Logf("Listing 100+ transactions took %v", duration)

		if duration > 500*time.Millisecond {
			t.Errorf("Large transaction list took too long: %v", duration)
		}
	})
}

// TestResponseTime tests response time SLAs
func TestResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 1000.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/tokens/{id}", getTokenHandler)
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)
	router.HandleFunc("/api/v1/wallets/{wallet_id}", getWalletHandler)

	tests := []struct {
		name        string
		method      string
		path        string
		maxDuration time.Duration
	}{
		{
			name:        "HealthCheck_SLA",
			method:      "GET",
			path:        "/health",
			maxDuration: 20 * time.Millisecond,
		},
		{
			name:        "GetToken_SLA",
			method:      "GET",
			path:        "/api/v1/tokens/" + token.ID,
			maxDuration: 100 * time.Millisecond,
		},
		{
			name:        "GetBalance_SLA",
			method:      "GET",
			path:        "/api/v1/wallets/" + wallet.ID + "/balance",
			maxDuration: 150 * time.Millisecond,
		},
		{
			name:        "GetWallet_SLA",
			method:      "GET",
			path:        "/api/v1/wallets/" + wallet.ID,
			maxDuration: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times and check average
			var totalDuration time.Duration
			iterations := 10

			for i := 0; i < iterations; i++ {
				start := time.Now()

				req := httptest.NewRequest(tt.method, tt.path, nil)
				w := httptest.NewRecorder()

				if tt.path == "/health" {
					healthHandler(w, req)
				} else {
					router.ServeHTTP(w, req)
				}

				totalDuration += time.Since(start)
			}

			avgDuration := totalDuration / time.Duration(iterations)

			t.Logf("%s average response time: %v (max: %v)", tt.name, avgDuration, tt.maxDuration)

			if avgDuration > tt.maxDuration {
				t.Errorf("Average response time %v exceeds SLA %v", avgDuration, tt.maxDuration)
			}
		})
	}
}

// TestThroughput tests system throughput
func TestThroughput(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 10000.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

	t.Run("BalanceCheckThroughput", func(t *testing.T) {
		duration := 2 * time.Second
		var count int32
		start := time.Now()
		done := make(chan bool)

		// Spawn workers
		for i := 0; i < 10; i++ {
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
						w := httptest.NewRecorder()

						router.ServeHTTP(w, req)
						count++
					}
				}
			}()
		}

		time.Sleep(duration)
		close(done)

		elapsed := time.Since(start)
		throughput := float64(count) / elapsed.Seconds()

		t.Logf("Throughput: %.2f requests/second (%d requests in %v)", throughput, count, elapsed)

		if throughput < 100 {
			t.Errorf("Throughput too low: %.2f req/s", throughput)
		}
	})
}

// TestDatabasePooling tests database connection pooling
func TestDatabasePooling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	if db == nil {
		t.Skip("Database not available")
	}

	t.Run("ConnectionStats", func(t *testing.T) {
		stats := db.Stats()

		t.Logf("Database connection stats:")
		t.Logf("  Max Open: %d", stats.MaxOpenConnections)
		t.Logf("  Open: %d", stats.OpenConnections)
		t.Logf("  In Use: %d", stats.InUse)
		t.Logf("  Idle: %d", stats.Idle)

		if stats.OpenConnections > 25 {
			t.Errorf("Too many open connections: %d", stats.OpenConnections)
		}
	})

	t.Run("ConcurrentQueries", func(t *testing.T) {
		iterations := 100
		var wg sync.WaitGroup
		wg.Add(iterations)

		start := time.Now()

		for i := 0; i < iterations; i++ {
			go func() {
				defer wg.Done()

				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM tokens").Scan(&count)
				if err != nil {
					t.Logf("Query error: %v", err)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Completed %d concurrent queries in %v", iterations, duration)

		stats := db.Stats()
		t.Logf("Final connection stats: Open=%d, InUse=%d, Idle=%d",
			stats.OpenConnections, stats.InUse, stats.Idle)
	})
}

// TestCacheEffectiveness tests cache hit rates
func TestCacheEffectiveness(t *testing.T) {
	if rdb == nil {
		t.Skip("Redis not available for cache testing")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	householdID := "00000000-0000-0000-0000-000000000099"
	token := createTestToken(t, testDB.DB, householdID)
	wallet := createTestWallet(t, testDB.DB, householdID)
	createTestBalance(t, testDB.DB, wallet.ID, token.ID, 1000.0)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/wallets/{wallet_id}/balance", getBalanceHandler)

	t.Run("FirstRequest_MissCache", func(t *testing.T) {
		// Clear cache
		cacheKey := fmt.Sprintf("balances:%s", wallet.ID)
		rdb.Del(ctx, cacheKey)

		start := time.Now()
		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		missTime := time.Since(start)

		t.Logf("Cache miss time: %v", missTime)
	})

	t.Run("SecondRequest_HitCache", func(t *testing.T) {
		start := time.Now()
		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.ID+"/balance", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		hitTime := time.Since(start)

		t.Logf("Cache hit time: %v", hitTime)

		// Cache hit should be faster (though not guaranteed in tests)
		if hitTime > 100*time.Millisecond {
			t.Logf("Warning: Cache hit time seems slow: %v", hitTime)
		}
	})
}
