package userconfig

import "github.com/vrooli/api-core/coreset"

// Default values for configuration
const (
	DefaultVersion = "1.0"

	// Global defaults
	DefaultGracePeriodSeconds          = 60
	DefaultTickIntervalSeconds         = 60
	DefaultVerifyDelaySeconds          = 30
	DefaultMaxRestartAttempts          = 3
	DefaultRestartCooldownSeconds      = 300
	DefaultHistoryRetentionHours       = 24
	DefaultActionTimeoutFastSeconds    = 30
	DefaultActionTimeoutRestartSeconds = 300
	DefaultTimeoutRetrySeconds         = 30

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
		GracePeriodSeconds:          DefaultGracePeriodSeconds,
		TickIntervalSeconds:         DefaultTickIntervalSeconds,
		VerifyDelaySeconds:          DefaultVerifyDelaySeconds,
		MaxRestartAttempts:          DefaultMaxRestartAttempts,
		RestartCooldownSeconds:      DefaultRestartCooldownSeconds,
		HistoryRetentionHours:       DefaultHistoryRetentionHours,
		ActionTimeoutFastSeconds:    DefaultActionTimeoutFastSeconds,
		ActionTimeoutRestartSeconds: DefaultActionTimeoutRestartSeconds,
		TimeoutRetrySeconds:         DefaultTimeoutRetrySeconds,
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
	monitoring := MonitoringConfig{
		Scenarios: map[string]MonitoredScenario{
			// Critical scenarios - will report StatusCritical when stopped
			"app-monitor": {Critical: true},
			// system-monitor provides the pressure evidence consumed by the
			// runtime recovery controller; its liveness is therefore explicit.
			"system-monitor":   {Critical: true},
			"template-manager": {Critical: true},
			// Non-critical scenarios - will report StatusWarning when stopped
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
			"whisper",
		},
	}
	ensureMandatoryCoreScenarios(&monitoring)
	return monitoring
}

// ensureMandatoryCoreScenarios keeps the platform's shared core-set healthy
// without requiring every consumer scenario to declare the same dependency.
// Core members are always critical; optional operator monitoring remains
// additive around this mandatory floor.
func ensureMandatoryCoreScenarios(monitoring *MonitoringConfig) {
	if monitoring.Scenarios == nil {
		monitoring.Scenarios = make(map[string]MonitoredScenario)
	}
	for _, name := range coreset.CoreSeedScenarios() {
		monitoring.Scenarios[name] = MonitoredScenario{Critical: true}
	}
}

// CheckDefaults contains default settings for a check
type CheckDefaults struct {
	Enabled         bool
	AutoHeal        bool
	AutoHealOn      string
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
		AutoHealOn:      "critical",
		IntervalSeconds: 30,
	},
	"infra-dns": {
		Enabled:         true,
		AutoHeal:        true, // Can restart systemd-resolved
		AutoHealOn:      "critical",
		IntervalSeconds: 30,
	},
	"infra-docker": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"infra-cloudflared": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"infra-ntp": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 300,
	},
	"infra-display": {
		Enabled:         true, // Monitors desktop session and RDP availability
		AutoHeal:        true, // Can restart GDM to recover desktop session
		AutoHealOn:      "critical",
		IntervalSeconds: 60, // Check frequently for RDP availability
	},
	"infra-rdp": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"infra-resolved": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"infra-certificate": {
		Enabled:         true,
		AutoHeal:        false, // Can't auto-renew certificates
		AutoHealOn:      "critical",
		IntervalSeconds: 3600,
	},

	// System checks
	"system-disk": {
		Enabled: true,
		// Disk pressure IS auto-healable: the request-cleanup action reports
		// to cleanup-manager, which reclaims safe-tier space unattended and
		// refuses anything above safe tier. This was previously false with the
		// comment "Can't auto-heal disk space", which is why the 2026-07-31
		// host filled to 100 percent overnight with nobody awake to act.
		AutoHeal:        true,
		AutoHealOn:      "critical",
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
		AutoHealOn:      "critical",
		IntervalSeconds: 120,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(85.0),
			CriticalPercent: ptr(95.0),
		},
	},
	"system-swap": {
		Enabled:         true,
		AutoHeal:        false,
		AutoHealOn:      "critical",
		IntervalSeconds: 120,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(50.0),
			CriticalPercent: ptr(80.0),
		},
	},
	"system-zombies": {
		Enabled:         true,
		AutoHeal:        true, // Can send SIGCHLD to parents
		AutoHealOn:      "critical",
		IntervalSeconds: 300,
		Thresholds: &Thresholds{
			WarningCount:  intPtr(5),
			CriticalCount: intPtr(20),
		},
	},
	"system-ports": {
		Enabled:         true,
		AutoHeal:        false,
		AutoHealOn:      "critical",
		IntervalSeconds: 300,
		Thresholds: &Thresholds{
			WarningPercent:  ptr(70.0),
			CriticalPercent: ptr(85.0),
		},
	},
	"system-claude-cache": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 3600,
	},

	// Resource checks - all enabled with auto-heal by default
	"resource-postgres": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 30,
	},
	"resource-redis": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 30,
	},
	"resource-ollama": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"resource-qdrant": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"resource-searxng": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 120,
	},
	"resource-whisper": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},

	"os-watchdog": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 300,
	},
	"vrooli-runtime-supervisor": {
		Enabled:         true,
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},

	// Vrooli lifecycle checks
	"vrooli-stale-locks": {
		Enabled:         true,
		AutoHeal:        true, // Safe - only removes lock files for dead processes
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	},
	"vrooli-orphans": {
		Enabled:         true,
		AutoHeal:        false, // Dangerous - killing processes requires explicit opt-in
		AutoHealOn:      "critical",
		IntervalSeconds: 120,
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
	if isMandatoryCoreScenarioCheck(checkID) {
		return CheckDefaults{Enabled: true, AutoHeal: true, AutoHealOn: "critical", IntervalSeconds: 60}
	}
	if defaults, ok := KnownCheckDefaults[checkID]; ok {
		return defaults
	}
	// Generic defaults for unknown checks
	return CheckDefaults{
		Enabled:         true,
		AutoHeal:        false,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	}
}

func isMandatoryCoreScenarioCheck(checkID string) bool {
	const prefix = "scenario-"
	if len(checkID) <= len(prefix) || checkID[:len(prefix)] != prefix {
		return false
	}
	return coreset.IsCoreSeed(checkID[len(prefix):])
}
