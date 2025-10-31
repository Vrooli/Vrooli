package main

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds validated environment configuration
type Config struct {
	// Database configuration
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	// Qdrant configuration
	QdrantURL string

	// Ollama configuration
	OllamaURL string

	// Server configuration
	APIPort string

	// Lifecycle configuration
	LifecycleManaged bool

	// Testing configuration
	UseMockTesting bool
}

// getEnv retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired retrieves a required environment variable or returns an error
func getEnvRequired(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

// getEnvBool retrieves a boolean environment variable with validation
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		// Log warning about invalid value and return default
		if logger != nil {
			logger.Warn("Invalid boolean environment variable", map[string]interface{}{
				"key":          key,
				"value":        value,
				"default_used": defaultValue,
				"error":        err.Error(),
			})
		}
		return defaultValue
	}

	return boolValue
}

// getEnvInt retrieves an integer environment variable with validation
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		// Log warning about invalid value and return default
		if logger != nil {
			logger.Warn("Invalid integer environment variable", map[string]interface{}{
				"key":          key,
				"value":        value,
				"default_used": defaultValue,
				"error":        err.Error(),
			})
		}
		return defaultValue
	}

	return intValue
}

// LoadConfig loads and validates all environment configuration
func LoadConfig() (*Config, error) {
	config := &Config{
		// Database defaults
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "prompt_injection_arena"),

		// Service defaults
		QdrantURL: getEnv("QDRANT_URL", "http://localhost:6333"),
		OllamaURL: getEnv("OLLAMA_URL", "http://localhost:11434"),

		// Server defaults
		APIPort: getEnv("API_PORT", "16018"),

		// Lifecycle flags
		LifecycleManaged: getEnvBool("VROOLI_LIFECYCLE_MANAGED", false),
		UseMockTesting:   getEnvBool("USE_MOCK_TESTING", false),
	}

	// Validate critical configuration
	if config.PostgresHost == "" {
		return nil, fmt.Errorf("PostgresHost cannot be empty")
	}
	if config.PostgresDB == "" {
		return nil, fmt.Errorf("PostgresDB cannot be empty")
	}

	// Log loaded configuration (without sensitive values)
	if logger != nil {
		logger.Info("Configuration loaded", map[string]interface{}{
			"postgres_host":     config.PostgresHost,
			"postgres_port":     config.PostgresPort,
			"postgres_db":       config.PostgresDB,
			"qdrant_url":        config.QdrantURL,
			"ollama_url":        config.OllamaURL,
			"api_port":          config.APIPort,
			"lifecycle_managed": config.LifecycleManaged,
			"use_mock_testing":  config.UseMockTesting,
			"password_set":      config.PostgresPassword != "",
		})
	}

	return config, nil
}

// GetDatabaseConnectionString returns the PostgreSQL connection string
func (c *Config) GetDatabaseConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresDB,
	)
}
