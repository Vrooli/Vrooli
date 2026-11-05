package main

import (
	"fmt"
	"os"
	"time"

	"github.com/vrooli/api-core/database"
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

	// Database configuration is owned by api-core/database. Keeping the
	// precedence and SSL mode in one seam prevents variant and platform drift.
	var err error
	cfg.DatabaseURL, err = database.ResolvePostgresDSN(os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("database configuration missing: %w", err)
	}

	return cfg, nil
}
