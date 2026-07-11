package agentmanager

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/scenario"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

// NewAgentServiceWithClient creates an agent service with a pre-built client (for testing).
func NewAgentServiceWithClient(client *Client, profileName, profileKey string, enabled bool) *AgentService {
	return &AgentService{
		client:      client,
		profileName: profileName,
		profileKey:  profileKey,
		enabled:     enabled,
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

// Initialize reconciles the scenario-owned role-only profile source.
// Call this at startup; profile definition remains in .vrooli/agent-profiles.
func (s *AgentService) Initialize(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	resp, err := s.client.ReconcileScenarioProfiles(ctx, scenario.Name())
	if err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}

	s.mu.Lock()
	for _, item := range resp.Results {
		if item.ProfileKey == s.profileKey {
			s.profileID = item.ProfileId
			break
		}
	}
	s.mu.Unlock()

	log.Printf("[agent-manager] Reconciled profiles for %s (created=%d updated=%d unchanged=%d failed=%d)",
		resp.Scenario, resp.Created, resp.Updated, resp.Unchanged, resp.Failed)

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

	// Pipeline investigations are diagnostic — the deliverable is a
	// report on build state, logs, and packaging, not repo changes.
	// ManualReview=true defers apply at run end so any file mutations
	// persist as pending-review for operator approval rather than
	// auto-applying. See workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
	if runReq.InlineConfig == nil {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
	}
	if runReq.InlineConfig.SandboxConfig == nil {
		runReq.InlineConfig.SandboxConfig = &domainpb.SandboxConfig{}
	}
	runReq.InlineConfig.SandboxConfig.ManualReview = true

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

	// Pipeline investigations are diagnostic — see Execute() above for
	// ManualReview=true rationale.
	if runReq.InlineConfig == nil {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{}
	}
	if runReq.InlineConfig.SandboxConfig == nil {
		runReq.InlineConfig.SandboxConfig = &domainpb.SandboxConfig{}
	}
	runReq.InlineConfig.SandboxConfig.ManualReview = true

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
