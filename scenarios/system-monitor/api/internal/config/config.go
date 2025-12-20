package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Monitoring   MonitoringConfig
	Resources    ResourcesConfig
	Alerts       AlertConfig
	Logging      LoggingConfig
	Health       HealthConfig
	AgentManager AgentManagerConfig
}

// AgentManagerConfig contains agent-manager service configuration
type AgentManagerConfig struct {
	// ProfileName is the name to use for the system-monitor agent profile
	ProfileName string
	// Timeout for API requests
	Timeout time.Duration
	// Enabled determines whether agent-manager is available for investigations
	Enabled bool
}

// ServerConfig contains server configuration
type ServerConfig struct {
	APIPort     string
	UIPort      string
	Environment string
	Version     string
	ServiceName string
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	URL                string
	MaxOpenConnections int
	MaxIdleConnections int
	ConnMaxLifetime    time.Duration
	EnableMigrations   bool
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	MetricsInterval    time.Duration
	AnomalyInterval    time.Duration
	ReportInterval     time.Duration
	ThresholdInterval  time.Duration
	RetentionDays      int
	EnableAutoResolve  bool
	MaxInvestigations  int
}

// ResourcesConfig contains resource service configurations
type ResourcesConfig struct {
	PostgresURL    string
	RedisURL       string
	QuestDBURL     string
	NodeRedURL     string
	OllamaURL      string
	GrafanaURL     string
}

// AlertConfig contains alerting configuration
type AlertConfig struct {
	WebhookURL       string
	EnableWebhooks   bool
	EnableEmail      bool
	EmailFrom        string
	EmailSMTPHost    string
	EmailSMTPPort    int
	MinSeverity      int
	CooldownMinutes  int
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string
	Format     string // json, text
	Output     string // stdout, file
	FilePath   string
	MaxSize    int // megabytes
	MaxBackups int
	MaxAge     int // days
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Enabled            bool
	Interval           time.Duration
	Timeout            time.Duration
	StartupGracePeriod time.Duration
	Endpoints          []HealthEndpoint
}

// HealthEndpoint defines a health check endpoint
type HealthEndpoint struct {
	Name     string
	URL      string
	Critical bool
	Timeout  time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			APIPort:     getEnvOrFatal("API_PORT"),
			UIPort:      getEnv("UI_PORT", ""),
			Environment: getEnv("ENVIRONMENT", "development"),
			Version:     getEnv("VERSION", "2.0.0"),
			ServiceName: "system-monitor",
		},
		Database: DatabaseConfig{
			URL:                getEnv("DATABASE_URL", ""),
			MaxOpenConnections: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConnections: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime:    time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)) * time.Minute,
			EnableMigrations:   getEnvAsBool("DB_ENABLE_MIGRATIONS", true),
		},
		Monitoring: MonitoringConfig{
			MetricsInterval:    time.Duration(getEnvAsInt("METRICS_INTERVAL_SECONDS", 10)) * time.Second,
			AnomalyInterval:    time.Duration(getEnvAsInt("ANOMALY_INTERVAL_SECONDS", 30)) * time.Second,
			ReportInterval:     time.Duration(getEnvAsInt("REPORT_INTERVAL_HOURS", 1)) * time.Hour,
			ThresholdInterval:  time.Duration(getEnvAsInt("THRESHOLD_INTERVAL_SECONDS", 20)) * time.Second,
			RetentionDays:      getEnvAsInt("RETENTION_DAYS", 30),
			EnableAutoResolve:  getEnvAsBool("ENABLE_AUTO_RESOLVE", false),
			MaxInvestigations:  getEnvAsInt("MAX_INVESTIGATIONS", 100),
		},
		Resources: ResourcesConfig{
			PostgresURL:  getEnv("POSTGRES_URL", "postgres://localhost:5432/system_monitor"),
			RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
			QuestDBURL:   getEnv("QUESTDB_URL", "http://localhost:9000"),
			NodeRedURL:   getEnv("NODE_RED_URL", "http://localhost:1880"),
			OllamaURL:    getEnv("OLLAMA_URL", "http://localhost:11434"),
			GrafanaURL:   getEnv("GRAFANA_URL", "http://localhost:3004"),
		},
		Alerts: AlertConfig{
			WebhookURL:      getEnv("ALERT_WEBHOOK_URL", ""),
			EnableWebhooks:  getEnvAsBool("ENABLE_WEBHOOKS", false),
			EnableEmail:     getEnvAsBool("ENABLE_EMAIL_ALERTS", false),
			EmailFrom:       getEnv("EMAIL_FROM", ""),
			EmailSMTPHost:   getEnv("EMAIL_SMTP_HOST", ""),
			EmailSMTPPort:   getEnvAsInt("EMAIL_SMTP_PORT", 587),
			MinSeverity:     getEnvAsInt("MIN_ALERT_SEVERITY", 2),
			CooldownMinutes: getEnvAsInt("ALERT_COOLDOWN_MINUTES", 5),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			FilePath:   getEnv("LOG_FILE_PATH", "/var/log/system-monitor.log"),
			MaxSize:    getEnvAsInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getEnvAsInt("LOG_MAX_BACKUPS", 3),
			MaxAge:     getEnvAsInt("LOG_MAX_AGE_DAYS", 7),
		},
		Health: HealthConfig{
			Enabled:            getEnvAsBool("HEALTH_CHECKS_ENABLED", true),
			Interval:           time.Duration(getEnvAsInt("HEALTH_INTERVAL_SECONDS", 30)) * time.Second,
			Timeout:            time.Duration(getEnvAsInt("HEALTH_TIMEOUT_SECONDS", 5)) * time.Second,
			StartupGracePeriod: time.Duration(getEnvAsInt("HEALTH_STARTUP_GRACE_SECONDS", 10)) * time.Second,
			Endpoints:          loadHealthEndpoints(),
		},
		AgentManager: AgentManagerConfig{
			ProfileName: getEnv("AGENT_MANAGER_PROFILE_NAME", "system-monitor-investigator"),
			Timeout:     time.Duration(getEnvAsInt("AGENT_MANAGER_TIMEOUT_SECONDS", 30)) * time.Second,
			Enabled:     getEnvAsBool("AGENT_MANAGER_ENABLED", true),
		},
	}

	cfg.validate()
	return cfg
}

// validate ensures configuration is valid
func (c *Config) validate() {
	// Validate server config
	if c.Server.APIPort == "" {
		log.Fatal("API_PORT is required")
	}

	// Validate database config if URL is provided
	if c.Database.URL != "" {
		if c.Database.MaxOpenConnections <= 0 {
			c.Database.MaxOpenConnections = 25
		}
		if c.Database.MaxIdleConnections <= 0 {
			c.Database.MaxIdleConnections = 5
		}
	}

	// Validate monitoring intervals
	if c.Monitoring.MetricsInterval < time.Second {
		c.Monitoring.MetricsInterval = 10 * time.Second
	}
	if c.Monitoring.AnomalyInterval < time.Second {
		c.Monitoring.AnomalyInterval = 30 * time.Second
	}

	// Validate alert configuration
	if c.Alerts.EnableWebhooks && c.Alerts.WebhookURL == "" {
		log.Println("Warning: Webhooks enabled but no webhook URL provided")
		c.Alerts.EnableWebhooks = false
	}

	if c.Alerts.EnableEmail {
		if c.Alerts.EmailFrom == "" || c.Alerts.EmailSMTPHost == "" {
			log.Println("Warning: Email alerts enabled but SMTP configuration incomplete")
			c.Alerts.EnableEmail = false
		}
	}
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// HasDatabase returns true if database is configured
func (c *Config) HasDatabase() bool {
	return c.Database.URL != ""
}

// Helper functions

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// Check alternative PORT variable for backward compatibility
		if key == "API_PORT" {
			value = os.Getenv("PORT")
		}
		if value == "" {
			log.Fatalf("❌ %s environment variable is required", key)
		}
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, strValue, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(strValue)
	if err != nil {
		log.Printf("Warning: Invalid boolean value for %s: %s, using default: %v", key, strValue, defaultValue)
		return defaultValue
	}
	return value
}

func loadHealthEndpoints() []HealthEndpoint {
	// Default health endpoints
	endpoints := []HealthEndpoint{
		{
			Name:     "api",
			URL:      fmt.Sprintf("http://localhost:%s/health", os.Getenv("API_PORT")),
			Critical: true,
			Timeout:  5 * time.Second,
		},
	}

	// Add UI endpoint if configured
	if uiPort := os.Getenv("UI_PORT"); uiPort != "" {
		endpoints = append(endpoints, HealthEndpoint{
			Name:     "ui",
			URL:      fmt.Sprintf("http://localhost:%s/", uiPort),
			Critical: false,
			Timeout:  5 * time.Second,
		})
	}

	return endpoints
}
