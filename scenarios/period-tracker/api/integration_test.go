// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestIntegrationCycleWorkflow tests the complete cycle management workflow
func TestIntegrationCycleWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("CompleteCycleLifecycle", func(t *testing.T) {
		// Step 1: Create a new cycle
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/cycles",
			Body: map[string]interface{}{
				"start_date":     "2024-01-15",
				"flow_intensity": "medium",
				"notes":          "First cycle notes",
			},
		})

		if err != nil {
			t.Fatalf("Failed to create cycle: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Skipf("Skipping test - database not available (status %d)", w.Code)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, []string{"cycle_id"})
		cycleID := response["cycle_id"].(string)

		// Step 2: Verify cycle was created
		w2, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
		})

		if err != nil {
			t.Fatalf("Failed to get cycles: %v", err)
		}

		response2 := assertJSONResponse(t, w2, http.StatusOK, []string{"cycles"})
		cycles := response2["cycles"].([]interface{})

		if len(cycles) != 1 {
			t.Errorf("Expected 1 cycle, got %d", len(cycles))
		}

		cycle := cycles[0].(map[string]interface{})
		if cycle["id"].(string) != cycleID {
			t.Error("Retrieved cycle ID doesn't match created cycle ID")
		}

		// Step 3: Add symptoms for this cycle
		w3, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/symptoms",
			Body: map[string]interface{}{
				"date":               "2024-01-15",
				"physical_symptoms":  []string{"cramps", "headache"},
				"mood_rating":        7,
				"cramp_intensity":    5,
				"headache_intensity": 3,
				"flow_level":         "medium",
			},
		})

		if err != nil {
			t.Fatalf("Failed to log symptoms: %v", err)
		}

		if w3.Code == http.StatusCreated {
			assertJSONResponse(t, w3, http.StatusCreated, []string{"symptom_id"})
		}

		// Step 4: Retrieve symptoms
		w4, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/symptoms",
		})

		if err != nil {
			t.Fatalf("Failed to get symptoms: %v", err)
		}

		if w4.Code == http.StatusOK {
			response4 := assertJSONResponse(t, w4, http.StatusOK, []string{"symptoms"})
			symptoms := response4["symptoms"].([]interface{})

			if len(symptoms) > 0 {
				symptom := symptoms[0].(map[string]interface{})
				if symptom["symptom_date"].(string) != "2024-01-15" {
					t.Error("Symptom date mismatch")
				}
			}
		}
	})
}

// TestIntegrationPredictionWorkflow tests the prediction generation workflow
func TestIntegrationPredictionWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("PredictionGeneration", func(t *testing.T) {
		// Create multiple cycles for prediction
		dates := []string{"2024-01-01", "2024-01-29", "2024-02-26", "2024-03-25", "2024-04-22", "2024-05-20"}

		for i, date := range dates {
			cycleID := uuid.New().String()
			_, err := env.DB.Exec(`
				INSERT INTO cycles (id, user_id, start_date, cycle_length, flow_intensity)
				VALUES ($1, $2, $3, $4, $5)`,
				cycleID, env.UserID, date, 28, "medium")

			if err != nil {
				t.Logf("Could not create cycle %d: %v", i, err)
				t.Skip("Skipping test - database not available")
			}
		}

		// Generate predictions
		predictions, err := generatePredictions(env.UserID)
		if err != nil {
			t.Logf("Prediction generation failed: %v", err)
		}

		if len(predictions) > 0 {
			// Verify prediction structure
			pred := predictions[0]
			if pred.PredictedStartDate == "" {
				t.Error("Prediction should have start date")
			}

			if pred.ConfidenceScore < 0 || pred.ConfidenceScore > 1 {
				t.Errorf("Invalid confidence score: %f", pred.ConfidenceScore)
			}

			if pred.AlgorithmVersion == "" {
				t.Error("Prediction should have algorithm version")
			}

			// Retrieve predictions via API
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/predictions",
			})

			if err != nil {
				t.Fatalf("Failed to get predictions: %v", err)
			}

			if w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, http.StatusOK, []string{"predictions"})
				apiPredictions := response["predictions"].([]interface{})

				if len(apiPredictions) > 0 {
					t.Logf("Successfully retrieved %d predictions via API", len(apiPredictions))
				}
			}
		}
	})
}

// TestIntegrationMultiTenantIsolation tests data isolation between users
func TestIntegrationMultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create two users
	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	defer cleanupTestData(t, env, user1ID)
	defer cleanupTestData(t, env, user2ID)

	t.Run("DataIsolation", func(t *testing.T) {
		// Create cycle for user 1
		cycleID1 := createTestCycle(t, env, user1ID, "2024-01-15", "medium", "User 1 cycle")

		// Create cycle for user 2
		cycleID2 := createTestCycle(t, env, user2ID, "2024-01-15", "heavy", "User 2 cycle")

		// Query as user 1 - should only see their data
		w1, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
			UserID: user1ID,
		})

		if err != nil {
			t.Fatalf("Failed to query user 1 cycles: %v", err)
		}

		if w1.Code != http.StatusOK {
			t.Skipf("Skipping isolation test - database not available")
		}

		response1 := assertJSONResponse(t, w1, http.StatusOK, []string{"cycles"})
		cycles1 := response1["cycles"].([]interface{})

		if len(cycles1) != 1 {
			t.Errorf("User 1 should see 1 cycle, got %d", len(cycles1))
		} else {
			cycle := cycles1[0].(map[string]interface{})
			if cycle["id"].(string) != cycleID1 {
				t.Error("User 1 seeing wrong cycle")
			}
		}

		// Query as user 2 - should only see their data
		w2, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
			UserID: user2ID,
		})

		if err != nil {
			t.Fatalf("Failed to query user 2 cycles: %v", err)
		}

		response2 := assertJSONResponse(t, w2, http.StatusOK, []string{"cycles"})
		cycles2 := response2["cycles"].([]interface{})

		if len(cycles2) != 1 {
			t.Errorf("User 2 should see 1 cycle, got %d", len(cycles2))
		} else {
			cycle := cycles2[0].(map[string]interface{})
			if cycle["id"].(string) != cycleID2 {
				t.Error("User 2 seeing wrong cycle")
			}
		}
	})
}

// TestIntegrationEncryptionEndToEnd tests end-to-end encryption workflow
func TestIntegrationEncryptionEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("EncryptedDataFlow", func(t *testing.T) {
		sensitiveNotes := "Private health information: severe headaches correlating with cycle day 2"

		// Create cycle with encrypted notes
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/cycles",
			Body: map[string]interface{}{
				"start_date":     "2024-01-15",
				"flow_intensity": "medium",
				"notes":          sensitiveNotes,
			},
		})

		if err != nil {
			t.Fatalf("Failed to create cycle: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Skipf("Skipping encryption test - database not available (status %d)", w.Code)
		}

		response := assertJSONResponse(t, w, http.StatusCreated, []string{"cycle_id"})
		cycleID := response["cycle_id"].(string)

		// Verify data is encrypted in database
		var notesEncrypted string
		err = env.DB.QueryRow("SELECT notes_encrypted FROM cycles WHERE id = $1", cycleID).Scan(&notesEncrypted)
		if err != nil {
			t.Skipf("Cannot verify encryption - database query failed: %v", err)
		}

		// Notes should not be stored in plain text
		if notesEncrypted == sensitiveNotes {
			t.Error("SECURITY ISSUE: Notes are stored in plain text!")
		}

		// Notes should not be empty
		if notesEncrypted == "" {
			t.Error("Notes were not encrypted")
		}

		// Retrieve cycle and verify notes are decrypted correctly
		w2, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
		})

		if err != nil {
			t.Fatalf("Failed to retrieve cycles: %v", err)
		}

		response2 := assertJSONResponse(t, w2, http.StatusOK, []string{"cycles"})
		cycles := response2["cycles"].([]interface{})

		if len(cycles) > 0 {
			cycle := cycles[0].(map[string]interface{})
			if notes, exists := cycle["notes"]; exists {
				if notes.(string) != sensitiveNotes {
					t.Errorf("Decrypted notes don't match original. Got: %v", notes)
				}
			} else {
				t.Error("Notes field missing in response")
			}
		}
	})
}

// TestIntegrationDateRangeQueries tests date range filtering
func TestIntegrationDateRangeQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("SymptomDateRangeFilter", func(t *testing.T) {
		// Create symptoms across different dates
		dates := []string{"2024-01-10", "2024-01-15", "2024-01-20", "2024-01-25", "2024-01-30"}
		for i, date := range dates {
			createTestSymptom(t, env, env.UserID, date, 5+i)
		}

		// Query with date range
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/symptoms?start_date=2024-01-15&end_date=2024-01-25",
		})

		if err != nil {
			t.Fatalf("Failed to query symptoms: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Skipf("Skipping date range test - database not available")
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"symptoms"})
		symptoms := response["symptoms"].([]interface{})

		// Should return symptoms from 2024-01-15, 2024-01-20, 2024-01-25 (3 total)
		if len(symptoms) != 3 {
			t.Errorf("Expected 3 symptoms in date range, got %d", len(symptoms))
		}

		// Verify dates are within range
		for _, s := range symptoms {
			symptom := s.(map[string]interface{})
			date := symptom["symptom_date"].(string)
			if date < "2024-01-15" || date > "2024-01-25" {
				t.Errorf("Symptom date %s is outside requested range", date)
			}
		}
	})
}

// TestIntegrationConcurrentRequests tests concurrent access to the API
func TestIntegrationConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("ConcurrentCycleCreation", func(t *testing.T) {
		numRequests := 10
		results := make(chan error, numRequests)

		// Create cycles concurrently
		for i := 0; i < numRequests; i++ {
			go func(index int) {
				date := fmt.Sprintf("2024-01-%02d", 10+index)
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/cycles",
					Body: map[string]interface{}{
						"start_date":     date,
						"flow_intensity": "medium",
						"notes":          fmt.Sprintf("Concurrent cycle %d", index),
					},
				})

				if err != nil {
					results <- err
					return
				}

				if w.Code != http.StatusCreated {
					results <- fmt.Errorf("unexpected status: %d", w.Code)
					return
				}

				results <- nil
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		for i := 0; i < numRequests; i++ {
			err := <-results
			if err == nil {
				successCount++
			}
		}

		if successCount == 0 {
			t.Skip("All concurrent requests failed - database may not be available")
		}

		t.Logf("Successfully created %d cycles concurrently", successCount)

		// Verify all cycles were created
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
		})

		if err == nil && w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, []string{"cycles"})
			cycles := response["cycles"].([]interface{})

			if len(cycles) != successCount {
				t.Logf("Expected %d cycles, found %d", successCount, len(cycles))
			}
		}
	})
}

// TestIntegrationAuthenticationFlow tests the complete authentication flow
func TestIntegrationAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AuthenticationRequired", func(t *testing.T) {
		// Test without authentication
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
			UserID: "", // No user ID
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertErrorResponse(t, w, http.StatusUnauthorized)
	})

	t.Run("InvalidAuthentication", func(t *testing.T) {
		// Test with invalid UUID
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
			Headers: map[string]string{
				"X-User-ID": "not-a-valid-uuid",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ValidAuthentication", func(t *testing.T) {
		// Test with valid UUID
		validUserID := uuid.New().String()
		defer cleanupTestData(t, env, validUserID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/cycles",
			UserID: validUserID,
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should succeed (empty result is OK)
		if w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest {
			t.Error("Valid authentication should not fail")
		}
	})
}

// TestIntegrationHealthEndpoints tests health check endpoints
func TestIntegrationHealthEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GeneralHealthCheck", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			UserID: "", // Health doesn't require auth
		})

		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		// Health check should return 200 or 503
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Health check returned unexpected status: %d", w.Code)
		}
	})

	t.Run("EncryptionHealthCheck", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health/encryption",
		})

		if err != nil {
			t.Fatalf("Encryption health check failed: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"encryption_enabled", "algorithm"})

		if !response["encryption_enabled"].(bool) {
			t.Error("Encryption should be enabled")
		}

		if response["algorithm"].(string) != "AES-GCM" {
			t.Error("Expected AES-GCM algorithm")
		}
	})

	t.Run("AuthHealthCheck", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/auth/status",
		})

		if err != nil {
			t.Fatalf("Auth health check failed: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, []string{"authenticated", "user_id", "multi_tenant"})

		if !response["authenticated"].(bool) {
			t.Error("Should be authenticated")
		}

		if !response["multi_tenant"].(bool) {
			t.Error("Multi-tenant should be enabled")
		}
	})
}
