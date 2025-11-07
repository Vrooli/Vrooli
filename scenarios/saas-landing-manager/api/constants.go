package main

import "time"

// Database connection pool configuration
const (
	DBMaxOpenConns    = 25
	DBMaxIdleConns    = 5
	DBConnMaxLifetime = 5 * time.Minute
)

// Database connection retry configuration
const (
	DBMaxRetries      = 10
	DBBaseRetryDelay  = 1 * time.Second
	DBMaxRetryDelay   = 30 * time.Second
	DBJitterRatio     = 0.25
	DBRetryLogEveryN  = 3
)

// Health check configuration
const (
	HealthCheckTimeout = 2 * time.Second
)

// Service metadata
const (
	ServiceName    = "saas-landing-manager-api"
	ServiceVersion = "1.0.0"
)

// SaaS detection thresholds and scoring
const (
	SaaSConfidenceThreshold         = 0.5
	SaaSTagMatchScore               = 0.2
	SaaSPostgresScore               = 0.3
	SaaSRevenuePatternScore         = 0.1
	SaaSRevenueExtractionScore      = 0.2
	SaaSUIPresenceScore             = 0.2
	SaaSAPIPresenceScore            = 0.1
	SaaSDetectionAnalysisVersion    = "1.0"
)

// Claude Code deployment configuration
const (
	ClaudeAgentEstimatedDuration = 10 * time.Minute
	ClaudeDefaultCLI             = "resource-claude-code"
)

// Default paths
const (
	DefaultScenariosPath = "../../"
	DefaultTemplatesPath = "./templates"
)
