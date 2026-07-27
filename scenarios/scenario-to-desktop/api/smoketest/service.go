// DOC: docs/reference/smoke-test-pipeline.md
package smoketest

import (
	"fmt"
	"log/slog"
	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
	"time"
)

// DefaultService is the default implementation of Service.
// See docs/reference/smoke-test-pipeline.md for execution flow and configuration.
type DefaultService struct {
	store             Store
	cancelManager     CancelManager
	telemetryIngestor TelemetryIngestor
	port              int
	logger            Logger

	// New injected components
	config             Config
	executor           ProcessExecutor
	platformResolver   PlatformResolver
	telemetryResolver  TelemetryPathResolver
	outputParser       OutputParser
	fileSystem         FileSystem
	prereqChecker      PrerequisiteCheckerI
	envReader          EnvironmentReader
	telemetryExtractor TelemetryErrorExtractor

	// Optional screen recording (nil = recording disabled)
	recorder   screenrecording.Recorder
	displayMgr screenrecording.DisplayManager

	// Optional process monitoring (nil = monitoring disabled)
	monitorFactory procmetrics.MonitorFactory
}

// NewService creates a new smoke test service with all required dependencies.
func NewService(
	store Store,
	cancelManager CancelManager,
	telemetryIngestor TelemetryIngestor,
	config Config,
	executor ProcessExecutor,
	platformResolver PlatformResolver,
	telemetryResolver TelemetryPathResolver,
	outputParser OutputParser,
	fileSystem FileSystem,
	logger Logger,
	port int,
	telemetryExtractor TelemetryErrorExtractor,
) *DefaultService {
	return &DefaultService{
		store:              store,
		cancelManager:      cancelManager,
		telemetryIngestor:  telemetryIngestor,
		config:             config,
		executor:           executor,
		platformResolver:   platformResolver,
		telemetryResolver:  telemetryResolver,
		outputParser:       outputParser,
		fileSystem:         fileSystem,
		logger:             logger,
		port:               port,
		telemetryExtractor: telemetryExtractor,
	}
}

// NewDefaultSmokeTestService creates a new smoke test service with default implementations.
// This is the factory function for production wiring.
func NewDefaultSmokeTestService(
	store Store,
	cancelManager CancelManager,
	telemetryIngestor TelemetryIngestor,
	port int,
	logger Logger,
) *DefaultService {
	config := DefaultConfig()
	envReader := NewEnvironmentReader()
	fs := NewFileSystem()
	executor := NewProcessExecutorWithLimit(logger, config.MaxOutputBytes)
	platformResolver := NewPlatformResolver(executor, config, envReader, fs)
	telemetryResolver := NewTelemetryPathResolver(config, envReader, fs)
	outputParser := NewOutputParser(config)
	prereqChecker := NewPrerequisiteChecker(envReader, fs, executor)
	telemetryExtractor := NewTelemetryErrorExtractor(fs)

	return &DefaultService{
		store:              store,
		cancelManager:      cancelManager,
		telemetryIngestor:  telemetryIngestor,
		config:             config,
		executor:           executor,
		platformResolver:   platformResolver,
		telemetryResolver:  telemetryResolver,
		outputParser:       outputParser,
		fileSystem:         fs,
		logger:             logger,
		port:               port,
		prereqChecker:      prereqChecker,
		envReader:          envReader,
		telemetryExtractor: telemetryExtractor,
	}
}

// WithRecording enables screen recording on an existing service.
func (s *DefaultService) WithRecording(recorder screenrecording.Recorder, displayMgr screenrecording.DisplayManager) {
	s.recorder = recorder
	s.displayMgr = displayMgr
}

// WithMonitor sets the process monitor factory for tracking app startup time and resource usage.
func (s *DefaultService) WithMonitor(factory procmetrics.MonitorFactory) {
	s.monitorFactory = factory
}

// CurrentPlatform returns the current platform identifier.
func (s *DefaultService) CurrentPlatform() string {
	return s.platformResolver.CurrentPlatform()
}

func (s *DefaultService) recordTypedFailure(smokeTestID string, err *Error) {
	s.transitionTo(smokeTestID, StateFailed, err.Message)
	s.store.Update(smokeTestID, func(status *Status) {
		status.Status = "failed"
		status.Error = err.Error()
		status.ErrorKind = &err.Kind
		status.ErrorContext = err.Context
		status.SuggestedAction = err.SuggestedAction
		status.Logs = append(status.Logs, fmt.Sprintf("FAILED: %s", err.Message))
		now := time.Now()
		status.CompletedAt = &now
	})

	s.logger.Error("smoke_test_failed",
		"smoke_test_id", smokeTestID,
		"error_kind", err.Kind.String(),
		"error", err.Error(),
		"recoverable", err.Recoverable,
	)
}

func (s *DefaultService) transitionTo(smokeTestID string, newState State, message string) {
	s.store.Update(smokeTestID, func(status *Status) {
		now := time.Now()
		var durationMs int64
		if len(status.Transitions) > 0 {
			lastTransition := status.Transitions[len(status.Transitions)-1]
			durationMs = now.Sub(lastTransition.Timestamp).Milliseconds()
		}
		transition := StateTransition{
			From:       status.CurrentState,
			To:         newState,
			Timestamp:  now,
			Message:    message,
			DurationMs: durationMs,
		}
		status.Transitions = append(status.Transitions, transition)
		status.CurrentState = newState
		status.Logs = append(status.Logs, fmt.Sprintf("[%s] %s", newState, message))
	})

	s.logger.Info("smoke_test_state_transition",
		"smoke_test_id", smokeTestID,
		"state", string(newState),
		"message", message,
	)
}

// SlogAdapter adapts slog.Logger to the Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new slog adapter.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: logger}
}

// Info logs an info message.
func (a *SlogAdapter) Info(msg string, args ...interface{}) {
	a.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (a *SlogAdapter) Warn(msg string, args ...interface{}) {
	a.logger.Warn(msg, args...)
}

// Error logs an error message.
func (a *SlogAdapter) Error(msg string, args ...interface{}) {
	a.logger.Error(msg, args...)
}
