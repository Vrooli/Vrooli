// Package config provides tunable configuration levers for the reference-react-vite API.
// All meaningful behavior controls are exposed here with sensible defaults.
//
// DOC: docs/reference/configuration.md#tunable-levers
package config

import (
	"os"
	"strconv"
)

// Pagination controls list endpoint behavior.
// These affect all list operations (tasks, projects, notes).
type Pagination struct {
	// DefaultLimit is the number of items returned when no limit is specified.
	// Higher values show more data at once; lower values reduce response size.
	// Range: 1-MaxLimit, Default: 20
	DefaultLimit int

	// MaxLimit is the maximum allowed limit parameter.
	// Prevents clients from requesting excessively large result sets.
	// Range: 1-1000, Default: 100
	MaxLimit int
}

// Validation controls input validation constraints.
// These define the boundaries for user-provided data.
type Validation struct {
	// TaskTitleMaxLength is the maximum characters allowed for task titles.
	// Longer titles are rejected. Default: 255
	TaskTitleMaxLength int

	// ProjectNameMaxLength is the maximum characters allowed for project names.
	// Longer names are rejected. Default: 100
	ProjectNameMaxLength int

	// NoteContentMaxLength is the maximum characters allowed for note content.
	// Longer content is rejected. Default: 10000
	NoteContentMaxLength int
}

// CORS controls Cross-Origin Resource Sharing behavior.
type CORS struct {
	// AllowedOrigins is a comma-separated list of allowed origins.
	// Use "http://localhost:*" for development (matches any localhost port).
	// Use specific origins like "https://example.com" for production.
	// Use "*" to allow all origins (not recommended for production).
	// Default: "http://localhost:*"
	AllowedOrigins string

	// MaxAge is the number of seconds browsers should cache preflight responses.
	// Higher values reduce preflight requests; lower values update CORS faster.
	// Range: 0-86400, Default: 86400 (24 hours)
	MaxAge int
}

// Server controls server-level settings.
type Server struct {
	// HealthVersion is the version string reported in health check responses.
	// Should match the deployed version for monitoring/debugging.
	// Default: "1.0.0"
	HealthVersion string
}

// Config holds all tunable configuration for the API.
type Config struct {
	Pagination Pagination
	Validation Validation
	CORS       CORS
	Server     Server
}

// Default returns the default configuration values.
// These are designed for development and serve as sensible production defaults.
func Default() *Config {
	return &Config{
		Pagination: Pagination{
			DefaultLimit: 20,
			MaxLimit:     100,
		},
		Validation: Validation{
			TaskTitleMaxLength:   255,
			ProjectNameMaxLength: 100,
			NoteContentMaxLength: 10000,
		},
		CORS: CORS{
			AllowedOrigins: "http://localhost:*",
			MaxAge:         86400,
		},
		Server: Server{
			HealthVersion: "1.0.0",
		},
	}
}

// LoadFromEnv loads configuration from environment variables, falling back to defaults.
// Environment variables:
//   - PAGINATION_DEFAULT_LIMIT: Default number of items per page
//   - PAGINATION_MAX_LIMIT: Maximum allowed items per page
//   - VALIDATION_TASK_TITLE_MAX: Max characters for task titles
//   - VALIDATION_PROJECT_NAME_MAX: Max characters for project names
//   - VALIDATION_NOTE_CONTENT_MAX: Max characters for note content
//   - CORS_ALLOWED_ORIGINS: Comma-separated allowed origins
//   - CORS_MAX_AGE: Preflight cache duration in seconds
//   - HEALTH_VERSION: Version string for health checks
func LoadFromEnv() *Config {
	cfg := Default()

	// Pagination
	if v := os.Getenv("PAGINATION_DEFAULT_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Pagination.DefaultLimit = n
		}
	}
	if v := os.Getenv("PAGINATION_MAX_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Pagination.MaxLimit = n
		}
	}

	// Validation
	if v := os.Getenv("VALIDATION_TASK_TITLE_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Validation.TaskTitleMaxLength = n
		}
	}
	if v := os.Getenv("VALIDATION_PROJECT_NAME_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Validation.ProjectNameMaxLength = n
		}
	}
	if v := os.Getenv("VALIDATION_NOTE_CONTENT_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Validation.NoteContentMaxLength = n
		}
	}

	// CORS
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.CORS.AllowedOrigins = v
	}
	if v := os.Getenv("CORS_MAX_AGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.CORS.MaxAge = n
		}
	}

	// Server
	if v := os.Getenv("HEALTH_VERSION"); v != "" {
		cfg.Server.HealthVersion = v
	}

	return cfg
}
