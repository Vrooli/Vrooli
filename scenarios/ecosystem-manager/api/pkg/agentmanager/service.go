package agentmanager

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// AgentService provides agent execution services for ecosystem-manager.
// It wraps the agent-manager client and handles profile management,
// run execution, and status tracking for ecosystem tasks.
type AgentService struct {
	client             *Client
	taskProfileKey     string
	insightsProfileKey string
	taskProfileID      string
	insightsProfileID  string
	vrooliRoot         string
	mu                 sync.RWMutex
}

// Config contains configuration for the agent service.
type Config struct {
	TaskProfileKey     string
	InsightsProfileKey string
	Timeout            time.Duration
	VrooliRoot         string
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg Config) *AgentService {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client := NewClient(timeout)
	return &AgentService{
		client:             client,
		taskProfileKey:     cfg.TaskProfileKey,
		insightsProfileKey: cfg.InsightsProfileKey,
		vrooliRoot:         cfg.VrooliRoot,
	}
}

// IsAvailable checks if agent-manager is reachable.
func (s *AgentService) IsAvailable(ctx context.Context) bool {
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// ResolveURL returns the current agent-manager base URL.
func (s *AgentService) ResolveURL(ctx context.Context) (string, error) {
	return s.client.ResolveURL(ctx)
}

// Initialize ensures both task and insights profiles exist.
// Call this at startup to create/update profiles.
func (s *AgentService) Initialize(ctx context.Context) error {
	// Initialize task profile
	taskCfg := s.buildTaskProfileConfig()
	taskResp, err := s.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     s.taskProfileKey,
		Defaults:       s.buildProfile(s.taskProfileKey, "ecosystem-manager-tasks", "Agent profile for ecosystem-manager task execution", taskCfg),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure task profile: %w", err)
	}

	s.mu.Lock()
	if taskResp.Profile != nil {
		s.taskProfileID = taskResp.Profile.Id
	}
	s.mu.Unlock()

	if taskResp.Created {
		log.Printf("[agent-manager] Created task profile '%s' (id=%s)", s.taskProfileKey, s.taskProfileID)
	} else {
		log.Printf("[agent-manager] Resolved task profile '%s' (id=%s)", s.taskProfileKey, s.taskProfileID)
	}

	// Initialize insights profile
	insightsCfg := s.buildInsightsProfileConfig()
	insightsResp, err := s.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     s.insightsProfileKey,
		Defaults:       s.buildProfile(s.insightsProfileKey, "ecosystem-manager-insights", "Agent profile for ecosystem-manager insight generation", insightsCfg),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure insights profile: %w", err)
	}

	s.mu.Lock()
	if insightsResp.Profile != nil {
		s.insightsProfileID = insightsResp.Profile.Id
	}
	s.mu.Unlock()

	if insightsResp.Created {
		log.Printf("[agent-manager] Created insights profile '%s' (id=%s)", s.insightsProfileKey, s.insightsProfileID)
	} else {
		log.Printf("[agent-manager] Resolved insights profile '%s' (id=%s)", s.insightsProfileKey, s.insightsProfileID)
	}

	return nil
}

// UpdateProfiles updates both profiles with current settings.
// Call this when settings change to propagate new config.
func (s *AgentService) UpdateProfiles(ctx context.Context) error {
	s.mu.RLock()
	taskID := s.taskProfileID
	insightsID := s.insightsProfileID
	s.mu.RUnlock()

	if taskID != "" {
		taskCfg := s.buildTaskProfileConfig()
		profile := s.buildProfile(s.taskProfileKey, "ecosystem-manager-tasks", "Agent profile for ecosystem-manager task execution", taskCfg)
		profile.Id = taskID
		if _, err := s.client.UpdateProfile(ctx, taskID, profile); err != nil {
			return fmt.Errorf("update task profile: %w", err)
		}
		log.Printf("[agent-manager] Updated task profile '%s'", s.taskProfileKey)
	}

	if insightsID != "" {
		insightsCfg := s.buildInsightsProfileConfig()
		profile := s.buildProfile(s.insightsProfileKey, "ecosystem-manager-insights", "Agent profile for ecosystem-manager insight generation", insightsCfg)
		profile.Id = insightsID
		if _, err := s.client.UpdateProfile(ctx, insightsID, profile); err != nil {
			return fmt.Errorf("update insights profile: %w", err)
		}
		log.Printf("[agent-manager] Updated insights profile '%s'", s.insightsProfileKey)
	}

	return nil
}

// =============================================================================
// EXECUTION
// =============================================================================

// ExecuteRequest contains parameters for task execution.
type ExecuteRequest struct {
	// Task being executed
	Task tasks.TaskItem
	// Prompt to send to the agent
	Prompt string
	// Timeout for this execution
	Timeout time.Duration
	// Optional tag override (defaults to ecosystem-{taskID})
	Tag string
}

// ExecuteResult contains the result of agent execution.
type ExecuteResult struct {
	RunID            string
	Success          bool
	Output           string
	ErrorMessage     string
	DurationSeconds  int
	TokensUsed       int32
	CostEstimate     float64
	RateLimited      bool
	Timeout          bool
	MaxTurnsExceeded bool
}

// ExecuteTask starts a task run and waits for completion.
func (s *AgentService) ExecuteTask(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	tag := req.Tag
	if tag == "" {
		tag = fmt.Sprintf("ecosystem-%s", req.Task.ID)
	}

	// Create task in agent-manager
	amTask := &domainpb.Task{
		Title:       req.Task.Title,
		Description: fmt.Sprintf("Ecosystem task: %s (%s/%s)", req.Task.Title, req.Task.Type, req.Task.Operation),
		ScopePath:   s.vrooliRoot,
		ProjectRoot: s.vrooliRoot,
		CreatedBy:   "ecosystem-manager",
	}

	createdTask, err := s.client.CreateTask(ctx, amTask)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build profile reference with current settings
	profileRef := s.buildTaskProfileRef()

	// Create run
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: profileRef,
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true, // Bypass capacity limits for ecosystem tasks
		Prompt:     proto.String(req.Prompt),
	}

	// Apply timeout override if specified
	if req.Timeout > 0 {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{
			Timeout: durationpb.New(req.Timeout),
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

	return s.buildExecuteResult(completedRun), nil
}

// ExecuteTaskAsync starts a task run without waiting for completion.
// Returns the run ID for tracking.
func (s *AgentService) ExecuteTaskAsync(ctx context.Context, req ExecuteRequest) (string, error) {
	tag := req.Tag
	if tag == "" {
		tag = fmt.Sprintf("ecosystem-%s", req.Task.ID)
	}

	// Create task in agent-manager
	amTask := &domainpb.Task{
		Title:       req.Task.Title,
		Description: fmt.Sprintf("Ecosystem task: %s (%s/%s)", req.Task.Title, req.Task.Type, req.Task.Operation),
		ScopePath:   s.vrooliRoot,
		ProjectRoot: s.vrooliRoot,
		CreatedBy:   "ecosystem-manager",
	}

	createdTask, err := s.client.CreateTask(ctx, amTask)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	// Build profile reference with current settings
	profileRef := s.buildTaskProfileRef()

	// Create run
	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: profileRef,
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true,
		Prompt:     proto.String(req.Prompt),
	}

	if req.Timeout > 0 {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{
			Timeout: durationpb.New(req.Timeout),
		}
	}

	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}

	return run.Id, nil
}

// InsightRequest contains parameters for insight generation.
type InsightRequest struct {
	// Task ID for tracking
	TaskID string
	// Prompt to send to the agent
	Prompt string
	// Timeout for this execution
	Timeout time.Duration
}

// ExecuteInsight runs an insight generation task.
func (s *AgentService) ExecuteInsight(ctx context.Context, req InsightRequest) (*ExecuteResult, error) {
	tag := fmt.Sprintf("insight-%s", req.TaskID)

	// Create task in agent-manager
	amTask := &domainpb.Task{
		Title:       fmt.Sprintf("Insight generation for %s", req.TaskID),
		Description: "Ecosystem insight generation",
		ScopePath:   s.vrooliRoot,
		ProjectRoot: s.vrooliRoot,
		CreatedBy:   "ecosystem-manager",
	}

	createdTask, err := s.client.CreateTask(ctx, amTask)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Build profile reference for insights
	profileRef := s.buildInsightsProfileRef()

	// Create run
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: profileRef,
		Tag:        &tag,
		RunMode:    domainpb.RunMode_RUN_MODE_IN_PLACE.Enum(),
		Force:      true,
		Prompt:     proto.String(req.Prompt),
		InlineConfig: &domainpb.RunConfigOverrides{
			Timeout: durationpb.New(timeout),
		},
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

	return s.buildExecuteResult(completedRun), nil
}

// GetRunStatus returns the current status of a run.
func (s *AgentService) GetRunStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	return s.client.GetRun(ctx, runID)
}

// StopRun stops an active run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	return s.client.StopRun(ctx, runID)
}

// GetRunEvents returns events for a run.
func (s *AgentService) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	return s.client.GetRunEvents(ctx, runID, afterSequence)
}

// WaitForRun waits for a run to complete.
func (s *AgentService) WaitForRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	return s.client.WaitForRun(ctx, runID, 2*time.Second)
}

// =============================================================================
// PROFILE CONFIGURATION
// =============================================================================

// ProfileConfig contains agent profile configuration.
type ProfileConfig struct {
	RunnerType       domainpb.RunnerType
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
}

func (s *AgentService) buildTaskProfileConfig() *ProfileConfig {
	currentSettings := settings.GetSettings()

	return &ProfileConfig{
		RunnerType:       s.getRunnerType(),
		MaxTurns:         int32(currentSettings.MaxTurns),
		TimeoutSeconds:   int32(currentSettings.TaskTimeout * 60), // Convert minutes to seconds
		AllowedTools:     parseToolsList(currentSettings.AllowedTools),
		SkipPermissions:  currentSettings.SkipPermissions,
		RequiresSandbox:  false, // In-place execution
		RequiresApproval: false, // Auto-apply
	}
}

func (s *AgentService) buildInsightsProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:       s.getRunnerType(),
		MaxTurns:         20,  // Lower turns for insights
		TimeoutSeconds:   300, // 5 minutes
		AllowedTools:     []string{"Read", "Glob", "Grep", "Bash"},
		SkipPermissions:  true,
		RequiresSandbox:  false,
		RequiresApproval: false,
	}
}

func (s *AgentService) getRunnerType() domainpb.RunnerType {
	currentSettings := settings.GetSettings()
	switch strings.ToLower(currentSettings.RunnerType) {
	case "codex":
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	case "opencode":
		return domainpb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	}
}

func (s *AgentService) buildProfile(profileKey, name, description string, cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 name,
		ProfileKey:           profileKey,
		Description:          description,
		RunnerType:           cfg.RunnerType,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "ecosystem-manager",
	}
}

func (s *AgentService) buildTaskProfileRef() *apipb.ProfileRef {
	cfg := s.buildTaskProfileConfig()
	return &apipb.ProfileRef{
		ProfileKey: s.taskProfileKey,
		Defaults:   s.buildProfile(s.taskProfileKey, "ecosystem-manager-tasks", "Agent profile for ecosystem-manager task execution", cfg),
	}
}

func (s *AgentService) buildInsightsProfileRef() *apipb.ProfileRef {
	cfg := s.buildInsightsProfileConfig()
	return &apipb.ProfileRef{
		ProfileKey: s.insightsProfileKey,
		Defaults:   s.buildProfile(s.insightsProfileKey, "ecosystem-manager-insights", "Agent profile for ecosystem-manager insight generation", cfg),
	}
}

func (s *AgentService) buildExecuteResult(run *domainpb.Run) *ExecuteResult {
	result := &ExecuteResult{
		RunID:   run.Id,
		Success: run.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE,
	}

	if run.Summary != nil {
		result.Output = run.Summary.Description
		result.TokensUsed = run.Summary.TokensUsed
		result.CostEstimate = run.Summary.CostEstimate
	}

	if run.ErrorMsg != "" {
		result.ErrorMessage = run.ErrorMsg
		lowerErr := strings.ToLower(run.ErrorMsg)
		result.RateLimited = strings.Contains(lowerErr, "rate limit") || strings.Contains(lowerErr, "429")
		result.Timeout = strings.Contains(lowerErr, "timeout")
		result.MaxTurnsExceeded = strings.Contains(lowerErr, "max_turns") || strings.Contains(lowerErr, "max turns")
	}

	// Calculate duration
	if run.StartedAt != nil && run.EndedAt != nil {
		duration := run.EndedAt.AsTime().Sub(run.StartedAt.AsTime())
		result.DurationSeconds = int(duration.Seconds())
	}

	return result
}

// parseToolsList splits a comma-separated tools string into a slice.
func parseToolsList(tools string) []string {
	if tools == "" {
		return nil
	}
	parts := strings.Split(tools, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
