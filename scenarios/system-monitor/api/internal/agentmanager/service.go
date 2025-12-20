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

// AgentService provides agent execution services for system-monitor.
// It wraps the agent-manager client and handles profile management,
// run execution, and status tracking.
type AgentService struct {
	client      *Client
	profileName string
	profileID   string
	mu          sync.RWMutex
	enabled     bool
}

// AgentServiceConfig contains configuration for the agent service.
type AgentServiceConfig struct {
	ProfileName string
	Timeout     time.Duration
	Enabled     bool
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg AgentServiceConfig) *AgentService {
	client := NewClient(cfg.Timeout)
	return &AgentService{
		client:      client,
		profileName: cfg.ProfileName,
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
// Call this at startup to create/update the system-monitor profile.
func (s *AgentService) Initialize(ctx context.Context, cfg *ProfileConfig) error {
	if !s.enabled {
		return nil
	}

	profile := s.buildProfile(cfg)
	created, isNew, err := s.client.UpsertProfile(ctx, profile)
	if err != nil {
		return fmt.Errorf("upsert profile: %w", err)
	}

	s.mu.Lock()
	s.profileID = created.Id
	s.mu.Unlock()

	if isNew {
		log.Printf("[agent-manager] Created profile '%s' (id=%s)", s.profileName, created.Id)
	} else {
		log.Printf("[agent-manager] Updated profile '%s' (id=%s)", s.profileName, created.Id)
	}

	return nil
}

// ProfileConfig contains agent profile configuration.
type ProfileConfig struct {
	RunnerType       domainpb.RunnerType
	Model            string
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
}

// DefaultProfileConfig returns the default configuration for system-monitor.
func DefaultProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:       domainpb.RunnerType_RUNNER_TYPE_CODEX,
		Model:            "codex-mini-latest",
		MaxTurns:         75,
		TimeoutSeconds:   600,
		AllowedTools:     []string{"read_file", "write_file", "append_file", "list_files", "analyze_code", "execute_command"},
		SkipPermissions:  true,
		RequiresSandbox:  false, // In-place execution for system control
		RequiresApproval: false, // Auto-apply changes
	}
}

func (s *AgentService) buildProfile(cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 s.profileName,
		Description:          "Agent profile for system-monitor investigations",
		RunnerType:           cfg.RunnerType,
		Model:                cfg.Model,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "system-monitor",
	}
}

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
}

// GetProfile returns the current agent profile.
func (s *AgentService) GetProfile(ctx context.Context) (*domainpb.AgentProfile, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	profileID := s.GetProfileID()
	if profileID == "" {
		return nil, fmt.Errorf("profile not initialized")
	}

	return s.client.GetProfile(ctx, profileID)
}

// UpdateProfile updates the agent profile configuration.
func (s *AgentService) UpdateProfile(ctx context.Context, cfg *ProfileConfig) (*domainpb.AgentProfile, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	profile := s.buildProfile(cfg)
	profile.Id = s.GetProfileID()
	if profile.Id == "" {
		return nil, fmt.Errorf("profile not initialized")
	}

	updated, err := s.client.UpdateProfile(ctx, profile.Id, profile)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return updated, nil
}

// =============================================================================
// RUNNERS
// =============================================================================

// RunnerInfo contains information about an available runner.
type RunnerInfo struct {
	Type            domainpb.RunnerType
	Name            string
	Available       bool
	Message         string
	InstallHint     string
	SupportedModels []string
	Capabilities    *domainpb.RunnerCapabilities
}

// GetAvailableRunners returns all available runners with their info.
func (s *AgentService) GetAvailableRunners(ctx context.Context) ([]RunnerInfo, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	runners, err := s.client.GetRunnerStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get runners: %w", err)
	}

	result := make([]RunnerInfo, 0, len(runners))
	for _, r := range runners {
		result = append(result, RunnerInfo{
			Type:            r.RunnerType,
			Name:            runnerTypeName(r.RunnerType),
			Available:       r.Available,
			Message:         r.Message,
			InstallHint:     r.InstallHint,
			SupportedModels: r.SupportedModels,
			Capabilities:    r.Capabilities,
		})
	}
	return result, nil
}

func runnerTypeName(rt domainpb.RunnerType) string {
	switch rt {
	case domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE:
		return "claude-code"
	case domainpb.RunnerType_RUNNER_TYPE_CODEX:
		return "codex"
	case domainpb.RunnerType_RUNNER_TYPE_OPENCODE:
		return "opencode"
	default:
		return "unknown"
	}
}

// =============================================================================
// EXECUTION
// =============================================================================

// ExecuteRequest contains parameters for agent execution.
type ExecuteRequest struct {
	// Unique ID for this execution (used for tagging)
	InvestigationID string
	// Prompt to send to the agent
	Prompt string
	// Working directory for execution
	WorkingDir string
	// Optional override for runner type (uses profile default if empty)
	RunnerType *domainpb.RunnerType
	// Optional override for model (uses profile default if empty)
	Model string
	// Optional inline config overrides
	InlineConfig *domainpb.RunConfig
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

	profileID := s.GetProfileID()
	if profileID == "" {
		return nil, fmt.Errorf("profile not initialized")
	}

	// Create task for this investigation
	task := &domainpb.Task{
		Title:       fmt.Sprintf("System Investigation %s", req.InvestigationID),
		Description: req.Prompt,
		ScopePath:   req.WorkingDir,
		ProjectRoot: req.WorkingDir,
		CreatedBy:   "system-monitor",
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Create run with tag for tracking
	tag := fmt.Sprintf("system-monitor-%s", req.InvestigationID)
	runReq := &apipb.CreateRunRequest{
		TaskId:         createdTask.Id,
		AgentProfileId: &profileID,
		Tag:            &tag,
		RunMode:        domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:          true, // Bypass capacity limits for investigations
	}

	// Apply inline config overrides if provided
	if req.InlineConfig != nil {
		runReq.InlineConfig = req.InlineConfig
	} else if req.RunnerType != nil || req.Model != "" {
		// Build inline config from individual overrides
		runReq.InlineConfig = &domainpb.RunConfig{}
		if req.RunnerType != nil {
			runReq.InlineConfig.RunnerType = *req.RunnerType
		}
		if req.Model != "" {
			runReq.InlineConfig.Model = req.Model
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

	profileID := s.GetProfileID()
	if profileID == "" {
		return "", fmt.Errorf("profile not initialized")
	}

	// Create task
	task := &domainpb.Task{
		Title:       fmt.Sprintf("System Investigation %s", req.InvestigationID),
		Description: req.Prompt,
		ScopePath:   req.WorkingDir,
		ProjectRoot: req.WorkingDir,
		CreatedBy:   "system-monitor",
	}

	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	// Create run
	tag := fmt.Sprintf("system-monitor-%s", req.InvestigationID)
	runReq := &apipb.CreateRunRequest{
		TaskId:         createdTask.Id,
		AgentProfileId: &profileID,
		Tag:            &tag,
		RunMode:        domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:          true,
	}

	if req.InlineConfig != nil {
		runReq.InlineConfig = req.InlineConfig
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

// GetRunByTag returns a run by its tag.
func (s *AgentService) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRunByTag(ctx, tag)
}

// ListActiveRuns returns all active system-monitor runs.
func (s *AgentService) ListActiveRuns(ctx context.Context) ([]*domainpb.Run, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}

	status := domainpb.RunStatus_RUN_STATUS_RUNNING
	return s.client.ListRuns(ctx, &status, "system-monitor-")
}

// StopRun stops an active run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	if !s.enabled {
		return fmt.Errorf("agent-manager not enabled")
	}
	return s.client.StopRun(ctx, runID)
}

// StopAllRuns stops all system-monitor runs.
func (s *AgentService) StopAllRuns(ctx context.Context) (int, error) {
	if !s.enabled {
		return 0, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.StopAllRuns(ctx, "system-monitor-")
}

// GetRunEvents returns events for a run.
func (s *AgentService) GetRunEvents(ctx context.Context, runID string) ([]*domainpb.RunEvent, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.GetRunEvents(ctx, runID, 0)
}
