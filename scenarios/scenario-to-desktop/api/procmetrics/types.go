// DOC: docs/reference/live-desktop-api.md#process-metrics
package procmetrics

import "time"

// Sample is a single point-in-time measurement of process resources.
type Sample struct {
	Timestamp  time.Time `json:"timestamp"`
	CPUPercent float64   `json:"cpu_percent"`
	RSSBytes   int64     `json:"rss_bytes"`
	PeakBytes  int64     `json:"peak_bytes"`
	Threads    int       `json:"threads"`
}

// Summary contains aggregate statistics computed from the sample series.
type Summary struct {
	PeakRSSBytes int64   `json:"peak_rss_bytes"`
	AvgRSSBytes  int64   `json:"avg_rss_bytes"`
	PeakCPU      float64 `json:"peak_cpu_percent"`
	AvgCPU       float64 `json:"avg_cpu_percent"`
	MaxThreads   int     `json:"max_threads"`
	SampleCount  int     `json:"sample_count"`
	DurationMs   int64   `json:"duration_ms"`
}

// ProcessRole identifies the startup component that owns a process.
type ProcessRole string

const (
	RoleElectronMain     ProcessRole = "electron_main"
	RoleElectronRender   ProcessRole = "electron_renderer"
	RoleElectronGPU      ProcessRole = "electron_gpu"
	RoleElectronUtility  ProcessRole = "electron_utility"
	RoleElectronCrashpad ProcessRole = "electron_crashpad"
	RoleBundledRuntime   ProcessRole = "bundled_runtime"
	RoleScenarioService  ProcessRole = "scenario_service"
	RoleLauncher         ProcessRole = "launcher"
	RoleUnknown          ProcessRole = "unknown"
	RoleShellMain        ProcessRole = "shell_main"
	RoleWebviewRenderer  ProcessRole = "webview_renderer"
	RoleWebviewGPU       ProcessRole = "webview_gpu"
	RoleWebviewOther     ProcessRole = "webview_other"
	RoleBackend          ProcessRole = "backend"
)

// ProcessInfo is one node in a sampled process tree.
type ProcessInfo struct {
	PID                  int         `json:"pid"`
	PPID                 int         `json:"ppid"`
	Command              string      `json:"command,omitempty"`
	Cmdline              []string    `json:"cmdline,omitempty"`
	Exe                  string      `json:"exe,omitempty"`
	Role                 ProcessRole `json:"role"`
	CPUJiffies           int64       `json:"cpu_jiffies"`
	RSSBytes             int64       `json:"rss_bytes"`
	PSSBytes             int64       `json:"pss_bytes,omitempty"`
	USSBytes             int64       `json:"uss_bytes,omitempty"`
	MemoryStatsAvailable bool        `json:"memory_stats_available"`
	PeakBytes            int64       `json:"peak_bytes"`
	Threads              int         `json:"threads"`
}

// RoleSummary aggregates unique process nodes. Unsupported roles remain
// unavailable instead of becoming zero-valued measurements.
type RoleSummary struct {
	Role                 ProcessRole `json:"role"`
	Available            bool        `json:"available"`
	Unsupported          bool        `json:"unsupported,omitempty"`
	ProcessCount         int         `json:"process_count"`
	CPUPercent           float64     `json:"cpu_percent"`
	PeakCPU              float64     `json:"peak_cpu_percent"`
	RSSBytes             int64       `json:"rss_bytes"`
	PSSBytes             int64       `json:"pss_bytes,omitempty"`
	USSBytes             int64       `json:"uss_bytes,omitempty"`
	MemoryStatsAvailable bool        `json:"memory_stats_available"`
	PeakRSSBytes         int64       `json:"peak_rss_bytes"`
	Threads              int         `json:"threads"`
	DurationMs           int64       `json:"duration_ms"`
	SampleCount          int         `json:"sample_count"`
}

// ProcessTreeReport describes the attribution scope and role totals.
type ProcessTreeReport struct {
	Supported bool                        `json:"supported"`
	Scope     string                      `json:"scope"`
	Roles     map[ProcessRole]RoleSummary `json:"roles,omitempty"`
}

// ProcessTreeOptions identifies the application executable when it is known.
// An empty name enables the conservative executable-path heuristic.
type ProcessTreeOptions struct {
	AppExecutableName string
}

// GenericProcessRole maps the current engine classifier onto the taxonomy
// shared by desktop and future mobile shells.
func GenericProcessRole(role ProcessRole) ProcessRole {
	switch role {
	case RoleElectronMain, RoleShellMain:
		return RoleShellMain
	case RoleElectronRender, RoleWebviewRenderer:
		return RoleWebviewRenderer
	case RoleElectronGPU, RoleWebviewGPU:
		return RoleWebviewGPU
	case RoleElectronUtility, RoleElectronCrashpad, RoleWebviewOther:
		return RoleWebviewOther
	case RoleBundledRuntime, RoleScenarioService, RoleBackend:
		return RoleBackend
	default:
		return RoleUnknown
	}
}

// StartupTiming records app startup in two phases:
//   - Splash: launch → first visible window of any size (e.g. splash screen)
//   - Ready:  launch → main application window (meets expected size threshold)
//
// If the app has no splash screen, both phases resolve at the same time.
type StartupTiming struct {
	LaunchAt time.Time `json:"launch_at"`

	// Phase 1: first visible window (splash screen or immediate main window).
	SplashVisibleAt  *time.Time `json:"splash_visible_at,omitempty"`
	SplashDurationMs *int64     `json:"splash_duration_ms,omitempty"`

	// Phase 2: main application window (meets size threshold).
	ReadyAt *time.Time `json:"ready_at,omitempty"`
	ReadyMs *int64     `json:"ready_ms,omitempty"`
}

// Report is the complete metrics output for a monitored process.
type Report struct {
	Startup     StartupTiming      `json:"startup"`
	Samples     []Sample           `json:"samples"`
	Summary     *Summary           `json:"summary,omitempty"`
	ProcessTree *ProcessTreeReport `json:"process_tree,omitempty"`
}
