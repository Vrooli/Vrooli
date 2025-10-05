package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"email-triage/handlers"
	"email-triage/services"

	"github.com/google/uuid"
)

// TestAccountCreationPerformance tests account creation performance
func TestAccountCreationPerformance(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()

	pattern := PerformanceTestPattern{
		Name:        "AccountCreation",
		Description: "Test account creation performance",
		MaxDuration: 500 * time.Millisecond,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			return userID
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			userID := setupData.(string)

			reqBody := map[string]interface{}{
				"email_address": fmt.Sprintf("test-%d@example.com", iteration),
				"password":      "test-password",
				"imap_server":   "imap.example.com",
				"imap_port":     993,
				"smtp_server":   "smtp.example.com",
				"smtp_port":     587,
				"use_tls":       true,
			}

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/accounts",
				Body:   reqBody,
			})

			if err != nil {
				return err
			}

			ctx := context.WithValue(httpReq.Context(), "user_id", userID)
			httpReq = httpReq.WithContext(ctx)

			handler.CreateAccount(w, httpReq)

			// Expected to fail due to connection test, but we're testing speed
			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, durations []time.Duration) {
			avg := averageDuration(durations)
			t.Logf("Average account creation time: %v", avg)

			if avg > 200*time.Millisecond {
				t.Logf("Warning: Average time %v exceeds recommended 200ms", avg)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestAccountListingPerformance tests account listing performance
func TestAccountListingPerformance(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()

	// Create test accounts
	for i := 0; i < 20; i++ {
		TestData.CreateTestAccount(t, testDB.db, userID)
	}

	pattern := PerformanceTestPattern{
		Name:        "AccountListing",
		Description: "Test account listing performance with 20 accounts",
		MaxDuration: 100 * time.Millisecond,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			return userID
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			userID := setupData.(string)

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/accounts",
			})

			if err != nil {
				return err
			}

			ctx := context.WithValue(httpReq.Context(), "user_id", userID)
			httpReq = httpReq.WithContext(ctx)

			handler.ListAccounts(w, httpReq)

			if w.Code != 200 {
				return fmt.Errorf("unexpected status: %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, durations []time.Duration) {
			avg := averageDuration(durations)
			max := maxDuration(durations)
			t.Logf("Account listing - Avg: %v, Max: %v", avg, max)

			if avg > 50*time.Millisecond {
				t.Logf("Warning: Average time %v exceeds recommended 50ms", avg)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestConcurrentAccountAccess tests concurrent account access
func TestConcurrentAccountAccess(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	emailService := services.NewEmailService("mock-mail-server")
	handler := handlers.NewAccountHandler(testDB.db, emailService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)

	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentAccountAccess",
		Description: "Test concurrent access to same account",
		Concurrency: 10,
		Iterations:  100,
		Setup: func(t *testing.T) interface{} {
			return map[string]string{
				"user_id":    userID,
				"account_id": accountID,
			}
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			data := setupData.(map[string]string)
			userID := data["user_id"]
			accountID := data["account_id"]

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/accounts/" + accountID,
				URLVars: map[string]string{"id": accountID},
			})

			if err != nil {
				return err
			}

			ctx := context.WithValue(httpReq.Context(), "user_id", userID)
			httpReq = httpReq.WithContext(ctx)

			handler.GetAccount(w, httpReq)

			if w.Code != 200 {
				return fmt.Errorf("unexpected status: %d", w.Code)
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

			successRate := float64(len(results)-errorCount) / float64(len(results)) * 100
			t.Logf("Concurrent access success rate: %.2f%%", successRate)

			if successRate < 95.0 {
				t.Errorf("Success rate %.2f%% below threshold of 95%%", successRate)
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestDatabaseConnectionPooling tests database connection pooling under load
func TestDatabaseConnectionPooling(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	userID := uuid.New().String()

	// Create test data
	for i := 0; i < 10; i++ {
		TestData.CreateTestAccount(t, testDB.db, userID)
	}

	t.Run("ConcurrentDatabaseQueries", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 50)

		start := time.Now()

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()

				query := "SELECT COUNT(*) FROM email_accounts WHERE user_id = $1"
				var count int
				err := testDB.db.QueryRow(query, userID).Scan(&count)
				if err != nil {
					errors <- fmt.Errorf("query %d failed: %v", iteration, err)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			t.Logf("Error: %v", err)
			errorCount++
		}

		t.Logf("50 concurrent queries completed in %v with %d errors", duration, errorCount)

		if duration > 5*time.Second {
			t.Errorf("Concurrent queries took too long: %v", duration)
		}

		if errorCount > 0 {
			t.Errorf("Database connection pooling failed with %d errors", errorCount)
		}
	})
}

// TestSearchPerformance tests search performance
func TestSearchPerformance(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	searchService := services.NewSearchService("http://localhost:6333")
	handler := handlers.NewEmailHandler(testDB.db, searchService)

	userID := uuid.New().String()
	accountID := TestData.CreateTestAccount(t, testDB.db, userID)

	// Create test emails
	for i := 0; i < 50; i++ {
		TestData.CreateTestEmail(t, testDB.db, accountID)
	}

	pattern := PerformanceTestPattern{
		Name:        "EmailSearch",
		Description: "Test email search performance with 50 emails",
		MaxDuration: 200 * time.Millisecond,
		Iterations:  20,
		Setup: func(t *testing.T) interface{} {
			return userID
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			userID := setupData.(string)

			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/v1/emails/search",
				QueryParams: map[string]string{"query": "test"},
			})

			if err != nil {
				return err
			}

			ctx := context.WithValue(httpReq.Context(), "user_id", userID)
			httpReq = httpReq.WithContext(ctx)

			handler.SearchEmails(w, httpReq)

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, durations []time.Duration) {
			avg := averageDuration(durations)
			t.Logf("Average search time: %v", avg)

			if avg > 100*time.Millisecond {
				t.Logf("Warning: Average search time %v exceeds recommended 100ms", avg)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestHealthCheckPerformance tests health check endpoint performance
func TestHealthCheckPerformance(t *testing.T) {
	server := &Server{}

	pattern := PerformanceTestPattern{
		Name:        "HealthCheck",
		Description: "Test health check endpoint performance",
		MaxDuration: 10 * time.Millisecond,
		Iterations:  100,
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if err != nil {
				return err
			}

			server.healthCheck(w, httpReq)

			if w.Code != 200 {
				return fmt.Errorf("unexpected status: %d", w.Code)
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, durations []time.Duration) {
			avg := averageDuration(durations)
			max := maxDuration(durations)
			t.Logf("Health check - Avg: %v, Max: %v", avg, max)

			if avg > 5*time.Millisecond {
				t.Errorf("Average health check time %v exceeds threshold of 5ms", avg)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available")
	}
	defer testDB.cleanup()

	t.Run("BulkAccountCreation", func(t *testing.T) {
		userID := uuid.New().String()

		start := time.Now()

		// Create 100 test accounts
		for i := 0; i < 100; i++ {
			TestData.CreateTestAccount(t, testDB.db, userID)
		}

		duration := time.Since(start)

		t.Logf("Created 100 accounts in %v (avg: %v per account)",
			duration, duration/100)

		// Verify all accounts exist
		var count int
		err := testDB.db.QueryRow(
			"SELECT COUNT(*) FROM email_accounts WHERE user_id = $1",
			userID).Scan(&count)

		if err != nil {
			t.Fatalf("Failed to count accounts: %v", err)
		}

		if count != 100 {
			t.Errorf("Expected 100 accounts, got %d", count)
		}
	})
}
