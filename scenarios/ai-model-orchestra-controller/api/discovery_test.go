package main

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitDatabase(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Save and restore environment variables
	cleanup := setTestEnv(t, map[string]string{
		"ORCHESTRATOR_HOST":       "invalid-host",
		"RESOURCE_PORTS_POSTGRES": "5432",
		"POSTGRES_USER":           "test",
		"POSTGRES_PASSWORD":       "test",
		"POSTGRES_DB":             "test",
	})
	defer cleanup()

	t.Run("InvalidHost", func(t *testing.T) {
		db, err := initDatabase(logger)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("MissingEnvironmentVariables", func(t *testing.T) {
		// Temporarily unset environment variables
		originalHost := os.Getenv("ORCHESTRATOR_HOST")
		os.Unsetenv("ORCHESTRATOR_HOST")
		defer os.Setenv("ORCHESTRATOR_HOST", originalHost)

		db, err := initDatabase(logger)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "ORCHESTRATOR_HOST")
	})
}

func TestInitRedis(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Save and restore environment variables
	cleanup := setTestEnv(t, map[string]string{
		"ORCHESTRATOR_HOST":    "invalid-host",
		"RESOURCE_PORTS_REDIS": "6379",
	})
	defer cleanup()

	t.Run("InvalidHost", func(t *testing.T) {
		rdb, err := initRedis(logger)
		assert.Error(t, err)
		assert.Nil(t, rdb)
	})

	t.Run("MissingEnvironmentVariables", func(t *testing.T) {
		// Temporarily unset environment variables
		originalHost := os.Getenv("ORCHESTRATOR_HOST")
		os.Unsetenv("ORCHESTRATOR_HOST")
		defer os.Setenv("ORCHESTRATOR_HOST", originalHost)

		rdb, err := initRedis(logger)
		assert.Error(t, err)
		assert.Nil(t, rdb)
		assert.Contains(t, err.Error(), "ORCHESTRATOR_HOST")
	})
}

func TestInitOllama(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Save and restore environment variables
	cleanup := setTestEnv(t, map[string]string{
		"ORCHESTRATOR_HOST":     "invalid-host",
		"RESOURCE_PORTS_OLLAMA": "11434",
	})
	defer cleanup()

	t.Run("InvalidHost", func(t *testing.T) {
		client, err := initOllama(logger)
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("MissingEnvironmentVariables", func(t *testing.T) {
		// Temporarily unset environment variables
		originalHost := os.Getenv("ORCHESTRATOR_HOST")
		os.Unsetenv("ORCHESTRATOR_HOST")
		defer os.Setenv("ORCHESTRATOR_HOST", originalHost)

		client, err := initOllama(logger)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "ORCHESTRATOR_HOST")
	})
}

func TestInitSchema(t *testing.T) {
	// Skip if database not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}

	db, cleanup := createMockDatabase(t)
	if db == nil {
		return
	}
	defer cleanup()

	// Schema should already be initialized by createMockDatabase
	// Verify tables exist
	tables := []string{"model_metrics", "orchestrator_requests", "system_resources"}

	for _, table := range tables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)
		`, table).Scan(&exists)

		assert.NoError(t, err)
		assert.True(t, exists, "Table %s should exist", table)
	}
}

func TestCheckAndReconnectDatabase(t *testing.T) {
	_ = log.New(os.Stdout, "[test] ", log.LstdFlags)

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	t.Run("WithNullDatabase", func(t *testing.T) {
		testApp.App.DB = nil

		// This should attempt reconnection but fail due to invalid config
		cleanup := setTestEnv(t, map[string]string{
			"ORCHESTRATOR_HOST":       "invalid-host",
			"RESOURCE_PORTS_POSTGRES": "5432",
			"POSTGRES_USER":           "test",
			"POSTGRES_PASSWORD":       "test",
		})
		defer cleanup()

		checkAndReconnectDatabase(testApp.App)

		// DB should still be nil after failed reconnection
		assert.Nil(t, testApp.App.DB)
	})
}

func TestCheckAndReconnectRedis(t *testing.T) {
	_ = log.New(os.Stdout, "[test] ", log.LstdFlags)

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	t.Run("WithNullRedis", func(t *testing.T) {
		testApp.App.Redis = nil

		// This should attempt connection but fail due to invalid config
		cleanup := setTestEnv(t, map[string]string{
			"ORCHESTRATOR_HOST":    "invalid-host",
			"RESOURCE_PORTS_REDIS": "6379",
		})
		defer cleanup()

		checkAndReconnectRedis(testApp.App)

		// Redis should still be nil after failed connection
		assert.Nil(t, testApp.App.Redis)
	})
}

// Test discovery-related functionality from presets
func TestPresets_DefaultModels(t *testing.T) {
	models := getDefaultModelCapabilities()

	assert.Greater(t, len(models), 0)

	// Test each default model
	for _, model := range models {
		t.Run(model.ModelName, func(t *testing.T) {
			assert.NotEmpty(t, model.ModelName)
			assert.Greater(t, len(model.Capabilities), 0)
			assert.Greater(t, model.RamRequiredGB, 0.0)
			assert.NotEmpty(t, model.Speed)
			assert.NotEmpty(t, model.QualityTier)
			assert.GreaterOrEqual(t, model.CostPer1KTokens, 0.0)
			assert.Greater(t, len(model.BestFor), 0)
		})
	}
}

func TestPresets_ModelSizes(t *testing.T) {
	models := getDefaultModelCapabilities()

	// Verify we have models of different sizes
	var smallModels, mediumModels, largeModels int

	for _, model := range models {
		switch model.Speed {
		case "fast":
			smallModels++
			assert.Less(t, model.RamRequiredGB, 3.0)
			assert.Equal(t, "basic", model.QualityTier)
		case "medium":
			mediumModels++
			assert.GreaterOrEqual(t, model.RamRequiredGB, 3.0)
			assert.LessOrEqual(t, model.RamRequiredGB, 8.0) // Medium models can be up to 8GB
		case "slow":
			largeModels++
			assert.GreaterOrEqual(t, model.RamRequiredGB, 6.0)
		}
	}

	assert.Greater(t, smallModels, 0, "Should have at least one small/fast model")
	assert.Greater(t, mediumModels, 0, "Should have at least one medium model")
	assert.Greater(t, largeModels, 0, "Should have at least one large/slow model")
}

func TestPresets_ModelCapabilities(t *testing.T) {
	models := getDefaultModelCapabilities()

	// Verify we have models with different capabilities
	capabilitiesMap := make(map[string]int)

	for _, model := range models {
		for _, cap := range model.Capabilities {
			capabilitiesMap[cap]++
		}
	}

	// Should have models for common task types
	assert.Greater(t, capabilitiesMap["completion"], 0, "Should have completion models")
	assert.Greater(t, capabilitiesMap["reasoning"], 0, "Should have reasoning models")
	assert.Greater(t, capabilitiesMap["code"], 0, "Should have code models")
}

func TestConvertOllamaModels_EdgeCases(t *testing.T) {
	t.Run("EmptyList", func(t *testing.T) {
		result := convertOllamaModelsToCapabilities([]OllamaModel{})
		assert.Equal(t, 0, len(result))
	})

	t.Run("VerySmallModel", func(t *testing.T) {
		models := []OllamaModel{
			{
				Name: "tiny-model",
				Size: 500 * 1024 * 1024, // 500MB
			},
		}

		result := convertOllamaModelsToCapabilities(models)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "fast", result[0].Speed)
		assert.Equal(t, "basic", result[0].QualityTier)
	})

	t.Run("VeryLargeModel", func(t *testing.T) {
		models := []OllamaModel{
			{
				Name: "huge-model",
				Size: 70 * 1024 * 1024 * 1024, // 70GB
			},
		}

		result := convertOllamaModelsToCapabilities(models)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, "slow", result[0].Speed)
		assert.Equal(t, "high", result[0].QualityTier)
		assert.Contains(t, result[0].Capabilities, "analysis")
	})

	t.Run("EmbeddingModel", func(t *testing.T) {
		models := []OllamaModel{
			{
				Name: "nomic-embed-text",
				Size: 2 * 1024 * 1024 * 1024, // 2GB
			},
		}

		result := convertOllamaModelsToCapabilities(models)
		assert.Equal(t, 1, len(result))
		assert.Contains(t, result[0].Capabilities, "embedding")
	})
}

// Benchmark tests for discovery functions
func BenchmarkInitOllama(b *testing.B) {
	logger := log.New(os.Stdout, "[bench] ", log.LstdFlags)

	// Set environment variables for benchmark
	os.Setenv("ORCHESTRATOR_HOST", "invalid-host")
	os.Setenv("RESOURCE_PORTS_OLLAMA", "11434")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		initOllama(logger)
	}
}
