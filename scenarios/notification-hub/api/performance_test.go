// +build testing

package main

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkProfileCreation benchmarks profile creation performance
func BenchmarkProfileCreation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := map[string]interface{}{
			"name": fmt.Sprintf("Benchmark Profile %d", i),
			"slug": fmt.Sprintf("bench-profile-%d", i),
			"plan": "test",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/admin/profiles",
			Body:   req,
		})
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		if w.Code != 201 {
			b.Fatalf("Expected status 201, got %d", w.Code)
		}
	}
}

// BenchmarkNotificationSending benchmarks notification creation
func BenchmarkNotificationSending(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil {
		b.Skip("Test environment not available")
		return
	}
	defer env.Cleanup()

	// Create test contact
	contact, err := createTestContact(env.DB, env.TestProfile.ID)
	if err != nil {
		b.Fatalf("Failed to create test contact: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := map[string]interface{}{
			"recipients": []map[string]interface{}{
				{
					"contact_id": contact.ID.String(),
					"variables": map[string]interface{}{
						"name":  fmt.Sprintf("User %d", i),
						"count": i,
					},
				},
			},
			"subject":  fmt.Sprintf("Benchmark Notification %d", i),
			"content": map[string]interface{}{
				"text": "Benchmark test notification {{name}} - {{count}}",
			},
			"channels": []string{"email"},
			"priority": "normal",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
			Body:    req,
			Headers: map[string]string{"X-API-Key": env.TestAPIKey},
		})
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		if w.Code != 201 {
			b.Fatalf("Expected status 201, got %d", w.Code)
		}
	}
}

// TestPerformancePatterns runs systematic performance tests
func TestPerformancePatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Build performance test suite
	performanceTests := NewPerformanceTestBuilder().
		AddEndpointLatencyTest("health_check", "/health", 200*time.Millisecond).
		AddEndpointLatencyTest("profile_list", "/api/v1/admin/profiles", 500*time.Millisecond).
		Build()

	RunPerformanceTests(t, env, performanceTests)
}

// TestBulkNotificationCreation tests creating many notifications
func TestBulkNotificationCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bulk test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test contact
	contact, err := createTestContact(env.DB, env.TestProfile.ID)
	if err != nil {
		t.Fatalf("Failed to create test contact: %v", err)
	}

	t.Run("Create100Notifications", func(t *testing.T) {
		start := time.Now()
		count := 100

		for i := 0; i < count; i++ {
			req := map[string]interface{}{
				"recipients": []map[string]interface{}{
					{
						"contact_id": contact.ID.String(),
						"variables": map[string]interface{}{
							"index": i,
						},
					},
				},
				"subject":  fmt.Sprintf("Bulk Test %d", i),
				"content": map[string]interface{}{
					"text": "Bulk notification {{index}}",
				},
				"channels": []string{"email"},
			}

			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  "POST",
				Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
				Body:    req,
				Headers: map[string]string{"X-API-Key": env.TestAPIKey},
			})

			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}
			if w.Code != 201 {
				t.Fatalf("Request %d expected status 201, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		throughput := float64(count) / duration.Seconds()

		t.Logf("Created %d notifications in %v (%.2f notifications/sec)", count, duration, throughput)

		// Performance target: should create 100 notifications in less than 10 seconds
		if duration > 10*time.Second {
			t.Errorf("Bulk creation took %v, expected less than 10s", duration)
		}

		// Throughput target: should achieve at least 10 notifications/second
		if throughput < 10.0 {
			t.Errorf("Throughput %.2f notifications/sec, expected at least 10/sec", throughput)
		}
	})
}

// TestTemplateRendering benchmarks template rendering
func TestTemplateRenderingPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := env.Server.processor

	template := "Hello {{name}}, your order {{order_id}} for {{product}} is {{status}}. Total: ${{price}}"
	variables := map[string]interface{}{
		"name":     "John Doe",
		"order_id": "ORD-12345",
		"product":  "Premium Widget",
		"status":   "shipped",
		"price":    "99.99",
	}

	iterations := 10000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		result := processor.renderTemplate(template, variables)
		if len(result) == 0 {
			t.Fatal("Template rendering failed")
		}
	}

	duration := time.Since(start)
	perOp := duration / time.Duration(iterations)

	t.Logf("Template rendering: %d iterations in %v (%.2f µs/op)", iterations, duration, float64(perOp.Microseconds()))

	// Performance target: less than 100 microseconds per render
	if perOp > 100*time.Microsecond {
		t.Errorf("Template rendering too slow: %.2f µs/op, expected < 100 µs/op", float64(perOp.Microseconds()))
	}
}

// TestConcurrentNotificationSending tests concurrent notification creation
func TestConcurrentNotificationSending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test contacts
	contacts := make([]*Contact, 10)
	for i := 0; i < len(contacts); i++ {
		contact, err := createTestContact(env.DB, env.TestProfile.ID)
		if err != nil {
			t.Fatalf("Failed to create contact: %v", err)
		}
		contacts[i] = contact
	}

	t.Run("Concurrent10Requests", func(t *testing.T) {
		concurrencyTests := NewConcurrencyTestBuilder().
			AddConcurrentRequests("notifications",
				fmt.Sprintf("/api/v1/profiles/%s/notifications", env.TestProfile.ID),
				"GET", 10, 50).
			Build()

		RunConcurrencyTests(t, env, concurrencyTests)
	})

	t.Run("ConcurrentNotificationCreation", func(t *testing.T) {
		start := time.Now()
		errChan := make(chan error, 50)
		doneChan := make(chan bool, 50)

		for i := 0; i < 50; i++ {
			go func(index int) {
				contact := contacts[index%len(contacts)]
				req := map[string]interface{}{
					"recipients": []map[string]interface{}{
						{
							"contact_id": contact.ID.String(),
							"variables":  map[string]interface{}{"index": index},
						},
					},
					"subject":  fmt.Sprintf("Concurrent Test %d", index),
					"content": map[string]interface{}{"text": "Concurrent notification"},
					"channels": []string{"email"},
				}

				_, err := makeHTTPRequest(env, HTTPTestRequest{
					Method:  "POST",
					Path:    fmt.Sprintf("/api/v1/profiles/%s/notifications/send", env.TestProfile.ID),
					Body:    req,
					Headers: map[string]string{"X-API-Key": env.TestAPIKey},
				})

				errChan <- err
				doneChan <- true
			}(i)
		}

		// Wait for all to complete
		errorCount := 0
		for i := 0; i < 50; i++ {
			<-doneChan
			if err := <-errChan; err != nil {
				errorCount++
			}
		}

		duration := time.Since(start)
		t.Logf("50 concurrent notifications created in %v", duration)

		if errorCount > 0 {
			t.Errorf("%d out of 50 concurrent requests failed", errorCount)
		}

		// Performance target: complete in less than 5 seconds
		if duration > 5*time.Second {
			t.Errorf("Concurrent creation took %v, expected less than 5s", duration)
		}
	})
}

// TestDatabaseQueryPerformance tests database query performance
func TestDatabaseQueryPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ListProfilesPerformance", func(t *testing.T) {
		// Create multiple profiles
		for i := 0; i < 20; i++ {
			req := map[string]interface{}{
				"name": fmt.Sprintf("Perf Test Profile %d", i),
				"plan": "test",
			}

			makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/admin/profiles",
				Body:   req,
			})
		}

		// Test list performance
		start := time.Now()
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/admin/profiles",
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("List profiles query took %v", duration)

		// Performance target: less than 500ms
		if duration > 500*time.Millisecond {
			t.Errorf("List profiles took %v, expected less than 500ms", duration)
		}
	})

	t.Run("NotificationQueryPerformance", func(t *testing.T) {
		// Create test data
		contact, _ := createTestContact(env.DB, env.TestProfile.ID)

		// Create multiple notifications
		for i := 0; i < 50; i++ {
			createTestNotification(env.DB, env.TestProfile.ID, contact.ID)
		}

		// Test query performance
		start := time.Now()
		count, err := getNotificationCount(env.DB, env.TestProfile.ID, "pending")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if count == 0 {
			t.Error("Expected notifications to exist")
		}

		t.Logf("Notification count query took %v", duration)

		// Performance target: less than 100ms
		if duration > 100*time.Millisecond {
			t.Errorf("Notification query took %v, expected less than 100ms", duration)
		}
	})
}

// TestRedisPerformance tests Redis operations performance
func TestRedisPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("RedisCacheOperations", func(t *testing.T) {
		ctx := env.Server.redis.Context()

		// Test write performance
		writeStart := time.Now()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("perf_test_%d", i)
			err := env.Redis.Set(ctx, key, fmt.Sprintf("value_%d", i), 1*time.Minute).Err()
			if err != nil {
				t.Fatalf("Redis write failed: %v", err)
			}
		}
		writeDuration := time.Since(writeStart)

		// Test read performance
		readStart := time.Now()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("perf_test_%d", i)
			_, err := env.Redis.Get(ctx, key).Result()
			if err != nil {
				t.Fatalf("Redis read failed: %v", err)
			}
		}
		readDuration := time.Since(readStart)

		t.Logf("Redis: 1000 writes in %v (%.2f ops/sec)", writeDuration, 1000.0/writeDuration.Seconds())
		t.Logf("Redis: 1000 reads in %v (%.2f ops/sec)", readDuration, 1000.0/readDuration.Seconds())

		// Performance targets
		if writeDuration > 2*time.Second {
			t.Errorf("Redis writes too slow: %v, expected < 2s", writeDuration)
		}
		if readDuration > 1*time.Second {
			t.Errorf("Redis reads too slow: %v, expected < 1s", readDuration)
		}
	})
}
