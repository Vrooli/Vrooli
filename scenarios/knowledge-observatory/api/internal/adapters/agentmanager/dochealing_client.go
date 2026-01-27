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

	"knowledge-observatory/internal/services/dochealing"
)

// DocHealingProfileConfig defines the agent profile for doc healing.
type DocHealingProfileConfig struct {
	ProfileKey       string
	ProfileName      string
	Description      string
	RunnerType       domainpb.RunnerType
	Model            string
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
	SandboxConfig    *domainpb.SandboxConfig
	CreatedBy        string
}

// DefaultDocHealingProfileConfig returns the default doc healing profile settings.
func DefaultDocHealingProfileConfig() DocHealingProfileConfig {
	return DocHealingProfileConfig{
		ProfileKey:       "doc-healer",
		ProfileName:      "Documentation Healing",
		Description:      "Agent profile for documentation healing tasks",
		RunnerType:       domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		Model:            "claude-opus-4.5",
		MaxTurns:         20,
		TimeoutSeconds:   600,
		AllowedTools:     []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "LS"},
		SkipPermissions:  true,
		RequiresSandbox:  true,
		RequiresApproval: true,
		SandboxConfig: &domainpb.SandboxConfig{
			Acceptance: &domainpb.SandboxAcceptanceConfig{
				Mode: domainpb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST,
				Allow: &domainpb.SandboxFileCriteria{
					PathGlobs:  []string{"scenarios/*/docs/**", "scenarios/*/*.md", "docs/**"},
					Extensions: []string{".md", ".json", ".txt"},
				},
				IgnoreBinary: true,
			},
		},
		CreatedBy: "knowledge-observatory",
	}
}

// DocHealingClient implements dochealing.AgentClient using agent-manager.
type DocHealingClient struct {
	client *Client
	cfg    DocHealingProfileConfig
	mu     sync.RWMutex
	id     string
}

// NewDocHealingClient creates a new doc healing client.
func NewDocHealingClient(timeout time.Duration, cfg DocHealingProfileConfig) *DocHealingClient {
	return &DocHealingClient{
		client: NewClient(timeout),
		cfg:    cfg,
	}
}

// NewDocHealingClientWithBaseURL creates a doc healing client with a fixed agent-manager base URL.
func NewDocHealingClientWithBaseURL(timeout time.Duration, cfg DocHealingProfileConfig, baseURL string) *DocHealingClient {
	return &DocHealingClient{
		client: NewClientWithBaseURL(timeout, baseURL),
		cfg:    cfg,
	}
}

func (c *DocHealingClient) EnsureProfile(ctx context.Context) error {
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

func (c *DocHealingClient) CreateRun(ctx context.Context, req dochealing.AgentRunRequest) (string, error) {
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

func (c *DocHealingClient) GetRun(ctx context.Context, runID string) (*dochealing.AgentRun, error) {
	run, err := c.client.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	summary := ""
	if run.Summary != nil {
		summary = strings.TrimSpace(run.Summary.Description)
	}
	return &dochealing.AgentRun{
		ID:      run.Id,
		Status:  mapDocHealingRunStatus(run.Status),
		Error:   strings.TrimSpace(run.ErrorMsg),
		Summary: summary,
	}, nil
}

func (c *DocHealingClient) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]dochealing.AgentRunEvent, error) {
	events, err := c.client.GetRunEvents(ctx, runID, afterSequence)
	if err != nil {
		return nil, err
	}
	out := make([]dochealing.AgentRunEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		normalized := dochealing.AgentRunEvent{
			Sequence: event.Sequence,
		}
		switch event.EventType {
		case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
			if msg := event.GetMessage(); msg != nil {
				normalized.Type = dochealing.EventMessage
				normalized.Role = msg.Role
				normalized.Content = msg.Content
			}
		default:
			if progress := event.GetProgress(); progress != nil {
				normalized.Type = dochealing.EventProgress
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

func (c *DocHealingClient) GetRunDiff(ctx context.Context, runID string) (*dochealing.RunDiff, error) {
	diff, err := c.client.GetRunDiff(ctx, runID)
	if err != nil {
		return nil, err
	}
	if diff == nil {
		return nil, nil
	}
	files := make([]dochealing.RunFileDiff, 0, len(diff.Files))
	for _, file := range diff.Files {
		if file == nil {
			continue
		}
		files = append(files, dochealing.RunFileDiff{
			Path:       strings.TrimSpace(file.Path),
			ChangeType: strings.TrimSpace(file.ChangeType),
			Additions:  file.Additions,
			Deletions:  file.Deletions,
		})
	}
	return &dochealing.RunDiff{
		Content: strings.TrimSpace(diff.Content),
		Files:   files,
	}, nil
}

func (c *DocHealingClient) ApproveRun(ctx context.Context, req dochealing.ApprovalRequest) (*dochealing.ApproveResult, error) {
	actor := strings.TrimSpace(req.Actor)
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}
	var commitPtr *string
	if msg := strings.TrimSpace(req.CommitMsg); msg != "" {
		commitPtr = &msg
	}
	resp, err := c.client.ApproveRun(ctx, &apipb.ApproveRunRequest{
		RunId:     strings.TrimSpace(req.RunID),
		Actor:     actorPtr,
		CommitMsg: commitPtr,
		Force:     req.Force,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Result == nil {
		return &dochealing.ApproveResult{Success: true}, nil
	}
	return &dochealing.ApproveResult{
		Success:      resp.Result.Success,
		FilesApplied: resp.Result.FilesApplied,
		CommitHash:   resp.Result.CommitHash,
		Message:      strings.TrimSpace(resp.Result.Message),
	}, nil
}

func (c *DocHealingClient) RejectRun(ctx context.Context, req dochealing.RejectRequest) error {
	actor := strings.TrimSpace(req.Actor)
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}
	reason := strings.TrimSpace(req.Reason)
	_, err := c.client.RejectRun(ctx, &apipb.RejectRunRequest{
		RunId:  strings.TrimSpace(req.RunID),
		Actor:  actorPtr,
		Reason: reason,
	})
	return err
}

func (c *DocHealingClient) buildProfile() *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 c.cfg.ProfileName,
		ProfileKey:           c.cfg.ProfileKey,
		Description:          c.cfg.Description,
		RunnerType:           c.cfg.RunnerType,
		Model:                c.cfg.Model,
		MaxTurns:             c.cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(c.cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         c.cfg.AllowedTools,
		SkipPermissionPrompt: c.cfg.SkipPermissions,
		RequiresSandbox:      c.cfg.RequiresSandbox,
		RequiresApproval:     c.cfg.RequiresApproval,
		SandboxConfig:        c.cfg.SandboxConfig,
		CreatedBy:            c.cfg.CreatedBy,
	}
}

func (c *DocHealingClient) profileID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

func (c *DocHealingClient) resolveRunMode() *domainpb.RunMode {
	if c.cfg.RequiresSandbox {
		mode := domainpb.RunMode_RUN_MODE_SANDBOXED
		return &mode
	}
	mode := domainpb.RunMode_RUN_MODE_IN_PLACE
	return &mode
}

func mapDocHealingRunStatus(status domainpb.RunStatus) dochealing.RunStatus {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return dochealing.RunStatusComplete
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return dochealing.RunStatusFailed
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return dochealing.RunStatusCancelled
	case domainpb.RunStatus_RUN_STATUS_RUNNING, domainpb.RunStatus_RUN_STATUS_STARTING:
		return dochealing.RunStatusRunning
	default:
		return dochealing.RunStatusPending
	}
}
