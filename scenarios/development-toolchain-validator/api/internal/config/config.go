// DOC: docs/reference/configuration.md#api-configuration
// Package config provides centralized configuration for the development-toolchain-validator API.
// All tunable levers are exposed here with sensible defaults, clear documentation,
// and consistent environment variable naming.
//
// Configuration Design Principles:
// 1. Prefer environment variables for deployment-time configuration
// 2. Use explicit defaults that work for common cases
// 3. Validate and constrain values to prevent misuse
// 4. Group related settings together
package config

import (
	"os"
	"strconv"
	"strings"
)

// PaginationConfig controls list endpoint behavior.
// These levers affect memory usage and response latency.
type PaginationConfig struct {
	// DefaultLimit is applied when no limit is specified or limit is invalid.
	// Higher values increase response payload size but reduce API calls.
	// Range: 1 to MaxLimit
	DefaultLimit int

	// MaxLimit is the upper bound for requested limits.
	// Prevents resource exhaustion from excessive page sizes.
	// Range: 1 to 1000 (hard maximum for safety)
	MaxLimit int
}

// ValidationConfig controls input validation constraints.
// These levers affect what values are accepted by the API.
type ValidationConfig struct {
	// SlugMinLength is the minimum allowed slug length.
	// Shorter slugs are rejected with ErrInvalidSlug.
	// Range: 1 to SlugMaxLength
	SlugMinLength int

	// SlugMaxLength is the maximum allowed slug length.
	// Longer slugs are rejected with ErrInvalidSlug.
	// Must be <= database VARCHAR limit (255).
	SlugMaxLength int
}

// CORSConfig controls cross-origin resource sharing.
// These levers affect which origins can access the API.
type CORSConfig struct {
	// AllowedOrigins is the list of origins permitted to make CORS requests.
	// In production, this should be set explicitly via CORS_ALLOWED_ORIGINS.
	AllowedOrigins []string
}

// Config holds all configuration for the API server.
type Config struct {
	Pagination PaginationConfig
	Validation ValidationConfig
	CORS       CORSConfig
}

// DefaultConfig returns configuration with sensible defaults.
// These defaults are designed for development and testing.
func DefaultConfig() Config {
	return Config{
		Pagination: PaginationConfig{
			DefaultLimit: 20,
			MaxLimit:     100,
		},
		Validation: ValidationConfig{
			SlugMinLength: 2,
			SlugMaxLength: 100,
		},
		CORS: CORSConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:5173",
				"http://127.0.0.1:3000",
				"http://127.0.0.1:5173",
			},
		},
	}
}

// LoadFromEnv creates configuration from environment variables.
// Environment variables override defaults where specified.
//
// Supported variables:
//   - DTV_PAGINATION_DEFAULT_LIMIT: Default pagination limit (default: 20)
//   - DTV_PAGINATION_MAX_LIMIT: Maximum pagination limit (default: 100)
//   - DTV_SLUG_MIN_LENGTH: Minimum slug length (default: 2)
//   - DTV_SLUG_MAX_LENGTH: Maximum slug length (default: 100)
//   - CORS_ALLOWED_ORIGINS: Comma-separated list of allowed origins
func LoadFromEnv() Config {
	cfg := DefaultConfig()

	// Pagination
	if v := os.Getenv("DTV_PAGINATION_DEFAULT_LIMIT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.Pagination.DefaultLimit = parsed
		}
	}
	if v := os.Getenv("DTV_PAGINATION_MAX_LIMIT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 1000 {
			cfg.Pagination.MaxLimit = parsed
		}
	}

	// Validation
	if v := os.Getenv("DTV_SLUG_MIN_LENGTH"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.Validation.SlugMinLength = parsed
		}
	}
	if v := os.Getenv("DTV_SLUG_MAX_LENGTH"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 255 {
			cfg.Validation.SlugMaxLength = parsed
		}
	}

	// Ensure min <= max
	if cfg.Validation.SlugMinLength > cfg.Validation.SlugMaxLength {
		cfg.Validation.SlugMinLength = cfg.Validation.SlugMaxLength
	}
	if cfg.Pagination.DefaultLimit > cfg.Pagination.MaxLimit {
		cfg.Pagination.DefaultLimit = cfg.Pagination.MaxLimit
	}

	// CORS
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		cleaned := make([]string, 0, len(origins))
		for _, o := range origins {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		if len(cleaned) > 0 {
			cfg.CORS.AllowedOrigins = cleaned
		}
	}

	return cfg
}

// ApplyPaginationLimit enforces pagination constraints on a limit value.
// Returns the constrained limit: defaults to DefaultLimit if invalid,
// caps at MaxLimit if too large.
func (c PaginationConfig) ApplyPaginationLimit(limit int) int {
	if limit <= 0 || limit > c.MaxLimit {
		return c.DefaultLimit
	}
	return limit
}

// IsValidSlugLength checks if a slug length is within configured bounds.
func (c ValidationConfig) IsValidSlugLength(length int) bool {
	return length >= c.SlugMinLength && length <= c.SlugMaxLength
}

// IsOriginAllowed checks if an origin is in the allowed list.
func (c CORSConfig) IsOriginAllowed(origin string) bool {
	for _, allowed := range c.AllowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}
