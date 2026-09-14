package agentmanager

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/workflowcontract"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// GoalRunRequest describes one Agent Manager run for a Swarm goal execution.
type GoalRunRequest struct {
	ProfileKey      string
	Prompt          string
	Until           string
	PreferredRunner string
	Model           string
	Effort          string
	ResultSpec      *domainpb.ResultSpec
	IdempotencyKey  string
	Tag             string
	ScopePath       string
	ProjectRoot     string
}

// GoalRunResult is the durable outcome of creating a goal run.
type GoalRunResult struct {
	RunID           string
	TaskID          string
	SelectionReason string
}

// GoalRunState is the terminal projection Swarm reads for a goal run.
type GoalRunState struct {
	RunID         string
	Status        string
	TerminalClass string
	StopReason    string
	LastHandoff   string
	ErrorMessage  string
}

// GetGoalRunState reads one Agent Manager run's typed terminal fields so Swarm
// can map a verdict to finalization and an interruption to `interrupted`.
func (s *AgentService) GetGoalRunState(ctx context.Context, runID string) (GoalRunState, error) {
	if !s.enabled {
		return GoalRunState{}, ErrNotAvailable
	}
	run, err := s.client.GetRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return GoalRunState{}, err
	}
	return GoalRunState{
		RunID:         run.GetId(),
		Status:        run.GetStatus().String(),
		TerminalClass: run.GetTerminalClass(),
		StopReason:    run.GetStopReason(),
		LastHandoff:   run.GetLastHandoff(),
		ErrorMessage:  run.GetErrorMsg(),
	}, nil
}

// GetGoalRunUsage reads a goal run's metered usage from the owner. terminal is
// false while the run is live. The usage flags stay false unless the owner
// reports a terminal, complete and measured receipt; one run has no workflow
// children, node attempts, retries or slices.
func (s *AgentService) GetGoalRunUsage(ctx context.Context, runID string) (*workflowcontract.Usage, bool, error) {
	if !s.enabled {
		return nil, false, ErrNotAvailable
	}
	accounting, err := s.client.GetRunAccounting(ctx, strings.TrimSpace(runID))
	if err != nil {
		return nil, false, err
	}
	return goalRunUsage(accounting), accounting.GetTerminal(), nil
}

func goalRunUsage(accounting *apipb.RunAccounting) *workflowcontract.Usage {
	terminal := accounting.GetTerminal()
	return &workflowcontract.Usage{
		Tokens:         accounting.GetTokens(),
		Turns:          accounting.GetTurns(),
		WallSeconds:    accounting.GetWallSeconds(),
		ChargeMicroUSD: accounting.GetChargeMicroUsd(),
		TokensKnown:    terminal && accounting.GetTokensKnown(),
		ChargeMeasured: terminal && accounting.GetChargeMeasured(),
	}
}

// CreateGoalRun creates exactly one Agent Manager run whose prompt is the
// composed goal message and whose until is the finish line. It is the goal
// mode's run-creation seam; the sliced route continues to use the workflow
// runner. Idempotency is enforced by the caller's key, so a re-queue reuses the
// same run.
func (s *AgentService) CreateGoalRun(ctx context.Context, req GoalRunRequest) (GoalRunResult, error) {
	if !s.enabled {
		return GoalRunResult{}, ErrNotAvailable
	}
	if strings.TrimSpace(req.Prompt) == "" || strings.TrimSpace(req.Until) == "" {
		return GoalRunResult{}, fmt.Errorf("%w: goal run requires a prompt and an until", ErrRequestFailed)
	}
	profileRef, err := s.profileRefFor(req.ProfileKey)
	if err != nil {
		return GoalRunResult{}, err
	}
	title := strings.TrimSpace(req.Tag)
	if title == "" {
		title = "swarm goal execution"
	}
	task, err := s.client.CreateTask(ctx, &domainpb.Task{
		Title:       title,
		ScopePath:   strings.TrimSpace(req.ScopePath),
		ProjectRoot: strings.TrimSpace(req.ProjectRoot),
		CreatedBy:   "swarm-manager",
	})
	if err != nil {
		return GoalRunResult{}, err
	}
	inline := &domainpb.RunConfigOverrides{}
	if until := strings.TrimSpace(req.Until); until != "" {
		inline.Until = &until
	}
	if req.ResultSpec != nil {
		inline.ResultSpec = req.ResultSpec
	}
	runReq := &apipb.CreateRunRequest{
		TaskId:       task.Id,
		ProfileRef:   profileRef,
		Force:        true,
		InlineConfig: inline,
		// Preferred runner/model/effort ride the typed execution preferences so
		// agent-manager's role-policy resolution can target the runner the
		// item asked for instead of the goal-work profile's default.
		ExecutionPreferences: &domainpb.ExecutionPreferences{
			PreferredRunner: strings.TrimSpace(req.PreferredRunner),
			Model:           strings.TrimSpace(req.Model),
			Effort:          strings.TrimSpace(req.Effort),
		},
	}
	prompt := req.Prompt
	runReq.Prompt = &prompt
	if tag := strings.TrimSpace(req.Tag); tag != "" {
		runReq.Tag = &tag
	}
	if key := strings.TrimSpace(req.IdempotencyKey); key != "" {
		runReq.IdempotencyKey = &key
	}
	run, err := s.client.CreateRun(ctx, runReq)
	if err != nil {
		return GoalRunResult{}, err
	}
	if strings.TrimSpace(run.GetId()) == "" {
		return GoalRunResult{}, fmt.Errorf("%w: agent-manager created a goal run with no id", ErrRequestFailed)
	}
	return GoalRunResult{RunID: run.Id, TaskID: task.Id}, nil
}
