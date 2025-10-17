package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOllamaClient_CircuitBreaker(t *testing.T) {
	// Test circuit breaker integration with Ollama client
	cb := &CircuitBreaker{
		maxFailures:  3,
		resetTimeout: time.Second,
		state:        Closed,
	}

	client := &OllamaClient{
		BaseURL:        "http://invalid-ollama:11434",
		Client:         &http.Client{Timeout: 100 * time.Millisecond},
		CircuitBreaker: cb,
	}

	ctx := context.Background()

	t.Run("OpenCircuitAfterFailures", func(t *testing.T) {
		// Reset circuit breaker
		cb.state = Closed
		cb.failures = 0

		// Make multiple failing requests
		for i := 0; i < 3; i++ {
			_, err := client.GetModels(ctx)
			assert.Error(t, err)
		}

		// Circuit should be open now
		assert.Equal(t, Open, cb.GetState())

		// Next request should fail immediately
		_, err := client.GetModels(ctx)
		assert.Error(t, err)
	})

	t.Run("HalfOpenAfterTimeout", func(t *testing.T) {
		// Reset circuit breaker
		cb.state = Closed
		cb.failures = 0
		cb.resetTimeout = 10 * time.Millisecond

		// Open the circuit
		for i := 0; i < 3; i++ {
			client.GetModels(ctx)
		}

		assert.Equal(t, Open, cb.GetState())

		// Wait for reset timeout
		time.Sleep(15 * time.Millisecond)

		// Check if can request (should transition to half-open)
		err := cb.CanRequest()
		assert.NoError(t, err)
		assert.Equal(t, HalfOpen, cb.GetState())
	})
}

func TestOllamaClient_IsHealthy(t *testing.T) {
	t.Run("WithInvalidEndpoint", func(t *testing.T) {
		client := &OllamaClient{
			BaseURL: "http://invalid-host:11434",
			Client:  &http.Client{Timeout: 100 * time.Millisecond},
			CircuitBreaker: &CircuitBreaker{
				maxFailures:  5,
				resetTimeout: 60 * time.Second,
				state:        Closed,
			},
		}

		ctx := context.Background()
		healthy := client.IsHealthy(ctx)
		assert.False(t, healthy)
	})

	t.Run("WithMockServer", func(t *testing.T) {
		// Create a mock Ollama server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"models": []}`))
			}
		}))
		defer mockServer.Close()

		client := &OllamaClient{
			BaseURL: mockServer.URL,
			Client:  &http.Client{Timeout: 5 * time.Second},
			CircuitBreaker: &CircuitBreaker{
				maxFailures:  5,
				resetTimeout: 60 * time.Second,
				state:        Closed,
			},
		}

		ctx := context.Background()
		healthy := client.IsHealthy(ctx)
		assert.True(t, healthy)
	})
}

func TestStoreOrchestratorRequest(t *testing.T) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}

	db, cleanup := createMockDatabase(t)
	if db == nil {
		return
	}
	defer cleanup()

	err := storeOrchestratorRequest(
		db,
		"test-request-id",
		"completion",
		"llama3.2:1b",
		false,
		150,
		true,
		nil,
		0.5,
		0.001,
	)

	assert.NoError(t, err)

	// Verify the record was stored
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM orchestrator_requests WHERE request_id = $1", "test-request-id").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStoreSystemResources(t *testing.T) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}

	db, cleanup := createMockDatabase(t)
	if db == nil {
		return
	}
	defer cleanup()

	err := storeSystemResources(db, 8.0, 6.0, 16.0, 25.5, 0.0)
	assert.NoError(t, err)

	// Verify the record was stored
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM system_resources").Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}

func TestUpdateModelMetrics(t *testing.T) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}

	db, cleanup := createMockDatabase(t)
	if db == nil {
		return
	}
	defer cleanup()

	// Insert initial metrics
	err := updateModelMetrics(db, "llama3.2:1b", 1, 1, 0, 150.0, 0.5, 2048.0, true)
	assert.NoError(t, err)

	// Update metrics
	err = updateModelMetrics(db, "llama3.2:1b", 1, 1, 0, 200.0, 0.6, 2048.0, true)
	assert.NoError(t, err)

	// Verify the record was updated
	var requestCount int
	err = db.QueryRow("SELECT request_count FROM model_metrics WHERE model_name = $1", "llama3.2:1b").Scan(&requestCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, requestCount) // Should be incremented
}

func TestDatabaseOperations_NilDatabase(t *testing.T) {
	var db *sql.DB = nil

	t.Run("StoreOrchestratorRequest", func(t *testing.T) {
		err := storeOrchestratorRequest(db, "test", "completion", "model", false, 100, true, nil, 0.5, 0.001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database not initialized")
	})

	t.Run("StoreSystemResources", func(t *testing.T) {
		err := storeSystemResources(db, 8.0, 6.0, 16.0, 25.5, 0.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database not initialized")
	})

	t.Run("UpdateModelMetrics", func(t *testing.T) {
		err := updateModelMetrics(db, "model", 1, 1, 0, 150.0, 0.5, 2048.0, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database not initialized")
	})
}

func TestRouteAIRequest(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	t.Run("WithoutOllamaClient", func(t *testing.T) {
		req := &RouteRequest{
			TaskType: "completion",
			Prompt:   "Hello, world!",
		}

		response, err := routeAIRequest(req, nil, nil, logger)

		// Should succeed with simulation
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.RequestID)
		assert.NotEmpty(t, response.SelectedModel)
		assert.NotEmpty(t, response.Response)
		assert.Contains(t, response.Response, "Simulated response")
	})

	t.Run("WithRequirements", func(t *testing.T) {
		req := &RouteRequest{
			TaskType: "completion",
			Prompt:   "Test prompt",
			Requirements: map[string]interface{}{
				"maxTokens":   float64(100),
				"temperature": 0.7,
				"costLimit":   0.005,
			},
		}

		response, err := routeAIRequest(req, nil, nil, logger)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotNil(t, response.Metrics)

		// Verify metrics structure
		metrics := response.Metrics
		assert.Contains(t, metrics, "responseTimeMs")
		assert.Contains(t, metrics, "memoryPressure")
		assert.Contains(t, metrics, "modelUsed")
	})
}

func TestRouteAIRequest_WithDatabase(t *testing.T) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}

	db, cleanup := createMockDatabase(t)
	if db == nil {
		return
	}
	defer cleanup()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	req := &RouteRequest{
		TaskType: "completion",
		Prompt:   "Test prompt",
	}

	response, err := routeAIRequest(req, db, nil, logger)

	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Verify request was logged to database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM orchestrator_requests WHERE request_id = $1", response.RequestID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify model metrics were updated
	err = db.QueryRow("SELECT COUNT(*) FROM model_metrics WHERE model_name = $1", response.SelectedModel).Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}

// Benchmark tests for orchestrator-specific functions
func BenchmarkStoreOrchestratorRequest(b *testing.B) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		b.Skip("Skipping database benchmark (SKIP_DB_TESTS=true)")
	}

	// This would require a test database, skip for now
	b.Skip("Database benchmarks require test database setup")
}
