// Package wire provides dependency injection and wiring for the browser-automation-studio API.
// It separates infrastructure setup from business logic, following the
// "boundary of responsibility enforcement" principle.
//
// This package is the single source of truth for:
//   - Production dependency construction
//   - Service interconnection
//   - Infrastructure initialization order
//
// Handlers and services should receive their dependencies through this package
// rather than constructing them directly. This enables:
//   - Easier testing through dependency injection
//   - Clear separation between infrastructure and business logic
//   - Single point of modification for dependency changes
package wire

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

// Dependencies holds all injectable dependencies for the application.
// This struct centralizes dependency management and makes it clear what
// components are needed to run the application.
type Dependencies struct {
	// Core services
	WorkflowService   *workflow.WorkflowService
	RecordModeService *livecapture.Service
	RecordingImport   archiveingestion.IngestionServiceInterface

	// Validators
	WorkflowValidator *workflowvalidator.Validator

	// Storage and infrastructure
	Storage        storage.StorageInterface
	RecordingsRoot string
	ReplayRenderer ReplayRenderer

	// Session management
	SessionProfiles *archiveingestion.SessionProfileStore

	// Optional integrations
	UXMetricsRepo uxmetrics.Repository
}

// ReplayRenderer is the interface for rendering replay videos.
type ReplayRenderer interface {
	Render(ctx context.Context, spec *replay.ReplayMovieSpec, format replay.RenderFormat, filename string) (*replay.RenderedMedia, error)
}

// Config holds configuration for dependency construction.
type Config struct {
	// UXMetricsRepo enables UX metrics collection when provided.
	UXMetricsRepo uxmetrics.Repository

	// SkipEngineValidation skips validation of the automation engine on startup.
	// Useful for environments where the engine isn't available.
	SkipEngineValidation bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		SkipEngineValidation: false,
	}
}

// BuildDependencies constructs all production dependencies.
// This is the main entry point for dependency injection.
func BuildDependencies(repo database.Repository, hub *wsHub.Hub, log *logrus.Logger, cfg Config) (*Dependencies, error) {
	// Initialize storage infrastructure
	screenshotRoot := paths.ResolveScreenshotsRoot(log)
	storageClient := storage.NewScreenshotStorage(log, screenshotRoot)

	// Initialize recordings infrastructure
	recordingsRoot := paths.ResolveRecordingsRoot(log)
	recordingImportSvc := archiveingestion.NewIngestionService(repo, storageClient, hub, log, recordingsRoot)
	sessionProfiles := archiveingestion.NewSessionProfileStore(paths.ResolveSessionProfilesRoot(log), log)

	// Wire automation stack
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactory(log)
	if engErr != nil && !cfg.SkipEngineValidation {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}

	// Persist execution artifacts under recordingsRoot
	autoRecorder := executionwriter.NewFileRecorder(repo, storageClient, log, recordingsRoot)

	// Configure event sink factory - optionally wrap with UX metrics collector
	var eventSinkFactory func() autoevents.Sink
	if cfg.UXMetricsRepo != nil {
		eventSinkFactory = func() autoevents.Sink {
			baseSink := autoevents.NewWSHubSink(hub, log, eventBufferLimits())
			return uxcollector.NewCollector(baseSink, cfg.UXMetricsRepo)
		}
		log.Debug("UX metrics collector enabled in event pipeline")
	}

	// Create workflow service with dependencies
	workflowSvc := workflow.NewWorkflowServiceWithDeps(repo, hub, log, workflow.WorkflowServiceOptions{
		Executor:          autoExecutor,
		EngineFactory:     autoEngineFactory,
		ArtifactRecorder:  autoRecorder,
		EventSinkFactory:  eventSinkFactory,
		ExecutionDataRoot: recordingsRoot,
	})

	// Ensure the demo project exists
	if workflowSvc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if _, err := workflowSvc.EnsureSeedProject(ctx); err != nil && log != nil {
			log.WithError(err).Warn("Failed to ensure seed project")
		}
		cancel()
	}

	// Create validator
	validatorInstance, err := workflowvalidator.NewValidator()
	if err != nil {
		return nil, err
	}

	// Create record mode service
	recordModeSvc := livecapture.NewService(log)

	return &Dependencies{
		WorkflowService:   workflowSvc,
		RecordModeService: recordModeSvc,
		RecordingImport:   recordingImportSvc,
		WorkflowValidator: validatorInstance,
		Storage:           storageClient,
		RecordingsRoot:    recordingsRoot,
		ReplayRenderer:    replay.NewReplayRenderer(log, recordingsRoot),
		SessionProfiles:   sessionProfiles,
		UXMetricsRepo:     cfg.UXMetricsRepo,
	}, nil
}

// eventBufferLimits returns validated event buffer limits from config.
func eventBufferLimits() autocontracts.EventBufferLimits {
	limits := autocontracts.EventBufferLimits{
		PerExecution: config.Load().Events.PerExecutionBuffer,
		PerAttempt:   config.Load().Events.PerAttemptBuffer,
	}
	if limits.Validate() != nil {
		return autocontracts.DefaultEventBufferLimits
	}
	return limits
}

// HandlerDeps converts Dependencies to the format expected by handlers.
// This provides backward compatibility while migrating to the new wire package.
type HandlerDeps struct {
	WorkflowCatalog   *workflow.WorkflowService
	ExecutionService  *workflow.WorkflowService
	ExportService     *workflow.WorkflowService
	WorkflowValidator *workflowvalidator.Validator
	Storage           storage.StorageInterface
	RecordingService  archiveingestion.IngestionServiceInterface
	RecordingsRoot    string
	ReplayRenderer    ReplayRenderer
	SessionProfiles   *archiveingestion.SessionProfileStore
	UXMetricsRepo     uxmetrics.Repository
}

// ToHandlerDeps converts Dependencies to HandlerDeps for backward compatibility.
func (d *Dependencies) ToHandlerDeps() HandlerDeps {
	return HandlerDeps{
		WorkflowCatalog:   d.WorkflowService,
		ExecutionService:  d.WorkflowService,
		ExportService:     d.WorkflowService,
		WorkflowValidator: d.WorkflowValidator,
		Storage:           d.Storage,
		RecordingService:  d.RecordingImport,
		RecordingsRoot:    d.RecordingsRoot,
		ReplayRenderer:    d.ReplayRenderer,
		SessionProfiles:   d.SessionProfiles,
		UXMetricsRepo:     d.UXMetricsRepo,
	}
}
