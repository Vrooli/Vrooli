package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
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
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		Status:      database.ExecutionStatusPending,
		TriggerType: "manual",
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
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

	s.startExecutionRunner(ctx, getResp.Workflow, exec.ID, parameters)
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
	// RequiresPerfTrace forces CDP performance-trace + web-vitals capture
	// capability on the execution plan metadata. The driver attaches a
	// PerformanceTracer to the session and emits performance.json +
	// performance.web-vitals.json into the execution's artifact root.
	RequiresPerfTrace bool
	// RequiresAccessibility forces a CDP accessibility-tree snapshot on the
	// execution plan metadata. The driver walks the AX tree after the page
	// settles and writes the normalized accessibility.json
	// (bas-accessibility-snapshot/v1) into the execution's artifact root.
	RequiresAccessibility bool
	// ElectronTarget attaches the workflow to a target-owned Electron
	// renderer. ValidationContext must be provided with it; the context binds
	// the workflow to its artifact, target, and leased test storage.
	ElectronTarget    *autodriver.ElectronTarget
	ValidationContext *autodriver.ValidationContext
}

// requiredVideoArtifactError turns the requires-video request flag into an
// evidence contract. Capture being enabled is not sufficient: callers asking
// for video need a durable artifact they can attach to a journey or release
// record, and a missing artifact must make the execution fail closed.
func requiredVideoArtifactError(required bool, videos []ExecutionVideoArtifact, lookupErr error) error {
	if !required {
		return nil
	}
	if lookupErr != nil {
		return fmt.Errorf("required video artifact could not be verified: %w", lookupErr)
	}
	if len(videos) == 0 {
		return errors.New("required video artifact was not produced")
	}
	return nil
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

	initialStore, initialParams, env, artifactCfg, execBrowserProfile, projectRoot, startURL, sessionProfileID, saveSessionProfileID, restoreTabs, navigationWaitUntil, continueOnError := executionParametersToMaps(req.Parameters)
	if err := validateProjectRoot(projectRoot); err != nil {
		return nil, err
	}
	if projectRoot == "" {
		projectRoot, err = executionProjectRoot(ctx, s.repo, workflowID)
		if err != nil {
			return nil, fmt.Errorf("resolve workflow project root: %w", err)
		}
	}
	if err := validateSeedRequirements(workflowSummary.FlowDefinition, initialStore, initialParams, env); err != nil {
		return nil, err
	}

	// Resolve session profile if provided for authenticated execution
	var storageState json.RawMessage
	var profileBrowserSettings *sessionprofilepersistence.BrowserProfile
	var openTabs []sessionprofilepersistence.TabState
	if sessionProfileID != "" && s.sessionProfileService != nil {
		profile, err := s.sessionProfileService.GetProfile(sessionprofilepersistence.ProfileID(sessionProfileID))
		if err != nil {
			return nil, fmt.Errorf("session profile %s not found: %w", sessionProfileID, err)
		}
		storageState = profile.StorageState
		profileBrowserSettings = profile.BrowserProfile

		// Load open tabs if tab restoration is requested
		if restoreTabs && len(profile.OpenTabs) > 0 {
			openTabs = profile.OpenTabs
		}

		// Update usage tracking to keep profile LRU order current
		if _, err := s.sessionProfileService.Touch(sessionprofilepersistence.ProfileID(sessionProfileID)); err != nil && s.log != nil {
			s.log.WithError(err).WithField("session_profile_id", sessionProfileID).Warn("Failed to update session profile usage timestamp")
		}
	}

	// Extract workflow default browser profile and merge with execution override
	// Priority order: execution params > session profile > workflow defaults
	var workflowBrowserProfile *sessionprofilepersistence.BrowserProfile
	if workflowSummary.FlowDefinition != nil && workflowSummary.FlowDefinition.Settings != nil && workflowSummary.FlowDefinition.Settings.BrowserProfile != nil {
		workflowBrowserProfile = sessionprofilepersistence.BrowserProfileFromProto(workflowSummary.FlowDefinition.Settings.BrowserProfile)
	}
	// First merge workflow defaults with session profile defaults
	baseBrowserProfile := sessionprofilepersistence.MergeBrowserProfiles(workflowBrowserProfile, profileBrowserSettings)
	// Then merge with execution-level overrides (highest priority)
	finalBrowserProfile := sessionprofilepersistence.MergeBrowserProfiles(baseBrowserProfile, execBrowserProfile)

	now := time.Now().UTC()
	exec := &database.ExecutionIndex{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		Status:      database.ExecutionStatusPending,
		TriggerType: "api",
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
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

	s.startExecutionRunnerWithOptions(ctx, workflowSummary, exec.ID, initialStore, initialParams, env, artifactCfg, finalBrowserProfile, storageState, opts, projectRoot, startURL, saveSessionProfileID, restoreTabs, openTabs, navigationWaitUntil, continueOnError)

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

// executionProjectCatalog is the narrow persistence seam needed to locate the
// selector manifest root for a persisted workflow execution.
type executionProjectCatalog interface {
	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.WorkflowIndex, error)
	GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
}

// executionProjectRoot returns the persisted workflow's project directory.
// Explicit execution parameters still win; this is the safe default for
// ordinary project-backed workflow runs, including CLI executions.
func executionProjectRoot(ctx context.Context, catalog executionProjectCatalog, workflowID uuid.UUID) (string, error) {
	index, err := catalog.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}
	if index == nil || index.ProjectID == nil {
		return "", nil
	}
	project, err := catalog.GetProject(ctx, *index.ProjectID)
	if err != nil {
		return "", err
	}
	if project == nil {
		return "", nil
	}
	return strings.TrimSpace(project.FolderPath), nil
}

// validateProjectRoot rejects relative project_root values. A relative path
// would be resolved against the BAS server's working directory rather than the
// caller's, silently selecting the wrong selector manifest and scenario root.
func validateProjectRoot(projectRoot string) error {
	if projectRoot != "" && !filepath.IsAbs(projectRoot) {
		return fmt.Errorf("execution parameter project_root must be an absolute path, got %q (relative paths resolve against the BAS server working directory, not the caller's)", projectRoot)
	}
	return nil
}

// executionParametersToMaps extracts namespace maps, artifact config, browser profile, project root, start URL, session profile IDs, and execution defaults from ExecutionParameters.
// Returns: store (@store/ namespace), params (@params/ namespace), env (environment), artifact config, browser profile, projectRoot, startURL, sessionProfileID, saveSessionProfileID, restoreTabs, navigationWaitUntil, continueOnError.
// projectRoot is used for filesystem-based subflow resolution when the calling workflow has no database project.
// sessionProfileID references a stored session profile for authenticated execution.
// saveSessionProfileID specifies where to save storage state after execution.
// restoreTabs indicates whether to restore tabs from the session profile before execution.
// navigationWaitUntil and continueOnError are execution-level defaults that override workflow settings.
func executionParametersToMaps(p *basexecution.ExecutionParameters) (store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *sessionprofilepersistence.BrowserProfile, projectRoot string, startURL string, sessionProfileID string, saveSessionProfileID string, restoreTabs bool, navigationWaitUntil string, continueOnError *bool) {
	store = map[string]any{}
	params = map[string]any{}
	env = map[string]any{}
	if p == nil {
		return store, params, env, nil, nil, "", "", "", "", false, "", nil
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

	// Resolve artifact collection config, applying the operator-configured global
	// default profile (BAS_ARTIFACT_DEFAULT_PROFILE, "standard" by default) when
	// no per-execution config is supplied. An explicit per-execution profile wins.
	settings := config.ResolveArtifactSettingsWithDefault(p.ArtifactConfig, config.Load().ArtifactLimits.DefaultProfile)
	artifactCfg = &settings

	// Extract browser profile for anti-detection and human-like behavior if provided
	if p.BrowserProfile != nil {
		browserProfile = sessionprofilepersistence.BrowserProfileFromProto(p.BrowserProfile)
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

	// Extract save session profile ID for post-execution state capture.
	// When set, the browser's storage state will be saved to this profile after execution.
	if p.SaveSessionProfileId != nil {
		saveSessionProfileID = strings.TrimSpace(*p.SaveSessionProfileId)
	}

	// Extract tab restoration flag for session profile.
	// When true, restores the profile's saved tabs before execution.
	if p.RestoreTabs != nil {
		restoreTabs = *p.RestoreTabs
	}

	// Extract execution-level defaults for navigation and error handling.
	// These override workflow defaults but can be further overridden by per-node settings.
	if p.NavigationWaitUntil != nil {
		navigationWaitUntil = navigateWaitEventToString(*p.NavigationWaitUntil)
	}
	continueOnError = p.ContinueOnError

	return store, params, env, artifactCfg, browserProfile, projectRoot, startURL, sessionProfileID, saveSessionProfileID, restoreTabs, navigationWaitUntil, continueOnError
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

func (s *WorkflowService) startExecutionRunner(parent context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, parameters map[string]any) {
	// Normalize legacy flat parameters into the namespaced model.
	// All legacy parameters go to @store/ namespace for backward compatibility.
	// Use default artifact config (full profile) and no projectRoot (legacy callers don't use subflows).
	s.startExecutionRunnerWithNamespaces(parent, workflow, executionID, parameters, nil, nil, nil, "")
}

func (s *WorkflowService) startExecutionRunnerWithNamespaces(parent context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, projectRoot string) {
	s.startExecutionRunnerWithOptions(parent, workflow, executionID, store, params, env, artifactCfg, nil, nil, nil, projectRoot, "", "", false, nil, "", nil)
}

func (s *WorkflowService) startExecutionRunnerWithOptions(parent context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *sessionprofilepersistence.BrowserProfile, storageState json.RawMessage, opts *ExecuteOptions, projectRoot string, startURL string, saveSessionProfileID string, restoreTabs bool, openTabs []sessionprofilepersistence.TabState, navigationWaitUntil string, continueOnError *bool) {
	if coredb.IsTestMode(parent) {
		browserProfile = withTestModeBrowserHeader(browserProfile)
	}
	// Async work must outlive the request, but its routed-isolation marker is a
	// correctness boundary. Rebuild a background context with that one durable
	// execution attribute instead of retaining the request cancellation chain.
	ctx, cancel := context.WithCancel(detachedExecutionContext(parent))
	s.storeExecutionCancel(executionID, cancel)
	go s.executeWorkflowAsyncWithOptions(ctx, workflow, executionID, store, params, env, artifactCfg, browserProfile, storageState, opts, projectRoot, startURL, saveSessionProfileID, restoreTabs, openTabs, navigationWaitUntil, continueOnError)
}

// withTestModeBrowserHeader clones the optional browser profile before adding
// the header that makes in-page API calls participate in the same routed test
// lease as the initiating ExecuteAdhoc request. Browser UI fetches are a
// separate HTTP hop, so retaining the marker only on the server context is not
// sufficient isolation.
func withTestModeBrowserHeader(profile *sessionprofilepersistence.BrowserProfile) *sessionprofilepersistence.BrowserProfile {
	if profile == nil {
		return &sessionprofilepersistence.BrowserProfile{ExtraHeaders: map[string]string{"X-Vrooli-Test-Mode": "1"}}
	}
	cloned := *profile
	cloned.ExtraHeaders = make(map[string]string, len(profile.ExtraHeaders)+1)
	for key, value := range profile.ExtraHeaders {
		cloned.ExtraHeaders[key] = value
	}
	cloned.ExtraHeaders["X-Vrooli-Test-Mode"] = "1"
	return &cloned
}

func detachedExecutionContext(parent context.Context) context.Context {
	// Preserve the durable execution metadata (including routed test isolation)
	// while explicitly dropping the request's cancellation and deadline. The
	// caller owns this context's lifecycle through storeExecutionCancel.
	return context.WithoutCancel(parent)
}

// executeWorkflowAsyncWithOptions runs a workflow asynchronously with optional settings.
// projectRoot is the absolute path to the project root for filesystem-based subflow resolution.
// browserProfile configures anti-detection and human-like behavior settings for the execution.
// storageState contains cookies/localStorage from a session profile for authenticated execution.
// saveSessionProfileID specifies where to save storage state after successful execution.
// restoreTabs indicates whether to restore tabs from the session profile before execution.
// openTabs contains the saved tab states to restore (only used when restoreTabs is true).
// navigationWaitUntil and continueOnError are execution-level defaults that override workflow settings.
func (s *WorkflowService) executeWorkflowAsyncWithOptions(ctx context.Context, workflow *basapi.WorkflowSummary, executionID uuid.UUID, store map[string]any, params map[string]any, env map[string]any, artifactCfg *config.ArtifactCollectionSettings, browserProfile *sessionprofilepersistence.BrowserProfile, storageState json.RawMessage, opts *ExecuteOptions, projectRoot string, startURL string, saveSessionProfileID string, restoreTabs bool, openTabs []sessionprofilepersistence.TabState, navigationWaitUntil string, continueOnError *bool) {
	defer s.cancelExecutionByID(executionID)

	// Execution cancellation must stop the runner, but it must not cancel the
	// persistence context used to publish the terminal status and evidence.
	// Otherwise StopExecution leaves the durable record stuck in `running` and
	// callers cannot distinguish a cancelled execution from a hung one.
	persistenceCtx := context.WithoutCancel(ctx)
	if coredb.IsTestMode(persistenceCtx) {
		// Development routing installs an empty, lease-owned database. Seed the
		// minimum scenario fixture inside that pool so browser validations exercise
		// the same project surface as normal startup without touching primary data.
		if _, err := s.EnsureSeedWorkflow(persistenceCtx); err != nil && s.log != nil {
			s.log.WithError(err).WithField("execution_id", executionID).Warn("Failed to prepare routed test fixture")
		}
	}
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

	// Resolve the artifact config for this execution. Legacy callers
	// (ExecuteWorkflow with flat params) pass nil; fall back to the operator-
	// configured default profile rather than the "collect everything" default.
	if artifactCfg == nil {
		settings := config.ResolveArtifactSettingsWithDefault(nil, config.Load().ArtifactLimits.DefaultProfile)
		artifactCfg = &settings
	}

	// Configure artifact collection settings scoped to this execution before it
	// starts. Per-execution scoping prevents concurrent executions from leaking
	// each other's profile through the shared recorder.
	if s.artifactRecorder != nil {
		s.artifactRecorder.SetArtifactConfigForExecution(executionID, artifactCfg)
		defer s.artifactRecorder.ForgetExecution(executionID)
	}

	compileCtx := ctx
	if projectRoot != "" {
		compileCtx = autocompiler.WithProjectRoot(ctx, projectRoot)
	}

	plan, _, err := autoexecutor.BuildContractsPlan(compileCtx, executionID, workflow)
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
	if opts != nil && opts.RequiresPerfTrace {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["requiresPerformanceTrace"] = true
	}
	if opts != nil && opts.RequiresAccessibility {
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["requiresAccessibility"] = true
	}
	if opts != nil && opts.ElectronTarget != nil {
		if opts.ValidationContext == nil || strings.TrimSpace(opts.ValidationContext.IsolationLeaseID) == "" {
			execIndex.Status = database.ExecutionStatusFailed
			execIndex.ErrorMessage = "Electron target requires a lease-bound validation context"
			now := time.Now().UTC()
			execIndex.CompletedAt = &now
			execIndex.UpdatedAt = now
			_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, &execIndex.ErrorMessage, execIndex.CompletedAt, execIndex.UpdatedAt)
			return
		}
		if plan.Metadata == nil {
			plan.Metadata = make(map[string]any)
		}
		plan.Metadata["electron_target"] = opts.ElectronTarget
		plan.Metadata["validation_context"] = opts.ValidationContext
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
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     s.engineFactory,
		Recorder:          s.artifactRecorder,
		EventSink:         eventSink,
		HeartbeatInterval: 2 * time.Second,
		// Leave this unset so the executor resolves the workflow's declared
		// sessionReuseMode metadata. The historical explicit "reuse" value
		// silently overrode an adhoc caller's isolation policy.
		ReuseMode:           "",
		WorkflowResolver:    s,
		PlanCompiler:        s.planCompiler,
		MaxSubflowDepth:     5,
		StartFromStepIndex:  -1,
		ProjectRoot:         projectRoot,
		InitialStore:        store,
		InitialParams:       params,
		Env:                 env,
		StartURL:            strings.TrimSpace(startURL),
		ArtifactConfig:      artifactCfg,
		BrowserProfile:      browserProfile,
		StorageState:        storageState,
		RestoreTabs:         restoreTabs,
		OpenTabs:            openTabs,
		NavigationWaitUntil: navigationWaitUntil,
		ContinueOnError:     continueOnError,
	}

	// Set up callback to save storage state after successful execution
	if saveSessionProfileID != "" && s.sessionProfileService != nil {
		if s.log != nil {
			s.log.WithFields(logrus.Fields{
				"execution_id":            executionID.String(),
				"save_session_profile_id": saveSessionProfileID,
			}).Info("Setting up storage state save callback")
		}
		profileID := saveSessionProfileID // capture for closure
		req.SaveStorageStateCallback = func(storageState json.RawMessage) error {
			if s.log != nil {
				s.log.WithFields(logrus.Fields{
					"execution_id":            executionID.String(),
					"save_session_profile_id": profileID,
					"storage_state_size":      len(storageState),
				}).Info("Saving storage state to session profile")
			}
			_, err := s.sessionProfileService.SaveStorageState(sessionprofilepersistence.ProfileID(profileID), storageState)
			if err != nil && s.log != nil {
				s.log.WithError(err).WithFields(logrus.Fields{
					"execution_id":            executionID.String(),
					"save_session_profile_id": profileID,
				}).Error("Failed to save storage state to session profile")
			}
			return err
		}
	} else if saveSessionProfileID != "" && s.sessionProfileService == nil {
		if s.log != nil {
			s.log.WithFields(logrus.Fields{
				"execution_id":            executionID.String(),
				"save_session_profile_id": saveSessionProfileID,
			}).Warn("Cannot save session state: session profiles service not available")
		}
	}

	executor := s.executor
	if executor == nil {
		executor = autoexecutor.NewSimpleExecutor(nil)
	}
	runErr := executor.Execute(ctx, req)
	if runErr == nil && opts != nil && opts.RequiresVideo {
		videos, artifactErr := s.GetExecutionVideoArtifacts(persistenceCtx, executionID)
		runErr = requiredVideoArtifactError(true, videos, artifactErr)
	}

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
