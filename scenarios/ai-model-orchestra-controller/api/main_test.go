package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Initialize logger for tests
func init() {
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with default value
	result := getEnvOrDefault("NONEXISTENT_KEY", "default")
	assert.Equal(t, "default", result)
}

func TestGetSystemMetrics(t *testing.T) {
	metrics := getSystemMetrics()
	
	// Verify expected keys exist
	assert.Contains(t, metrics, "memoryPressure")
	assert.Contains(t, metrics, "availableMemoryGb")
	assert.Contains(t, metrics, "totalMemoryGb")
	assert.Contains(t, metrics, "usedMemoryGb")
	assert.Contains(t, metrics, "cpuUsage")
	
	// Verify types
	assert.IsType(t, float64(0), metrics["memoryPressure"])
	assert.IsType(t, float64(0), metrics["availableMemoryGb"])
	assert.IsType(t, float64(0), metrics["totalMemoryGb"])
	assert.IsType(t, float64(0), metrics["usedMemoryGb"])
	
	// Verify memory pressure is between 0 and 1
	memoryPressure := metrics["memoryPressure"].(float64)
	assert.GreaterOrEqual(t, memoryPressure, 0.0)
	assert.LessOrEqual(t, memoryPressure, 1.0)
}

func TestGetDefaultModelCapabilities(t *testing.T) {
	capabilities := getDefaultModelCapabilities()
	
	// Should have at least one model
	assert.Greater(t, len(capabilities), 0)
	
	// Check first model has required fields
	model := capabilities[0]
	assert.NotEmpty(t, model.ModelName)
	assert.Greater(t, len(model.Capabilities), 0)
	assert.Greater(t, model.RamRequiredGB, 0.0)
	assert.NotEmpty(t, model.Speed)
	assert.NotEmpty(t, model.QualityTier)
	assert.GreaterOrEqual(t, model.CostPer1KTokens, 0.0)
}

func TestConvertOllamaModelsToCapabilities(t *testing.T) {
	// Test with mock Ollama models
	ollamaModels := []OllamaModel{
		{
			Name: "llama3.2:1b",
			Size: 1024 * 1024 * 1024, // 1GB
		},
		{
			Name: "codellama:7b",
			Size: 7 * 1024 * 1024 * 1024, // 7GB
		},
	}
	
	capabilities := convertOllamaModelsToCapabilities(ollamaModels)
	
	assert.Equal(t, 2, len(capabilities))
	
	// Check first model (small model)
	smallModel := capabilities[0]
	assert.Equal(t, "llama3.2:1b", smallModel.ModelName)
	assert.Equal(t, "fast", smallModel.Speed)
	assert.Equal(t, "basic", smallModel.QualityTier)
	assert.Equal(t, 1.0, smallModel.RamRequiredGB)
	
	// Check second model (large model)
	largeModel := capabilities[1]
	assert.Equal(t, "codellama:7b", largeModel.ModelName)
	assert.Equal(t, "slow", largeModel.Speed)
	assert.Equal(t, "high", largeModel.QualityTier)
	assert.Equal(t, 7.0, largeModel.RamRequiredGB)
	assert.Contains(t, largeModel.Capabilities, "analysis") // Large models get analysis capability
}

func TestCircuitBreaker(t *testing.T) {
	cb := &CircuitBreaker{
		maxFailures:  3,
		resetTimeout: time.Second,
		state:        Closed,
	}
	
	// Test initial state
	assert.Equal(t, Closed, cb.GetState())
	assert.Nil(t, cb.CanRequest())
	
	// Test recording failures
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, Closed, cb.GetState()) // Still closed after 2 failures
	
	cb.RecordFailure() // Third failure should open circuit
	assert.Equal(t, Open, cb.GetState())
	assert.Error(t, cb.CanRequest()) // Should reject requests when open
	
	// Test success resets the circuit
	cb.RecordSuccess()
	assert.Equal(t, Closed, cb.GetState())
	assert.Equal(t, 0, cb.failures)
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := &CircuitBreaker{
		maxFailures:  2,
		resetTimeout: 10 * time.Millisecond, // Very short timeout for testing
		state:        Closed,
	}
	
	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, Open, cb.GetState())
	
	// Wait for reset timeout
	time.Sleep(15 * time.Millisecond)
	
	// Should transition to half-open on next request check
	assert.Nil(t, cb.CanRequest())
	assert.Equal(t, HalfOpen, cb.GetState())
}

func TestSelectOptimalModelBasic(t *testing.T) {
	// This test requires no external dependencies
	response, err := selectOptimalModel("completion", map[string]interface{}{})
	
	// Should not error even without Ollama connection (uses defaults)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.SelectedModel)
	assert.Equal(t, "completion", response.TaskType)
	assert.NotNil(t, response.SystemMetrics)
	assert.NotNil(t, response.ModelInfo)
}

func TestSelectOptimalModelWithRequirements(t *testing.T) {
	requirements := map[string]interface{}{
		"complexity":    "complex",
		"priority":      "high",
		"costLimit":     0.005,
		"maxTokens":     float64(1000),
	}
	
	response, err := selectOptimalModel("completion", requirements)
	
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.SelectedModel)
	
	// With cost limit, should prefer cost-effective models
	modelInfo := response.ModelInfo
	assert.NotNil(t, modelInfo)
	
	if costPer1K, ok := modelInfo["cost_per_1k"].(float64); ok {
		assert.LessOrEqual(t, costPer1K, 0.005)
	}
}

func TestSelectOptimalModelHighMemoryPressure(t *testing.T) {
	// This would require mocking getSystemMetrics to return high memory pressure
	// For now, just test that it doesn't crash
	response, err := selectOptimalModel("reasoning", map[string]interface{}{})
	
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.SelectedModel)
}

// Benchmark tests
func BenchmarkGetSystemMetrics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getSystemMetrics()
	}
}

func BenchmarkConvertOllamaModelsToCapabilities(b *testing.B) {
	ollamaModels := []OllamaModel{
		{Name: "llama3.2:1b", Size: 1024 * 1024 * 1024},
		{Name: "llama3.2:3b", Size: 3 * 1024 * 1024 * 1024},
		{Name: "llama3.2:8b", Size: 8 * 1024 * 1024 * 1024},
		{Name: "codellama:7b", Size: 7 * 1024 * 1024 * 1024},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertOllamaModelsToCapabilities(ollamaModels)
	}
}