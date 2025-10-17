// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Request        HTTPTestRequest
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds a test case for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Request with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Headers: map[string]string{
				"X-User-ID": "invalid-uuid-format",
			},
		},
	})
	return b
}

// AddMissingAuth adds a test case for missing authentication
func (b *TestScenarioBuilder) AddMissingAuth(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Request without authentication header",
		ExpectedStatus: http.StatusUnauthorized,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Headers: map[string]string{
				// No X-User-ID header
			},
			UserID: "", // Explicitly no user ID
		},
	})
	return b
}

// AddInvalidJSON adds a test case for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Request with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Malformed JSON string
		},
	})
	return b
}

// AddMissingRequiredField adds a test case for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Request missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]interface{}{}, // Empty body
		},
	})
	return b
}

// AddNonExistentResource adds a test case for accessing non-existent resources
func (b *TestScenarioBuilder) AddNonExistentResource(method, pathTemplate string) *TestScenarioBuilder {
	// Generate a non-existent UUID
	nonExistentID := uuid.New().String()
	path := fmt.Sprintf(pathTemplate, nonExistentID)

	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Request for non-existent resource",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
	})
	return b
}

// Build returns the accumulated error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration
	Cleanup        func(t *testing.T, env *TestEnvironment, setupData interface{})
	ValidateTiming func(t *testing.T, duration time.Duration, maxDuration time.Duration)
}

// RunPerformanceTest executes a single performance test
func RunPerformanceTest(t *testing.T, env *TestEnvironment, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, env)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(t, env, setupData)
		}

		// Execute and measure
		duration := pattern.Execute(t, env, setupData)

		// Validate timing
		if pattern.ValidateTiming != nil {
			pattern.ValidateTiming(t, duration, pattern.MaxDuration)
		} else {
			// Default validation
			if duration > pattern.MaxDuration {
				t.Errorf("Operation took %v, expected max %v", duration, pattern.MaxDuration)
			} else {
				t.Logf("Performance: %v (max: %v) ✓", duration, pattern.MaxDuration)
			}
		}
	})
}

// Common performance patterns

// CreateCyclePerformancePattern tests cycle creation performance
func CreateCyclePerformancePattern() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "CreateCyclePerformance",
		Description: "Measure cycle creation performance",
		MaxDuration: 100 * time.Millisecond,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			return map[string]interface{}{
				"userID": env.UserID,
			}
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)

			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/cycles",
				Body: map[string]interface{}{
					"start_date":     "2024-01-15",
					"flow_intensity": "medium",
					"notes":          "Test cycle",
				},
				UserID: userID,
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201, got %d", w.Code)
			}

			return duration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)
			cleanupTestData(t, env, userID)
		},
	}
}

// GetCyclesPerformancePattern tests cycle retrieval performance
func GetCyclesPerformancePattern(cycleCount int) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        fmt.Sprintf("GetCyclesPerformance_%dCycles", cycleCount),
		Description: fmt.Sprintf("Measure performance of retrieving %d cycles", cycleCount),
		MaxDuration: 200 * time.Millisecond,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			userID := env.UserID
			// Create test cycles
			for i := 0; i < cycleCount; i++ {
				date := time.Now().AddDate(0, -i, 0).Format("2006-01-02")
				createTestCycle(t, env, userID, date, "medium", fmt.Sprintf("Test cycle %d", i))
			}
			return map[string]interface{}{
				"userID":     userID,
				"cycleCount": cycleCount,
			}
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)

			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/cycles",
				UserID: userID,
			})
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)
			cleanupTestData(t, env, userID)
		},
	}
}

// EncryptionPerformancePattern tests encryption/decryption performance
func EncryptionPerformancePattern() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "EncryptionPerformance",
		Description: "Measure encryption and decryption performance",
		MaxDuration: 50 * time.Millisecond,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			testData := "This is sensitive health data that needs to be encrypted"

			start := time.Now()

			// Encrypt
			encrypted, err := encryptString(testData)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Decrypt
			decrypted, err := decryptString(encrypted)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			duration := time.Since(start)

			if decrypted != testData {
				t.Errorf("Decrypted data doesn't match original")
			}

			return duration
		},
	}
}

// DatabaseConnectionPattern tests database connection pool performance
func DatabaseConnectionPattern() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "DatabaseConnectionPerformance",
		Description: "Measure database connection and query performance",
		MaxDuration: 50 * time.Millisecond,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			err := env.DB.Ping()
			if err != nil {
				t.Fatalf("Database ping failed: %v", err)
			}

			// Execute a simple query
			var result int
			err = env.DB.QueryRow("SELECT 1").Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			duration := time.Since(start)

			if result != 1 {
				t.Errorf("Query returned unexpected result: %d", result)
			}

			return duration
		},
	}
}

// PredictionGenerationPattern tests prediction algorithm performance
func PredictionGenerationPattern() PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "PredictionGenerationPerformance",
		Description: "Measure prediction generation performance",
		MaxDuration: 300 * time.Millisecond,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			userID := env.UserID

			// Create cycles with cycle_length for prediction algorithm
			for i := 0; i < 6; i++ {
				cycleID := uuid.New().String()
				startDate := time.Now().AddDate(0, -i, 0).Format("2006-01-02")
				_, err := env.DB.Exec(`
					INSERT INTO cycles (id, user_id, start_date, cycle_length, flow_intensity)
					VALUES ($1, $2, $3, $4, $5)`,
					cycleID, userID, startDate, 28, "medium")
				if err != nil {
					t.Fatalf("Failed to create test cycle: %v", err)
				}
			}

			return map[string]interface{}{
				"userID": userID,
			}
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)

			start := time.Now()
			predictions, err := generatePredictions(userID)
			duration := time.Since(start)

			if err != nil {
				t.Logf("Prediction generation returned error (may be expected): %v", err)
			} else if len(predictions) == 0 {
				t.Logf("No predictions generated (may need more data)")
			}

			return duration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			data := setupData.(map[string]interface{})
			userID := data["userID"].(string)
			cleanupTestData(t, env, userID)
		},
	}
}
