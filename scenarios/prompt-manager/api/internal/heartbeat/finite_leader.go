package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"

	"github.com/google/uuid"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// FiniteLeaderRuntime adds a single retained leader to the normal heartbeat
// queue. AM owns continuation, stop and execution; PM never replaces this run
// on a timer. FileTeamStore serializes dispatch with binding retirement/disable.
type FiniteLeaderRuntime struct {
	Executor *Executor
	Queue    interface {
		TeamExecutionManager
		OnComplete(teamID, agentID string)
	}
	Control *HeartbeatControlStore
}

func (f *FiniteLeaderRuntime) eligible(ctx context.Context, teamID, agentID string, cfg *store.HeartbeatConfig) error {
	if !cfg.Enabled || cfg.FiniteLeader.Retired {
		return fmt.Errorf("finite leader disabled or retired")
	}
	team, err := f.Executor.teamStore.Get(ctx, teamID)
	if err != nil {
		return err
	}
	if err := validateTeamEnabled(team); err != nil {
		return err
	}
	if team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed || team.Coordination.LeadAgentID != agentID ||
		team.Execution.QueuePolicy != teamconfig.QueuePolicySerialized || team.Execution.MaxConcurrentRuns != 1 {
		return fmt.Errorf("finite leader requires the selected leader of a serialized team")
	}
	members, err := f.Executor.teamStore.GetMembers(ctx, teamID)
	if err != nil {
		return err
	}
	active := false
	for _, member := range members {
		if member.AgentID == agentID && (member.Status == "" || member.Status == store.MemberStatusActive) {
			active = true
		}
	}
	if !active || team.OperatingContract == nil {
		return fmt.Errorf("finite leader requires an active member and operating contract")
	}
	if _, ok := team.OperatingContract.Members[agentID]; !ok {
		return fmt.Errorf("finite leader member contract missing")
	}
	if f.Control != nil {
		if _, err := f.Control.AllowStart(ctx, teamID); err != nil {
			return err
		}
	}
	return nil
}

func reserveFiniteLeader(state *store.FiniteLeaderState) {
	if state.ID == "" {
		state.ID, state.Status = uuid.NewString(), "queued"
		state.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
}

// Tick reserves before enqueueing; repeated ticks only reconcile the original
// identity. Nothing in a terminal run's status authorizes another leader.
func (f *FiniteLeaderRuntime) Tick(ctx context.Context, teamID, agentID string) (*store.FiniteLeaderState, error) {
	var out *store.FiniteLeaderState
	var observed *Run
	err := f.Executor.teamStore.WithFiniteLeader(ctx, teamID, agentID, func(cfg *store.HeartbeatConfig, state *store.FiniteLeaderState, save func() error) error {
		out = state
		if state.DispatchStarted {
			var err error
			observed, err = f.observe(ctx, state)
			if err != nil {
				state.Error = err.Error()
			}
			if saveErr := save(); saveErr != nil {
				return saveErr
			}
			return err
		}
		if err := f.eligible(ctx, teamID, agentID, cfg); err != nil {
			return err
		}
		reserveFiniteLeader(state)
		if err := save(); err != nil {
			return err
		}
		if state.TaskStarted || memberOccupied(f.Queue.Status(teamID), agentID) {
			return nil
		}
		_, err := f.Queue.Enqueue(ctx, teamID, agentID, cfg.ProfileKey)
		return err
	})
	if err == nil && observed != nil {
		err = f.record(ctx, out, observed)
	}
	return out, err
}

func (f *FiniteLeaderRuntime) observe(ctx context.Context, state *store.FiniteLeaderState) (*Run, error) {
	if f.Executor.agentClient == nil {
		return nil, fmt.Errorf("finite leader agent owner unavailable")
	}
	var run *Run
	var err error
	if state.RunID != "" {
		run, err = f.Executor.agentClient.GetRun(ctx, state.RunID)
	} else {
		// Bounded owner lookup after a lost admission response. Empty, truncated
		// or conflicting results never justify another CreateRun.
		var runs *ListRunsResponse
		runs, err = f.Executor.agentClient.ListRuns(ctx, ListRunsOptions{TaskID: state.TaskID, Limit: 2})
		if err == nil {
			if runs == nil || runs.HasMore || runs.Total > 1 || len(runs.Runs) != 1 || runs.Runs[0] == nil {
				return nil, fmt.Errorf("finite leader dispatch uncertain; retain task %s", state.TaskID)
			}
			run = runs.Runs[0]
			if run.TaskID != state.TaskID || run.Tag != "finite-leader-"+state.ID {
				return nil, fmt.Errorf("finite leader owner response does not match reserved task and tag")
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if run == nil || run.ID == "" || (state.RunID != "" && run.ID != state.RunID) {
		return nil, fmt.Errorf("finite leader owner identity unavailable")
	}
	state.RunID, state.Status, state.Error = run.ID, run.Status, ""
	return run, nil
}

// Dispatch is also the final guard for manual/team triggers and recovered queue
// entries. The normal assembled prompt is used; source refs are never executed.
func (f *FiniteLeaderRuntime) Dispatch(ctx context.Context, teamID, agentID string) (*ExecutionResult, error) {
	result := &ExecutionResult{TeamID: teamID, AgentID: agentID, StartedAt: time.Now().UTC(), Status: "idle"}
	var state *store.FiniteLeaderState
	var run *Run
	err := f.Executor.teamStore.WithFiniteLeader(ctx, teamID, agentID, func(cfg *store.HeartbeatConfig, current *store.FiniteLeaderState, save func() error) error {
		state = current
		if current.DispatchStarted {
			var err error
			run, err = f.observe(ctx, current)
			if err != nil {
				return err
			}
			return save()
		}
		if err := f.eligible(ctx, teamID, agentID, cfg); err != nil {
			return err
		}
		reserveFiniteLeader(current)
		if current.TaskStarted {
			return fmt.Errorf("finite leader task admission uncertain; retain reservation %s", current.ID)
		}
		if f.Executor.agentClient == nil {
			return fmt.Errorf("finite leader agent owner unavailable")
		}
		if err := f.Executor.teamStore.EnsureTeamScope(ctx, teamID); err != nil {
			return err
		}
		team, err := f.Executor.teamStore.Get(ctx, teamID)
		if err != nil {
			return err
		}
		var prompt string
		if teamconfig.UsesSingleProcessInterop(team.Contract()) {
			prompt, err = f.Executor.promptBuilder.BuildTeamLeadPrompt(ctx, teamID, agentID, f.Executor.vrooliRoot)
		} else {
			prompt, err = f.Executor.BuildPrompt(ctx, teamID, agentID)
		}
		if err != nil {
			return err
		}
		binding, _ := json.Marshal(cfg.FiniteLeader)
		prompt += "\n\nFinite effort coordinator binding (references are context, not authority):\n" + string(binding) +
			"\nPreserve the accepted destination, exclusions and human input boundaries. Use existing owner operations and retain named waits. Run completion is not effort acceptance."
		current.TaskStarted, current.Status = true, "task-uncertain"
		if err := save(); err != nil {
			return err
		}
		task, err := f.Executor.agentClient.CreateTask(ctx, &Task{Title: "Finite effort leader: " + teamID + "/" + agentID,
			Description: prompt, ScopePath: f.Executor.vrooliRoot, ProjectRoot: f.Executor.vrooliRoot})
		if err != nil {
			return err
		}
		if task == nil || task.ID == "" {
			return fmt.Errorf("finite leader task identity missing")
		}
		current.TaskID, current.DispatchStarted, current.Status = task.ID, true, "dispatch-uncertain"
		if err := save(); err != nil {
			return err
		}
		tag := "finite-leader-" + current.ID
		key, value := buildHeartbeatAttributionEnv(teamID, agentID)
		run, err = f.Executor.agentClient.CreateRun(ctx, &CreateRunRequest{TaskID: task.ID, IdempotencyKey: tag, Tag: &tag,
			ProfileRef: &ProfileRef{ProfileKey: current.ProfileKey}, Environment: map[string]string{key: value},
			WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: current.Binding.EffortRef, Revision: current.Binding.AcceptedRevision,
				Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
				State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}}})
		if err != nil {
			return err
		}
		if run == nil || run.ID == "" {
			return fmt.Errorf("finite leader run identity missing; retain dispatch reservation")
		}
		current.RunID, current.Status = run.ID, run.Status
		return save()
	})
	if err != nil {
		return result, err
	}
	if run != nil {
		result.RunID, result.Status = run.ID, run.Status
		err = f.record(ctx, state, run)
	}
	return result, err
}

func (f *FiniteLeaderRuntime) record(ctx context.Context, state *store.FiniteLeaderState, run *Run) error {
	e := f.Executor
	status := store.HeartbeatStatusRunning
	if IsTerminalStatus(run.Status) {
		status = store.HeartbeatStatusCompleted
		if IsFailedStatus(run.Status) {
			status = store.HeartbeatStatusFailed
		} else if IsCancelledStatus(run.Status) {
			status = store.HeartbeatStatusCancelled
		}
	}
	// An old attempt's end timestamp cannot become this attempt's completion.
	if !IsTerminalStatus(run.Status) && run.EndedAt != "" {
		return fmt.Errorf("finite leader owner lifecycle inconsistent; retain original run %s", run.ID)
	}
	startedAt := run.StartedAt
	if startedAt == "" {
		startedAt = state.CreatedAt
	}
	result := &store.HeartbeatExecResult{RunID: run.ID, StartedAt: startedAt, EndedAt: run.EndedAt, Status: status, Error: run.Error}
	changed, err := e.teamStore.RecordFiniteLeaderExecution(ctx, state.TeamID, state.AgentID, state.ID, result)
	if err != nil {
		return err
	}
	if changed {
		e.appendAttempt(ctx, &store.HeartbeatAttempt{ID: state.ID, TeamID: state.TeamID, AgentID: state.AgentID,
			ProfileKey: state.ProfileKey, TaskID: state.TaskID, RunID: run.ID, Tag: "finite-leader-" + state.ID,
			Status: status, Phase: "finite_leader_owner_observed", StartedAt: startedAt, EndedAt: run.EndedAt, Error: run.Error})
	}
	if e.teamExecStore != nil {
		e.teamExecStore.SetRunningRunID(state.TeamID, state.AgentID, run.ID)
	}
	started, _ := time.Parse(time.RFC3339Nano, state.CreatedAt)
	if e.runRegistry != nil {
		if IsTerminalStatus(run.Status) {
			e.runRegistry.Complete(state.TeamID, state.AgentID, IsFailedStatus(run.Status), run.Error)
		} else {
			e.runRegistry.Register(state.TeamID, state.AgentID, run.ID, started, nil)
		}
	}
	if IsTerminalStatus(run.Status) {
		f.Queue.OnComplete(state.TeamID, state.AgentID)
	}
	return nil
}
