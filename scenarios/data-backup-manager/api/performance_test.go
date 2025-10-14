package main

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkHealthEndpoint benchmarks health check performance
func BenchmarkHealthEndpoint(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/health", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Health check failed with status %d", rr.Code)
		}
	}
}

// BenchmarkBackupCreate benchmarks backup creation performance
func BenchmarkBackupCreate(b *testing.B) {
	router := createTestRouter()

	reqBody := BackupCreateRequest{
		Type:        "incremental",
		Targets:     []string{"postgres"},
		Description: "Benchmark backup",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Backup creation failed with status %d", rr.Code)
		}
	}
}

// BenchmarkBackupStatus benchmarks status endpoint performance
func BenchmarkBackupStatus(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/api/v1/backup/status", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Status check failed with status %d", rr.Code)
		}
	}
}

// BenchmarkBackupList benchmarks backup list retrieval
func BenchmarkBackupList(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/api/v1/backup/list", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Backup list failed with status %d", rr.Code)
		}
	}
}

// BenchmarkRestoreCreate benchmarks restore creation
func BenchmarkRestoreCreate(b *testing.B) {
	router := createTestRouter()

	reqBody := RestoreCreateRequest{
		BackupJobID:         "backup-123",
		Targets:             []string{"postgres"},
		VerifyBeforeRestore: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/restore/create", reqBody)
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Restore creation failed with status %d", rr.Code)
		}
	}
}

// BenchmarkScheduleList benchmarks schedule list retrieval
func BenchmarkScheduleList(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/api/v1/schedules", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Schedule list failed with status %d", rr.Code)
		}
	}
}

// BenchmarkScheduleCreate benchmarks schedule creation
func BenchmarkScheduleCreate(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reqBody := map[string]interface{}{
			"name":            fmt.Sprintf("bench-schedule-%d", i),
			"cron_expression": "0 2 * * *",
			"backup_type":     "full",
			"targets":         []string{"postgres"},
			"retention_days":  7,
		}

		req := makeHTTPRequest("POST", "/api/v1/schedules", reqBody)
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Schedule creation failed with status %d", rr.Code)
		}
	}
}

// BenchmarkComplianceReport benchmarks compliance report generation
func BenchmarkComplianceReport(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/api/v1/compliance/report", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Compliance report failed with status %d", rr.Code)
		}
	}
}

// BenchmarkMaintenanceStatus benchmarks maintenance status retrieval
func BenchmarkMaintenanceStatus(b *testing.B) {
	router := createTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/api/v1/maintenance/status", nil)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("Maintenance status failed with status %d", rr.Code)
		}
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	router := createTestRouter()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := makeHTTPRequest("GET", "/health", nil)
			rr := executeRequest(router, req)

			if rr.Code != 200 {
				b.Fatalf("Health check failed with status %d", rr.Code)
			}
		}
	})
}

// BenchmarkJSONSerialization benchmarks JSON encoding/decoding
func BenchmarkJSONSerialization(b *testing.B) {
	router := createTestRouter()

	complexBody := map[string]interface{}{
		"type":           "full",
		"targets":        []string{"postgres", "files", "minio"},
		"description":    "Complex backup with multiple targets and metadata",
		"retention_days": 30,
		"metadata": map[string]interface{}{
			"priority": "high",
			"tags":     []string{"production", "critical", "daily"},
			"owner":    "ops-team",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/backup/create", complexBody)
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Complex backup creation failed with status %d", rr.Code)
		}
	}
}

// BenchmarkBackupJobCreation benchmarks BackupJob struct creation
func BenchmarkBackupJobCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &BackupJob{
			ID:          fmt.Sprintf("backup-%d", time.Now().Unix()),
			Type:        "full",
			Target:      "postgres",
			TargetID:    "main-db",
			Status:      "pending",
			StartedAt:   time.Now(),
			Description: "Benchmark backup job",
		}
	}
}

// BenchmarkErrorHandling benchmarks error response generation
func BenchmarkErrorHandling(b *testing.B) {
	router := createTestRouter()

	invalidBody := BackupCreateRequest{
		Type:    "invalid-type",
		Targets: []string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/backup/create", invalidBody)
		rr := executeRequest(router, req)

		if rr.Code != 400 {
			b.Fatalf("Expected 400 error, got %d", rr.Code)
		}
	}
}

// BenchmarkMultipleEndpoints benchmarks round-robin access to multiple endpoints
func BenchmarkMultipleEndpoints(b *testing.B) {
	router := createTestRouter()

	endpoints := []struct {
		method string
		path   string
		body   interface{}
	}{
		{"GET", "/health", nil},
		{"GET", "/api/v1/backup/status", nil},
		{"GET", "/api/v1/backup/list", nil},
		{"GET", "/api/v1/schedules", nil},
		{"GET", "/api/v1/compliance/report", nil},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		endpoint := endpoints[i%len(endpoints)]
		req := makeHTTPRequest(endpoint.method, endpoint.path, endpoint.body)
		rr := executeRequest(router, req)

		if rr.Code != 200 {
			b.Fatalf("%s %s failed with status %d", endpoint.method, endpoint.path, rr.Code)
		}
	}
}

// BenchmarkLargeTargetList benchmarks handling of many targets
func BenchmarkLargeTargetList(b *testing.B) {
	router := createTestRouter()

	// Create a large target list
	targets := make([]string, 100)
	for i := 0; i < 100; i++ {
		targets[i] = fmt.Sprintf("target-%d", i)
	}

	reqBody := BackupCreateRequest{
		Type:    "incremental",
		Targets: targets,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Large target backup failed with status %d", rr.Code)
		}
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	router := createTestRouter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/backup/create", BackupCreateRequest{
			Type:    "full",
			Targets: []string{"postgres"},
		})
		rr := executeRequest(router, req)

		if rr.Code != 201 {
			b.Fatalf("Backup creation failed")
		}
	}
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	router := createTestRouter()

	// Test that health checks complete within acceptable time
	t.Run("HealthCheckLatency", func(t *testing.T) {
		maxLatency := 10 * time.Millisecond
		iterations := 100

		for i := 0; i < iterations; i++ {
			start := time.Now()
			req := makeHTTPRequest("GET", "/health", nil)
			rr := executeRequest(router, req)
			elapsed := time.Since(start)

			if elapsed > maxLatency {
				t.Errorf("Health check took %v, expected < %v", elapsed, maxLatency)
			}

			if rr.Code != 200 {
				t.Fatalf("Health check failed")
			}
		}
	})

	// Test that backup creation completes within acceptable time
	t.Run("BackupCreationLatency", func(t *testing.T) {
		maxLatency := 50 * time.Millisecond
		iterations := 50

		for i := 0; i < iterations; i++ {
			start := time.Now()
			req := makeHTTPRequest("POST", "/api/v1/backup/create", BackupCreateRequest{
				Type:    "incremental",
				Targets: []string{"postgres"},
			})
			rr := executeRequest(router, req)
			elapsed := time.Since(start)

			if elapsed > maxLatency {
				t.Errorf("Backup creation took %v, expected < %v", elapsed, maxLatency)
			}

			if rr.Code != 201 {
				t.Fatalf("Backup creation failed")
			}
		}
	})

	// Test concurrent request handling doesn't degrade
	t.Run("ConcurrentPerformance", func(t *testing.T) {
		numGoroutines := 10
		requestsPerGoroutine := 10
		done := make(chan time.Duration, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				start := time.Now()
				for j := 0; j < requestsPerGoroutine; j++ {
					req := makeHTTPRequest("GET", "/health", nil)
					rr := executeRequest(router, req)
					if rr.Code != 200 {
						t.Errorf("Concurrent health check failed")
					}
				}
				done <- time.Since(start)
			}()
		}

		maxExpected := 1 * time.Second
		for i := 0; i < numGoroutines; i++ {
			elapsed := <-done
			if elapsed > maxExpected {
				t.Errorf("Concurrent requests took %v, expected < %v", elapsed, maxExpected)
			}
		}
	})
}
