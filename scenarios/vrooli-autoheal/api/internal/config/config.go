// Package config handles runtime configuration loading
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/api-core/database"
)

// Config holds runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	dbURL, err := database.ResolvePostgresDSN(nil)
	if err != nil {
		return nil, err
	}

	port := requireEnv("API_PORT")
	if port == "" {
		return nil, fmt.Errorf("API_PORT environment variable is required")
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
	}, nil
}

// requireEnv returns the value of an environment variable, or empty string if not set
func requireEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// resolveDatabaseURL delegates URL precedence and component reconstruction to
// api-core/database, the owning Postgres environment seam.
func resolveDatabaseURL() (string, error) {
	return database.ResolvePostgresDSN(nil)
}
