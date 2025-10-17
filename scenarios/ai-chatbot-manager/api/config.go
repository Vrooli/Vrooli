package main

import (
	"fmt"
	"os"
	"time"
)

const (
	apiVersion         = "1.0.0"
	serviceName        = "ai-chatbot-manager"
	httpTimeout        = 30 * time.Second
	maxDBConnections   = 25
	maxIdleConnections = 5
	connMaxLifetime    = 5 * time.Minute
)

// Config holds all configuration for the service
type Config struct {
	APIPort      string
	APIBaseURL   string
	OllamaURL    string
	DatabaseURL  string
	DatabaseHost string
	DatabasePort string
	DatabaseUser string
	DatabasePass string
	DatabaseName string
}

// LoadConfig loads configuration from environment variables with NO DEFAULTS
func LoadConfig() (*Config, error) {
	// Check lifecycle management
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		return nil, fmt.Errorf(`this binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start ai-chatbot-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported`)
	}

	cfg := &Config{}

	// API Port - REQUIRED, no default
	cfg.APIPort = os.Getenv("API_PORT")
	if cfg.APIPort == "" {
		return nil, fmt.Errorf("API_PORT environment variable is required")
	}

	// API Base URL - OPTIONAL, used for widget embed code generation
	cfg.APIBaseURL = os.Getenv("API_BASE_URL")

	// Ollama URL - REQUIRED, no default
	cfg.OllamaURL = os.Getenv("OLLAMA_URL")
	if cfg.OllamaURL == "" {
		return nil, fmt.Errorf("OLLAMA_URL environment variable is required")
	}

	// Database configuration - try POSTGRES_URL first
	cfg.DatabaseURL = os.Getenv("POSTGRES_URL")
	if cfg.DatabaseURL == "" {
		// Build from components - ALL REQUIRED
		cfg.DatabaseHost = os.Getenv("POSTGRES_HOST")
		cfg.DatabasePort = os.Getenv("POSTGRES_PORT")
		cfg.DatabaseUser = os.Getenv("POSTGRES_USER")
		cfg.DatabasePass = os.Getenv("POSTGRES_PASSWORD")
		cfg.DatabaseName = os.Getenv("POSTGRES_DB")

		if cfg.DatabaseHost == "" || cfg.DatabasePort == "" ||
			cfg.DatabaseUser == "" || cfg.DatabasePass == "" || cfg.DatabaseName == "" {
			return nil, fmt.Errorf("database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			cfg.DatabaseUser, cfg.DatabasePass, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)
	}

	return cfg, nil
}
