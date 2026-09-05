package agentmanager

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Service is the higher-level spawn/poll seam the L3 agent depends on. It hides
// the HTTP/proto details behind two methods: Spawn (CreateTask -> CreateRun)
// and GetRunState (GetRun + status normalization). Tests fake this directly.
type Service interface {
	// Spawn creates a research task + run and returns the run handle.
	Spawn(ctx context.Context, req SpawnRequest) (RunResult, error)
	// GetRunState polls a run by id.
	GetRunState(ctx context.Context, runID string) (RunState, error)
}

// SpawnRequest describes an L3 research run to start.
type SpawnRequest struct {
	Query  string
	Title  string
	Prompt string
}

// RunResult returns agent-manager identifiers for a started run.
type RunResult struct {
	TaskID string
	RunID  string
	Status string
}

// RunState captures externally visible lifecycle state for a run.
type RunState struct {
	RunID      string
	Status     string
	Summary    string
	StartedAt  string
	FinishedAt string
	ErrorMsg   string
}

// agentService implements Service over a Client.
type agentService struct {
	client Client
}

// NewService constructs the production agent-manager research service over the
// given client (typically NewHTTPClient()).
func NewService(client Client) Service {
	return &agentService{client: client}
}

func (s *agentService) Spawn(ctx context.Context, req SpawnRequest) (RunResult, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "L3 research: " + strings.TrimSpace(req.Query)
	}
	task := &domainpb.Task{
		Title:       title,
		Description: strings.TrimSpace(req.Prompt),
		ScopePath:   ".",
		ProjectRoot: ".",
		CreatedBy:   "web-search",
	}
	createdTask, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return RunResult{}, err
	}

	conversationID := uuid.NewString()
	force := true
	runReq := &apipb.CreateRunRequest{
		ConversationId: &conversationID,
		TaskId:         createdTask.Id,
		Force:          force,
	}
	if prompt := strings.TrimSpace(req.Prompt); prompt != "" {
		runReq.Prompt = &prompt
	}
	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{
		TaskID: createdTask.Id,
		RunID:  run.Id,
		Status: NormalizeRunStatus(run.Status),
	}, nil
}

func (s *agentService) GetRunState(ctx context.Context, runID string) (RunState, error) {
	run, err := s.client.GetRun(ctx, runID)
	if err != nil {
		return RunState{}, err
	}
	state := RunState{
		RunID:    strings.TrimSpace(run.Id),
		Status:   NormalizeRunStatus(run.Status),
		ErrorMsg: strings.TrimSpace(run.ErrorMsg),
	}
	if run.StartedAt != nil {
		state.StartedAt = run.StartedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.EndedAt != nil {
		state.FinishedAt = run.EndedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if run.Summary != nil && strings.TrimSpace(run.Summary.Description) != "" {
		state.Summary = strings.TrimSpace(run.Summary.Description)
	}
	return state, nil
}

// NormalizeRunStatus maps the agent-manager RunStatus enum to a stable lowercase
// string surfaced on the research status RPC.
func NormalizeRunStatus(status domainpb.RunStatus) string {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_PENDING:
		return "pending"
	case domainpb.RunStatus_RUN_STATUS_STARTING:
		return "starting"
	case domainpb.RunStatus_RUN_STATUS_RUNNING:
		return "running"
	case domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return "needs_review"
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return "complete"
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}

var _ Service = (*agentService)(nil)
