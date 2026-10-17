// Package build provides desktop application build services.
// This domain handles building Electron applications for multiple platforms.
package build

import "log/slog"

// Service orchestrates desktop build operations.
type Service interface {
	// PerformDesktopBuild builds a desktop application from the given path.
	PerformDesktopBuild(buildID string, request *BuildRequest)

	// PerformScenarioDesktopBuild builds a scenario's desktop application.
	PerformScenarioDesktopBuild(buildID, scenarioName, desktopPath string, platforms []string, clean bool)

	// BuildPlatform builds for a specific platform.
	BuildPlatform(buildID, scenarioName, desktopPath, distPath, platform string)
}

// Store manages build status tracking.
type Store interface {
	// Save inserts or replaces a build status.
	Save(status *Status)

	// Get returns the status for the given build if it exists.
	Get(id string) (*Status, bool)

	// Update executes fn while holding a write lock on the requested build.
	// It returns false when the build ID is unknown.
	Update(id string, fn func(status *Status)) bool

	// UpdatePlatform updates a specific platform's build result.
	UpdatePlatform(buildID, platform string, fn func(status *Status, result *PlatformResult)) bool

	// Snapshot returns a shallow copy of the current build status map.
	Snapshot() map[string]*Status

	// Len reports how many builds are tracked.
	Len() int
}

// WineChecker checks if Wine is available for Windows builds.
type WineChecker interface {
	// IsWineInstalled checks if Wine is installed on the system.
	IsWineInstalled() bool
}

// PackageFinder locates built packages in dist directories.
type PackageFinder interface {
	// FindBuiltPackage finds the built package file for a specific platform.
	FindBuiltPackage(distPath, platform string) (string, error)
}

// Logger provides structured logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// SlogLogger adapts slog.Logger to the Logger interface.
type SlogLogger struct {
	*slog.Logger
}

// Info logs an info message.
func (l *SlogLogger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *SlogLogger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message.
func (l *SlogLogger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}
