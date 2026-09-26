// DOC: docs/reference/smoke-test-pipeline.md
package smoketest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
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
	config             SmokeTestConfig
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
	captures   *captures.Service

	// Optional process monitoring (nil = monitoring disabled)
	monitorFactory       procmetrics.MonitorFactory
	windowDetector       *procmetrics.XdotoolDetector
	journeyDriver        DesktopDriver
	journeyClock         Clock
	journeyWaiter        JourneyWaiter
	journeyCapture       JourneyCapture
	journeyAPI           JourneyAPIProbe
	journeyProcess       JourneyProcessObserver
	journeyCapability    string
	evidenceReporter     EvidenceReporter
	manifestWriter       EvidenceManifestWriter
	rendererURLResolver  func(context.Context, string) (string, error)
	apiURLResolver       func(context.Context, string) (string, error)
	targetResolver       func(context.Context, *Status) (JourneyTarget, error)
	providerJourney      ProviderJourneyExecutor
	providerScenarioRoot string
}

// WithProviderJourneyExecutor enables provider-owned BAS execution inside the
// existing smoke journey, before its single evidence manifest is written.
func (s *DefaultService) WithProviderJourneyExecutor(executor ProviderJourneyExecutor, scenarioRoot string) {
	s.providerJourney = executor
	s.providerScenarioRoot = scenarioRoot
}

// NewService creates a new smoke test service with all required dependencies.
func NewService(
	store Store,
	cancelManager CancelManager,
	telemetryIngestor TelemetryIngestor,
	config SmokeTestConfig,
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
		targetResolver:     resolveSmokeTarget,
	}
}

func (s *DefaultService) validationRendererEnv(ctx context.Context, scenarioName string) []string {
	if s == nil || scenarioName == "" {
		return nil
	}
	env := make([]string, 0, 2)
	if s.rendererURLResolver != nil {
		rendererURL, err := s.rendererURLResolver(ctx, scenarioName)
		if err != nil || rendererURL == "" {
			if err != nil && s.logger != nil {
				s.logger.Warn("validation_renderer_url_unavailable", "scenario", scenarioName, "error", err)
			}
		} else {
			env = append(env, "VROOLI_VALIDATION_RENDERER_URL="+rendererURL)
		}
	}
	if s.apiURLResolver != nil {
		apiURL, err := s.apiURLResolver(ctx, scenarioName)
		if err != nil || apiURL == "" {
			if err != nil && s.logger != nil {
				s.logger.Warn("validation_api_url_unavailable", "scenario", scenarioName, "error", err)
			}
		} else {
			env = append(env, "VROOLI_VALIDATION_API_URL="+apiURL)
		}
	}
	return env
}

func (s *DefaultService) validationRendererEnvForStatus(ctx context.Context, status *Status) []string {
	if status == nil {
		return nil
	}
	if status.DeploymentMode == "bundled" {
		// The bundled runtime discovers its private service ports through its
		// control API. Never inject a Tier 1 URL into a bundled launch.
		return nil
	}
	if status.DeploymentMode == "proxy" && s.targetResolver != nil {
		target, err := s.targetResolver(ctx, status)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("isolated_smoke_target_unavailable", "scenario", status.ScenarioName, "error", err)
			}
			return nil
		}
		result := make([]string, 0, 2)
		if target.RendererURL != "" {
			result = append(result, "VROOLI_VALIDATION_RENDERER_URL="+target.RendererURL)
		}
		if target.APIURL != "" {
			result = append(result, "VROOLI_VALIDATION_API_URL="+target.APIURL)
		}
		return result
	}
	// Unknown-mode legacy requests are intentionally targetless. A smoke run
	// must not silently rediscover the live instance.
	return nil
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
		targetResolver:     resolveSmokeTarget,
	}
}

// WithRecording enables screen recording on an existing service.
func (s *DefaultService) WithRecording(recorder screenrecording.Recorder, displayMgr screenrecording.DisplayManager) {
	s.recorder = recorder
	s.displayMgr = displayMgr
}

// WithCaptures makes the captures domain the only durable home for smoke-test
// recordings. Status retains metadata and a capture ID, never a local path.
func (s *DefaultService) WithCaptures(service *captures.Service) { s.captures = service }

func (s *DefaultService) annotateCapture(capture *captures.Capture, pipelineID string) {
	if capture == nil || pipelineID == "" || s == nil || s.captures == nil {
		return
	}
	if annotator, ok := s.captures.Store().(captures.PipelineAnnotator); ok {
		if err := annotator.UpdatePipelineID(capture.ScenarioName, capture.ID, pipelineID); err != nil && s.logger != nil {
			s.logger.Warn("capture_pipeline_id_persist_failed", "capture_id", capture.ID, "error", err)
		}
	}
}

// WithMonitor sets the process monitor factory for tracking app startup time and resource usage.
func (s *DefaultService) WithMonitor(factory procmetrics.MonitorFactory) {
	s.monitorFactory = factory
}

// WithWindowDetector wires the shared xdotool detector into the evidence
// journey. The same detector is used by startup metrics, so availability and
// visible-window behavior stay consistent across both evidence paths.
func (s *DefaultService) WithWindowDetector(detector *procmetrics.XdotoolDetector) {
	s.windowDetector = detector
	if detector != nil {
		s.journeyDriver = procmetricsDesktopDriver{detector: detector}
	}
}

// WithJourneySeams replaces platform, timing, readiness, capture, and API
// dependencies for deterministic contract tests. Production callers normally
// use WithWindowDetector and the default clock/wait/capture adapters.
func (s *DefaultService) WithJourneySeams(driver DesktopDriver, clock Clock, waiter JourneyWaiter, capture JourneyCapture, api JourneyAPIProbe) {
	s.journeyDriver = driver
	s.journeyClock = clock
	s.journeyWaiter = waiter
	s.journeyCapture = capture
	s.journeyAPI = api
}

// WithJourneyProcessObserver adds a credential-free process observation seam
// to the journey without making process monitoring part of desktop actions.
func (s *DefaultService) WithJourneyProcessObserver(observer JourneyProcessObserver) {
	s.journeyProcess = observer
}

// WithJourneyCapability selects an explicit behavior fixture for the next
// smoke journey. Empty values retain registry lookup by scenario identity.
func (s *DefaultService) WithJourneyCapability(capability string) {
	s.journeyCapability = capability
}

// EvidenceReporter is optional so local smoke tests do not acquire a startup
// dependency on deployment-manager. When configured, it receives references
// after the journey settles and reports transport failures explicitly.
type EvidenceReporter interface {
	ReportJourney(context.Context, EvidenceReportInput) error
}

// EvidenceManifestWriter persists the producer-owned, reviewable manifest
// after capture and governance reporting have settled.
type EvidenceManifestWriter interface {
	WriteManifest(context.Context, EvidenceManifestInput) error
}

type EvidenceManifestInput struct {
	RunID                   string
	ScenarioName            string
	Platform                string
	ArtifactPath            string
	Profile                 string
	StartedAt               time.Time
	CompletedAt             time.Time
	Journey                 *deliveryramp.JourneyResult
	WorkflowReference       *deliveryramp.WorkflowExecutionReference
	Captures                []captures.Capture
	GovernanceReported      bool
	ProtocolTracePath       string
	DemoTracePath           string
	PerformanceArtifacts    []PerformanceArtifact
	ProtocolResourceSummary *procmetrics.Summary
	ProtocolProcessTree     *procmetrics.ProcessTreeReport
	DemoResourceSummary     *procmetrics.Summary
	DemoProcessTree         *procmetrics.ProcessTreeReport
	ProtocolProfileDir      string
	DemoProfileDir          string
	ProfileMode             string
	ProtocolPassed          bool
	VisualReadiness         string
	JourneyCaptureID        string
	RecordingCaptureID      string
	ScreenContentSource     string
	IsolationObservations   []deliveryramp.IsolationObservation
}

// PerformanceArtifact is a producer-owned file with an immutable checksum.
// LocalPath is retained only for local review; remote consumers receive the
// immutable reference, checksum, and size through the manifest.
type PerformanceArtifact struct {
	ImmutableRef string
	LocalPath    string
	Kind         string
	Checksum     string
	SizeBytes    int64
	Available    bool
	Reason       string
}

type EvidenceReportInput struct {
	ProfileID             string
	GitCommit             string
	ArtifactDigest        string
	CandidateID           string
	DestinationRevisionID string
	AuthorizationEpoch    uint64
	PolicyVersion         int
	Channel               string
	ScenarioName          string
	Platform              string
	RunID                 string
	Disposition           string
	Target                *domainv1.EvidenceTarget
	Captures              []captures.Capture
	Journey               *deliveryramp.JourneyResult
	ProducerBaseURL       string
}

func (s *DefaultService) WithEvidenceReporter(reporter EvidenceReporter) {
	s.evidenceReporter = reporter
}

func (s *DefaultService) WithEvidenceManifestWriter(writer EvidenceManifestWriter) {
	s.manifestWriter = writer
}

// CurrentPlatform returns the current platform identifier.
func (s *DefaultService) CurrentPlatform() string {
	return s.platformResolver.CurrentPlatform()
}

func (s *DefaultService) recordTargetObservation(smokeTestID string, target JourneyTarget) {
	if s.store == nil || target.Isolation == nil {
		return
	}
	observations := []deliveryramp.IsolationObservation{*target.Isolation}
	s.store.Update(smokeTestID, func(status *Status) {
		status.IsolationObservations = observations
		status.ScreenContentSource = deriveScreenContentSource(observations)
	})
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
