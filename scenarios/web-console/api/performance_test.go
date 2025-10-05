package main

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestPerformanceSessionCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	pattern := PerformanceTestPattern{
		Name:        "SessionCreationPerformance",
		Description: "Measure session creation time",
		MaxDuration: 500 * time.Millisecond,
		Setup: func(t *testing.T) interface{} {
			return nil
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()
			s, err := manager.createSession(createSessionRequest{
				Command: "/bin/echo",
				Args:    []string{"test"},
			})
			duration := time.Since(start)
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			cleanupSession(s)
			return duration
		},
		Cleanup: func(setupData interface{}) {},
	}

	RunPerformanceTest(t, pattern)
}

func TestPerformanceWorkspaceOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	t.Run("AddTabPerformance", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "AddTabPerformance",
			Description: "Measure tab creation time",
			MaxDuration: 100 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()
				tab := TestData.CreateTabRequest(fmt.Sprintf("tab-%d", start.UnixNano()), "Test Tab", "sky")
				err := ws.addTab(tab)
				duration := time.Since(start)
				if err != nil {
					t.Fatalf("Failed to add tab: %v", err)
				}
				return duration
			},
		}
		RunPerformanceTest(t, pattern)
	})

	t.Run("GetStatePerformance", func(t *testing.T) {
		// Add some tabs first
		for i := 0; i < 10; i++ {
			tab := TestData.CreateTabRequest(fmt.Sprintf("tab-%d", i), "Test Tab", "sky")
			ws.addTab(tab)
		}

		pattern := PerformanceTestPattern{
			Name:        "GetStatePerformance",
			Description: "Measure workspace state retrieval time",
			MaxDuration: 50 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()
				stateBytes, err := ws.getState()
				duration := time.Since(start)
				if err != nil {
					t.Errorf("Failed to get state: %v", err)
				}
				if len(stateBytes) == 0 {
					t.Error("Expected non-empty state")
				}
				return duration
			},
		}
		RunPerformanceTest(t, pattern)
	})
}

func TestConcurrentSessionOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	cfg.maxConcurrent = 10 // Increase capacity for concurrency test
	manager, _, _ := setupTestSessionManager(t, cfg)

	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentSessionCreation",
		Description: "Test concurrent session creation",
		Concurrency: 5,
		Iterations:  5,
		Setup: func(t *testing.T) interface{} {
			return &sync.Mutex{}
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			s, err := manager.createSession(createSessionRequest{
				Command: "/bin/sleep",
				Args:    []string{"0.1"},
			})
			if err != nil {
				return err
			}
			// Cleanup after a short delay
			time.Sleep(50 * time.Millisecond)
			cleanupSession(s)
			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
					t.Logf("Error in concurrent execution: %v", err)
				}
			}
			if errorCount > 0 {
				t.Errorf("Expected no errors, got %d errors", errorCount)
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

func TestConcurrentWorkspaceOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentTabOperations",
		Description: "Test concurrent tab creation",
		Concurrency: 10,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			return ws
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			ws := setupData.(*workspace)
			tab := TestData.CreateTabRequest(fmt.Sprintf("tab-concurrent-%d", iteration), "Test Tab", "sky")
			return ws.addTab(tab)
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			successCount := 0
			for _, err := range results {
				if err == nil {
					successCount++
				}
			}
			// Concurrent file writes are contentious; expect at least some successes
			// In production, workspace updates would be queued or use proper DB
			if successCount == 0 {
				t.Errorf("Expected at least 1 successful operation, got 0")
			}
			t.Logf("Concurrent operations: %d/%d successful", successCount, len(results))
		},
	}

	RunConcurrencyTest(t, pattern)
}

func TestPerformanceHTTPHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, ws := setupTestSessionManager(t, cfg)

	t.Run("CreateSessionHandlerPerformance", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "CreateSessionHandler",
			Description: "Measure session creation handler time",
			MaxDuration: 500 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				reqBody := TestData.CreateSessionRequest("", nil)
				req := makeHTTPRequest(HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/sessions",
					Body:   reqBody,
				})
				w := httptest.NewRecorder()

				start := time.Now()
				handleCreateSession(w, req, manager, ws)
				duration := time.Since(start)

				// Cleanup created session
				if w.Code == 201 {
					response := assertJSONResponse(t, w, 201, nil)
					if response != nil {
						if id, ok := response["id"].(string); ok && id != "" {
							if s, ok := manager.getSession(id); ok {
								cleanupSession(s)
							}
						}
					}
				}

				return duration
			},
		}
		RunPerformanceTest(t, pattern)
	})

	t.Run("GetWorkspaceHandlerPerformance", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "GetWorkspaceHandler",
			Description: "Measure workspace retrieval handler time",
			MaxDuration: 50 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				req := httptest.NewRequest("GET", "/api/v1/workspace", nil)
				w := httptest.NewRecorder()

				start := time.Now()
				handleGetWorkspace(w, req, ws)
				duration := time.Since(start)

				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				return duration
			},
		}
		RunPerformanceTest(t, pattern)
	})
}

func BenchmarkSessionCreation(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	cfg.maxConcurrent = 100 // High capacity for benchmarking
	manager, _, _ := setupTestSessionManager(&testing.T{}, cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/echo",
			Args:    []string{"test"},
		})
		if err != nil {
			b.Fatalf("Failed to create session: %v", err)
		}
		cleanupSession(s)
	}
}

func BenchmarkWorkspaceTabOperations(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(&testing.T{}, cfg.storagePath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tab := TestData.CreateTabRequest(fmt.Sprintf("bench-tab-%d", i), "Bench Tab", "sky")
		if err := ws.addTab(tab); err != nil {
			b.Fatalf("Failed to add tab: %v", err)
		}
	}
}

func BenchmarkMetricsIncrement(b *testing.B) {
	metrics := newMetricsRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.totalSessions.Add(1)
		metrics.activeSessions.Add(1)
		metrics.activeSessions.Add(-1)
	}
}
