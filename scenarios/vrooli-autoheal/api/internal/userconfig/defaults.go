package userconfig

// Default values for configuration
const (
	DefaultVersion = "1.0"

	// Global defaults
	DefaultGracePeriodSeconds     = 60
	DefaultTickIntervalSeconds    = 60
	DefaultVerifyDelaySeconds     = 30
	DefaultMaxRestartAttempts     = 3
	DefaultRestartCooldownSeconds = 300
	DefaultHistoryRetentionHours  = 24

	// UI defaults
	DefaultAutoRefreshSeconds = 30
	DefaultTheme              = "system"
	DefaultDefaultTab         = "dashboard"

	// Check settings defaults
	DefaultLogLinesToCapture = 100
)

// DefaultGlobal returns the default global configuration
func DefaultGlobal() GlobalConfig {
	return GlobalConfig{
		GracePeriodSeconds:     DefaultGracePeriodSeconds,
		TickIntervalSeconds:    DefaultTickIntervalSeconds,
		VerifyDelaySeconds:     DefaultVerifyDelaySeconds,
		MaxRestartAttempts:     DefaultMaxRestartAttempts,
		RestartCooldownSeconds: DefaultRestartCooldownSeconds,
		HistoryRetentionHours:  DefaultHistoryRetentionHours,
	}
}

// DefaultUI returns the default UI configuration
func DefaultUI() UIConfig {
	return UIConfig{
		AutoRefreshSeconds: DefaultAutoRefreshSeconds,
		Theme:              DefaultTheme,
		ShowDisabledChecks: false,
		DefaultTab:         DefaultDefaultTab,
	}
}

// DefaultConfig returns a configuration with all defaults applied
func DefaultConfig() *Config {
	return &Config{
		Version:    DefaultVersion,
		Global:     DefaultGlobal(),
		Checks:     make(map[string]Check),
		UI:         DefaultUI(),
		Monitoring: DefaultMonitoring(),
	}
}

// DefaultMonitoring returns the default monitoring configuration
// This defines which scenarios and resources are monitored by default
func DefaultMonitoring() MonitoringConfig {
	return MonitoringConfig{
		Scenarios: map[string]MonitoredScenario{
			// Critical scenarios - will report StatusCritical when stopped
			"app-monitor":       {Critical: true},
			"ecosystem-manager": {Critical: true},
			// Non-critical scenarios - will report StatusWarning when stopped
			"landing-manager":           {Critical: false},
			"browser-automation-studio": {Critical: false},
			"test-genie":                {Critical: false},
			"deployment-manager":        {Critical: false},
			"git-control-tower":         {Critical: false},
			"tidiness-manager":          {Critical: false},
		},
		Resources: []string{
			"postgres",
			"redis",
			"ollama",
			"qdrant",
			"searxng",
			"browserless",
		},
	}
}

// CheckDefaults contains default settings for a check
type CheckDefaults struct {
	Enabled         bool
	AutoHeal        bool
	IntervalSeconds int
	Thresholds      *Thresholds
}

// KnownCheckDefaults maps check IDs to their default configurations
// These are used when a check isn't explicitly configured
var KnownCheckDefaults = map[string]CheckDefaults{
	// Infrastructure checks
	"infra-network": {
		Enabled:         true,
		AutoHeal:        false, // Can't auto-heal network issues
		IntervalSeconds: 30,
	},
	"infra-dns": {
		Enabled:         true,
		AutoHeal:        true, // Can restart systemd-resolved
		IntervalSeconds: 30,
	},
	"infra-docker": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"infra-cloudflared": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"infra-ntp": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 300,
	},
	"infra-display": {
		Enabled:         false, // Dangerous on desktop - disabled by default
		AutoHeal:        false,
		IntervalSeconds: 120,
	},
	"infra-rdp": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"infra-resolved": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"infra-certificate": {
		Enabled:         true,
		AutoHeal:        false, // Can't auto-renew certificates
		IntervalSeconds: 3600,
	},

	// System checks
	"system-disk": {
		Enabled:         true,
		AutoHeal:        false, // Can't auto-heal disk space
		IntervalSeconds: 120,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(80.0),
			CriticalPercent: ptr(90.0),
			Partitions:      []string{"/", "/home"},
		},
	},
	"system-inode": {
		Enabled:         true,
		AutoHeal:        false,
		IntervalSeconds: 120,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(85.0),
			CriticalPercent: ptr(95.0),
		},
	},
	"system-swap": {
		Enabled:         true,
		AutoHeal:        false,
		IntervalSeconds: 120,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(50.0),
			CriticalPercent: ptr(80.0),
		},
	},
	"system-zombies": {
		Enabled:         true,
		AutoHeal:        true, // Can send SIGCHLD to parents
		IntervalSeconds: 300,
		Thresholds: &Thresholds{
			WarningCount:  intPtr(5),
			CriticalCount: intPtr(20),
		},
	},
	"system-ports": {
		Enabled:         true,
		AutoHeal:        false,
		IntervalSeconds: 300,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(70.0),
			CriticalPercent: ptr(85.0),
		},
	},
	"system-claude-cache": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 3600,
	},

	// Resource checks - all enabled with auto-heal by default
	"resource-postgres": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 30,
	},
	"resource-redis": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 30,
	},
	"resource-ollama": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"resource-qdrant": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 60,
	},
	"resource-searxng": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 120,
	},
	"resource-browserless": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 120,
	},

	// Vrooli API check
	"vrooli-api": {
		Enabled:         true,
		AutoHeal:        true,
		IntervalSeconds: 30,
	},
}

// Helper functions for creating pointers
func ptr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

// GetCheckDefaults returns the default configuration for a check
// If the check isn't in KnownCheckDefaults, returns generic defaults
func GetCheckDefaults(checkID string) CheckDefaults {
	if defaults, ok := KnownCheckDefaults[checkID]; ok {
		return defaults
	}
	// Generic defaults for unknown checks
	return CheckDefaults{
		Enabled:         true,
		AutoHeal:        false,
		IntervalSeconds: 60,
	}
}
