package procmetrics

import "log/slog"

// DefaultMonitorFactory creates DefaultMonitor instances with shared dependencies.
type DefaultMonitorFactory struct {
	proc   ProcReader
	window WindowDetector
	logger *slog.Logger
}

// NewDefaultMonitorFactory creates a factory with the given dependencies.
func NewDefaultMonitorFactory(proc ProcReader, window WindowDetector, logger *slog.Logger) *DefaultMonitorFactory {
	return &DefaultMonitorFactory{
		proc:   proc,
		window: window,
		logger: logger,
	}
}

// NewMonitor creates a fresh DefaultMonitor.
func (f *DefaultMonitorFactory) NewMonitor() Monitor {
	return NewDefaultMonitor(f.proc, f.window, f.logger)
}
