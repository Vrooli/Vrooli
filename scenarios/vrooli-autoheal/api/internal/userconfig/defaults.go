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
	// DefaultHistoryRetentionHours is the named operational-history window.
	// The manifest-owned scheduled sweeps for health_results and action_logs
	// are contract-tested against this value so configuration and enforcement
	// cannot silently drift apart.
	DefaultHistoryRetentionHours       = 24
	DefaultActionTimeoutFastSeconds    = 30
	DefaultActionTimeoutRestartSeconds = 300
	DefaultTimeoutRetrySeconds         = 30
	DefaultHealInterlockSeconds        = 30
	// DefaultContainStorm is automatic (plan decision D5): agents only, never
	// a supervisor or a resource.
	DefaultContainStorm = "automatic"

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
		HealInterlockSeconds:        DefaultHealInterlockSeconds,
		ContainStorm:                DefaultContainStorm,
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
		Version: DefaultVersion,
		Global:  DefaultGlobal(),
		Checks: map[string]Check{
			// Host-integrity checks are part of the operator-visible contract.
			// Keep their explicit toggles in the returned config so the API and
			// CLI expose the same controls that the registry uses.
			"host-kernel-module-drift":   {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
			"host-device-driver-binding": {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
			"host-runtime-integrity":     {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
			"host-package-state":         {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
			"host-kernel-error-signals":  {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
			"host-capability-drift":      {Enabled: boolPtr(true), AutoHeal: boolPtr(false)},
		},
		UI:         DefaultUI(),
		Monitoring: DefaultMonitoring(),
	}
}

// DefaultMonitoring returns the default monitoring configuration
// This defines which scenarios and resources are monitored by default
func DefaultMonitoring() MonitoringConfig {
	// Supervision authority comes exclusively from `vrooli supervision-set`.
	// Persisted monitoring entries are additive operator overrides, so the
	// built-in default must remain empty rather than becoming a rival core list.
	return MonitoringConfig{Scenarios: map[string]MonitoredScenario{}}
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
	"system-stale-service-binary": {
		Enabled: true,
		// A supervised service running a replaced binary is exactly the kind of
		// condition an operator should never have to notice: the fix is a
		// restart of a unit designed to be restarted, and the restart already
		// passes through the host-pressure gate that defers it while the
		// machine is saturated.
		//
		// AutoHealOn must be warning+critical. The check reports a warning —
		// stale code is wrong, not on fire — so the default "critical" trigger
		// would leave it detected and never acted on.
		AutoHeal:        true,
		AutoHealOn:      "warning+critical",
		IntervalSeconds: 300,
	},
	"system-disk": {
		Enabled: true,
		// Disk pressure IS auto-healable: the request-cleanup action reports
		// to storage-manager, which reclaims safe-tier space unattended and
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
	"system-emergency-watchdog-report": {
		Enabled: true,
		// The storm authority (plan decision D5): a sustained fork storm the
		// watchdog attributes to an agent session scope is frozen from the
		// auto-heal pass. The action only ever targets a scope under
		// vrooli-agents.slice, and Global.ContainStorm=propose_only marks it
		// Dangerous so this pass never selects it.
		AutoHeal:        true,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
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

	// Host-integrity checks are intentionally observable by default, but their
	// mutations remain operator-approved until each remediation action has an
	// explicit safety contract.
	"host-kernel-module-drift":   {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},
	"host-device-driver-binding": {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},
	"host-runtime-integrity":     {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},
	"host-package-state":         {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},
	"host-kernel-error-signals":  {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},
	"host-capability-drift":      {Enabled: true, AutoHeal: false, AutoHealOn: "critical", IntervalSeconds: 300},

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

// scenarioAutoHealOptOut is deliberately explicit and empty by default. A
// scenario that must never be restarted can be added here with an operator
// decision; ordinary scenario checks must not require a hand-maintained allow
// list to recover from an outage.
var scenarioAutoHealOptOut = map[string]struct{}{}

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
	if isScenarioCheck(checkID) && !isScenarioAutoHealOptedOut(checkID) {
		return CheckDefaults{Enabled: true, AutoHeal: true, AutoHealOn: "critical", IntervalSeconds: 60}
	}
	if defaults, ok := KnownCheckDefaults[checkID]; ok {
		return defaults
	}
	// A resource check gets resource defaults from its prefix, exactly as a
	// scenario check does. Before this, a resource added to the monitored list
	// without a matching KnownCheckDefaults entry fell through to the generic
	// default with AutoHeal false and was monitored but never healed — which is
	// what left resource-reranker logging "auto-heal not enabled for this
	// check" on every cycle for weeks.
	if isResourceCheck(checkID) {
		return CheckDefaults{Enabled: true, AutoHeal: true, AutoHealOn: "critical", IntervalSeconds: 60}
	}
	// Generic defaults for unknown checks
	return CheckDefaults{
		Enabled:         true,
		AutoHeal:        false,
		AutoHealOn:      "critical",
		IntervalSeconds: 60,
	}
}

// isResourceCheck reports whether a check id names a resource check. The
// mode-drift family shares the prefix, so it inherits the same defaults.
func isResourceCheck(checkID string) bool {
	const prefix = "resource-"
	return len(checkID) > len(prefix) && checkID[:len(prefix)] == prefix
}

func isScenarioCheck(checkID string) bool {
	const prefix = "scenario-"
	return len(checkID) > len(prefix) && checkID[:len(prefix)] == prefix
}

func isScenarioAutoHealOptedOut(checkID string) bool {
	const prefix = "scenario-"
	if !isScenarioCheck(checkID) {
		return false
	}
	_, optedOut := scenarioAutoHealOptOut[checkID[len(prefix):]]
	return optedOut
}
