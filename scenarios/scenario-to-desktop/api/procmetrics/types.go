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
	Startup StartupTiming `json:"startup"`
	Samples []Sample      `json:"samples"`
	Summary *Summary      `json:"summary,omitempty"`
}
