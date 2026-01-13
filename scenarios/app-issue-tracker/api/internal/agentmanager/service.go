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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Runner defines the agent-manager methods used by app-issue-tracker.
type Runner interface {
	IsAvailable(ctx context.Context) bool
	Initialize(ctx context.Context, cfg ProfileConfig) error
	UpdateProfile(ctx context.Context, cfg ProfileConfig) error
	CreateRun(ctx context.Context, req RunRequest, cfg ProfileConfig) (string, error)
	WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*domainpb.Run, error)
	GetRun(ctx context.Context, runID string) (*domainpb.Run, error)
	GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error)
	StopRun(ctx context.Context, runID string) error
}

// RunnerType aliases the agent-manager runner type enum.
type RunnerType = domainpb.RunnerType

const (
	RunnerTypeClaudeCode = domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	RunnerTypeCodex      = domainpb.RunnerType_RUNNER_TYPE_CODEX
	RunnerTypeOpenCode   = domainpb.RunnerType_RUNNER_TYPE_OPENCODE
)

// AgentService provides agent execution services for app-issue-tracker.
type AgentService struct {
	client     *Client
	profileKey string
	profileID  string
	vrooliRoot string
	mu         sync.RWMutex
}

// Config contains configuration for the agent service.
type Config struct {
	ProfileKey string
	Timeout    time.Duration
	VrooliRoot string
}

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

// RunRequest contains parameters for a run.
type RunRequest struct {
	IssueID    string
	IssueTitle string
	Prompt     string
	Timeout    time.Duration
	Tag        string
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg Config) *AgentService {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &AgentService{
		client:     NewClient(timeout),
		profileKey: cfg.ProfileKey,
		vrooliRoot: cfg.VrooliRoot,
	}
}

// IsAvailable checks if agent-manager is reachable.
func (s *AgentService) IsAvailable(ctx context.Context) bool {
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// Initialize ensures the profile exists with current settings.
func (s *AgentService) Initialize(ctx context.Context, cfg ProfileConfig) error {
	resp, err := s.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     s.profileKey,
		Defaults:       s.buildProfile(s.profileKey, "app-issue-tracker-investigations", "Agent profile for app-issue-tracker investigations", cfg),
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
		log.Printf("[agent-manager] Created profile '%s' (id=%s)", s.profileKey, s.profileID)
	} else {
		log.Printf("[agent-manager] Resolved profile '%s' (id=%s)", s.profileKey, s.profileID)
	}

	return nil
}

// UpdateProfile updates the profile with new settings.
func (s *AgentService) UpdateProfile(ctx context.Context, cfg ProfileConfig) error {
	s.mu.RLock()
	profileID := s.profileID
	s.mu.RUnlock()

	if profileID == "" {
		return s.Initialize(ctx, cfg)
	}

	profile := s.buildProfile(s.profileKey, "app-issue-tracker-investigations", "Agent profile for app-issue-tracker investigations", cfg)
	profile.Id = profileID
	if _, err := s.client.UpdateProfile(ctx, profileID, profile); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	log.Printf("[agent-manager] Updated profile '%s'", s.profileKey)
	return nil
}

// CreateRun creates a run and returns the run ID.
func (s *AgentService) CreateRun(ctx context.Context, req RunRequest, cfg ProfileConfig) (string, error) {
	tag := req.Tag
	if tag == "" {
		tag = fmt.Sprintf("app-issue-tracker-%s", req.IssueID)
	}

	title := strings.TrimSpace(req.IssueTitle)
	if title == "" {
		title = fmt.Sprintf("Issue %s", req.IssueID)
	}

	amTask := &domainpb.Task{
		Title:       title,
		Description: fmt.Sprintf("Issue investigation: %s", req.IssueID),
		ScopePath:   s.vrooliRoot,
		ProjectRoot: s.vrooliRoot,
		CreatedBy:   "app-issue-tracker",
	}

	createdTask, err := s.client.CreateTask(ctx, amTask)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	runReq := &apipb.CreateRunRequest{
		TaskId:     createdTask.Id,
		ProfileRef: s.buildProfileRef(cfg),
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

// WaitForRun waits for a run to complete.
func (s *AgentService) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*domainpb.Run, error) {
	return s.client.WaitForRun(ctx, runID, pollInterval)
}

// GetRun retrieves a run by ID.
func (s *AgentService) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	return s.client.GetRun(ctx, runID)
}

// GetRunEvents retrieves events for a run.
func (s *AgentService) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	return s.client.GetRunEvents(ctx, runID, afterSequence)
}

// StopRun stops an active run.
func (s *AgentService) StopRun(ctx context.Context, runID string) error {
	return s.client.StopRun(ctx, runID)
}

func (s *AgentService) buildProfile(profileKey, name, description string, cfg ProfileConfig) *domainpb.AgentProfile {
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
		CreatedBy:            "app-issue-tracker",
	}
}

func (s *AgentService) buildProfileRef(cfg ProfileConfig) *apipb.ProfileRef {
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
		Defaults:   s.buildProfile(s.profileKey, "app-issue-tracker-investigations", "Agent profile for app-issue-tracker investigations", cfg),
	}
}
