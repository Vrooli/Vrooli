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

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	"github.com/vrooli/browser-automation-studio/services/export/render"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	unifiedrecording "github.com/vrooli/browser-automation-studio/services/recording"
	unifiedpersistence "github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofile "github.com/vrooli/browser-automation-studio/services/session-profile"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	uxcollector "github.com/vrooli/browser-automation-studio/services/uxmetrics/collector"
	"github.com/vrooli/browser-automation-studio/services/vision"
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

	// Unified recording service
	// DOC: docs/architecture/recording.md#unified-recording
	UnifiedRecordingService *unifiedrecording.Service
	UnifiedRecordingRepo    unifiedpersistence.Repository

	// Vision navigation
	PlaywrightNavigator *vision.PlaywrightVisionNavigator
	NavigatorRegistry   *vision.NavigatorRegistry

	// Validators
	WorkflowValidator *workflowvalidator.Validator

	// Storage and infrastructure
	Storage        storage.StorageInterface
	RecordingsRoot string
	ReplayRenderer ReplayRenderer

	// Session management
	SessionProfileService *sessionprofile.Service

	// Optional integrations
	UXMetricsRepo uxmetrics.Repository
}

// ReplayRenderer is the interface for rendering replay videos.
type ReplayRenderer interface {
	Render(ctx context.Context, spec *render.ReplayMovieSpec, format render.RenderFormat, filename string) (*render.RenderedMedia, error)
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
func BuildDependencies(repo database.Repository, db *database.DB, hub *wsHub.Hub, log *logrus.Logger, cfg Config) (*Dependencies, error) {
	// Initialize recordings infrastructure
	recordingsRoot := paths.ResolveRecordingsRoot(log)
	// Store screenshots alongside other execution artifacts under recordingsRoot.
	storageClient := storage.NewScreenshotStorage(log, recordingsRoot)
	recordingImportSvc := archiveingestion.NewIngestionService(repo, storageClient, hub, log, recordingsRoot)
	sessionProfileSvc := sessionprofile.NewServiceWithPath(paths.ResolveSessionProfilesRoot(log), log)

	// Wire automation stack
	autoExecutor := autoexecutor.NewSimpleExecutor(nil)
	autoEngineFactory, engErr := autoengine.DefaultFactoryWithRecordingsRoot(log, recordingsRoot)
	if engErr != nil && !cfg.SkipEngineValidation {
		log.WithError(engErr).Warn("Failed to initialize automation engine; automation executor will be disabled")
	}

	// Persist execution artifacts under recordingsRoot
	autoRecorder := executionwriter.NewFileWriter(repo, storageClient, log, recordingsRoot)

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
		Executor:              autoExecutor,
		EngineFactory:         autoEngineFactory,
		ArtifactRecorder:      autoRecorder,
		EventSinkFactory:      eventSinkFactory,
		ExecutionDataRoot:     recordingsRoot,
		SessionProfileService: sessionProfileSvc,
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

	// Create unified recording service first (needed by live-capture service)
	// DOC: docs/architecture/recording.md#unified-recording
	var unifiedRecordingRepo unifiedpersistence.Repository
	var unifiedRecordingSvc *unifiedrecording.Service
	if db != nil {
		// Access underlying *sql.DB from *sqlx.DB embedded in database.DB
		unifiedRecordingRepo = unifiedpersistence.NewSQLiteRepository(db.DB.DB, log)
		unifiedRecordingSvc = unifiedrecording.NewService(
			unifiedRecordingRepo,
			hub,
			log,
			unifiedrecording.ServiceConfig{},
		)
		log.Info("✅ Unified recording service initialized")
	}

	// Create record mode service with unified recording service injected
	// This ensures all browser actions flow through a single recording pipeline
	recordModeSvc := livecapture.NewService(log, unifiedRecordingSvc)

	// Create vision navigators with recording callbacks
	playwrightNav := vision.NewPlaywrightVisionNavigator(
		log,
		vision.WithPlaywrightHub(hub),
	)

	// Connect vision navigator to live-capture service for unified AI action recording.
	// AI actions flow through the same AddTimelineAction path as manual actions,
	// with "source": "ai" in the payload to distinguish them.
	// DOC: docs/architecture/recording.md#unified-recording
	playwrightNav.SetActionRecordCallback(func(sessionID string, action *vision.RecordedNavigationAction) {
		// Convert AI navigation action to driver.RecordedAction
		driverAction := &driver.RecordedAction{
			ID:          uuid.New().String(),
			SessionID:   sessionID,
			SequenceNum: action.StepNumber,
			Timestamp:   action.Timestamp,
			ActionType:  action.ActionType,
			Confidence:  0.9, // AI actions have high confidence
			URL:         action.URL,
		}
		if action.Selector != "" {
			driverAction.Selector = &driver.SelectorSet{
				Primary: action.Selector,
			}
		}
		// Include AI-specific metadata and source marker
		driverAction.Payload = map[string]interface{}{
			"source": "ai",
		}
		if action.Reasoning != "" {
			driverAction.Payload["reasoning"] = action.Reasoning
		}

		// Route through live-capture service's unified recording pipeline
		// The source is determined from payload["source"] = "ai"
		recordModeSvc.AddTimelineAction(sessionID, driverAction, uuid.Nil)
	})
	log.Info("✅ Vision navigator connected to unified recording via live-capture service")

	// Create navigator registry - first registered navigator becomes the default
	navigatorRegistry := vision.NewNavigatorRegistry()
	navigatorRegistry.Register(playwrightNav)

	// Register ClaudeCode navigator (stub)
	claudeCodeNav := vision.NewClaudeCodeVisionNavigator(log)
	navigatorRegistry.Register(claudeCodeNav)

	return &Dependencies{
		WorkflowService:         workflowSvc,
		RecordModeService:       recordModeSvc,
		RecordingImport:         recordingImportSvc,
		UnifiedRecordingService: unifiedRecordingSvc,
		UnifiedRecordingRepo:    unifiedRecordingRepo,
		PlaywrightNavigator:     playwrightNav,
		NavigatorRegistry:       navigatorRegistry,
		WorkflowValidator:       validatorInstance,
		Storage:                 storageClient,
		RecordingsRoot:          recordingsRoot,
		ReplayRenderer:          render.NewReplayRenderer(log, recordingsRoot),
		SessionProfileService:   sessionProfileSvc,
		UXMetricsRepo:           cfg.UXMetricsRepo,
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
	WorkflowCatalog       *workflow.WorkflowService
	ExecutionService      *workflow.WorkflowService
	ExportService         *workflow.WorkflowService
	WorkflowValidator     *workflowvalidator.Validator
	Storage               storage.StorageInterface
	RecordingService      archiveingestion.IngestionServiceInterface
	RecordingsRoot        string
	ReplayRenderer        ReplayRenderer
	SessionProfileService *sessionprofile.Service
	UXMetricsRepo         uxmetrics.Repository

	// Unified recording service
	// DOC: docs/architecture/recording.md#unified-recording
	UnifiedRecordingService *unifiedrecording.Service

	// Vision navigation
	PlaywrightNavigator *vision.PlaywrightVisionNavigator
	NavigatorRegistry   *vision.NavigatorRegistry
}

// ToHandlerDeps converts Dependencies to HandlerDeps for backward compatibility.
func (d *Dependencies) ToHandlerDeps() HandlerDeps {
	return HandlerDeps{
		WorkflowCatalog:         d.WorkflowService,
		ExecutionService:        d.WorkflowService,
		ExportService:           d.WorkflowService,
		WorkflowValidator:       d.WorkflowValidator,
		Storage:                 d.Storage,
		RecordingService:        d.RecordingImport,
		RecordingsRoot:          d.RecordingsRoot,
		ReplayRenderer:          d.ReplayRenderer,
		SessionProfileService:   d.SessionProfileService,
		UXMetricsRepo:           d.UXMetricsRepo,
		UnifiedRecordingService: d.UnifiedRecordingService,
		PlaywrightNavigator:     d.PlaywrightNavigator,
		NavigatorRegistry:       d.NavigatorRegistry,
	}
}
