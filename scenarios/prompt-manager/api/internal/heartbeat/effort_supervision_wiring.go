package heartbeat

import (
	"context"
	"fmt"
	"time"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

// WireStandingSupervisor shares the canonical queue, context, resource profile
// and engagement controls. It does not enable any member or policy.
func WireStandingSupervisor(owner EffortSupervisionOwner, executor *Executor, queue *TeamExecutionStore, scheduler *Scheduler, control *HeartbeatControlStore, runtimeRoot string) *StandingSupervisor {
	s := &StandingSupervisor{
		Owner: owner, Agent: executor.agentClient, Queue: queue,
		State: FileSupervisionStateStore{Root: runtimeRoot}, Root: executor.vrooliRoot,
		Prompt:             executor.BuildSupervisionPrompt,
		Record:             executor.recordSupervisionExecution,
		DispatchCredential: resolveSupervisorDispatchCredential,
	}
	s.Config = func(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, error) {
		team, err := executor.teamStore.Get(ctx, teamID)
		if err != nil {
			return nil, err
		}
		if err := validateTeamEnabled(team); err != nil {
			return nil, err
		}
		if team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed || team.Coordination.LeadAgentID != agentID ||
			team.Execution.QueuePolicy != teamconfig.QueuePolicySerialized || team.Execution.MaxConcurrentRuns != 1 {
			return nil, fmt.Errorf("standing supervision requires the selected leader of a serialized team with one concurrent run")
		}
		members, err := executor.teamStore.GetMembers(ctx, teamID)
		if err != nil {
			return nil, err
		}
		active := false
		for _, member := range members {
			if member.AgentID == agentID && (member.Status == "" || member.Status == store.MemberStatusActive) {
				active = true
			}
		}
		if !active || team.OperatingContract == nil {
			return nil, fmt.Errorf("standing supervision requires an active leader and operating contract")
		}
		if _, ok := team.OperatingContract.Members[agentID]; !ok {
			return nil, fmt.Errorf("standing supervision leader is missing its member contract")
		}
		if control != nil {
			if _, err := control.AllowStart(ctx, teamID); err != nil {
				return nil, err
			}
		}
		cfg, err := executor.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
		if err != nil || cfg == nil {
			return cfg, err
		}
		if cfg.ProfileKey == "" {
			cfg.ProfileKey, err = DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
		}
		return cfg, err
	}
	executor.EffortSupervisor = s
	scheduler.SetEffortSupervisor(s)
	return s
}

// recordSupervisionExecution projects exact owner observations into the existing
// heartbeat history. Repeated recovery reads do not append duplicate attempts.
func (e *Executor) recordSupervisionExecution(ctx context.Context, teamID, agentID string, wake *SupervisionWake, run *Run) error {
	cfg, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("supervision heartbeat config missing during run observation")
	}
	status, phase := store.HeartbeatStatusRunning, "supervision_run_created"
	ended := ""
	if IsTerminalStatus(run.Status) {
		status, phase = store.HeartbeatStatusCompleted, "supervision_run_terminal"
		if IsFailedStatus(run.Status) {
			status = store.HeartbeatStatusFailed
		}
		if IsCancelledStatus(run.Status) {
			status = store.HeartbeatStatusCancelled
		}
		ended = run.EndedAt
		if ended == "" {
			ended = time.Now().UTC().Format(time.RFC3339)
		}
	}
	if cfg.LastExecution != nil && cfg.LastExecution.RunID == run.ID && cfg.LastExecution.Status == status {
		return nil
	}
	cfg.LastExecution = &store.HeartbeatExecResult{RunID: run.ID, StartedAt: wake.CreatedAt.Format(time.RFC3339), EndedAt: ended, Status: status, Error: run.Error}
	if ended != "" {
		cfg.RecordHeartbeatStatus(status)
	}
	if err := e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, cfg); err != nil {
		return err
	}
	e.appendAttempt(ctx, &store.HeartbeatAttempt{
		ID: wake.ID, TeamID: teamID, AgentID: agentID, ProfileKey: wake.ProfileKey,
		TaskID: wake.TaskID, RunID: run.ID, Tag: "supervision-" + wake.ID, Status: status, Phase: phase,
		StartedAt: wake.CreatedAt.Format(time.RFC3339), EndedAt: ended, Error: run.Error,
	})
	if e.runRegistry != nil {
		if ended != "" {
			e.runRegistry.Complete(teamID, agentID, status == store.HeartbeatStatusFailed, run.Error)
		} else {
			e.runRegistry.Register(teamID, agentID, run.ID, wake.CreatedAt, nil)
		}
	}
	if ended == "" && e.teamExecStore != nil {
		e.teamExecStore.SetRunningRunID(teamID, agentID, run.ID)
	}
	return nil
}
