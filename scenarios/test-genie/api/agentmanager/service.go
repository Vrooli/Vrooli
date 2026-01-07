package agentmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

// AgentService provides agent execution services for test-genie.
// It wraps the agent-manager client and handles profile management,
// batch spawning, and status tracking for test generation.
type AgentService struct {
	client      *Client
	profileName string
	profileKey  string
	profileID   string
	mu          sync.RWMutex
	enabled     bool
}

// Config contains configuration for the agent service.
type Config struct {
	ProfileName string
	ProfileKey  string
	Timeout     time.Duration
	Enabled     bool
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg Config) *AgentService {
	client := NewClient(cfg.Timeout)
	return &AgentService{
		client:      client,
		profileName: cfg.ProfileName,
		profileKey:  cfg.ProfileKey,
		enabled:     cfg.Enabled,
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

// Initialize ensures the agent profile exists.
// Call this at startup to create/update the test-genie profile.
func (s *AgentService) Initialize(ctx context.Context, cfg *ProfileConfig) error {
	if !s.enabled {
		return nil
	}

	resp, err := s.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     s.profileKey,
		Defaults:       s.buildProfile(cfg),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure profile: %w", err)
	}

	s.mu.Lock()
	if resp.Profile != nil {
		s.profileID = resp.Profile.Id
	}
	s.mu.Unlock()

	if resp.Created {
		log.Printf("[agent-manager] Created profile '%s' (id=%s)", s.profileName, s.profileID)
	} else {
		log.Printf("[agent-manager] Resolved profile '%s' (id=%s)", s.profileName, s.profileID)
	}

	return nil
}

// ProfileConfig contains agent profile configuration.
type ProfileConfig struct {
	RunnerType       domainpb.RunnerType
	Model            string
	ModelPreset      domainpb.ModelPreset
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
}

// DefaultProfileConfig returns the default configuration for test generation.
func DefaultProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:  domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		ModelPreset: domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:    50,
		// 15 minute timeout for thorough test generation
		TimeoutSeconds: 900,
		AllowedTools: []string{
			"Read",   // Read files
			"Write",  // Create/overwrite files
			"Edit",   // Modify files
			"Glob",   // Find files by pattern
			"Grep",   // Search file contents
			"Bash",   // Execute allowed commands
		},
		SkipPermissions:  false, // Require confirmation for safety
		RequiresSandbox:  false, // In-place execution for test-genie
		RequiresApproval: false, // Auto-apply test files
	}
}

func (s *AgentService) buildProfile(cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 s.profileName,
		ProfileKey:           s.profileKey,
		Description:          "Agent profile for test-genie test generation",
		RunnerType:           cfg.RunnerType,
		Model:                cfg.Model,
		ModelPreset:          cfg.ModelPreset,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "test-genie",
	}
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
		Defaults:   s.buildProfile(DefaultProfileConfig()),
	}
}

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
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
	Model       string
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

	// Determine working directory
	repoRoot := os.Getenv("VROOLI_ROOT")
	if repoRoot == "" {
		repoRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	scenarioPath := filepath.Join(repoRoot, "scenarios", req.Scenario)

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
				ScopePath:   scenarioPath,
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

			// Create run for this task
			runReq := &apipb.CreateRunRequest{
				TaskId:     createdTask.Id,
				ProfileRef: s.defaultProfileRef(),
				Tag:        &tag,
				RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
				Force:      true, // Bypass capacity limits
			}

			// Apply model override if specified
			if req.Model != "" {
				runReq.InlineConfig = &domainpb.RunConfigOverrides{
					Model: &req.Model,
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
