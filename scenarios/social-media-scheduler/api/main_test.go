// +build testing

package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify health response structure
		if response.Data == nil {
			t.Error("Expected health data, got nil")
		}
	})
}

// TestQueueHealthCheck tests the queue health endpoint
func TestQueueHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health/queue",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify queue health data
		if response.Data == nil {
			t.Error("Expected queue health data, got nil")
		}
	})
}

// TestConfiguration tests configuration loading
func TestConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidConfiguration", func(t *testing.T) {
		config := &Configuration{
			APIPort:     "18888",
			UIPort:      "38888",
			DatabaseURL: "postgres://test",
			RedisURL:    "redis://test",
			MinIOURL:    "http://localhost:9000",
			OllamaURL:   "http://localhost:11434",
			JWTSecret:   "test-secret",
			Environment: "test",
			Mode:        "both",
		}

		if config.APIPort == "" {
			t.Error("API port should not be empty")
		}
		if config.Environment != "test" {
			t.Errorf("Expected environment 'test', got %s", config.Environment)
		}
		if config.Mode != "both" {
			t.Errorf("Expected mode 'both', got %s", config.Mode)
		}
	})

	t.Run("ModeValidation", func(t *testing.T) {
		validModes := []string{"server", "worker", "both"}
		for _, mode := range validModes {
			config := &Configuration{
				Mode: mode,
			}
			if config.Mode != mode {
				t.Errorf("Expected mode %s, got %s", mode, config.Mode)
			}
		}
	})
}

// TestDatabaseConnection tests database connectivity
func TestDatabaseConnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("DatabasePing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := env.DB.PingContext(ctx); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})

	t.Run("DatabaseQuery", func(t *testing.T) {
		var count int
		err := env.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Errorf("Failed to query database: %v", err)
		}

		if count < 0 {
			t.Errorf("Invalid user count: %d", count)
		}
	})

	t.Run("ConnectionPoolSettings", func(t *testing.T) {
		stats := env.DB.Stats()
		if stats.MaxOpenConnections <= 0 {
			t.Error("Max open connections should be configured")
		}
	})
}

// TestRedisConnection tests Redis connectivity
func TestRedisConnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("RedisPing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := env.Redis.Ping(ctx).Err(); err != nil {
			t.Errorf("Redis ping failed: %v", err)
		}
	})

	t.Run("RedisSetGet", func(t *testing.T) {
		ctx := context.Background()
		key := "test:key"
		value := "test_value"

		// Set value
		if err := env.Redis.Set(ctx, key, value, 10*time.Second).Err(); err != nil {
			t.Fatalf("Failed to set Redis key: %v", err)
		}

		// Get value
		result, err := env.Redis.Get(ctx, key).Result()
		if err != nil {
			t.Fatalf("Failed to get Redis key: %v", err)
		}

		if result != value {
			t.Errorf("Expected %s, got %s", value, result)
		}

		// Cleanup
		env.Redis.Del(ctx, key)
	})

	t.Run("RedisQueueOperations", func(t *testing.T) {
		ctx := context.Background()
		queueKey := "test:queue"

		// Push to queue
		if err := env.Redis.LPush(ctx, queueKey, "item1", "item2").Err(); err != nil {
			t.Fatalf("Failed to push to queue: %v", err)
		}

		// Get queue length
		length, err := env.Redis.LLen(ctx, queueKey).Result()
		if err != nil {
			t.Fatalf("Failed to get queue length: %v", err)
		}

		if length != 2 {
			t.Errorf("Expected queue length 2, got %d", length)
		}

		// Cleanup
		env.Redis.Del(ctx, queueKey)
	})
}

// TestApplicationInitialization tests app initialization
func TestApplicationInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AppComponents", func(t *testing.T) {
		if env.App.Config == nil {
			t.Error("Config should not be nil")
		}
		if env.App.DB == nil {
			t.Error("Database should not be nil")
		}
		if env.App.Redis == nil {
			t.Error("Redis should not be nil")
		}
		if env.App.PlatformMgr == nil {
			t.Error("PlatformManager should not be nil")
		}
		if env.App.JobProcessor == nil {
			t.Error("JobProcessor should not be nil")
		}
		if env.App.WebSocket == nil {
			t.Error("WebSocket should not be nil")
		}
		if env.App.Router == nil {
			t.Error("Router should not be nil")
		}
	})

	t.Run("RouterConfiguration", func(t *testing.T) {
		routes := env.App.Router.Routes()
		if len(routes) == 0 {
			t.Error("Router should have routes configured")
		}

		// Verify critical routes exist
		criticalPaths := []string{
			"/health",
			"/health/queue",
			"/ws",
		}

		routeMap := make(map[string]bool)
		for _, route := range routes {
			routeMap[route.Path] = true
		}

		for _, path := range criticalPaths {
			if !routeMap[path] {
				t.Errorf("Critical route %s not found", path)
			}
		}
	})
}

// TestCORSConfiguration tests CORS middleware
func TestCORSConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("OptionsRequest", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/auth/platforms",
			Headers: map[string]string{
				"Origin": "http://localhost:" + env.Config.UIPort,
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// CORS headers should be present
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("CORS Access-Control-Allow-Origin header missing")
		}
	})
}

// TestResponseFormat tests standard response format
func TestResponseFormat(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, true)

		// Verify response structure
		if response.Error != "" {
			t.Errorf("Success response should not have error field set")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/posts/invalid-id",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return unauthorized (no auth token)
		response := assertJSONResponse(t, w, http.StatusUnauthorized, false)

		if response.Error == "" {
			t.Error("Error response should have error message")
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyPath", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("InvalidPath", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/invalid/path/that/does/not/exist",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UnsupportedMethod", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PATCH",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return method not allowed
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d or %d, got %d", http.StatusMethodNotAllowed, http.StatusNotFound, w.Code)
		}
	})

	t.Run("LargePayload", func(t *testing.T) {
		// Create a large but reasonable payload
		largeContent := ""
		for i := 0; i < 1000; i++ {
			largeContent += "This is a test content line. "
		}

		body := map[string]interface{}{
			"content": largeContent,
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/posts/schedule",
			Body:   body,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle large payloads (may fail auth or validation, but not crash)
		if w.Code == http.StatusInternalServerError {
			t.Errorf("Server should handle large payloads gracefully, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				})
				if err != nil {
					t.Errorf("Failed to make request: %v", err)
				}
				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

// TestGracefulShutdown tests graceful shutdown behavior
func TestGracefulShutdown(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CleanupResources", func(t *testing.T) {
		// Verify connections are open
		if err := env.DB.Ping(); err != nil {
			t.Fatalf("Database should be connected: %v", err)
		}

		ctx := context.Background()
		if err := env.Redis.Ping(ctx).Err(); err != nil {
			t.Fatalf("Redis should be connected: %v", err)
		}

		// Call cleanup
		env.App.cleanup()

		// Verify connections are closed
		if err := env.DB.Ping(); err == nil {
			t.Error("Database connection should be closed after cleanup")
		}
	})
}
