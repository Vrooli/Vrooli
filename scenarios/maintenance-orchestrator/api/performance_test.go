package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformance_ScenarioActivation tests activation performance
func TestPerformance_ScenarioActivation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create 100 test scenarios
	for i := 0; i < 100; i++ {
		scenario := createTestScenario(
			fmt.Sprintf("perf-scenario-%d", i),
			fmt.Sprintf("perf-scenario-%d", i),
			fmt.Sprintf("Performance Test Scenario %d", i),
			[]string{"test", "performance"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)
	}

	t.Run("ActivateMultipleScenarios", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			scenarioID := fmt.Sprintf("perf-scenario-%d", i)
			env.Orchestrator.ActivateScenario(scenarioID)
		}

		duration := time.Since(start)

		// Should complete in under 100ms for 50 activations
		if duration > 100*time.Millisecond {
			t.Errorf("Activation took too long: %v (expected < 100ms)", duration)
		}

		t.Logf("Activated 50 scenarios in %v", duration)
	})

	t.Run("DeactivateMultipleScenarios", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			scenarioID := fmt.Sprintf("perf-scenario-%d", i)
			env.Orchestrator.DeactivateScenario(scenarioID)
		}

		duration := time.Since(start)

		// Should complete in under 100ms for 50 deactivations
		if duration > 100*time.Millisecond {
			t.Errorf("Deactivation took too long: %v (expected < 100ms)", duration)
		}

		t.Logf("Deactivated 50 scenarios in %v", duration)
	})
}

// TestPerformance_ConcurrentActivation tests concurrent scenario operations
func TestPerformance_ConcurrentActivation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test scenarios
	for i := 0; i < 50; i++ {
		scenario := createTestScenario(
			fmt.Sprintf("concurrent-scenario-%d", i),
			fmt.Sprintf("concurrent-scenario-%d", i),
			fmt.Sprintf("Concurrent Test Scenario %d", i),
			[]string{"test", "concurrent"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)
	}

	t.Run("ConcurrentActivation", func(t *testing.T) {
		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				scenarioID := fmt.Sprintf("concurrent-scenario-%d", idx)
				env.Orchestrator.ActivateScenario(scenarioID)
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		// Should handle concurrent operations efficiently
		if duration > 200*time.Millisecond {
			t.Errorf("Concurrent activation took too long: %v (expected < 200ms)", duration)
		}

		t.Logf("Activated 50 scenarios concurrently in %v", duration)

		// Verify all are active
		activeCount := 0
		scenarios := env.Orchestrator.GetScenarios()
		for _, s := range scenarios {
			if s.IsActive {
				activeCount++
			}
		}

		if activeCount != 50 {
			t.Errorf("Expected 50 active scenarios, got %d", activeCount)
		}
	})
}

// TestPerformance_PresetApplication tests preset application performance
func TestPerformance_PresetApplication(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create scenarios
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		scenarioID := fmt.Sprintf("preset-perf-scenario-%d", i)
		scenario := createTestScenario(
			scenarioID,
			scenarioID,
			fmt.Sprintf("Preset Performance Scenario %d", i),
			[]string{"test", "preset-perf"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)

		// Include 50 scenarios in preset
		if i < 50 {
			states[scenarioID] = true
		}
	}

	// Create large preset
	preset := createTestPreset("large-preset", "Large Preset", "Preset with 50 scenarios", states)
	env.Orchestrator.presets[preset.ID] = preset

	t.Run("ApplyLargePreset", func(t *testing.T) {
		start := time.Now()

		env.Orchestrator.ApplyPreset("large-preset")

		duration := time.Since(start)

		// Should apply preset with 50 scenarios in under 150ms
		if duration > 150*time.Millisecond {
			t.Errorf("Preset application took too long: %v (expected < 150ms)", duration)
		}

		t.Logf("Applied preset with 50 scenarios in %v", duration)

		// Verify correct scenarios are active
		activeCount := 0
		scenarios := env.Orchestrator.GetScenarios()
		for _, s := range scenarios {
			if s.IsActive {
				activeCount++
			}
		}

		if activeCount != 50 {
			t.Errorf("Expected 50 active scenarios, got %d", activeCount)
		}
	})
}

// TestPerformance_APIEndpoints tests API endpoint response times
func TestPerformance_APIEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	populateTestScenarios(env.Orchestrator)
	initializeDefaultPresets(env.Orchestrator)

	endpoints := []struct {
		name   string
		method string
		path   string
		maxMS  int64
	}{
		{"GetScenarios", "GET", "/api/v1/scenarios", 50},
		{"GetPresets", "GET", "/api/v1/presets", 50},
		{"GetStatus", "GET", "/api/v1/status", 100},
		{"GetHealth", "GET", "/health", 20},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name+"_ResponseTime", func(t *testing.T) {
			start := time.Now()

			req := HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
			}

			w := makeHTTPRequest(env, req)
			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			durationMS := duration.Milliseconds()
			if durationMS > endpoint.maxMS {
				t.Errorf("%s took too long: %dms (expected < %dms)",
					endpoint.name, durationMS, endpoint.maxMS)
			}

			t.Logf("%s completed in %v", endpoint.name, duration)
		})
	}
}

// TestPerformance_StopAll tests stop-all performance with many active scenarios
func TestPerformance_StopAll(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create and activate many scenarios
	for i := 0; i < 100; i++ {
		scenario := createTestScenario(
			fmt.Sprintf("stopall-scenario-%d", i),
			fmt.Sprintf("stopall-scenario-%d", i),
			fmt.Sprintf("Stop All Test Scenario %d", i),
			[]string{"test", "stopall"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)
		env.Orchestrator.ActivateScenario(scenario.ID)
	}

	t.Run("StopAllPerformance", func(t *testing.T) {
		start := time.Now()

		deactivated := env.Orchestrator.StopAll()

		duration := time.Since(start)

		if len(deactivated) != 100 {
			t.Errorf("Expected 100 deactivated scenarios, got %d", len(deactivated))
		}

		// Should stop all 100 scenarios in under 200ms
		if duration > 200*time.Millisecond {
			t.Errorf("StopAll took too long: %v (expected < 200ms)", duration)
		}

		t.Logf("Stopped 100 scenarios in %v", duration)

		// Verify all are inactive
		scenarios := env.Orchestrator.GetScenarios()
		for _, s := range scenarios {
			if s.IsActive {
				t.Errorf("Scenario %s should be inactive", s.ID)
			}
		}
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LargeNumberOfScenarios", func(t *testing.T) {
		// Create 1000 scenarios
		for i := 0; i < 1000; i++ {
			scenario := createTestScenario(
				fmt.Sprintf("mem-scenario-%d", i),
				fmt.Sprintf("mem-scenario-%d", i),
				fmt.Sprintf("Memory Test Scenario %d", i),
				[]string{"test", "memory"},
				false,
			)
			env.Orchestrator.AddScenario(scenario)
		}

		scenarios := env.Orchestrator.GetScenarios()
		if len(scenarios) != 1000 {
			t.Errorf("Expected 1000 scenarios, got %d", len(scenarios))
		}

		// Quick activation/deactivation cycle
		start := time.Now()
		for i := 0; i < 100; i++ {
			scenarioID := fmt.Sprintf("mem-scenario-%d", i)
			env.Orchestrator.ActivateScenario(scenarioID)
		}
		for i := 0; i < 100; i++ {
			scenarioID := fmt.Sprintf("mem-scenario-%d", i)
			env.Orchestrator.DeactivateScenario(scenarioID)
		}
		duration := time.Since(start)

		t.Logf("Completed 200 operations on 1000-scenario orchestrator in %v", duration)

		if duration > 300*time.Millisecond {
			t.Errorf("Operations took too long with many scenarios: %v", duration)
		}
	})
}

// TestPerformance_ConcurrentPresetApplication tests concurrent preset operations
func TestPerformance_ConcurrentPresetApplication(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create scenarios
	for i := 0; i < 50; i++ {
		scenario := createTestScenario(
			fmt.Sprintf("preset-concurrent-%d", i),
			fmt.Sprintf("preset-concurrent-%d", i),
			fmt.Sprintf("Preset Concurrent %d", i),
			[]string{"test"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)
	}

	// Create multiple presets
	for i := 0; i < 10; i++ {
		states := make(map[string]bool)
		for j := i * 5; j < (i+1)*5 && j < 50; j++ {
			states[fmt.Sprintf("preset-concurrent-%d", j)] = true
		}
		preset := createTestPreset(
			fmt.Sprintf("preset-%d", i),
			fmt.Sprintf("Preset %d", i),
			fmt.Sprintf("Test preset %d", i),
			states,
		)
		env.Orchestrator.presets[preset.ID] = preset
	}

	t.Run("ConcurrentPresetApplication", func(t *testing.T) {
		start := time.Now()
		var wg sync.WaitGroup

		// Apply presets concurrently (though only one should be active at a time)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				presetID := fmt.Sprintf("preset-%d", idx)
				env.Orchestrator.ApplyPreset(presetID)
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Applied 10 presets concurrently in %v", duration)

		if duration > 300*time.Millisecond {
			t.Errorf("Concurrent preset application took too long: %v", duration)
		}
	})
}

// BenchmarkScenarioActivation benchmarks single scenario activation
func BenchmarkScenarioActivation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	scenario := createTestScenario("bench-scenario", "bench-scenario", "Benchmark Scenario", []string{"test"}, false)
	env.Orchestrator.AddScenario(scenario)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.Orchestrator.ActivateScenario("bench-scenario")
		env.Orchestrator.DeactivateScenario("bench-scenario")
	}
}

// BenchmarkPresetApplication benchmarks preset application
func BenchmarkPresetApplication(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Create scenarios
	states := make(map[string]bool)
	for i := 0; i < 10; i++ {
		scenarioID := fmt.Sprintf("bench-preset-scenario-%d", i)
		scenario := createTestScenario(scenarioID, scenarioID, fmt.Sprintf("Bench Scenario %d", i), []string{"test"}, false)
		env.Orchestrator.AddScenario(scenario)
		states[scenarioID] = true
	}

	preset := createTestPreset("bench-preset", "Benchmark Preset", "Preset for benchmarking", states)
	env.Orchestrator.presets[preset.ID] = preset

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.Orchestrator.ApplyPreset("bench-preset")
	}
}

// BenchmarkGetScenarios benchmarks scenario listing
func BenchmarkGetScenarios(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Create many scenarios
	for i := 0; i < 100; i++ {
		scenario := createTestScenario(
			fmt.Sprintf("bench-get-scenario-%d", i),
			fmt.Sprintf("bench-get-scenario-%d", i),
			fmt.Sprintf("Bench Get Scenario %d", i),
			[]string{"test"},
			false,
		)
		env.Orchestrator.AddScenario(scenario)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = env.Orchestrator.GetScenarios()
	}
}
