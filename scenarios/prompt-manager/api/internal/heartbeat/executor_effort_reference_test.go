package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

func finiteEffortCoordinatorFixture(t *testing.T) (*store.FileTeamStore, *store.FileAgentStore, store.RelationStore, string) {
	t.Helper()
	fileStore := newFileStore(t, paths.RootsForTest(t))
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relations := fileStore.Relations()
	ctx := context.Background()
	team := newIndependentTestTeam("team-1", "Team One")
	team.Purpose = teamconfig.PurposeDelivery
	team.PurposeSet = true
	team.Lifetime = teamconfig.LifetimeFinite
	team.LifetimeSet = true
	team.EffortRefs = []string{"effort:delivery"}
	team.Runtime.Mode = "single-process"
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "agent-1"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	team.Coordination.MessagingMode = teamconfig.MessagingModeInSession
	team.Execution.QueuePolicy = teamconfig.QueuePolicySerialized
	team.Execution.MaxConcurrentRuns = 1
	if err := teamStore.Create(ctx, team); err != nil {
		t.Fatal(err)
	}
	if err := agentStore.Create(ctx, &store.Agent{ID: "agent-1", DisplayName: "Agent One"}); err != nil {
		t.Fatal(err)
	}
	if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	profile, err := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true, ProfileKey: profile}); err != nil {
		t.Fatal(err)
	}
	return teamStore, agentStore, relations, profile
}

func TestConfiguredFiniteEffortLeaderRefusesUnconfirmedIdentityBeforeTaskOrRun(t *testing.T) {
	tests := []struct {
		name    string
		profile func(string) string
		mutate  func(*store.FileTeamStore, store.RelationStore, *mockAgentClient)
	}{
		{name: "profile mismatch", profile: func(profile string) string { return profile + "-wrong" }},
		{name: "inactive member", mutate: func(_ *store.FileTeamStore, relations store.RelationStore, _ *mockAgentClient) {
			if err := relations.SetTeamMember(context.Background(), &store.TeamMemberRelation{TeamID: "team-1", AgentID: "agent-1", Status: "inactive"}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "enrollment unavailable", mutate: func(_ *store.FileTeamStore, _ store.RelationStore, client *mockAgentClient) {
			client.effortEnrollmentErr = errors.New("owner read unavailable")
		}},
		{name: "withdrawn enrollment", mutate: func(_ *store.FileTeamStore, _ store.RelationStore, client *mockAgentClient) {
			client.effortEnrollment.Withdrawn = true
		}},
		{name: "missing target revision", mutate: func(_ *store.FileTeamStore, _ store.RelationStore, client *mockAgentClient) {
			client.effortEnrollment.TargetRevision = ""
		}},
		{name: "invalid team purpose", mutate: func(teamStore *store.FileTeamStore, _ store.RelationStore, _ *mockAgentClient) {
			team, err := teamStore.Get(context.Background(), "team-1")
			if err != nil {
				t.Fatal(err)
			}
			team.Purpose = teamconfig.PurposeSupervision
			team.PurposeSet = true
			if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing declared leader", mutate: func(teamStore *store.FileTeamStore, _ store.RelationStore, _ *mockAgentClient) {
			team, err := teamStore.Get(context.Background(), "team-1")
			if err != nil {
				t.Fatal(err)
			}
			team.Coordination.LeadAgentID = ""
			team.Runtime.Mode = teamconfig.RuntimeModeMultiProcess
			team.Coordination, err = teamconfig.BuildCoordinationPreset(teamconfig.CoordinationPatternIndependent, team.Runtime.Mode, "")
			if err != nil {
				t.Fatal(err)
			}
			if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			teamStore, agentStore, relations, profile := finiteEffortCoordinatorFixture(t)
			client := newMockAgentClient().WithEffortEnrollment(&domainpb.EffortEnrollment{EffortRef: "effort:delivery", TargetRevision: "accepted-1"})
			if tc.mutate != nil {
				tc.mutate(teamStore, relations, client)
			}
			if tc.profile != nil {
				profile = tc.profile(profile)
			}
			executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
			if _, err := executor.Execute(context.Background(), "team-1", "agent-1", profile); err == nil {
				t.Fatal("invalid configured finite coordinator identity unexpectedly dispatched")
			}
			if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
				t.Fatalf("invalid identity produced task/run effects: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
			}
		})
	}
}

func TestFiniteEffortNonLeadMemberDispatchesWithoutOrchestratorReference(t *testing.T) {
	teamStore, agentStore, relations, profile := finiteEffortCoordinatorFixture(t)
	ctx := context.Background()
	if err := agentStore.Create(ctx, &store.Agent{ID: "reviewer-1", DisplayName: "Acceptance Reviewer"}); err != nil {
		t.Fatal(err)
	}
	if err := relations.SetTeamMember(ctx, &store.TeamMemberRelation{TeamID: "team-1", AgentID: "reviewer-1", Status: store.MemberStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "reviewer-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "reviewer-1", Schedule: "0 * * * *", Enabled: true, ProfileKey: profile}); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().
		WithCreateRunResponse(&Run{ID: "review-run", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "review-run", Status: "RUN_STATUS_COMPLETE"})
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	result, err := executor.Execute(ctx, "team-1", "reviewer-1", profile)
	if err != nil {
		t.Fatalf("Execute non-lead reviewer: %v", err)
	}
	if result.RunID != "review-run" {
		t.Fatalf("run id=%q, want review-run", result.RunID)
	}
	if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 1 {
		t.Fatalf("reviewer task/run calls=%d/%d, want 1/1", len(client.createTaskCalls), len(client.createRunCalls))
	}
	if got := client.createRunCalls[0].WorkReferences; len(got) != 0 {
		t.Fatalf("non-lead reviewer received orchestrator work references: %+v", got)
	}
}

func TestFiniteEffortLeadReclaimsExactPriorOwnerBeforeFreshDispatch(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		status                string
		unavailable           bool
		enrollmentUnavailable bool
		mismatch              bool
		fresh                 bool
		currentRevision       string
		ownerRevision         string
		ownerProfileKey       string
		ownerProfileID        string
	}{
		{name: "resumed owner with historical revision and profile", status: "RUN_STATUS_RUNNING", currentRevision: "accepted-2", ownerRevision: "accepted-1", ownerProfileKey: "historical-profile", ownerProfileID: "historical-profile-id"},
		{name: "parked owner", status: "RUN_STATUS_PARKED"},
		{name: "owner read unavailable", unavailable: true},
		{name: "enrollment read unavailable retains old owner", enrollmentUnavailable: true},
		{name: "profile mismatch", status: "RUN_STATUS_RUNNING", mismatch: true},
		{name: "terminal owner permits fresh dispatch after revision and profile change", status: "RUN_STATUS_COMPLETE", fresh: true, currentRevision: "accepted-2", ownerRevision: "accepted-1", ownerProfileKey: "historical-profile", ownerProfileID: "historical-profile-id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			teamStore, agentStore, _, profileKey := finiteEffortCoordinatorFixture(t)
			ctx := context.Background()
			const priorRunID = "run-prior"
			const priorTaskID = "task-prior"
			if tc.currentRevision == "" {
				tc.currentRevision = "accepted-1"
			}
			if tc.ownerRevision == "" {
				tc.ownerRevision = "accepted-1"
			}
			if tc.ownerProfileKey == "" {
				tc.ownerProfileKey = profileKey
			}
			if tc.ownerProfileID == "" {
				tc.ownerProfileID = "resolved-profile"
			}
			config, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
			if err != nil {
				t.Fatal(err)
			}
			config.LastExecution = &store.HeartbeatExecResult{RunID: priorRunID, Status: store.HeartbeatStatusRunning, StartedAt: "2026-10-19T12:00:00Z"}
			if err := teamStore.SetHeartbeatConfig(ctx, "team-1", "agent-1", config); err != nil {
				t.Fatal(err)
			}
			if err := teamStore.AppendHeartbeatAttempt(ctx, "team-1", &store.HeartbeatAttempt{
				TeamID: "team-1", AgentID: "agent-1", ProfileKey: tc.ownerProfileKey,
				TaskID: priorTaskID, RunID: priorRunID, Status: store.HeartbeatStatusRunning,
				Phase: "run_started", StartedAt: "2026-10-19T12:00:00Z",
			}); err != nil {
				t.Fatal(err)
			}
			workRef, err := protojson.Marshal(&eventpb.WorkReference{
				Kind: "effort", Id: "effort:delivery", Revision: tc.ownerRevision,
				Relationship: "orchestrator", Verified: true,
				Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
				State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE,
			})
			if err != nil {
				t.Fatal(err)
			}
			runRefs, err := json.Marshal([]json.RawMessage{workRef})
			if err != nil {
				t.Fatal(err)
			}
			run := &Run{ID: priorRunID, TaskID: priorTaskID, ProfileID: tc.ownerProfileID, Status: tc.status, StartedAt: "2026-10-19T12:00:00Z", WorkReferences: runRefs}
			client := newMockAgentClient().
				WithEffortEnrollment(&domainpb.EffortEnrollment{EffortRef: "effort:delivery", TargetRevision: tc.currentRevision}).
				WithResolvedProfile(&AgentProfile{Id: tc.ownerProfileID, ProfileKey: tc.ownerProfileKey}).
				WithGetRunResponse(priorRunID, run).
				WithWaitRunError(errors.New("keep recovered owner observable"))
			if tc.unavailable {
				client.getRunErr = errors.New("AM unavailable")
			}
			if tc.enrollmentUnavailable {
				client.effortEnrollmentErr = errors.New("enrollment temporarily unavailable")
			}
			if tc.mismatch {
				run.ProfileID = "different-profile"
			}
			if tc.fresh {
				client.WithCreateRunResponse(&Run{ID: "run-fresh", Status: "RUN_STATUS_RUNNING"})
			}
			dir := t.TempDir()
			executor := newTestExecutor(t, teamStore, agentStore, client, dir, nil, nil)
			execStore := NewTeamExecutionStore(teamStore, executor, dir, client)
			t.Cleanup(execStore.Shutdown)
			executor.SetTeamExecStore(execStore)
			if _, err := execStore.Enqueue(ctx, "team-1", "agent-1", profileKey); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
			if tc.fresh {
				waitFor(t, 5*time.Second, func() bool { return len(client.createRunCalls) == 1 }, "terminal prior run to permit fresh dispatch")
				if len(client.createTaskCalls) != 1 || len(client.createRunCalls) != 1 || client.createRunCalls[0].TaskID == priorTaskID {
					t.Fatalf("terminal prior owner did not permit exactly one fresh task/run: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
				}
				return
			}
			waitFor(t, 5*time.Second, func() bool {
				data, readErr := os.ReadFile(filepath.Join(dir, "team-queue-team-1.json"))
				if readErr != nil {
					return false
				}
				var persisted persistedTeamQueue
				if json.Unmarshal(data, &persisted) != nil || len(persisted.Running) != 1 {
					return false
				}
				return persisted.Running[0].RunID == priorRunID
			}, "exact prior owner to be retained")
			if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
				t.Fatalf("prior owner reconciliation created competing work: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
			}
			persisted := readQueueFile(t, dir, "team-1")
			if len(persisted.Running) != 1 || persisted.Running[0].RunID != priorRunID {
				t.Fatalf("prior owner obligation not retained with exact run id: %+v", persisted.Running)
			}
			if tc.enrollmentUnavailable {
				retained, err := teamStore.GetHeartbeatConfig(ctx, "team-1", "agent-1")
				if err != nil {
					t.Fatal(err)
				}
				if retained.LastExecution == nil || retained.LastExecution.RunID != priorRunID {
					t.Fatalf("enrollment failure overwrote prior LastExecution owner: %+v", retained.LastExecution)
				}
				if _, err := execStore.Enqueue(ctx, "team-1", "agent-1", profileKey); err == nil {
					t.Fatal("next tick was admitted while prior owner association remained unverified")
				}
				if len(client.createTaskCalls) != 0 || len(client.createRunCalls) != 0 {
					t.Fatalf("next tick created competing work: tasks=%d runs=%d", len(client.createTaskCalls), len(client.createRunCalls))
				}
			}
		})
	}
}

func TestExecuteFreezesAndReplaysCanonicalEffortReference(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	team, err := teamStore.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	team.EffortRefs = []string{"effort:delivery"}
	team.Purpose = teamconfig.PurposeDelivery
	team.PurposeSet = true
	team.Lifetime = teamconfig.LifetimeFinite
	team.LifetimeSet = true
	team.Runtime.Mode = "single-process"
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "agent-1"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	team.Coordination.MessagingMode = teamconfig.MessagingModeInSession
	team.Execution.QueuePolicy = teamconfig.QueuePolicySerialized
	team.Execution.MaxConcurrentRuns = 1
	if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
		t.Fatal(err)
	}
	profile, err := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true, ProfileKey: profile}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	base := newMockAgentClient().
		WithEffortEnrollment(&domainpb.EffortEnrollment{EffortRef: "effort:delivery", TargetRevision: "accepted-2026-10-19"}).
		WithCreateRunResponse(&Run{ID: "run-effort-replay", Status: "RUN_STATUS_RUNNING", StartedAt: "2026-10-19T12:00:00Z"}).
		WithEchoRunIdentity()
	client := &lostTaskCreateResponseClient{mockAgentClient: base}
	client.beforeTaskPost = func(task *Task) error {
		data, err := os.ReadFile(filepath.Join(dir, "team-queue-team-1.json"))
		if err != nil {
			return err
		}
		var persisted persistedTeamQueue
		if err := json.Unmarshal(data, &persisted); err != nil {
			return err
		}
		if len(persisted.Running) != 1 {
			return fmt.Errorf("want one frozen dispatch, got %d", len(persisted.Running))
		}
		var frozen CreateRunRequest
		if err := json.Unmarshal([]byte(persisted.Running[0].CreateRunRequestSnapshot), &frozen); err != nil {
			return err
		}
		if frozen.TaskID != task.ID || len(frozen.WorkReferences) != 1 || frozen.WorkReferences[0].GetId() != "effort:delivery" || frozen.WorkReferences[0].GetRevision() != "accepted-2026-10-19" {
			return fmt.Errorf("frozen request lost canonical effort identity: %+v", frozen)
		}
		return nil
	}
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	execStore := NewTeamExecutionStore(teamStore, executor, dir, client)
	t.Cleanup(execStore.Shutdown)
	executor.SetTeamExecStore(execStore)
	NewScheduler(executor, client, teamStore, execStore).executeHeartbeat(context.Background(), "team-1", "agent-1")
	waitFor(t, 5*time.Second, func() bool {
		status := execStore.Status("team-1")
		return len(status.UncertainAgentIDs) == 1 && status.UncertainAgentIDs[0] == "agent-1"
	}, "accepted task with lost response to remain uncertain")
	execStore.Shutdown()
	if err := client.gateError(); err != nil {
		t.Fatal(err)
	}
	persisted := readQueueFile(t, dir, "team-1")
	if len(persisted.Running) != 1 || persisted.Running[0].CreateRunRequestSnapshot == "" {
		t.Fatalf("expected frozen run request, got %+v", persisted.Running)
	}

	restartedExecutor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	restartedStore := NewTeamExecutionStore(teamStore, restartedExecutor, dir, client)
	t.Cleanup(restartedStore.Shutdown)
	restartedExecutor.SetTeamExecStore(restartedStore)
	restartedStore.Recover(context.Background())
	NewScheduler(restartedExecutor, client, teamStore, restartedStore).executeHeartbeat(context.Background(), "team-1", "agent-1")
	if len(base.createRunCalls) != 1 {
		t.Fatalf("CreateRun calls=%d, want one replay", len(base.createRunCalls))
	}
	actual, err := json.Marshal(base.createRunCalls[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != persisted.Running[0].CreateRunRequestSnapshot {
		t.Fatalf("replay changed frozen work reference: got %s want %s", actual, persisted.Running[0].CreateRunRequestSnapshot)
	}
	if base.createRunCalls[0].WorkReferences[0].GetRevision() != "accepted-2026-10-19" {
		t.Fatalf("replayed revision=%q", base.createRunCalls[0].WorkReferences[0].GetRevision())
	}
}
