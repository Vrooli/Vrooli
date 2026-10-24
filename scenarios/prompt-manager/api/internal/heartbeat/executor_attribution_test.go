package heartbeat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

// TestExecute_PropagatesAttributionInCreateRunEnv asserts the spawner-side
// integration: when the heartbeat executor calls CreateRun, the request's
// Environment map carries VROOLI_PROMPT_MANAGER_ATTRIBUTION with a
// well-formed agent-member payload describing the spawned member's identity.
//
// This is the load-bearing test for the spawner half of the runtime
// attribution contract. It ensures no future refactor drops the env
// injection silently — every heartbeat-spawned agent's CLI must inherit
// attribution, otherwise the API would record those writes as
// operator-direct (the CLI's fallback when the env var is absent).
func TestExecute_PropagatesAttributionInCreateRunEnv(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	team, err := teamStore.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	team.EffortRefs = []string{"effort:test"}
	team.Purpose = teamconfig.PurposeDelivery
	team.PurposeSet = true
	team.Lifetime = teamconfig.LifetimeFinite
	team.LifetimeSet = true
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "agent-1"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
		t.Fatalf("set team effort identity: %v", err)
	}
	if err := teamStore.SetHeartbeatConfig(context.Background(), "team-1", "agent-1", &store.HeartbeatConfig{TeamID: "team-1", AgentID: "agent-1", Schedule: "0 * * * *", Enabled: true, ProfileKey: "test-profile"}); err != nil {
		t.Fatal(err)
	}

	mockClient := newMockAgentClient().
		WithEffortEnrollment(&domainpb.EffortEnrollment{Delivery: runningTestDelivery(), EffortRef: "effort:test", TargetRevision: "revision-7"}).
		WithCreateTaskResponse(&Task{ID: "task-100", Title: "test"}).
		WithCreateRunResponse(&Run{ID: "run-200", Status: "RUN_STATUS_RUNNING"}).
		WithWaitRunResponse(&Run{ID: "run-200", Status: "RUN_STATUS_COMPLETE"})

	registry := NewRunRegistry(t.TempDir())
	executor := newTestExecutor(t, teamStore, agentStore, mockClient, t.TempDir(), registry, nil)

	// Block on completion so this test doesn't race with the async waitForCompletion goroutine.
	var completeCalled sync.WaitGroup
	completeCalled.Add(1)
	executor.OnRunComplete = func(_, _, _ string) { completeCalled.Done() }

	if _, err := executor.Execute(context.Background(), "team-1", "agent-1", "test-profile"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(mockClient.createRunCalls) != 1 {
		t.Fatalf("expected 1 CreateRun call, got %d", len(mockClient.createRunCalls))
	}
	runReq := mockClient.createRunCalls[0]

	// Environment must be populated.
	if runReq.Environment == nil {
		t.Fatal("CreateRunRequest.Environment is nil — attribution propagation is broken")
	}
	encoded, ok := runReq.Environment["VROOLI_PROMPT_MANAGER_ATTRIBUTION"]
	if !ok {
		t.Fatalf("Environment missing VROOLI_PROMPT_MANAGER_ATTRIBUTION; keys = %v", keysOf(runReq.Environment))
	}
	if encoded == "" {
		t.Fatal("VROOLI_PROMPT_MANAGER_ATTRIBUTION value is empty")
	}

	// Decode and check structure end-to-end.
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var info store.AttributionInfo
	if err := json.Unmarshal(decoded, &info); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if info.Kind != store.KnowledgeKindAgentMember {
		t.Errorf("Kind = %q, want agent-member", info.Kind)
	}
	if info.SpawnOrigin != store.SpawnOriginHeartbeat {
		t.Errorf("SpawnOrigin = %q, want heartbeat", info.SpawnOrigin)
	}
	if info.MemberID == nil || *info.MemberID != "agent-1" {
		t.Errorf("MemberID = %v, want agent-1", info.MemberID)
	}
	if info.TeamID == nil || *info.TeamID != "team-1" {
		t.Errorf("TeamID = %v, want team-1", info.TeamID)
	}
	// run_id is intentionally null at construction time: agent-manager
	// assigns the run UUID after CreateRun returns, and the validator's
	// heartbeat-relax permits this (canon:
	// docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge).
	if info.RunID != nil {
		t.Errorf("RunID = %v, want nil", info.RunID)
	}
	if got := runReq.Environment[workloadKindEnv]; got != "scheduled" {
		t.Fatalf("workload kind = %q, want scheduled", got)
	}
	if got := runReq.Environment[workloadKeyEnv]; got != "effort:test" {
		t.Fatalf("workload key = %q, want effort:test", got)
	}
	if got := runReq.Environment[workloadInstanceEnv]; got == "" {
		t.Fatal("workload instance is missing")
	}
	if len(runReq.WorkReferences) != 1 {
		t.Fatalf("WorkReferences = %+v, want one canonical effort reference", runReq.WorkReferences)
	}
	work := runReq.WorkReferences[0]
	if work.GetKind() != "effort" || work.GetId() != "effort:test" || work.GetRevision() != "revision-7" || work.GetRelationship() != "orchestrator" || !work.GetVerified() || work.GetVisibility() != eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC || work.GetState() != eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE {
		t.Fatalf("WorkReference = %+v, want exact active public verified orchestrator enrollment", work)
	}

	// Drain the async-completion goroutine so t.TempDir cleanup doesn't race
	// with config writes.
	done := make(chan struct{})
	go func() {
		completeCalled.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("run-scoped completion callback not called within 5s — async waitForCompletion stuck")
	}
}

func TestOrchestratorWorkReferencesLeaveUnknownAndMismatchedIdentityUnlinked(t *testing.T) {
	teamStore, agentStore, _ := setupExecutorTestEnv(t)
	team, err := teamStore.Get(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	team.EffortRefs = []string{"effort:test"}
	team.Purpose = teamconfig.PurposeDelivery
	team.PurposeSet = true
	team.Lifetime = teamconfig.LifetimeFinite
	team.LifetimeSet = true
	team.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	team.Coordination.LeadAgentID = "agent-1"
	team.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	if err := teamStore.Update(context.Background(), "team-1", team); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient().WithEffortEnrollment(&domainpb.EffortEnrollment{Delivery: runningTestDelivery(), EffortRef: "effort:test", TargetRevision: "revision-7"})
	executor := newTestExecutor(t, teamStore, agentStore, client, t.TempDir(), nil, nil)
	config := &store.HeartbeatConfig{ProfileKey: "leader-profile"}

	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "leader-profile", config); err != nil || len(refs) != 1 {
		t.Fatalf("valid leader enrollment refs=%+v err=%v, want one", refs, err)
	}
	team.Coordination.LeadAgentID = "someone-else"
	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "leader-profile", config); err != nil || len(refs) != 0 {
		t.Fatalf("non-lead member should remain an ordinary heartbeat: refs=%+v err=%v", refs, err)
	}
	team.Coordination.LeadAgentID = "agent-1"
	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "other-profile", config); err == nil || len(refs) != 0 {
		t.Fatalf("mismatched configured profile linked: %+v", refs)
	}
	client.effortEnrollment = &domainpb.EffortEnrollment{Delivery: runningTestDelivery(), EffortRef: "effort:test", TargetRevision: "revision-7", Withdrawn: true}
	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "leader-profile", config); err == nil || len(refs) != 0 {
		t.Fatalf("withdrawn enrollment linked: %+v", refs)
	}
	client.effortEnrollment = nil
	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "leader-profile", config); err == nil || len(refs) != 0 {
		t.Fatalf("unavailable enrollment linked: %+v", refs)
	}
	team.Purpose = teamconfig.PurposeSupervision
	team.EffortRefs = nil
	if refs, err := executor.orchestratorWorkReferences(context.Background(), team, "agent-1", "leader-profile", config); err != nil || len(refs) != 0 {
		t.Fatalf("non-delivery finite heartbeat should remain unassociated: refs=%+v err=%v", refs, err)
	}
}

// keysOf is a tiny helper for failure messages — sorted-set output keeps
// test failures deterministic.
func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
