package agentmanager

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

// AgentService provides agent execution services for scenario-to-desktop.
// It wraps the agent-manager client and handles profile management,
// run execution, and status tracking for pipeline investigations.
type AgentService struct {
	client      *Client
	profileName string
	profileKey  string
	profileID   string
	mu          sync.RWMutex
	enabled     bool
}

// AgentServiceConfig contains configuration for the agent service.
type AgentServiceConfig struct {
	ProfileName string
	ProfileKey  string
	Timeout     time.Duration
	Enabled     bool
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg AgentServiceConfig) *AgentService {
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
// Call this at startup to create/update the scenario-to-desktop profile.
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

// DefaultProfileConfig returns the default configuration for pipeline investigations.
func DefaultProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:  domainpb.RunnerType_RUNNER_TYPE_CODEX,
		ModelPreset: domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:    75,
		// 10 minute timeout for thorough build investigation
		TimeoutSeconds: 600,
		AllowedTools: []string{
			"read_file",       // Read build logs, config files
			"list_files",      // Browse generated desktop wrapper
			"execute_command", // Run build commands, npm, etc.
			"analyze_code",    // Understand build scripts
			"write_file",      // Write investigation report
		},
		SkipPermissions:  true,  // Auto-approve for automated investigations
		RequiresSandbox:  false, // In-place execution for local access
		RequiresApproval: false, // Auto-apply (report-only by default)
	}
}

func (s *AgentService) buildProfile(cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 s.profileName,
		ProfileKey:           s.profileKey,
		Description:          "Agent profile for scenario-to-desktop pipeline investigations",
		RunnerType:           cfg.RunnerType,
		Model:                cfg.Model,
		ModelPreset:          cfg.ModelPreset,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "scenario-to-desktop",
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
// EXECUTION
// =============================================================================

// ExecuteRequest contains parameters for agent execution.
type ExecuteRequest struct {
	// Unique ID for this execution (used for tagging)
	InvestigationID string
	// Optional additional tag for investigation classification.
	AdditionalTag string
	// Prompt to send to the agent
	Prompt string
	// Working directory for execution
	WorkingDir string
	// Optional override for runner type (uses profile default if empty)
	RunnerType *domainpb.RunnerType
	// Optional override for model (uses profile default if empty)
	Model string
	// Context attachments for structured context (optional)
	ContextAttachments []*domainpb.ContextAttachment
}

// ExecuteResult contains the result of agent execution.
type ExecuteResult struct {
	RunID           string
	Success         bool
	Output          string
	ErrorMessage    string
	DurationSeconds int
	TokensUsed      int32
	CostEstimate    float64
	RateLimited     bool
	Timeout         bool
}

// Execute starts an agent run and waits for completion.
func (s *AgentService) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	// Create task for this investigation
	task := &domainpb.Task{
		Title:              fmt.Sprintf("Pipeline Investigation %s", req.InvestigationID),
		Description:        req.Prompt,
		ScopePath:          req.WorkingDir,
		ProjectRoot:        req.WorkingDir,
		CreatedBy:          "scenario-to-desktop",
		ContextAttachments: req.ContextAttachments,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Create run with tag for tracking
	tag := buildRunTag(req.InvestigationID, req.AdditionalTag)
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.defaultProfileRef(),
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true, // Bypass capacity limits for investigations
	}

	// Apply inline config overrides if provided
	if req.RunnerType != nil || req.Model != "" {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
		if req.RunnerType != nil {
			runReq.InlineConfig.RunnerType = req.RunnerType
		}
		if req.Model != "" {
			runReq.InlineConfig.Model = &req.Model
		}
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	// Wait for run to complete
	completedRun, err := s.client.WaitForRun(ctx, run.Id, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("wait for run: %w", err)
	}

	// Build result
	result := &ExecuteResult{
		RunID:   completedRun.Id,
		Success: completedRun.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE,
	}

	if completedRun.Summary != nil {
		result.Output = completedRun.Summary.Description
		result.TokensUsed = completedRun.Summary.TokensUsed
		result.CostEstimate = completedRun.Summary.CostEstimate
	}

	if completedRun.ErrorMsg != "" {
		result.ErrorMessage = completedRun.ErrorMsg
		// Check for rate limit in error message
		if strings.Contains(strings.ToLower(completedRun.ErrorMsg), "rate limit") {
			result.RateLimited = true
		}
		if strings.Contains(strings.ToLower(completedRun.ErrorMsg), "timeout") {
			result.Timeout = true
		}
	}

	// Calculate duration
	if completedRun.StartedAt != nil && completedRun.EndedAt != nil {
		duration := completedRun.EndedAt.AsTime().Sub(completedRun.StartedAt.AsTime())
		result.DurationSeconds = int(duration.Seconds())
	}

	return result, nil
}

// ExecuteAsync starts an agent run without waiting for completion.
// Returns the run ID for tracking.
func (s *AgentService) ExecuteAsync(ctx context.Context, req ExecuteRequest) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("agent-manager not enabled")
	}

	// Create task
	task := &domainpb.Task{
		Title:              fmt.Sprintf("Pipeline Investigation %s", req.InvestigationID),
		Description:        req.Prompt,
		ScopePath:          req.WorkingDir,
		ProjectRoot:        req.WorkingDir,
		CreatedBy:          "scenario-to-desktop",
		ContextAttachments: req.ContextAttachments,
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	// Create run
	tag := buildRunTag(req.InvestigationID, req.AdditionalTag)
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.defaultProfileRef(),
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true,
	}

	if req.RunnerType != nil || req.Model != "" {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
		if req.RunnerType != nil {
			runReq.InlineConfig.RunnerType = req.RunnerType
		}
		if req.Model != "" {
			runReq.InlineConfig.Model = &req.Model
		}
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}

	return run.Id, nil
}

// GetRunStatus returns the current status of a run.
func (s *AgentService) GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRun(ctx, runID)
}

// StopRun stops an active run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return fmt.Errorf("agent-manager not enabled")
	}
	return s.client.StopRun(ctx, runID)
}

// GetRunEvents returns events for a run.
func (s *AgentService) GetRunEvents(ctx context.Context, runID string) ([]*domainpb.RunEvent, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRunEvents(ctx, runID, 0)
}

func buildRunTag(investigationID, additionalTag string) string {
	baseTag := fmt.Sprintf("scenario-to-desktop-%s", investigationID)
	if strings.TrimSpace(additionalTag) == "" {
		return baseTag
	}
	return fmt.Sprintf("%s|%s", baseTag, additionalTag)
}
