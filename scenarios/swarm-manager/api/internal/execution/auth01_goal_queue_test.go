package execution

import (
	"context"
	"github.com/vrooli/api-core/owneridentity"
	"net/http"
	"path/filepath"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"
	"testing"
	"time"
)

type auth01GoalQueueCreator struct{}

func (auth01GoalQueueCreator) PrepareCreateRunCaller(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
func (auth01GoalQueueCreator) CreateGoalRun(context.Context, agentmanager.GoalRunRequest) (agentmanager.GoalRunResult, error) {
	panic("unexpected dispatch")
}

type auth01QueueStore struct {
	records []Record
	saves   int
}

func (s *auth01QueueStore) Load() ([]Record, error)     { return s.records, nil }
func (s *auth01QueueStore) Save(records []Record) error { s.records = records; s.saves++; return nil }
func TestAuth01RecoveredGoalRejectsBeforeOtherEffects(t *testing.T) {
	store := &auth01QueueStore{records: []Record{{ExecutionID: "fixture", Status: StatusPending, ExecutionMode: transitions.ExecutionModeGoal}}}
	s := &Service{store: store, goalRunCreator: auth01GoalQueueCreator{}}
	if _, err := s.Start(context.Background(), "fixture"); err == nil || store.saves != 0 {
		t.Fatal("recovered goal reached effects")
	}
	now := time.Now()
	expired, err := owneridentity.AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer old"}}, auth01GoalVerifier{now.Add(-time.Hour)}, now.Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	s.goalQueueCallers = map[string]context.Context{"fixture": expired}
	if _, err := s.Start(context.Background(), "fixture"); err == nil || store.saves != 0 {
		t.Fatal("expired goal reached effects")
	}
	if len(s.goalQueueCallers) != 0 {
		t.Fatal("expired proof retained")
	}
}

type auth01GoalVerifier struct{ expiry time.Time }

func (v auth01GoalVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: v.expiry}, nil
}
func TestAuth01GoalQueueProofRetainsExpiry(t *testing.T) {
	now := time.Now()
	expiry := now.Add(time.Hour)
	ctx, err := owneridentity.AuthorizeCreateRunCaller(context.Background(), http.Header{"Authorization": []string{"Bearer secret"}}, auth01GoalVerifier{expiry}, now)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := owneridentity.CarryCreateRunCaller(ctx, context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if owneridentity.RequireCreateRunCaller(queued, expiry) == nil {
		t.Fatal("expiry renewed")
	}
	if _, err := owneridentity.CarryCreateRunCaller(context.Background(), context.Background(), now); err == nil {
		t.Fatal("anonymous carry")
	}
}

type auth01PositiveGoalCreator struct{ proof string }

func (f *auth01PositiveGoalCreator) PrepareCreateRunCaller(ctx context.Context) (context.Context, error) {
	return owneridentity.AuthorizeBoundCreateRunCaller(ctx, auth01GoalVerifier{time.Now().Add(time.Hour)}, time.Now())
}
func (f *auth01PositiveGoalCreator) CreateGoalRun(ctx context.Context, _ agentmanager.GoalRunRequest) (agentmanager.GoalRunResult, error) {
	var err error
	f.proof, err = owneridentity.CreateRunAuthorization(ctx, time.Now())
	return agentmanager.GoalRunResult{RunID: "fixture-run", TaskID: "fixture-task"}, err
}
func TestAuth01GoalQueueThroughStartCarriesOriginalProof(t *testing.T) {
	root := t.TempDir()
	name := "auth01-goal"
	mustWriteBacklogItem(t, root, "execute", name, map[string]any{"name": name, "title": "Goal", "description": "bounded fixture", "status": "ready", "priority": 2, "tags": []string{}, "acceptance_allow": []string{"scenarios/swarm-manager/**"}, "execution_mode": "goal"})
	creator := &auth01PositiveGoalCreator{}
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer(), TransitionRegistry: testTransitionRegistry(t), GoalRunCreator: creator})
	ctx, cancel := context.WithCancel(owneridentity.WithCreateRunHeaders(context.Background(), http.Header{"Authorization": []string{"Bearer original-goal-fixture"}}))
	queued, err := service.QueueBacklog(ctx, CreateRequest{BacklogKind: "execute", BacklogName: name, Mode: ModeManual})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := service.Start(context.Background(), queued.ExecutionID); err != nil {
		t.Fatal(err)
	}
	if creator.proof != "Bearer original-goal-fixture" {
		t.Fatal("proof not carried through goal creation")
	}
}
