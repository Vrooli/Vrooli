package procmetrics

import "context"

// Monitor observes a running process and collects startup timing and resource usage.
type Monitor interface {
	// Start begins monitoring the given PID on the given X display.
	// expectedWidth/expectedHeight define the main window size threshold for
	// distinguishing a splash screen from the ready state. Pass 0,0 to skip
	// size-based detection (any visible window counts as ready).
	Start(ctx context.Context, pid int, display string, expectedWidth, expectedHeight int) error
	// Stop halts monitoring and computes the summary. Safe to call multiple times.
	Stop()
	// Report returns the current metrics report (safe to call while running).
	Report() *Report
	// Done returns a channel that closes when monitoring has finished.
	Done() <-chan struct{}
}

// MonitorFactory creates new Monitor instances.
type MonitorFactory interface {
	NewMonitor() Monitor
}

// ProcReader abstracts reading process stats from the OS.
type ProcReader interface {
	// ReadStat reads CPU time from /proc/<pid>/stat. Returns utime and stime in clock ticks.
	ReadStat(pid int) (utime, stime int64, err error)
	// ReadStatus reads memory and thread info from /proc/<pid>/status.
	ReadStatus(pid int) (rssBytes, peakBytes int64, threads int, err error)
	// IsAlive checks if the process still exists.
	IsAlive(pid int) bool
}

// ProcessTreeReader is optional. Platforms without a process-tree adapter
// must report unsupported attribution instead of treating the root as the
// complete application.
type ProcessTreeReader interface {
	ProcessTree(rootPID int) ([]ProcessInfo, error)
}

// WindowGeometry describes the size of a detected window.
type WindowGeometry struct {
	X      int
	Y      int
	Width  int
	Height int
}

// WindowDetector checks for visible X11 windows belonging to a process.
type WindowDetector interface {
	// HasVisibleWindow returns true if the process has at least one visible window on the display.
	HasVisibleWindow(ctx context.Context, pid int, display string) (bool, error)

	// LargestVisibleWindow returns the geometry of the largest visible window on the display
	// belonging to the process (or any window on the display as fallback for multi-process apps).
	// Returns nil if no visible window exists.
	LargestVisibleWindow(ctx context.Context, pid int, display string) (*WindowGeometry, error)
}

// ShellFunc abstracts shell command execution.
type ShellFunc func(ctx context.Context, env []string, name string, args ...string) (stdout []byte, err error)
