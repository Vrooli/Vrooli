// DOC: docs/reference/configuration.md
//
// Package config provides centralized configuration for the Lifestyle Dashboard API.
// All tunable levers are defined here with sensible defaults, making it easy for
// operators and agents to understand and adjust behavior without code changes.
//
// Configuration values can be overridden via environment variables where noted.
// Each setting documents its impact and valid range.
package config

import (
	"os"
	"strconv"
	"time"
)

// =============================================================================
// Database Configuration
// =============================================================================

// DatabaseConfig contains database connection and retry settings.
// These values are optimized for SQLite's single-writer constraint.
type DatabaseConfig struct {
	// MaxOpenConns sets the maximum number of open database connections.
	// For SQLite with WAL mode, this should be 1 to enforce single-writer.
	// Higher values may cause "database is locked" errors.
	// Default: 1, Valid: 1 (SQLite), 1-100 (PostgreSQL)
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of idle connections in the pool.
	// Should be <= MaxOpenConns to avoid wasted resources.
	// Default: 1, Valid: 0-100
	MaxIdleConns int

	// BusyTimeout sets how long SQLite waits for locks before returning BUSY.
	// Higher values reduce contention errors but increase latency under load.
	// Default: 5s, Valid: 1s-60s
	BusyTimeout time.Duration

	// RetryMaxAttempts is the number of connection attempts before failing.
	// For local SQLite, 3 is usually sufficient. For network databases, use 5+.
	// Default: 3, Valid: 1-10
	RetryMaxAttempts int

	// RetryBaseDelay is the initial delay between retry attempts.
	// Uses exponential backoff with jitter, so actual delays will vary.
	// Default: 100ms, Valid: 10ms-1s
	RetryBaseDelay time.Duration

	// RetryMaxDelay caps the maximum delay between retry attempts.
	// Prevents excessive waits during extended outages.
	// Default: 500ms (SQLite), 30s (network databases)
	RetryMaxDelay time.Duration
}

// DefaultDatabaseConfig returns the default database configuration for SQLite.
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		BusyTimeout:      5 * time.Second,
		RetryMaxAttempts: 3,
		RetryBaseDelay:   100 * time.Millisecond,
		RetryMaxDelay:    500 * time.Millisecond,
	}
}

// =============================================================================
// Query Configuration
// =============================================================================

// QueryConfig contains settings for API query behavior.
type QueryConfig struct {
	// DefaultEventLimit is the maximum number of events returned when
	// no limit is specified. Prevents accidental large result sets.
	// Clients can override with ?limit=N up to MaxEventLimit.
	// Default: 100, Valid: 10-1000
	DefaultEventLimit int

	// MaxEventLimit is the absolute maximum events a client can request.
	// Protects against denial-of-service via expensive queries.
	// Default: 1000, Valid: 100-10000
	MaxEventLimit int

	// DefaultTimelineDays is the number of days shown in timeline views
	// when no days parameter is specified.
	// Default: 7, Valid: 1-365
	DefaultTimelineDays int

	// MaxTimelineDays caps the maximum timeline range to prevent
	// expensive aggregation queries.
	// Default: 365, Valid: 30-3650
	MaxTimelineDays int
}

// DefaultQueryConfig returns the default query configuration.
func DefaultQueryConfig() QueryConfig {
	cfg := QueryConfig{
		DefaultEventLimit:   100,
		MaxEventLimit:       1000,
		DefaultTimelineDays: 7,
		MaxTimelineDays:     365,
	}

	// Allow environment override for default event limit
	if v := os.Getenv("LD_DEFAULT_EVENT_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.DefaultEventLimit = n
		}
	}

	// Allow environment override for default timeline days
	if v := os.Getenv("LD_DEFAULT_TIMELINE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.DefaultTimelineDays = n
		}
	}

	return cfg
}

// =============================================================================
// Health Check Configuration
// =============================================================================

// HealthCheckConfig contains settings for domain health checks.
type HealthCheckConfig struct {
	// Timeout is how long to wait for a domain's health endpoint to respond.
	// Set lower for quick feedback, higher for slow services.
	// Default: 5s, Valid: 1s-30s
	Timeout time.Duration

	// UnhealthyThreshold is the HTTP status code threshold above which
	// a response is considered unhealthy. Status >= this value = unhealthy.
	// Default: 300, Valid: 300-600
	UnhealthyThreshold int
}

// DefaultHealthCheckConfig returns the default health check configuration.
func DefaultHealthCheckConfig() HealthCheckConfig {
	cfg := HealthCheckConfig{
		Timeout:            5 * time.Second,
		UnhealthyThreshold: 300,
	}

	// Allow environment override for timeout
	if v := os.Getenv("LD_HEALTH_CHECK_TIMEOUT_SECS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Timeout = time.Duration(n) * time.Second
		}
	}

	return cfg
}

// =============================================================================
// CORS Configuration
// =============================================================================

// CORSConfig contains settings for Cross-Origin Resource Sharing.
type CORSConfig struct {
	// AllowedOrigins lists origins that can make cross-origin requests.
	// Empty slice means accept any origin (development mode).
	// In production, restrict to specific UI domains.
	// Default: [] (allow any), Production: ["https://your-domain.com"]
	AllowedOrigins []string

	// AllowCredentials determines if credentials (cookies, auth headers)
	// can be included in cross-origin requests.
	// Default: true
	AllowCredentials bool
}

// DefaultCORSConfig returns the default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // Allow any in development
		AllowCredentials: true,
	}
}

// =============================================================================
// Scoring Configuration
// =============================================================================

// ScoringConfig contains settings for lifestyle score calculation and trends.
// These thresholds determine how changes are categorized and displayed.
type ScoringConfig struct {
	// TrendThreshold is the minimum percentage change to indicate up/down trend.
	// Changes within ±TrendThreshold are considered "stable".
	// Default: 5.0, Valid: 1.0-20.0
	TrendThreshold float64

	// DomainTrendThreshold is the threshold for individual domain direction.
	// Typically higher than composite trend since domains are more volatile.
	// Default: 10.0, Valid: 5.0-30.0
	DomainTrendThreshold float64

	// NotableChangeThreshold marks domain changes as notable when exceeded.
	// Notable changes appear in highlights and focus recommendations.
	// Default: 20.0, Valid: 10.0-50.0
	NotableChangeThreshold float64

	// DataQualityGoodThreshold is minimum active domains for "good" quality.
	// Default: 3, Valid: 2-5
	DataQualityGoodThreshold int

	// DataQualityLimitedThreshold is minimum domains for "limited" quality.
	// Below this is "insufficient".
	// Default: 1, Valid: 1-3
	DataQualityLimitedThreshold int

	// ScoreExcellentThreshold is the score level for "Excellent day!" message.
	// Default: 80, Valid: 70-95
	ScoreExcellentThreshold int

	// ScoreGoodThreshold is the score level for "Good progress" message.
	// Default: 60, Valid: 50-80
	ScoreGoodThreshold int

	// ScoreModerateThreshold is the score level for "Moderate activity" message.
	// Below this shows "Light activity".
	// Default: 40, Valid: 30-60
	ScoreModerateThreshold int

	// PointsPerEvent is how many score points each event contributes.
	// Default: 20, Valid: 10-50
	PointsPerEvent int

	// MaxDomainScore caps the score any single domain can contribute.
	// Default: 100, Valid: 50-100
	MaxDomainScore int
}

// DefaultScoringConfig returns the default scoring configuration.
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		TrendThreshold:              5.0,
		DomainTrendThreshold:        10.0,
		NotableChangeThreshold:      20.0,
		DataQualityGoodThreshold:    3,
		DataQualityLimitedThreshold: 1,
		ScoreExcellentThreshold:     80,
		ScoreGoodThreshold:          60,
		ScoreModerateThreshold:      40,
		PointsPerEvent:              20,
		MaxDomainScore:              100,
	}
}

// =============================================================================
// Full Configuration
// =============================================================================

// Config aggregates all configuration sections.
// Use Load() to create with defaults and environment overrides.
type Config struct {
	Database    DatabaseConfig
	Query       QueryConfig
	HealthCheck HealthCheckConfig
	CORS        CORSConfig
	Scoring     ScoringConfig
}

// Load creates a Config with defaults and applies environment overrides.
// This is the primary entry point for configuration.
func Load() Config {
	return Config{
		Database:    DefaultDatabaseConfig(),
		Query:       DefaultQueryConfig(),
		HealthCheck: DefaultHealthCheckConfig(),
		CORS:        DefaultCORSConfig(),
		Scoring:     DefaultScoringConfig(),
	}
}
