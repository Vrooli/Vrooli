package agentmanager

// DOC: docs/concepts/ARCHITECTURE.md#integrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"

	"knowledge-observatory/internal/services/deepsearch"
)

// DeepSearchProfileConfig defines the agent profile for deep search.
type DeepSearchProfileConfig struct {
	ProfileKey string
	CreatedBy  string
}

// DefaultDeepSearchProfileConfig returns the default deep search profile settings.
func DefaultDeepSearchProfileConfig() DeepSearchProfileConfig {
	return DeepSearchProfileConfig{
		ProfileKey: "knowledge-observatory/deep-search",
		CreatedBy:  "knowledge-observatory",
	}
}

// DeepSearchClient implements deepsearch.AgentClient using agent-manager.
type DeepSearchClient struct {
	client *Client
	cfg    DeepSearchProfileConfig
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

func (c *DeepSearchClient) reconcileProfiles(ctx context.Context) error {
	resp, err := c.client.ReconcileScenarioProfiles(ctx, "knowledge-observatory")
	if err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}
	for _, result := range resp.Results {
		if result.GetProfileKey() == c.cfg.ProfileKey && result.GetProfileId() != "" {
			return nil
		}
	}
	return fmt.Errorf("reconciliation returned no profile %q", c.cfg.ProfileKey)
}

func (c *DeepSearchClient) CreateRun(ctx context.Context, req deepsearch.AgentRunRequest) (string, error) {
	if err := c.reconcileProfiles(ctx); err != nil {
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
		TaskId: createdTask.Id,
		Force:  true,
		Prompt: &req.Prompt,
	}
	if req.Tag != "" {
		tag := req.Tag
		runReq.Tag = &tag
	}
	runReq.ProfileRef = &apipb.ProfileRef{ProfileKey: c.cfg.ProfileKey}
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
