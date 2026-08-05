// Package server provides HTTP server setup, configuration, and routing.
package server

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

// Config holds minimal runtime configuration.
type Config struct {
	Port        string
	DatabaseURL string
}

// RequireEnv retrieves a required environment variable or exits with an error.
func RequireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

// ResolveDatabaseURL constructs a database connection URL from environment variables.
func ResolveDatabaseURL() (string, error) {
	// Prefer explicit DATABASE_URL if set
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		if raw == "" {
			return "", fmt.Errorf("DATABASE_URL is set but empty")
		}
		return raw, nil
	}

	// Fall back to individual components
	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or all of POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
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
