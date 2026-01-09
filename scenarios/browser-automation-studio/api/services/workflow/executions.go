package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// ExecuteWorkflow starts a workflow execution and returns the DB index record.
// Detailed execution results are written to disk by the artifact recorder.
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	if s == nil {
		return nil, errors.New("workflow service not configured")
	}

	getResp, err := s.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: workflowID.String()})
	if err != nil {
		return nil, err
	}
	if getResp == nil || getResp.Workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	now := time.Now().UTC()
	exec := &database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     database.ExecutionStatusPending,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	_ = s.writeExecutionSnapshot(ctx, exec, &basexecution.Execution{
		ExecutionId: exec.ID.String(),
		WorkflowId:  workflowID.String(),
		Status:      enums.StringToExecutionStatus(exec.Status),
		TriggerType: basbase.TriggerType_TRIGGER_TYPE_MANUAL,
		StartedAt:   autocontracts.TimeToTimestamp(now),
		CreatedAt:   autocontracts.TimeToTimestamp(now),
		UpdatedAt:   autocontracts.TimeToTimestamp(now),
	})

	s.startExecutionRunner(getResp.Workflow, exec.ID, parameters)
	return exec, nil
}

// ExecuteOptions contains optional settings for workflow execution.
type ExecuteOptions struct {
	// EnableFrameStreaming enables live frame streaming during execution.
	// When true, the execution will stream browser frames to connected WebSocket clients.
	EnableFrameStreaming bool
	// FrameStreamingQuality sets the JPEG quality (1-100). Default: 55.
	FrameStreamingQuality int
	// FrameStreamingFPS sets the target frames per second. Default: 6.
	FrameStreamingFPS int
	// RequiresVideo forces video capture capability on the execution plan metadata.
	RequiresVideo bool
	// RequiresTrace forces trace capture capability on the execution plan metadata.
	RequiresTrace bool
	// RequiresHAR forces HAR capture capability on the execution plan metadata.
	RequiresHAR bool
}

// ExecuteWorkflowAPI starts a workflow execution using proto request/response types.
func (s *WorkflowService) ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error) {
	return s.ExecuteWorkflowAPIWithOptions(ctx, req, nil)
}

// ExecuteWorkflowAPIWithOptions starts a workflow execution with additional options.
func (s *WorkflowService) ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID, err := uuid.Parse(strings.TrimSpace(req.WorkflowId))
	if err != nil {
		return nil, fmt.Errorf("invalid workflow_id: %w", err)
	}
	version := int(req.GetWorkflowVersion())

	workflowSummary, err := s.resolveWorkflowForExecution(ctx, workflowID, version)
	if err != nil {
		return nil, err
	}

	initialStore, initialParams, env, artifactCfg, execBrowserProfile, projectRoot, startURL, sessionProfileID, navigationWaitUntil, continueOnError := executionParametersToMaps(req.Parameters)
	if err := validateSeedRequirements(workflowSummary.FlowDefinition, initialStore, initialParams, env); err != nil {
		return nil, err
	}

	// Resolve session profile if provided for authenticated execution
	var storageState json.RawMessage
	var profileBrowserSettings *archiveingestion.BrowserProfile
	if sessionProfileID != "" && s.sessionProfiles != nil {
		profile, err := s.sessionProfiles.Get(sessionProfileID)
		if err != nil {
			return nil, fmt.Errorf("session profile %s not found: %w", sessionProfileID, err)
		}
		storageState = profile.StorageState
		profileBrowserSettings = profile.BrowserProfile

		// Update usage tracking to keep profile LRU order current
		if _, err := s.sessionProfiles.Touch(sessionProfileID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("session_profile_id", sessionProfileID).Warn("Failed to update session profile usage timestamp")
		}
	}

	// Extract workflow default browser profile and merge with execution override
	// Priority order: execution params > session profile > workflow defaults
	var workflowBrowserProfile *archiveingestion.BrowserProfile
	if workflowSummary.FlowDefinition != nil && workflowSummary.FlowDefinition.Settings != nil && workflowSummary.FlowDefinition.Settings.BrowserProfile != nil {
		workflowBrowserProfile = archiveingestion.BrowserProfileFromProto(workflowSummary.FlowDefinition.Settings.BrowserProfile)
	}
	// First merge workflow defaults with session profile defaults
	baseBrowserProfile := archiveingestion.MergeBrowserProfiles(workflowBrowserProfile, profileBrowserSettings)
	// Then merge with execution-level overrides (highest priority)
	finalBrowserProfile := archiveingestion.MergeBrowserProfiles(baseBrowserProfile, execBrowserProfile)

	now := time.Now().UTC()
	exec := &database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     database.ExecutionStatusPending,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return nil, err
	}

	// Persist an initial proto snapshot immediately so the filesystem is the source of truth
	// for parameters/trigger metadata and other rich execution fields not stored in the DB index.
	snapshot := &basexecution.Execution{
		ExecutionId:     exec.ID.String(),
		WorkflowId:      workflowID.String(),
		WorkflowVersion: int32(version),
		Status:          enums.StringToExecutionStatus(exec.Status),
		TriggerType:     basbase.TriggerType_TRIGGER_TYPE_API,
		StartedAt:       autocontracts.TimeToTimestamp(now),
		CreatedAt:       autocontracts.TimeToTimestamp(now),
		UpdatedAt:       autocontracts.TimeToTimestamp(now),
		Parameters:      req.Parameters,
	}
	_ = s.writeExecutionSnapshot(ctx, exec, snapshot)

	s.startExecutionRunnerWithOptions(workflowSummary, exec.ID, initialStore, initialParams, env, artifactCfg, finalBrowserProfile, storageState, opts, projectRoot, startURL, navigationWaitUntil, continueOnError)

	if req.WaitForCompletion {
		// Poll for completion; execution updates are persisted to the DB index by the runner.
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				latest, err := s.repo.GetExecution(ctx, exec.ID)
				if err != nil {
					return nil, err
				}
				if latest.CompletedAt != nil {
					resp := &basapi.ExecuteWorkflowResponse{
						ExecutionId: latest.ID.String(),
						Status:      enums.StringToExecutionStatus(latest.Status),
						CompletedAt: autocontracts.TimePtrToTimestamp(latest.CompletedAt),
					}
					if strings.TrimSpace(latest.ErrorMessage) != "" {
						msg := latest.ErrorMessage
						resp.Error = &msg
					}
					return resp, nil
				}
			}
		}
	}

	return &basapi.ExecuteWorkflowResponse{
		ExecutionId: exec.ID.String(),
		Status:      enums.StringToExecutionStatus(exec.Status),
	}, nil
}

func (s *WorkflowService) resolveWorkflowForExecution(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	if version > 0 {
		return s.GetWorkflowVersion(ctx, workflowID, version)
	}
	getResp, err := s.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: workflowID.String()})
	if err != nil {
		return nil, err
	}
	if getResp == nil || getResp.Workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	return getResp.Workflow, nil
}

// executionParametersToMaps extracts namespace maps, artifact config, browser profile, project root, start URL, and session profile ID from ExecutionParameters.
// Returns: store (@store/ namespace), params (@params/ namespace), env (environment), artifact config, browser profile, projectRoot, startURL, sessionProfileID, navigationWaitUntil, continueOnError.
// projectRoot is used for filesystem-based subflow resolution when the calling workflow has no database project.
// sessionProfileID references a stored session profile for authenticated execution.
// navigationWaitUntil and continueOnError are execution-level defaults that override workflow settings.
func executionParametersToMaps(p *basexecution.ExecutionParameters) (store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *archiveingestion.BrowserProfile, projectRoot string, startURL string, sessionProfileID string, navigationWaitUntil string, continueOnError *bool) {
	store = map[string]any{}
	params = map[string]any{}
	env = map[string]any{}
	if p == nil {
		return store, params, env, nil, nil, "", "", "", "", nil
	}

	for k, v := range p.InitialStore {
		store[k] = jsonValueToAny(v)
	}
	for k, v := range p.InitialParams {
		params[k] = jsonValueToAny(v)
	}
	for k, v := range p.Env {
		env[k] = jsonValueToAny(v)
	}
	// Backward compatibility: variables map<string,string> feeds @store/ as strings.
	for k, v := range p.Variables {
		if _, exists := store[k]; !exists {
			store[k] = v
		}
	}

	// Extract artifact collection config if provided
	if p.ArtifactConfig != nil {
		settings := config.ResolveArtifactSettings(p.ArtifactConfig)
		artifactCfg = &settings
	}

	// Extract browser profile for anti-detection and human-like behavior if provided
	if p.BrowserProfile != nil {
		browserProfile = archiveingestion.BrowserProfileFromProto(p.BrowserProfile)
	}

	// Extract project_root for filesystem-based subflow resolution.
	// Used by adhoc workflows that need to resolve workflowPath references.
	if p.ProjectRoot != nil {
		projectRoot = strings.TrimSpace(*p.ProjectRoot)
	}

	startURL = strings.TrimSpace(p.GetStartUrl())

	// Extract session profile ID for authenticated execution.
	// When set, the profile's storage state (cookies, localStorage) will be loaded into the browser context.
	if p.SessionProfileId != nil {
		sessionProfileID = strings.TrimSpace(*p.SessionProfileId)
	}

	// Extract execution-level defaults for navigation and error handling.
	// These override workflow defaults but can be further overridden by per-node settings.
	if p.NavigationWaitUntil != nil {
		navigationWaitUntil = navigateWaitEventToString(*p.NavigationWaitUntil)
	}
	continueOnError = p.ContinueOnError

	return store, params, env, artifactCfg, browserProfile, projectRoot, startURL, sessionProfileID, navigationWaitUntil, continueOnError
}

func jsonValueToAny(v *commonv1.JsonValue) any {
	return typeconv.JsonValueToAny(v)
}

// navigateWaitEventToString converts the NavigateWaitEvent enum to a string value.
func navigateWaitEventToString(e basactions.NavigateWaitEvent) string {
	switch e {
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_LOAD:
		return "load"
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED:
		return "domcontentloaded"
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE:
		return "networkidle"
	default:
		return ""
	}
}

func (s *WorkflowService) startExecutionRunner(workflow *basapi.WorkflowSummary, executionID uuid.UUID, parameters map[string]any) {
	// Normalize legacy flat parameters into the namespaced model.
	// All legacy parameters go to @store/ namespace for backward compatibility.
	// Use default artifact config (full profile) and no projectRoot (legacy callers don't use subflows).
	s.startExecutionRunnerWithNamespaces(workflow, executionID, parameters, nil, nil, nil, "")
}

func (s *WorkflowService) startExecutionRunnerWithNamespaces(workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, projectRoot string) {
	s.startExecutionRunnerWithOptions(workflow, executionID, store, params, env, artifactCfg, nil, nil, nil, projectRoot, "", "", nil)
}

func (s *WorkflowService) startExecutionRunnerWithOptions(workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *archiveingestion.BrowserProfile, storageState json.RawMessage, opts *ExecuteOptions, projectRoot string, startURL string, navigationWaitUntil string, continueOnError *bool) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(executionID, cancel)
	go s.executeWorkflowAsyncWithOptions(ctx, workflow, executionID, store, params, env, artifactCfg, browserProfile, storageState, opts, projectRoot, startURL, navigationWaitUntil, continueOnError)
}

// executeWorkflowAsync is the single implementation for running workflows asynchronously.
// It handles both legacy (flat parameters in store) and new (namespaced store/params/env) callers.
// artifactCfg controls what artifacts are collected; nil means use default (full profile).
func (s *WorkflowService) executeWorkflowAsync(ctx context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings) {
	s.executeWorkflowAsyncWithOptions(ctx, workflow, executionID, store, params, env, artifactCfg, nil, nil, nil, "", "", "", nil)
}

// executeWorkflowAsyncWithOptions runs a workflow asynchronously with optional settings.
// projectRoot is the absolute path to the project root for filesystem-based subflow resolution.
// browserProfile configures anti-detection and human-like behavior settings for the execution.
// storageState contains cookies/localStorage from a session profile for authenticated execution.
// navigationWaitUntil and continueOnError are execution-level defaults that override workflow settings.
func (s *WorkflowService) executeWorkflowAsyncWithOptions(ctx context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *archiveingestion.BrowserProfile, storageState json.RawMessage, opts *ExecuteOptions, projectRoot string, startURL string, navigationWaitUntil string, continueOnError *bool) {
	defer s.cancelExecutionByID(executionID)

	persistenceCtx := context.Background()
	execIndex, err := s.repo.GetExecution(persistenceCtx, executionID)
	if err != nil {
		return
	}

	execIndex.Status = database.ExecutionStatusRunning
	execIndex.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, nil, nil, execIndex.UpdatedAt)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      enums.StringToExecutionStatus(execIndex.Status),
		StartedAt:   autocontracts.TimeToTimestamp(execIndex.StartedAt),
		CreatedAt:   autocontracts.TimeToTimestamp(execIndex.CreatedAt),
		UpdatedAt:   autocontracts.TimeToTimestamp(execIndex.UpdatedAt),
	})

	engineName := autoengine.FromEnv().Resolve("")
	eventSink := s.newEventSink()

	// Configure artifact collection settings on the recorder before execution starts.
	// This allows per-execution customization of what artifacts are collected.
	if s.artifactRecorder != nil {
		s.artifactRecorder.SetArtifactConfig(artifactCfg)
	}

	plan, _, err := autoexecutor.BuildContractsPlan(ctx, executionID, workflow)
	if err != nil {
		execIndex.Status = database.ExecutionStatusFailed
		execIndex.ErrorMessage = err.Error()
		now := time.Now().UTC()
		execIndex.CompletedAt = &now
		execIndex.UpdatedAt = now
		errMsg := execIndex.ErrorMessage
		_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, &errMsg, execIndex.CompletedAt, execIndex.UpdatedAt)
		return
	}

	if opts != nil && opts.RequiresVideo {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["requiresVideo"] = true
	}
	if opts != nil && opts.RequiresTrace {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["requiresTracing"] = true
	}
	if opts != nil && opts.RequiresHAR {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["requiresHar"] = true
	}

	// Inject frame streaming config into plan metadata if enabled.
	// The SimpleExecutor reads this from plan.Metadata["frameStreaming"] to configure
	// the driver session with frame streaming callbacks.
	if opts != nil && opts.EnableFrameStreaming {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		fsConfig := map[string]any{
			"enabled": true,
		}
		// Apply custom quality if specified, otherwise use default (55)
		if opts.FrameStreamingQuality > 0 {
			fsConfig["quality"] = opts.FrameStreamingQuality
		}
		// Apply custom FPS if specified, otherwise use default (6)
		if opts.FrameStreamingFPS > 0 {
			fsConfig["fps"] = opts.FrameStreamingFPS
		}
		plan.Metadata["frameStreaming"] = fsConfig
	}

	req := autoexecutor.Request{
		Plan:               plan,
		EngineName:         engineName,
		EngineFactory:      s.engineFactory,
		Recorder:           s.artifactRecorder,
		EventSink:          eventSink,
		HeartbeatInterval:  2 * time.Second,
		ReuseMode:          autoengine.ReuseModeReuse,
		WorkflowResolver:   s,
		PlanCompiler:       s.planCompiler,
		MaxSubflowDepth:    5,
		StartFromStepIndex: -1,
		ProjectRoot:        projectRoot,
		InitialStore:       store,
		InitialParams:      params,
		Env:                 env,
		StartURL:            strings.TrimSpace(startURL),
		ArtifactConfig:      artifactCfg,
		BrowserProfile:      browserProfile,
		StorageState:        storageState,
		NavigationWaitUntil: navigationWaitUntil,
		ContinueOnError:     continueOnError,
	}

	executor := s.executor
	if executor == nil {
		executor = autoexecutor.NewSimpleExecutor(nil)
	}
	runErr := executor.Execute(ctx, req)

	status := database.ExecutionStatusCompleted
	errMsg := ""
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || strings.Contains(strings.ToLower(runErr.Error()), "cancel") {
			status = database.ExecutionStatusFailed
			errMsg = "execution cancelled"
		} else {
			status = database.ExecutionStatusFailed
			errMsg = runErr.Error()
		}
	}

	now := time.Now().UTC()
	execIndex.Status = status
	execIndex.ErrorMessage = errMsg
	execIndex.CompletedAt = &now
	execIndex.UpdatedAt = now
	var errPtr *string
	if strings.TrimSpace(execIndex.ErrorMessage) != "" {
		errPtr = &execIndex.ErrorMessage
	}
	_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, errPtr, execIndex.CompletedAt, execIndex.UpdatedAt)
	_ = s.writeExecutionSnapshot(persistenceCtx, execIndex, &basexecution.Execution{
		ExecutionId: execIndex.ID.String(),
		WorkflowId:  execIndex.WorkflowID.String(),
		Status:      enums.StringToExecutionStatus(execIndex.Status),
		StartedAt:   autocontracts.TimeToTimestamp(execIndex.StartedAt),
		CreatedAt:   autocontracts.TimeToTimestamp(execIndex.CreatedAt),
		UpdatedAt:   autocontracts.TimeToTimestamp(execIndex.UpdatedAt),
		CompletedAt: autocontracts.TimeToTimestamp(now),
		Error: func() *string {
			if strings.TrimSpace(execIndex.ErrorMessage) == "" {
				return nil
			}
			msg := execIndex.ErrorMessage
			return &msg
		}(),
	})

	// Emit execution completion event via WebSocket so UI gets notified of the final status
	if eventSink != nil {
		eventKind := autocontracts.EventKindExecutionCompleted
		if status == database.ExecutionStatusFailed {
			eventKind = autocontracts.EventKindExecutionFailed
		}
		payload := map[string]any{
			"status": status,
		}
		if errMsg != "" {
			payload["error"] = errMsg
		}
		_ = eventSink.Publish(persistenceCtx, autocontracts.EventEnvelope{
			SchemaVersion:  autocontracts.EventEnvelopeSchemaVersion,
			PayloadVersion: autocontracts.PayloadVersion,
			Kind:           eventKind,
			ExecutionID:    executionID,
			WorkflowID:     execIndex.WorkflowID,
			Timestamp:      now,
			Payload:        payload,
		})

		// Close the execution on the event sink to clean up resources
		if wsSink, ok := eventSink.(*autoevents.WSHubSink); ok {
			wsSink.CloseExecution(executionID)
		}
	}
}

func (s *WorkflowService) storeExecutionCancel(executionID uuid.UUID, cancel context.CancelFunc) {
	if s == nil {
		return
	}
	s.executionCancels.Store(executionID, cancel)
}

func (s *WorkflowService) cancelExecutionByID(executionID uuid.UUID) {
	if s == nil {
		return
	}
	if value, ok := s.executionCancels.Load(executionID); ok {
		if cancel, ok := value.(context.CancelFunc); ok && cancel != nil {
			cancel()
		}
	}
	s.executionCancels.Delete(executionID)
}

func (s *WorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	if s == nil {
		return errors.New("workflow service not configured")
	}
	_ = ctx
	s.cancelExecutionByID(executionID)
	return nil
}
