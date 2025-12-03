// Package config handles runtime configuration loading
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config holds runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	dbURL, err := resolveDatabaseURL()
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

// resolveDatabaseURL builds the database URL from environment variables
func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}
