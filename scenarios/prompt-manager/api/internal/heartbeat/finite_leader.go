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

// finiteLeaderGuidance is the durable leader doctrine appended to every finite
// coordinator prompt. It grants no authority; it states how to hold a finite
// effort across the owner boundary.
const finiteLeaderGuidance = "\n\nFinite effort coordinator guidance:\n" +
	"- Prefer durable owner reads (files, journals, owner APIs) over re-deriving state; never invent a private outcome ledger.\n" +
	"- Do independent, verifiable work before waiting; never hold a child wait inside the run.\n" +
	"- When delegated children are pending, obtain the verified parent ID with `agent-manager run identity --json`, create durable child lineage with `agent-manager run create --parent-run-id`, and create one Agent Manager cohort watch containing the exact child run IDs and this parent run ID.\n" +
	"- Park or checkpoint the parent on that watch with `agent-manager run park <parent-run-id> --producer supervision --key <watch-id> --deadline-unix <watch-deadline>`; parking must be an actual owner operation, not a statement in the handoff.\n" +
	"- A supervision wake resumes this same parent run with terminal child evidence. Reconcile the exact child IDs, cancel the completed watch, and select the next bounded action; never create a replacement coordinator run.\n" +
	"- If the watch deadline wakes the parent, classify the children as active, terminal, missing, or uncertain. Do not retry an uncertain child or invent a fresh grant.\n" +
	"- Before ending a pass, write the final handoff: changed, verified, remaining, unverified and the exact next action.\n" +
	"- Accept the effort only through the explicit completion receipt; a terminal run is not effort acceptance. After acceptance, retire the finite binding so future heartbeat ticks are fenced."

// finiteMemberGuidance describes how a finite leader delegates to its members.
// It grants no authority; it states the planner/worker contract and the exact
// identity a delegated assignment must carry.
const finiteMemberGuidance = "\n\nFinite member delegation guidance:\n" +
	"- Finite members act as planners or workers, never permanent roles: a subplanner owns one narrower outcome, a worker returns one retained handoff to its assigning parent.\n" +
	"- Every delegated assignment names its parent handoff identity, the task/attempt identity, and the exact model/context reference; descendants do not inherit the leader's model by default.\n" +
	"- Independent review is bounded work assigned to a distinct member, not a permanent role and not the author.\n" +
	"- A member handoff reports requirement coverage, changed paths, evidence, remaining outcomes and usage; a successful finite child stays terminal.\n" +
	"- A member never reopens an accepted completion or replaces an owner identity; it returns the gap to the leader."

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
		// Completion is terminal until an explicit reopen. A tick may still
		// reconcile an already-dispatched run so its accounting settles.
		if state.Completed != nil && !state.DispatchStarted {
			return store.ErrFiniteLeaderCompleted
		}
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
		// Bounded owner lookup after a lost admission response. Truncated or
		// conflicting results never justify another CreateRun; an authoritative
		// zero-run result permits only the original idempotency-key replay.
		var runs *ListRunsResponse
		runs, err = f.Executor.agentClient.ListRuns(ctx, ListRunsOptions{TaskID: state.TaskID, Limit: 2})
		if err == nil {
			if runs == nil || runs.HasMore || runs.Total > 1 {
				return nil, fmt.Errorf("finite leader dispatch uncertain; retain task %s", state.TaskID)
			}
			if runs.Total == 0 && len(runs.Runs) == 0 {
				// The task exists but the original run admission was definitively
				// absent. Replay the exact idempotency key; this is reconciliation,
				// not a replacement dispatch.
				binding := state.Binding
				tag := "finite-leader-" + state.ID
				attributionKey, attributionValue := buildHeartbeatAttributionEnv(state.TeamID, state.AgentID)
				run, err = f.Executor.agentClient.CreateRun(ctx, &CreateRunRequest{
					TaskID: state.TaskID, IdempotencyKey: tag, Tag: &tag,
					ProfileRef:  &ProfileRef{ProfileKey: state.ProfileKey},
					Environment: map[string]string{attributionKey: attributionValue},
					WorkReferences: []*eventpb.WorkReference{{
						Kind: "effort", Id: binding.EffortRef,
						Revision: binding.AcceptedRevision, Relationship: "orchestrator", Verified: true,
						Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
						State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
					}},
				})
				if err != nil {
					return nil, fmt.Errorf("finite leader dispatch remains uncertain after same-key replay: %w", err)
				}
			} else if len(runs.Runs) != 1 || runs.Runs[0] == nil {
				return nil, fmt.Errorf("finite leader dispatch uncertain; retain task %s", state.TaskID)
			} else {
				run = runs.Runs[0]
			}
			if run == nil || run.TaskID != state.TaskID || run.Tag != "finite-leader-"+state.ID {
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
		// A completed effort refuses every later scheduled or manual start; an
		// already-dispatched run is still reconciled rather than replaced.
		if current.Completed != nil && !current.DispatchStarted {
			return store.ErrFiniteLeaderCompleted
		}
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
			"\nPreserve the accepted destination, exclusions and human input boundaries. Use existing owner operations and retain named waits. Run completion is not effort acceptance." +
			finiteLeaderGuidance + finiteMemberGuidance
		current.TaskStarted, current.Status = true, "task-uncertain"
		if err := save(); err != nil {
			return err
		}
		task, err := f.Executor.agentClient.CreateTask(ctx, &Task{
			Title:       "Finite effort leader: " + teamID + "/" + agentID,
			Description: prompt, ScopePath: f.Executor.vrooliRoot, ProjectRoot: f.Executor.vrooliRoot,
		})
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
		run, err = f.Executor.agentClient.CreateRun(ctx, &CreateRunRequest{
			TaskID: task.ID, IdempotencyKey: tag, Tag: &tag,
			ProfileRef: &ProfileRef{ProfileKey: current.ProfileKey}, Environment: map[string]string{key: value},
			WorkReferences: []*eventpb.WorkReference{{
				Kind: "effort", Id: current.Binding.EffortRef, Revision: current.Binding.AcceptedRevision,
				Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
				State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
			}},
		})
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
		e.appendAttempt(ctx, &store.HeartbeatAttempt{
			ID: state.ID, TeamID: state.TeamID, AgentID: state.AgentID,
			ProfileKey: state.ProfileKey, TaskID: state.TaskID, RunID: run.ID, Tag: "finite-leader-" + state.ID,
			Status: status, Phase: "finite_leader_owner_observed", StartedAt: startedAt, EndedAt: run.EndedAt, Error: run.Error,
		})
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

// Complete records the retained completion receipt for the accepted revision.
// It is idempotent for a repeated exact completion and never reopens work.
func (f *FiniteLeaderRuntime) Complete(ctx context.Context, teamID, agentID, revision, evidenceRef string) (*store.FiniteLeaderCompletion, bool, error) {
	return f.Executor.teamStore.CompleteFiniteLeader(ctx, teamID, agentID, revision, evidenceRef)
}

// Restart explicitly recovers a terminal dispatched owner run. The exact
// Agent Manager run is reread before the persistent reservation is cleared;
// an unavailable or nonterminal owner never authorizes a replacement.
func (f *FiniteLeaderRuntime) Restart(ctx context.Context, teamID, agentID, revision, evidenceRef string) error {
	state, err := f.Executor.teamStore.ReadFiniteLeader(ctx, teamID, agentID)
	if err != nil {
		return err
	}
	if state == nil || !state.DispatchStarted || state.RunID == "" {
		return fmt.Errorf("finite leader restart requires a dispatched owner run")
	}
	run, err := f.Executor.agentClient.GetRun(ctx, state.RunID)
	if err != nil {
		return fmt.Errorf("read exact finite leader owner run: %w", err)
	}
	if run == nil || run.ID != state.RunID || !IsTerminalStatus(run.Status) {
		return fmt.Errorf("finite leader owner run is not terminal; replacement dispatch is refused")
	}
	return f.Executor.teamStore.RestartFiniteLeader(ctx, teamID, agentID, revision, evidenceRef, run.Status)
}

// Reopen is the explicit authorized counter-operation to completion.
func (f *FiniteLeaderRuntime) Reopen(ctx context.Context, teamID, agentID, revision, evidenceRef string) error {
	return f.Executor.teamStore.ReopenFiniteLeader(ctx, teamID, agentID, revision, evidenceRef)
}
