package agentmanager

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

// AgentService provides agent execution services for test-genie.
// It wraps the agent-manager client and handles profile management,
// batch spawning, and status tracking for test generation.
type AgentService struct {
	client     *Client
	profileKey string
	profileID  string
	mu         sync.RWMutex
	enabled    bool
}

// Config contains configuration for the agent service.
type Config struct {
	ProfileKey string
	Timeout    time.Duration
	Enabled    bool
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg Config) *AgentService {
	client := NewClient(cfg.Timeout)
	return &AgentService{
		client:     client,
		profileKey: cfg.ProfileKey,
		enabled:    cfg.Enabled,
	}
}

// IsEnabled returns whether agent-manager integration is enabled.
func (s *AgentService) IsEnabled() bool {
	return s.enabled
}

// IsAvailable checks if agent-manager is reachable.
func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if !s.enabled {
		return false
	}
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// ResolveURL returns the current agent-manager base URL.
func (s *AgentService) ResolveURL(ctx context.Context) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("agent-manager not enabled")
	}
	return s.client.ResolveURL(ctx)
}

// Initialize reconciles Test Genie's manifest-declared profile source.
func (s *AgentService) Initialize(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	resp, err := s.client.ReconcileScenarioProfiles(ctx, "test-genie")
	if err != nil {
		return fmt.Errorf("reconcile profile: %w", err)
	}

	s.mu.Lock()
	for _, item := range resp.Results {
		if item.ProfileKey == s.profileKey {
			s.profileID = item.ProfileId
		}
	}
	s.mu.Unlock()

	if s.profileID == "" {
		return fmt.Errorf("reconciliation returned no profile %q", s.profileKey)
	}
	return nil
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
	}
}

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
}

// GetRoleCatalog returns portable role choices for Test Genie's agent-improvement UI.
func (s *AgentService) GetRoleCatalog(ctx context.Context) (*apipb.GetRolePolicyCatalogResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRolePolicyCatalog(ctx)
}

// =============================================================================
// BATCH SPAWNING
// =============================================================================

// PromptConfig contains configuration for a single prompt in a batch.
type PromptConfig struct {
	Text       string   // The prompt text (with preamble already injected)
	Phases     []string // Test phases this prompt targets
	TargetPath string   // Optional specific path this prompt targets
}

// BatchSpawnRequest contains parameters for batch agent spawning.
type BatchSpawnRequest struct {
	Scenario    string
	Scope       []string
	Prompts     []PromptConfig
	RoleRef     string
	Concurrency int
	MaxTurns    int
	Timeout     time.Duration
}

// RunInfo contains information about a spawned run.
type RunInfo struct {
	RunID  string
	Tag    string
	TaskID string
}

// BatchSpawnResult contains the result of batch spawning.
type BatchSpawnResult struct {
	BatchID string
	Runs    []SpawnResult
	Errors  []string
}

// SpawnResult contains the result of a single agent spawn.
type SpawnResult struct {
	PromptIndex int
	RunID       string
	Tag         string
	Status      string
	Error       string
}

// SpawnBatch creates multiple Tasks and Runs for batch test generation.
// Each prompt becomes a separate Task with its own Run.
func (s *AgentService) SpawnBatch(ctx context.Context, req BatchSpawnRequest) (*BatchSpawnResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	batchID := NewUUID()
	results := make([]SpawnResult, len(req.Prompts))
	var errors []string

	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	scenarioPath, err := repocontract.ResolveScenarioPath(repoRoot, req.Scenario)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario path: %w", err)
	}
	// scope_path must be a repo-relative scope (e.g. "scenarios/foo"), not an
	// absolute path. agent-manager forwards it as VROOLI_SANDBOX_SCOPE; the
	// CLI's sandbox-aware path resolution (repocontract.ScenarioScopeMatch)
	// expects a relative form starting with the layout's scenario dir.
	scenarioScope := path.Join("scenarios", req.Scenario)

	// Semaphore for concurrency control
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, prompt := range req.Prompts {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, pc PromptConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			result := SpawnResult{
				PromptIndex: idx,
				Status:      "pending",
			}

			// Create task for this prompt
			tag := fmt.Sprintf("test-genie-%s-%d", batchID, idx)
			task := &domainpb.Task{
				Title:       fmt.Sprintf("Test Generation - %s [%d]", req.Scenario, idx),
				Description: pc.Text,
				ScopePath:   scenarioScope,
				ProjectRoot: scenarioPath,
				CreatedBy:   "test-genie",
				ContextAttachments: []*domainpb.ContextAttachment{
					{
						Type:    "note",
						Key:     "test-generation-context",
						Label:   "Test Generation Context",
						Content: fmt.Sprintf("Scenario: %s\nPhases: %v\nTarget: %s", req.Scenario, pc.Phases, pc.TargetPath),
					},
				},
			}

			createdTask, err := s.client.CreateTask(ctx, task)
			if err != nil {
				result.Status = "failed"
				result.Error = fmt.Sprintf("create task: %v", err)
				mu.Lock()
				results[idx] = result
				errors = append(errors, result.Error)
				mu.Unlock()
				return
			}

			// Create run for this task. RunMode is left unset so the
			// orchestrator resolves it via DeriveRunMode from the
			// profile's SandboxConfig.Mode (Protected by default) —
			// required to get VROOLI_SANDBOX_* env vars injected into
			// the agent process and inherited by the test-genie CLI
			// subprocess.
			runReq := &apipb.CreateRunRequest{
				TaskId:     createdTask.Id,
				ProfileRef: s.defaultProfileRef(),
				Tag:        &tag,
				Force:      true, // Bypass capacity limits
			}

			// Apply portable role override if specified. Concrete model selection
			// remains resource-owned evidence in Agent Manager's run snapshot.
			if req.RoleRef != "" {
				runReq.InlineConfig = &domainpb.RunConfigOverrides{
					RoleRef: &req.RoleRef,
				}
			}

			// Apply max turns override if specified
			if req.MaxTurns > 0 {
				if runReq.InlineConfig == nil {
					runReq.InlineConfig = &domainpb.RunConfigOverrides{}
				}
				maxTurns := int32(req.MaxTurns)
				runReq.InlineConfig.MaxTurns = &maxTurns
			}

			// Apply timeout override if specified
			if req.Timeout > 0 {
				if runReq.InlineConfig == nil {
					runReq.InlineConfig = &domainpb.RunConfigOverrides{}
				}
				runReq.InlineConfig.Timeout = durationpb.New(req.Timeout)
			}

			run, err := s.client.CreateRun(ctx, runReq)
			if err != nil {
				result.Status = "failed"
				result.Error = fmt.Sprintf("create run: %v", err)
				mu.Lock()
				results[idx] = result
				errors = append(errors, result.Error)
				mu.Unlock()
				return
			}

			result.RunID = run.Id
			result.Tag = tag
			result.Status = MapRunStatus(run.Status)

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, prompt)
	}

	wg.Wait()

	return &BatchSpawnResult{
		BatchID: batchID,
		Runs:    results,
		Errors:  errors,
	}, nil
}

// =============================================================================
// STATUS AND MANAGEMENT
// =============================================================================

// RunStatus contains the current status of a run.
type RunStatus struct {
	RunID           string
	Tag             string
	Status          string
	Output          string
	Error           string
	DurationSeconds int
	TokensUsed      int32
	CostEstimate    float64
}

// GetBatchStatus returns the status of all runs in a batch.
func (s *AgentService) GetBatchStatus(ctx context.Context, batchID string) ([]RunStatus, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	tagPrefix := fmt.Sprintf("test-genie-%s-", batchID)
	runs, err := s.client.ListRuns(ctx, ListRunsOptions{
		TagPrefix: tagPrefix,
	})
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	statuses := make([]RunStatus, 0, len(runs))
	for _, run := range runs {
		status := RunStatus{
			RunID:  run.Id,
			Tag:    run.GetTag(),
			Status: MapRunStatus(run.Status),
		}

		if run.ErrorMsg != "" {
			status.Error = run.ErrorMsg
		}

		if run.Summary != nil {
			status.Output = run.Summary.Description
			status.TokensUsed = run.Summary.TokensUsed
			status.CostEstimate = run.Summary.CostEstimate
		}

		if run.StartedAt != nil && run.EndedAt != nil {
			duration := run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
			status.DurationSeconds = int(duration.Seconds())
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// ListActiveRuns returns all active (non-terminal) test-genie runs.
func (s *AgentService) ListActiveRuns(ctx context.Context) ([]*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	// Get all runs filtered by tag prefix
	runs, err := s.client.ListRuns(ctx, ListRunsOptions{
		TagPrefix: "test-genie-",
	})
	if err != nil {
		return nil, err
	}

	// Filter to only active (non-terminal) runs
	var activeRuns []*domainpb.Run
	for _, r := range runs {
		if r.Status == domainpb.RunStatus_RUN_STATUS_PENDING ||
			r.Status == domainpb.RunStatus_RUN_STATUS_STARTING ||
			r.Status == domainpb.RunStatus_RUN_STATUS_RUNNING ||
			r.Status == domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW {
			activeRuns = append(activeRuns, r)
		}
	}
	return activeRuns, nil
}

// ListAllRuns returns all test-genie runs (including completed).
func (s *AgentService) ListAllRuns(ctx context.Context) ([]*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	return s.client.ListRuns(ctx, ListRunsOptions{
		TagPrefix: "test-genie-",
	})
}

// GetRun returns a specific run by ID.
func (s *AgentService) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	return s.client.GetRun(ctx, runID)
}

// GetRunByTag returns a run by its tag.
func (s *AgentService) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	return s.client.GetRunByTag(ctx, tag)
}

// StopRun stops a running agent by run ID.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return fmt.Errorf("agent-manager not enabled")
	}

	return s.client.StopRun(ctx, runID)
}

// StopBatch stops all runs in a batch.
func (s *AgentService) StopBatch(ctx context.Context, batchID string) (*StopAllRunsResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	tagPrefix := fmt.Sprintf("test-genie-%s-", batchID)
	return s.client.StopAllRuns(ctx, tagPrefix)
}

// StopAllRuns stops all test-genie runs.
func (s *AgentService) StopAllRuns(ctx context.Context) (*StopAllRunsResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	return s.client.StopAllRuns(ctx, "test-genie-")
}

// GetRunEvents returns events for a run.
func (s *AgentService) GetRunEvents(ctx context.Context, runID string) ([]*domainpb.RunEvent, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	return s.client.GetRunEvents(ctx, runID, 0)
}

// =============================================================================
// SINGLE TASK SPAWNING (for fix service)
// =============================================================================

// SpawnSingleRequest contains parameters for spawning a single agent task.
type SpawnSingleRequest struct {
	Task *domainpb.Task
	Tag  string
}

// SpawnSingleResult contains the result of spawning a single agent.
type SpawnSingleResult struct {
	TaskID string
	RunID  string
	Tag    string
	Status string
	Error  string
}

// SpawnSingle creates a single Task and Run for agent execution.
// This is simpler than SpawnBatch when you only need one agent.
//
// req.Task.ScopePath must be a repo-relative scope (e.g. "scenarios/foo")
// because the test-genie profile runs sandboxed and agent-manager forwards
// ScopePath as VROOLI_SANDBOX_SCOPE. cliutil.IsSandboxActive() requires both
// VROOLI_SANDBOX_MERGED and VROOLI_SANDBOX_SCOPE to be non-empty — without a
// scope, the CLI's sandbox-aware path resolution silently falls back to the
// real repo and the agent's edits in the overlay become invisible to it.
func (s *AgentService) SpawnSingle(ctx context.Context, req SpawnSingleRequest) (*SpawnSingleResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	if req.Task == nil {
		return nil, fmt.Errorf("task is required")
	}
	if strings.TrimSpace(req.Task.ScopePath) == "" {
		return nil, fmt.Errorf("task.scope_path is required: must be a repo-relative scope like \"scenarios/<name>\" so VROOLI_SANDBOX_SCOPE is set and the test-genie CLI's sandbox-aware path resolution stays active")
	}

	result := &SpawnSingleResult{
		Tag:    req.Tag,
		Status: "pending",
	}

	// Create task
	createdTask, err := s.client.CreateTask(ctx, req.Task)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("create task: %v", err)
		return result, err
	}
	result.TaskID = createdTask.Id

	// Create run. RunMode unset → orchestrator derives sandboxed via
	// the profile's SandboxConfig.Mode (Protected). See SpawnBatch for
	// the full rationale.
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.defaultProfileRef(),
		Tag:        &req.Tag,
		Force:      true,
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("create run: %v", err)
		return result, err
	}

	result.RunID = run.Id
	result.Status = MapRunStatus(run.Status)

	return result, nil
}

// =============================================================================
// STATUS MAPPING
// =============================================================================

// MapRunStatus converts agent-manager RunStatus to test-genie status strings.
func MapRunStatus(status domainpb.RunStatus) string {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING, domainpb.RunStatus_RUN_STATUS_STARTING:
		return "pending"
	case domainpb.RunStatus_RUN_STATUS_RUNNING, domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return "running"
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return "completed"
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return "stopped"
	default:
		return "unknown"
	}
}

// MapStatusToRun converts test-genie status strings to agent-manager RunStatus.
func MapStatusToRun(status string) domainpb.RunStatus {
	switch status {
	case "pending":
		return domainpb.RunStatus_RUN_STATUS_PENDING
	case "running":
		return domainpb.RunStatus_RUN_STATUS_RUNNING
	case "completed":
		return domainpb.RunStatus_RUN_STATUS_COMPLETE
	case "failed":
		return domainpb.RunStatus_RUN_STATUS_FAILED
	case "stopped":
		return domainpb.RunStatus_RUN_STATUS_CANCELLED
	default:
		return domainpb.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}
