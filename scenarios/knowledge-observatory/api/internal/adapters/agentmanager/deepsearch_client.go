package agentmanager

// DOC: docs/concepts/ARCHITECTURE.md#integrations

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"

	"knowledge-observatory/internal/services/deepsearch"
)

// DeepSearchProfileConfig defines the agent profile for deep search.
type DeepSearchProfileConfig struct {
	ProfileKey      string
	ProfileName     string
	Description     string
	RunnerType      domainpb.RunnerType
	Model           string
	MaxTurns        int32
	TimeoutSeconds  int32
	AllowedTools    []string
	SkipPermissions bool
	// SandboxMode selects the sandbox execution mode. Read-only deep
	// search runs in-place because there are no writes to audit. See
	// agent-manager domain.DeriveRunMode.
	SandboxMode domainpb.SandboxMode
	CreatedBy   string
}

// DefaultDeepSearchProfileConfig returns the default deep search profile settings.
func DefaultDeepSearchProfileConfig() DeepSearchProfileConfig {
	return DeepSearchProfileConfig{
		ProfileKey:      "deep-documentation-search",
		ProfileName:     "Deep Documentation Search",
		Description:     "Agent profile for read-only documentation deep search",
		RunnerType:      domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		Model:           "claude-3-haiku",
		MaxTurns:        10,
		TimeoutSeconds:  60,
		AllowedTools:    []string{"Read", "Glob", "Grep"},
		SkipPermissions: true,
		// Read-only — no sandbox needed.
		SandboxMode: domainpb.SandboxMode_SANDBOX_MODE_OFF,
		CreatedBy:   "knowledge-observatory",
	}
}

// DeepSearchClient implements deepsearch.AgentClient using agent-manager.
type DeepSearchClient struct {
	client *Client
	cfg    DeepSearchProfileConfig
	mu     sync.RWMutex
	id     string
}

// NewDeepSearchClient creates a new deep search client.
func NewDeepSearchClient(timeout time.Duration, cfg DeepSearchProfileConfig) *DeepSearchClient {
	return &DeepSearchClient{
		client: NewClient(timeout),
		cfg:    cfg,
	}
}

// NewDeepSearchClientWithBaseURL creates a deep search client with a fixed agent-manager base URL.
func NewDeepSearchClientWithBaseURL(timeout time.Duration, cfg DeepSearchProfileConfig, baseURL string) *DeepSearchClient {
	return &DeepSearchClient{
		client: NewClientWithBaseURL(timeout, baseURL),
		cfg:    cfg,
	}
}

func (c *DeepSearchClient) EnsureProfile(ctx context.Context) error {
	resp, err := c.client.EnsureProfile(ctx, &apipb.EnsureProfileRequest{
		ProfileKey:     c.cfg.ProfileKey,
		Defaults:       c.buildProfile(),
		UpdateExisting: false,
	})
	if err != nil {
		return fmt.Errorf("ensure profile: %w", err)
	}
	if resp.Profile != nil && resp.Profile.Id != "" {
		c.mu.Lock()
		c.id = resp.Profile.Id
		c.mu.Unlock()
	}
	return nil
}

func (c *DeepSearchClient) CreateRun(ctx context.Context, req deepsearch.AgentRunRequest) (string, error) {
	if err := c.EnsureProfile(ctx); err != nil {
		return "", err
	}
	createdTask, err := c.client.CreateTask(ctx, &domainpb.Task{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		ScopePath:   strings.TrimSpace(req.ScopePath),
		ProjectRoot: strings.TrimSpace(req.ProjectRoot),
		CreatedBy:   c.cfg.CreatedBy,
	})
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}
	runReq := &apipb.CreateRunRequest{
		TaskId:  createdTask.Id,
		Force:   true,
		Prompt:  &req.Prompt,
		RunMode: c.resolveRunMode(),
	}
	if req.Tag != "" {
		tag := req.Tag
		runReq.Tag = &tag
	}
	if c.profileID() != "" {
		id := c.profileID()
		runReq.AgentProfileId = &id
	} else {
		runReq.ProfileRef = &apipb.ProfileRef{
			ProfileKey: c.cfg.ProfileKey,
			Defaults:   c.buildProfile(),
		}
	}
	if req.Timeout > 0 {
		runReq.InlineConfig = &domainpb.RunConfigOverrides{
			Timeout: durationpb.New(req.Timeout),
		}
	}
	run, err := c.client.CreateRun(ctx, runReq)
	if err != nil {
		return "", fmt.Errorf("create run: %w", err)
	}
	return run.Id, nil
}

func (c *DeepSearchClient) GetRun(ctx context.Context, runID string) (*deepsearch.AgentRun, error) {
	run, err := c.client.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	return &deepsearch.AgentRun{
		ID:     run.Id,
		Status: mapRunStatus(run.Status),
		Error:  strings.TrimSpace(run.ErrorMsg),
	}, nil
}

func (c *DeepSearchClient) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]deepsearch.AgentRunEvent, error) {
	events, err := c.client.GetRunEvents(ctx, runID, afterSequence)
	if err != nil {
		return nil, err
	}
	out := make([]deepsearch.AgentRunEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		normalized := deepsearch.AgentRunEvent{
			Sequence: event.Sequence,
		}
		switch event.EventType {
		case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
			if msg := event.GetMessage(); msg != nil {
				normalized.Type = deepsearch.EventMessage
				normalized.Role = msg.Role
				normalized.Content = msg.Content
			}
		default:
			if progress := event.GetProgress(); progress != nil {
				normalized.Type = deepsearch.EventProgress
				normalized.ProgressPercent = progress.PercentComplete
				normalized.ProgressAction = progress.CurrentAction
				normalized.ProgressPhase = progress.Phase.String()
			}
		}
		if normalized.Type != "" {
			out = append(out, normalized)
		}
	}
	return out, nil
}

func (c *DeepSearchClient) buildProfile() *domainpb.AgentProfile {
	profile := &domainpb.AgentProfile{
		Name:                 c.cfg.ProfileName,
		ProfileKey:           c.cfg.ProfileKey,
		Description:          c.cfg.Description,
		RunnerType:           c.cfg.RunnerType,
		Model:                c.cfg.Model,
		MaxTurns:             c.cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(c.cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         c.cfg.AllowedTools,
		SkipPermissionPrompt: c.cfg.SkipPermissions,
		CreatedBy:            c.cfg.CreatedBy,
	}
	if c.cfg.SandboxMode != domainpb.SandboxMode_SANDBOX_MODE_UNSPECIFIED {
		profile.SandboxConfig = &domainpb.SandboxConfig{Mode: c.cfg.SandboxMode}
	}
	return profile
}

func (c *DeepSearchClient) profileID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

// resolveRunMode lets agent-manager derive the mode from the resolved
// SandboxConfig (single source of truth). Returning nil here is the
// preferred default; the orchestrator's domain.DeriveRunMode handles
// the OFF→InPlace and other-→Sandboxed mapping.
func (c *DeepSearchClient) resolveRunMode() *domainpb.RunMode {
	return nil
}

func mapRunStatus(status domainpb.RunStatus) deepsearch.RunStatus {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return deepsearch.RunStatusComplete
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return deepsearch.RunStatusFailed
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return deepsearch.RunStatusCancelled
	case domainpb.RunStatus_RUN_STATUS_RUNNING, domainpb.RunStatus_RUN_STATUS_STARTING:
		return deepsearch.RunStatusRunning
	default:
		return deepsearch.RunStatusPending
	}
}
