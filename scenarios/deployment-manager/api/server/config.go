// Package server provides HTTP server setup, configuration, and routing.
package server

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/vrooli/api-core/database"
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

// ResolveDatabaseURL delegates Postgres precedence and reconstruction to the
// package that owns the resource environment contract.
func ResolveDatabaseURL() (string, error) {
	return database.ResolvePostgresDSN(os.Getenv)
}
