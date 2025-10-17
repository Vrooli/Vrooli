package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// BenchmarkSchemaCreation benchmarks schema creation performance
func BenchmarkSchemaCreation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil || env.Database == nil {
		b.Skip("Database not available")
		return
	}
	defer env.Cleanup()

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)

	requestBody := map[string]interface{}{
		"name":        "benchmark-schema",
		"description": "Benchmark test schema",
		"schema_definition": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field1": map[string]interface{}{"type": "string"},
				"field2": map[string]interface{}{"type": "integer"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		requestBody["name"] = fmt.Sprintf("benchmark-schema-%d", i)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)
		if w.Code != 201 {
			b.Fatalf("Failed to create schema: %d", w.Code)
		}
	}
}

// BenchmarkSchemaRetrieval benchmarks schema retrieval performance
func BenchmarkSchemaRetrieval(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	if env == nil || env.Database == nil {
		b.Skip("Database not available")
		return
	}
	defer env.Cleanup()

	// Create test schema
	schema := createTestSchema(&testing.T{}, env.Database, "benchmark-retrieval")
	if schema == nil {
		b.Skip("Failed to create test schema")
		return
	}

	// Setup router
	env.Router.GET("/api/v1/schemas/:id", getSchema)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/" + schema.ID.String(),
		}

		w := makeHTTPRequest(env.Router, req)
		if w.Code != 200 {
			b.Fatalf("Failed to retrieve schema: %d", w.Code)
		}
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization
func BenchmarkJSONSerialization(b *testing.B) {
	schema := &Schema{
		Name:        "benchmark-json",
		Description: "Test schema for JSON benchmarking",
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field1": map[string]interface{}{"type": "string"},
				"field2": map[string]interface{}{"type": "number"},
				"field3": map[string]interface{}{"type": "boolean"},
			},
		},
		Version:  1,
		IsActive: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(schema)
		if err != nil {
			b.Fatalf("JSON marshaling failed: %v", err)
		}
	}
}

// TestSchemaCreationPerformance tests schema creation under load
func TestSchemaCreationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)

	pattern := PerformanceTestPattern{
		Name:        "SchemaCreationLoad",
		Description: "Test schema creation performance under load",
		MaxDuration: 5 * time.Second,
		Iterations:  50,
		Setup: func(t *testing.T) interface{} {
			return env
		},
		Execute: func(t *testing.T, setupData interface{}) {
			requestBody := map[string]interface{}{
				"name":        fmt.Sprintf("test-perf-%d", time.Now().UnixNano()),
				"description": "Performance test schema",
				"schema_definition": map[string]interface{}{
					"type": "object",
				},
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/schemas",
				Body:   requestBody,
			}

			w := makeHTTPRequest(env.Router, req)
			assert.Contains(t, []int{200, 201}, w.Code)
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			avgTime := duration / 50
			t.Logf("Average schema creation time: %v", avgTime)

			// Each schema creation should take less than 100ms on average
			assert.Less(t, avgTime, 100*time.Millisecond,
				"Schema creation taking too long")
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestConcurrentSchemaReads tests concurrent read performance
func TestConcurrentSchemaReads(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Create test schema
	schema := createTestSchema(t, env.Database, "concurrent-reads")
	assert.NotNil(t, schema)

	// Setup router
	env.Router.GET("/api/v1/schemas/:id", getSchema)

	t.Run("ConcurrentReads", func(t *testing.T) {
		concurrency := 10
		iterations := 50

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < iterations; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/schemas/" + schema.ID.String(),
					}

					w := makeHTTPRequest(env.Router, req)
					assert.Equal(t, 200, w.Code)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		totalOps := concurrency * iterations
		opsPerSecond := float64(totalOps) / duration.Seconds()

		t.Logf("Concurrent reads: %d operations in %v (%.2f ops/sec)",
			totalOps, duration, opsPerSecond)

		// Should handle at least 100 ops/sec
		assert.Greater(t, opsPerSecond, 100.0,
			"Throughput too low for concurrent reads")
	})
}

// TestDatabaseQueryPerformance tests database query performance
func TestDatabaseQueryPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("ListSchemasPerformance", func(t *testing.T) {
		// Create multiple schemas
		for i := 0; i < 20; i++ {
			createTestSchema(t, env.Database, fmt.Sprintf("perf-list-%d", i))
		}

		start := time.Now()

		rows, err := env.Database.DB.Query(`
			SELECT id, name, description, version, is_active, created_at, updated_at
			FROM schemas
			WHERE name LIKE 'test-%'
			ORDER BY created_at DESC
			LIMIT 100
		`)

		duration := time.Since(start)

		assert.NoError(t, err)
		if rows != nil {
			rows.Close()
		}

		t.Logf("List schemas query took: %v", duration)

		// Query should complete in less than 100ms
		assert.Less(t, duration, 100*time.Millisecond,
			"List schemas query too slow")
	})

	t.Run("CountQueryPerformance", func(t *testing.T) {
		start := time.Now()

		var count int
		err := env.Database.DB.QueryRow(`
			SELECT COUNT(*) FROM schemas WHERE name LIKE 'test-%'
		`).Scan(&count)

		duration := time.Since(start)

		assert.NoError(t, err)

		t.Logf("Count query took: %v for %d records", duration, count)

		// Count should be very fast
		assert.Less(t, duration, 50*time.Millisecond,
			"Count query too slow")
	})
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("LargeSchemaHandling", func(t *testing.T) {
		// Create a large schema definition
		largeProperties := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeProperties[fmt.Sprintf("field%d", i)] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Field %d description", i),
			}
		}

		schema := &Schema{
			Name:        "test-large-schema",
			Description: "Large schema for memory testing",
			SchemaDefinition: map[string]interface{}{
				"type":       "object",
				"properties": largeProperties,
			},
			Version:  1,
			IsActive: true,
		}

		// Serialize and check size
		data, err := json.Marshal(schema)
		assert.NoError(t, err)

		t.Logf("Large schema JSON size: %d bytes", len(data))

		// Should handle large schemas efficiently
		assert.Less(t, len(data), 1024*1024, // Less than 1MB
			"Schema too large")
	})
}

// TestAPIResponseTimes tests API endpoint response times
func TestAPIResponseTimes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup all routes
	env.Router.GET("/api/v1/schemas", getSchemas)
	env.Router.POST("/api/v1/schemas", createSchema)
	env.Router.GET("/api/v1/templates", getSchemaTemplates)

	tests := []struct {
		name        string
		method      string
		path        string
		body        interface{}
		maxDuration time.Duration
	}{
		{
			name:        "ListSchemas",
			method:      "GET",
			path:        "/api/v1/schemas",
			maxDuration: 200 * time.Millisecond,
		},
		{
			name:        "ListTemplates",
			method:      "GET",
			path:        "/api/v1/templates",
			maxDuration: 200 * time.Millisecond,
		},
		{
			name:   "CreateSchema",
			method: "POST",
			path:   "/api/v1/schemas",
			body: map[string]interface{}{
				"name":        "test-response-time",
				"description": "Response time test",
				"schema_definition": map[string]interface{}{
					"type": "object",
				},
			},
			maxDuration: 300 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			req := HTTPTestRequest{
				Method: tt.method,
				Path:   tt.path,
				Body:   tt.body,
			}

			w := makeHTTPRequest(env.Router, req)
			duration := time.Since(start)

			assert.Contains(t, []int{200, 201}, w.Code)

			t.Logf("%s response time: %v", tt.name, duration)

			assert.Less(t, duration, tt.maxDuration,
				"%s response time exceeded limit", tt.name)
		})
	}
}

// TestBulkOperations tests performance of bulk operations
func TestBulkOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("BulkSchemaCreation", func(t *testing.T) {
		bulkSize := 25
		start := time.Now()

		for i := 0; i < bulkSize; i++ {
			schema := createTestSchema(t, env.Database, fmt.Sprintf("bulk-%d", i))
			assert.NotNil(t, schema)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(bulkSize)

		t.Logf("Bulk creation of %d schemas took %v (avg: %v per schema)",
			bulkSize, duration, avgTime)

		// Total time should be reasonable
		assert.Less(t, duration, 5*time.Second,
			"Bulk operation too slow")
	})

	t.Run("BulkDataRetrieval", func(t *testing.T) {
		// Create test data
		schema := createTestSchema(t, env.Database, "bulk-retrieval")
		for i := 0; i < 10; i++ {
			createTestProcessedData(t, env.Database, schema.ID)
		}

		start := time.Now()

		rows, err := env.Database.DB.Query(`
			SELECT id, schema_id, source_file_name, structured_data, confidence_score
			FROM processed_data
			WHERE schema_id = $1
			ORDER BY created_at DESC
		`, schema.ID)

		duration := time.Since(start)

		assert.NoError(t, err)
		if rows != nil {
			defer rows.Close()

			count := 0
			for rows.Next() {
				count++
			}
			t.Logf("Retrieved %d records in %v", count, duration)
		}

		// Bulk retrieval should be fast
		assert.Less(t, duration, 100*time.Millisecond,
			"Bulk retrieval too slow")
	})
}

// TestHealthCheckPerformance tests health check endpoint performance
func TestHealthCheckPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	env.Router.GET("/health", func(c *gin.Context) {
		handleHealthCheck(c, env.Database.DB)
	})

	t.Run("HealthCheckLatency", func(t *testing.T) {
		iterations := 10
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w := makeHTTPRequest(env.Router, req)
			duration := time.Since(start)

			assert.Equal(t, 200, w.Code)
			totalDuration += duration
		}

		avgDuration := totalDuration / time.Duration(iterations)

		t.Logf("Average health check latency: %v (%d iterations)", avgDuration, iterations)

		// Health checks should be very fast
		assert.Less(t, avgDuration, 50*time.Millisecond,
			"Health check too slow")
	})
}
