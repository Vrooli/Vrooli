// +build testing

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPerformanceDeviceOperations tests performance of device operations
func TestPerformanceDeviceOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	pattern := PerformanceTestPattern{
		Name:        "DeviceRegistration",
		Description: "Test device registration performance",
		MaxDuration: 5 * time.Second,
		Iterations:  100,
		Concurrent:  false,
		Execute: func(t *testing.T, ts *TestServer, iteration int, setupData interface{}) {
			device := map[string]interface{}{
				"name":         fmt.Sprintf("Perf Device %d", iteration),
				"type":         "mobile",
				"platform":     "android",
				"capabilities": []string{"clipboard", "files"},
			}

			w, err := ts.makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/devices",
				Body:   device,
				Headers: map[string]string{
					"X-User-ID": user.ID,
				},
			})

			if err != nil {
				t.Errorf("Iteration %d failed: %v", iteration, err)
				return
			}

			if w.Code != 201 {
				t.Errorf("Iteration %d: Expected status 201, got %d", iteration, w.Code)
			}
		},
		Validate: func(t *testing.T, duration time.Duration, iterations int) {
			avgDuration := duration / time.Duration(iterations)
			t.Logf("Average device registration time: %v", avgDuration)

			if avgDuration > 50*time.Millisecond {
				t.Errorf("Average registration time %v exceeds target 50ms", avgDuration)
			}
		},
	}

	RunPerformanceTest(t, ts, pattern)
}

// TestPerformanceConcurrentDeviceRegistration tests concurrent device registration
func TestPerformanceConcurrentDeviceRegistration(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	pattern := PerformanceTestPattern{
		Name:        "ConcurrentDeviceRegistration",
		Description: "Test concurrent device registration performance",
		MaxDuration: 3 * time.Second,
		Iterations:  50,
		Concurrent:  true,
		Execute: func(t *testing.T, ts *TestServer, iteration int, setupData interface{}) {
			device := map[string]interface{}{
				"name":         fmt.Sprintf("Concurrent Device %d", iteration),
				"type":         "mobile",
				"platform":     "android",
				"capabilities": []string{"clipboard"},
			}

			w, err := ts.makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/devices",
				Body:   device,
				Headers: map[string]string{
					"X-User-ID": user.ID,
				},
			})

			if err != nil {
				t.Errorf("Concurrent iteration %d failed: %v", iteration, err)
				return
			}

			if w.Code != 201 {
				t.Errorf("Concurrent iteration %d: Expected status 201, got %d", iteration, w.Code)
			}
		},
		Validate: func(t *testing.T, duration time.Duration, iterations int) {
			t.Logf("Concurrent registrations: %d in %v", iterations, duration)

			// Should complete all concurrent operations in reasonable time
			if duration > 3*time.Second {
				t.Errorf("Concurrent operations took %v, exceeds 3s target", duration)
			}
		},
	}

	RunPerformanceTest(t, ts, pattern)
}

// TestPerformanceSyncItemCreation tests sync item creation performance
func TestPerformanceSyncItemCreation(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	pattern := PerformanceTestPattern{
		Name:        "SyncItemCreation",
		Description: "Test sync item creation performance",
		MaxDuration: 5 * time.Second,
		Iterations:  100,
		Concurrent:  false,
		Execute: func(t *testing.T, ts *TestServer, iteration int, setupData interface{}) {
			clipboardData := map[string]interface{}{
				"content": fmt.Sprintf("Performance test content %d", iteration),
			}

			w, err := ts.makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/sync/clipboard",
				Body:   clipboardData,
				Headers: map[string]string{
					"X-User-ID": user.ID,
				},
			})

			if err != nil {
				t.Errorf("Iteration %d failed: %v", iteration, err)
				return
			}

			if w.Code != 200 {
				t.Errorf("Iteration %d: Expected status 200, got %d", iteration, w.Code)
			}
		},
		Validate: func(t *testing.T, duration time.Duration, iterations int) {
			avgDuration := duration / time.Duration(iterations)
			t.Logf("Average sync item creation time: %v", avgDuration)

			if avgDuration > 50*time.Millisecond {
				t.Errorf("Average creation time %v exceeds target 50ms", avgDuration)
			}
		},
	}

	RunPerformanceTest(t, ts, pattern)
}

// TestPerformanceConcurrentSyncOperations tests concurrent sync operations
func TestPerformanceConcurrentSyncOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	pattern := PerformanceTestPattern{
		Name:        "ConcurrentSyncOperations",
		Description: "Test concurrent clipboard sync performance",
		MaxDuration: 3 * time.Second,
		Iterations:  100,
		Concurrent:  true,
		Execute: func(t *testing.T, ts *TestServer, iteration int, setupData interface{}) {
			clipboardData := map[string]interface{}{
				"content": fmt.Sprintf("Concurrent content %d", iteration),
			}

			w, err := ts.makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/sync/clipboard",
				Body:   clipboardData,
				Headers: map[string]string{
					"X-User-ID": user.ID,
				},
			})

			if err != nil {
				t.Errorf("Concurrent iteration %d failed: %v", iteration, err)
				return
			}

			if w.Code != 200 {
				t.Errorf("Concurrent iteration %d: Expected status 200, got %d", iteration, w.Code)
			}
		},
		Validate: func(t *testing.T, duration time.Duration, iterations int) {
			throughput := float64(iterations) / duration.Seconds()
			t.Logf("Concurrent sync throughput: %.2f ops/sec", throughput)

			// Should handle at least 30 operations per second
			if throughput < 30 {
				t.Errorf("Throughput %.2f ops/sec is below target 30 ops/sec", throughput)
			}
		},
	}

	RunPerformanceTest(t, ts, pattern)
}

// TestPerformanceListOperations tests list operation performance with varying data sizes
func TestPerformanceListOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	// Create test data
	for i := 0; i < 100; i++ {
		ts.createTestDevice(t, user.ID)
		ts.createTestSyncItem(t, user.ID, "clipboard")
	}

	t.Run("ListDevices", func(t *testing.T) {
		start := time.Now()

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/devices",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to list devices: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Listed 100 devices in %v", duration)

		if duration > 500*time.Millisecond {
			t.Errorf("List devices took %v, exceeds 500ms target", duration)
		}
	})

	t.Run("ListSyncItems", func(t *testing.T) {
		start := time.Now()

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/sync/items",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to list sync items: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Listed 100 sync items in %v", duration)

		if duration > 500*time.Millisecond {
			t.Errorf("List sync items took %v, exceeds 500ms target", duration)
		}
	})
}

// TestPerformanceCleanupOperations tests cleanup operation performance
func TestPerformanceCleanupOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	// Create 1000 expired items
	for i := 0; i < 1000; i++ {
		_, err := ts.DB.Exec(`
			INSERT INTO sync_items (id, user_id, type, content, source_device, target_devices, created_at, expires_at, status)
			VALUES ($1, $2, 'clipboard', '{"data": "test"}', $3, '[]', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 hour', 'active')
		`, fmt.Sprintf("expired-%d", i), user.ID, fmt.Sprintf("device-%d", i))

		if err != nil {
			t.Fatalf("Failed to create expired item %d: %v", i, err)
		}
	}

	start := time.Now()
	cleaned := ts.cleanupExpiredItems(t)
	duration := time.Since(start)

	t.Logf("Cleaned up %d expired items in %v", cleaned, duration)

	if cleaned != 1000 {
		t.Errorf("Expected to clean 1000 items, cleaned %d", cleaned)
	}

	if duration > 2*time.Second {
		t.Errorf("Cleanup took %v, exceeds 2s target for 1000 items", duration)
	}

	// Verify cleanup completed
	remainingCount := ts.getSyncItemCount(t, user.ID)
	if remainingCount != 0 {
		t.Errorf("Expected 0 remaining items, got %d", remainingCount)
	}
}

// TestPerformanceDatabaseOperations tests database operation performance
func TestPerformanceDatabaseOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("BulkInsert", func(t *testing.T) {
		const numInserts = 500
		start := time.Now()

		for i := 0; i < numInserts; i++ {
			ts.createTestSyncItem(t, user.ID, "clipboard")
		}

		duration := time.Since(start)
		avgDuration := duration / numInserts

		t.Logf("Bulk insert: %d items in %v (avg: %v)", numInserts, duration, avgDuration)

		if avgDuration > 10*time.Millisecond {
			t.Errorf("Average insert time %v exceeds 10ms target", avgDuration)
		}
	})

	t.Run("BulkRead", func(t *testing.T) {
		// Create test data
		for i := 0; i < 100; i++ {
			ts.createTestSyncItem(t, user.ID, "clipboard")
		}

		start := time.Now()

		var items []SyncItem
		rows, err := ts.DB.Query(`
			SELECT id, user_id, type, content, source_device, target_devices, created_at, expires_at, status
			FROM sync_items
			WHERE user_id = $1 AND status = 'active'
		`, user.ID)

		if err != nil {
			t.Fatalf("Failed to query items: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item SyncItem
			var contentJSON, targetDevicesJSON []byte

			err := rows.Scan(&item.ID, &item.UserID, &item.Type, &contentJSON,
				&item.SourceDevice, &targetDevicesJSON, &item.CreatedAt, &item.ExpiresAt, &item.Status)

			if err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}

			items = append(items, item)
		}

		duration := time.Since(start)

		t.Logf("Bulk read: %d items in %v", len(items), duration)

		if duration > 100*time.Millisecond {
			t.Errorf("Bulk read took %v, exceeds 100ms target", duration)
		}
	})

	t.Run("BulkDelete", func(t *testing.T) {
		// Create test data
		itemIDs := make([]string, 100)
		for i := 0; i < 100; i++ {
			item := ts.createTestSyncItem(t, user.ID, "clipboard")
			itemIDs[i] = item.ID
		}

		start := time.Now()

		_, err := ts.DB.Exec(`
			DELETE FROM sync_items
			WHERE user_id = $1
		`, user.ID)

		if err != nil {
			t.Fatalf("Failed to delete items: %v", err)
		}

		duration := time.Since(start)

		t.Logf("Bulk delete: 100 items in %v", duration)

		if duration > 50*time.Millisecond {
			t.Errorf("Bulk delete took %v, exceeds 50ms target", duration)
		}

		// Verify deletion
		count := ts.getSyncItemCount(t, user.ID)
		if count != 0 {
			t.Errorf("Expected 0 items after deletion, got %d", count)
		}
	})
}

// TestPerformanceMemoryUsage tests memory efficiency
func TestPerformanceMemoryUsage(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("LargePayloadHandling", func(t *testing.T) {
		// Create a large clipboard content (1MB)
		largeContent := string(make([]byte, 1024*1024))

		start := time.Now()

		clipboardData := map[string]interface{}{
			"content": largeContent,
		}

		w, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/sync/clipboard",
			Body:   clipboardData,
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to handle large payload: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Handled 1MB clipboard payload in %v", duration)

		if duration > 500*time.Millisecond {
			t.Errorf("Large payload handling took %v, exceeds 500ms target", duration)
		}
	})
}

// TestPerformanceRateLimiting tests system behavior under high load
func TestPerformanceRateLimiting(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	t.Run("HighFrequencyRequests", func(t *testing.T) {
		const numRequests = 200
		const concurrency = 20

		var wg sync.WaitGroup
		var successCount, errorCount int32
		semaphore := make(chan struct{}, concurrency)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			semaphore <- struct{}{}

			go func(iteration int) {
				defer wg.Done()
				defer func() { <-semaphore }()

				clipboardData := map[string]interface{}{
					"content": fmt.Sprintf("High frequency request %d", iteration),
				}

				w, err := ts.makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/sync/clipboard",
					Body:   clipboardData,
					Headers: map[string]string{
						"X-User-ID": user.ID,
					},
				})

				if err != nil || w.Code != 200 {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		throughput := float64(numRequests) / duration.Seconds()
		successRate := float64(successCount) / float64(numRequests) * 100

		t.Logf("High frequency test: %d requests in %v (%.2f req/sec)", numRequests, duration, throughput)
		t.Logf("Success: %d, Errors: %d (%.2f%% success rate)", successCount, errorCount, successRate)

		if successRate < 95 {
			t.Errorf("Success rate %.2f%% is below 95%% target", successRate)
		}

		if throughput < 50 {
			t.Errorf("Throughput %.2f req/sec is below 50 req/sec target", throughput)
		}
	})
}

// TestPerformanceResponseTime tests response time under various loads
func TestPerformanceResponseTime(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	user := createTestUser()

	// Create baseline data
	for i := 0; i < 50; i++ {
		ts.createTestDevice(t, user.ID)
		ts.createTestSyncItem(t, user.ID, "clipboard")
	}

	measurements := make([]time.Duration, 100)

	for i := 0; i < 100; i++ {
		start := time.Now()

		_, err := ts.makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/sync/items",
			Headers: map[string]string{
				"X-User-ID": user.ID,
			},
		})

		measurements[i] = time.Since(start)

		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}

	// Calculate statistics
	var total time.Duration
	var max time.Duration
	min := measurements[0]

	for _, d := range measurements {
		total += d
		if d > max {
			max = d
		}
		if d < min {
			min = d
		}
	}

	avg := total / time.Duration(len(measurements))

	// Calculate p95
	sorted := make([]time.Duration, len(measurements))
	copy(sorted, measurements)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	p95Index := int(float64(len(sorted)) * 0.95)
	p95 := sorted[p95Index]

	t.Logf("Response time statistics:")
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Avg: %v", avg)
	t.Logf("  P95: %v", p95)

	if p95 > 200*time.Millisecond {
		t.Errorf("P95 response time %v exceeds 200ms target", p95)
	}

	if avg > 100*time.Millisecond {
		t.Errorf("Average response time %v exceeds 100ms target", avg)
	}
}
