package config

// DOC: docs/reference/configuration.md

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
)

// Config holds all application configuration
type Config struct {
	Environment  Environment
	Server       ServerConfig
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
	// ProfileKey is the stable key used for profile lookup/creation
	ProfileKey string
	// Timeout for API requests
	Timeout time.Duration
	// Enabled determines whether agent-manager is available for investigations
	Enabled bool
}

// seam: Environment supplies process settings to configuration and runtime owners.
// Runtime code receives this seam from LoadWithEnvironment instead of reading
// process globals directly.
type Environment interface {
	LookupEnv(key string) (string, bool)
}

type systemEnvironment struct{}

func (systemEnvironment) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

var _ Environment = systemEnvironment{}

// ServerConfig contains server configuration
type ServerConfig struct {
	APIPort          string
	UIPort           string
	Environment      string
	Version          string
	ServiceName      string
	APIBaseURL       string
	ManagedLifecycle bool
	BridgeToken      string
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	MetricsInterval   time.Duration
	AnomalyInterval   time.Duration
	ReportInterval    time.Duration
	ThresholdInterval time.Duration
	RetentionDays     int
	EnableAutoResolve bool
	MaxInvestigations int

	// ProcSampleInterval is the cadence of the per-process /proc sampler that
	// feeds the attribution timeline (SYSTEM_MONITOR_PROC_SAMPLE_INTERVAL).
	ProcSampleInterval time.Duration
	// ProcSampleTopN caps the global per-rank process set persisted per cycle;
	// configured owners can add their own CPU and RSS leaders.
	// (SYSTEM_MONITOR_PROC_SAMPLE_TOP_N).
	ProcSampleTopN int
	// ProcSampleOwners retains each configured owner's CPU and RSS leaders in
	// addition to the global leaders (SYSTEM_MONITOR_PROC_SAMPLE_OWNERS).
	ProcSampleOwners []string
	// RawRetention is how long raw per-process rows are kept before they are
	// downsampled into per-minute rollups (SYSTEM_MONITOR_RAW_RETENTION).
	RawRetention time.Duration
	// RollupRetention is how long per-minute rollups are kept
	// (SYSTEM_MONITOR_ROLLUP_RETENTION).
	RollupRetention time.Duration
	// CollectorRetention shortens the metrics window for specific collectors
	// (SYSTEM_MONITOR_COLLECTOR_RETENTION, e.g. "device_graph=48h,network=168h").
	// The device graph is served live and its ~125 KB snapshots are only
	// forensic history; network/pressure history is read by reports for at
	// most 7 days. Other collectors keep the global metrics_retention_days.
	CollectorRetention         map[string]time.Duration
	StorageMode                string
	CompactionWindow           string
	InvestigationRoot          string
	SocketAttributionThreshold int
	GoCachePath                string
	PlaywrightCachePath        string
	RuntimeHome                string
}

// DefaultCollectorRetention is the per-collector metrics window applied when
// SYSTEM_MONITOR_COLLECTOR_RETENTION is unset.
var DefaultCollectorRetention = map[string]time.Duration{
	"device_graph": 48 * time.Hour,
	"network":      7 * 24 * time.Hour,
	"pressure":     7 * 24 * time.Hour,
}

// ParseCollectorRetention parses "name=duration,name=duration". Invalid or
// non-positive entries are skipped; an empty value yields the defaults.
func ParseCollectorRetention(raw string) map[string]time.Duration {
	out := map[string]time.Duration{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		for name, window := range DefaultCollectorRetention {
			out[name] = window
		}
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		window, err := time.ParseDuration(strings.TrimSpace(value))
		name = strings.TrimSpace(name)
		if err != nil || window <= 0 || name == "" {
			continue
		}
		out[name] = window
	}
	return out
}

const maxProcSampleOwners = 16

// ParseProcSampleOwners parses a comma-separated list of scenario owners to
// retain in every process-sampling cycle. Blank entries are ignored and
// duplicates are removed while preserving order.
func ParseProcSampleOwners(raw string) ([]string, error) {
	seen := make(map[string]struct{})
	owners := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		owner := strings.TrimSpace(part)
		if owner == "" {
			continue
		}
		if _, exists := seen[owner]; exists {
			continue
		}
		seen[owner] = struct{}{}
		owners = append(owners, owner)
		if len(owners) > maxProcSampleOwners {
			return nil, fmt.Errorf("at most %d process sample owners may be configured", maxProcSampleOwners)
		}
	}
	return owners, nil
}

// ResourcesConfig contains resource service configurations
type ResourcesConfig struct {
	NodeRedURL        string
	GrafanaURL        string
	StorageManagerURL string
	AutohealURL       string
}

// AlertConfig contains alerting configuration
type AlertConfig struct {
	WebhookURL      string
	EnableWebhooks  bool
	EnableEmail     bool
	EmailFrom       string
	EmailSMTPHost   string
	EmailSMTPPort   int
	MinSeverity     int
	CooldownMinutes int
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
	BuildIdentity      string
	OllamaBaseURL      string
}

// HealthEndpoint defines a health check endpoint
type HealthEndpoint struct {
	Name     string
	URL      string
	Critical bool
	Timeout  time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	return LoadWithEnvironment(systemEnvironment{})
}

// LoadFrom builds and validates configuration using the supplied environment
// lookup. Keeping environment access at this boundary makes configuration
// deterministic in tests and returns startup errors to the entrypoint.
func LoadFrom(lookup func(string) (string, bool)) (*Config, error) {
	if lookup == nil {
		return nil, fmt.Errorf("environment lookup is required")
	}
	return LoadWithEnvironment(environmentFunc(lookup))
}

// LoadWithEnvironment loads and validates configuration from an explicit
// environment seam. Runtime consumers can use Config.Getenv for values that
// remain intentionally dynamic after startup.
func LoadWithEnvironment(environment Environment) (*Config, error) {
	return LoadWithEnvironmentAndLogger(environment, logx.Default())
}

// LoadWithEnvironmentAndLogger makes configuration diagnostics controllable
// while retaining the environment seam as the sole settings source.
func LoadWithEnvironmentAndLogger(environment Environment, logger logx.Logger) (*Config, error) {
	if environment == nil {
		return nil, fmt.Errorf("environment lookup is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("configuration logger is required")
	}
	lookup := environment.LookupEnv
	apiPort := env(lookup, "API_PORT", "")
	if apiPort == "" {
		apiPort = env(lookup, "PORT", "")
	}
	if apiPort == "" {
		return nil, fmt.Errorf("API_PORT is required (PORT is accepted for compatibility)")
	}
	procSampleOwners, err := ParseProcSampleOwners(env(lookup, "SYSTEM_MONITOR_PROC_SAMPLE_OWNERS", ""))
	if err != nil {
		logger.Warn("invalid process sample owner configuration; using no pinned owners", "error", err)
		procSampleOwners = nil
	}
	socketThreshold := envInt(lookup, logger, "SYSTEM_MONITOR_SOCKET_ATTRIBUTION_THRESHOLD", 5000)
	if socketThreshold < 0 {
		socketThreshold = 5000
	}
	cfg := &Config{
		Environment: environment,
		Server: ServerConfig{
			APIPort:          apiPort,
			UIPort:           env(lookup, "UI_PORT", ""),
			Environment:      env(lookup, "ENVIRONMENT", "development"),
			Version:          env(lookup, "VERSION", "2.0.0"),
			ServiceName:      "system-monitor",
			APIBaseURL:       env(lookup, "API_BASE_URL", ""),
			ManagedLifecycle: envBool(lookup, logger, "VROOLI_LIFECYCLE_MANAGED", false),
			BridgeToken:      env(lookup, "VROOLI_BRIDGE_API_TOKEN", env(lookup, "VROOLI_API_TOKEN", "")),
		},
		Monitoring: MonitoringConfig{
			MetricsInterval:   time.Duration(envInt(lookup, logger, "METRICS_INTERVAL_SECONDS", 20)) * time.Second,
			AnomalyInterval:   time.Duration(envInt(lookup, logger, "ANOMALY_INTERVAL_SECONDS", 30)) * time.Second,
			ReportInterval:    time.Duration(envInt(lookup, logger, "REPORT_INTERVAL_HOURS", 1)) * time.Hour,
			ThresholdInterval: time.Duration(envInt(lookup, logger, "THRESHOLD_INTERVAL_SECONDS", 20)) * time.Second,
			RetentionDays:     envInt(lookup, logger, "RETENTION_DAYS", 30),
			EnableAutoResolve: envBool(lookup, logger, "ENABLE_AUTO_RESOLVE", false),
			MaxInvestigations: envInt(lookup, logger, "MAX_INVESTIGATIONS", 100),

			ProcSampleInterval:         envDuration(lookup, logger, "SYSTEM_MONITOR_PROC_SAMPLE_INTERVAL", 20*time.Second),
			ProcSampleTopN:             envInt(lookup, logger, "SYSTEM_MONITOR_PROC_SAMPLE_TOP_N", 50),
			ProcSampleOwners:           procSampleOwners,
			RawRetention:               envDuration(lookup, logger, "SYSTEM_MONITOR_RAW_RETENTION", 6*time.Hour),
			RollupRetention:            envDuration(lookup, logger, "SYSTEM_MONITOR_ROLLUP_RETENTION", 30*24*time.Hour),
			CollectorRetention:         ParseCollectorRetention(env(lookup, "SYSTEM_MONITOR_COLLECTOR_RETENTION", "")),
			StorageMode:                env(lookup, "SYSTEM_MONITOR_STORAGE_MODE", ""),
			CompactionWindow:           env(lookup, "SYSTEM_MONITOR_COMPACTION_WINDOW", ""),
			InvestigationRoot:          env(lookup, "SYSTEM_MONITOR_INVESTIGATION_ROOT", ""),
			SocketAttributionThreshold: socketThreshold,
			GoCachePath:                env(lookup, "GOCACHE", ""),
			PlaywrightCachePath:        env(lookup, "PLAYWRIGHT_BROWSERS_PATH", ""),
			RuntimeHome:                env(lookup, "VROOLI_HOME", ""),
		},
		Resources: ResourcesConfig{
			NodeRedURL:        env(lookup, "NODE_RED_URL", ""),
			GrafanaURL:        env(lookup, "GRAFANA_URL", "http://localhost:3004"),
			StorageManagerURL: env(lookup, "VROOLI_CLEANUP_MANAGER_API_URL", ""),
			AutohealURL:       env(lookup, "VROOLI_AUTOHEAL_API_URL", ""),
		},
		Alerts: AlertConfig{
			WebhookURL:      env(lookup, "ALERT_WEBHOOK_URL", ""),
			EnableWebhooks:  envBool(lookup, logger, "ENABLE_WEBHOOKS", false),
			EnableEmail:     envBool(lookup, logger, "ENABLE_EMAIL_ALERTS", false),
			EmailFrom:       env(lookup, "EMAIL_FROM", ""),
			EmailSMTPHost:   env(lookup, "EMAIL_SMTP_HOST", ""),
			EmailSMTPPort:   envInt(lookup, logger, "EMAIL_SMTP_PORT", 587),
			MinSeverity:     envInt(lookup, logger, "MIN_ALERT_SEVERITY", 2),
			CooldownMinutes: envInt(lookup, logger, "ALERT_COOLDOWN_MINUTES", 5),
		},
		Logging: LoggingConfig{
			Level:      env(lookup, "LOG_LEVEL", "info"),
			Format:     env(lookup, "LOG_FORMAT", "json"),
			Output:     env(lookup, "LOG_OUTPUT", "stdout"),
			FilePath:   env(lookup, "LOG_FILE_PATH", "/var/log/system-monitor.log"),
			MaxSize:    envInt(lookup, logger, "LOG_MAX_SIZE_MB", 100),
			MaxBackups: envInt(lookup, logger, "LOG_MAX_BACKUPS", 3),
			MaxAge:     envInt(lookup, logger, "LOG_MAX_AGE_DAYS", 7),
		},
		Health: HealthConfig{
			Enabled:            envBool(lookup, logger, "HEALTH_CHECKS_ENABLED", true),
			Interval:           time.Duration(envInt(lookup, logger, "HEALTH_INTERVAL_SECONDS", 30)) * time.Second,
			Timeout:            time.Duration(envInt(lookup, logger, "HEALTH_TIMEOUT_SECONDS", 5)) * time.Second,
			StartupGracePeriod: time.Duration(envInt(lookup, logger, "HEALTH_STARTUP_GRACE_SECONDS", 10)) * time.Second,
			Endpoints:          loadHealthEndpoints(lookup, apiPort),
			BuildIdentity:      env(lookup, "VROOLI_BUILD_IDENTITY", ""),
			OllamaBaseURL:      env(lookup, "OLLAMA_BASE_URL", ""),
		},
		AgentManager: AgentManagerConfig{
			ProfileName: env(lookup, "AGENT_MANAGER_PROFILE_NAME", "system-monitor-investigator"),
			ProfileKey:  env(lookup, "AGENT_MANAGER_PROFILE_KEY", ""),
			Timeout:     time.Duration(envInt(lookup, logger, "AGENT_MANAGER_TIMEOUT_SECONDS", 30)) * time.Second,
			Enabled:     envBool(lookup, logger, "AGENT_MANAGER_ENABLED", true),
		},
	}

	if cfg.AgentManager.ProfileKey == "" {
		cfg.AgentManager.ProfileKey = "system-monitor/default"
	}

	if cfg.Server.APIBaseURL == "" {
		cfg.Server.APIBaseURL = fmt.Sprintf("http://localhost:%s", cfg.Server.APIPort)
	}

	if err := cfg.validate(logger); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate ensures configuration is valid
func (c *Config) validate(logger logx.Logger) error {
	// Validate server config
	if c.Server.APIPort == "" {
		return fmt.Errorf("API_PORT is required")
	}

	// Validate monitoring intervals
	if c.Monitoring.MetricsInterval < time.Second {
		c.Monitoring.MetricsInterval = 20 * time.Second
	}
	if c.Monitoring.AnomalyInterval < time.Second {
		c.Monitoring.AnomalyInterval = 30 * time.Second
	}
	if c.Alerts.EnableWebhooks && c.Alerts.WebhookURL == "" {
		logger.Warn("webhooks enabled without a webhook URL; disabling webhook delivery")
		c.Alerts.EnableWebhooks = false
	}
	if c.Alerts.EnableEmail && (c.Alerts.EmailFrom == "" || c.Alerts.EmailSMTPHost == "") {
		logger.Warn("email alerts enabled without complete SMTP configuration; disabling email delivery")
		c.Alerts.EnableEmail = false
	}
	return nil
}

// IsProduction returns true if running in production
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment returns true if running in development
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// Getenv reads a setting from the configuration's injected environment. A
// nil environment is empty, which keeps zero-value test configs deterministic.
func (c *Config) Getenv(key string) string {
	if c == nil || c.Environment == nil {
		return ""
	}
	value, _ := c.Environment.LookupEnv(key)
	return value
}

type environmentFunc func(string) (string, bool)

func (f environmentFunc) LookupEnv(key string) (string, bool) { return f(key) }

// Helper functions

func env(lookup func(string) (string, bool), key, defaultValue string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func envInt(lookup func(string) (string, bool), logger logx.Logger, key string, defaultValue int) int {
	strValue := env(lookup, key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		logger.Warn("invalid integer configuration; using default", "key", key, "value", strValue, "default", defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsDuration parses a Go duration string (e.g. "20s", "6h", "30d" is NOT
// valid Go — use "720h"). Falls back to defaultValue on empty or invalid input.
func envDuration(lookup func(string) (string, bool), logger logx.Logger, key string, defaultValue time.Duration) time.Duration {
	strValue := env(lookup, key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(strValue)
	if err != nil || value <= 0 {
		logger.Warn("invalid duration configuration; using default", "key", key, "value", strValue, "default", defaultValue)
		return defaultValue
	}
	return value
}

func envBool(lookup func(string) (string, bool), logger logx.Logger, key string, defaultValue bool) bool {
	strValue := env(lookup, key, "")
	if strValue == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(strValue)
	if err != nil {
		logger.Warn("invalid boolean configuration; using default", "key", key, "value", strValue, "default", defaultValue)
		return defaultValue
	}
	return value
}

func loadHealthEndpoints(lookup func(string) (string, bool), apiPort string) []HealthEndpoint {
	// Default health endpoints
	endpoints := []HealthEndpoint{
		{
			Name:     "api",
			URL:      fmt.Sprintf("http://localhost:%s/health", apiPort),
			Critical: true,
			Timeout:  5 * time.Second,
		},
	}

	// Add UI endpoint if configured
	if uiPort := env(lookup, "UI_PORT", ""); uiPort != "" {
		endpoints = append(endpoints, HealthEndpoint{
			Name:     "ui",
			URL:      fmt.Sprintf("http://localhost:%s/", uiPort),
			Critical: false,
			Timeout:  5 * time.Second,
		})
	}

	return endpoints
}
