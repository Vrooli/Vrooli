package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/modules"
	"agent-manager/internal/supervision"
	"context"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	coreidentity "github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCommissionedWorkerDescendantsRetainEffortCustody(t *testing.T) {
	ref := &eventpb.WorkReference{Kind: "effort", Id: "effort:delivery", Relationship: "worker", Revision: "product-1", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
	parent := &domain.Run{WorkReferences: []*eventpb.WorkReference{ref}, ResolvedConfig: &domain.RunConfig{DeliveryCommissions: map[string]string{ref.Id: "commission-1"}}}
	inherited, err := inheritParentEffortReferences(parent, nil)
	if err != nil || len(inherited) != 1 || inherited[0].Id != ref.Id || inherited[0].Relationship != "worker" || inherited[0].Revision != ref.Revision {
		t.Fatalf("delegation lost commission custody: %+v %v", inherited, err)
	}
	if _, err := inheritParentEffortReferences(parent, []*eventpb.WorkReference{{Kind: "effort", Id: ref.Id, Relationship: "orchestrator"}}); err == nil {
		t.Fatal("worker descendant promoted itself to orchestrator")
	}
	parent.ResolvedConfig = nil
	ordinary, err := inheritParentEffortReferences(parent, nil)
	if err != nil || len(ordinary) != 0 {
		t.Fatal("ordinary worker semantics changed", err)
	}
}

func TestDeliveryActualNativeWorkersOverlapWithinBound(t *testing.T) {
	ctx := context.Background()
	o := newDeclarationOrchestrator(t)
	newCurrentModelPolicyFixtureOption(t)(o)
	o.identitySecret = []byte("disposable-keyring-lifecycle-secret")
	o.runStateRoot = t.TempDir()
	root := t.TempDir()
	o.config.DefaultProjectRoot = root
	o.config.RequireSandboxByDefault = false
	defer o.dispatcher.Close()
	type launched struct {
		request runner.ExecuteRequest
		token   string
	}
	starts := make(chan launched, 8)
	release := make(chan struct{})
	var released sync.Once
	finish := func() { released.Do(func() { close(release) }) }
	defer finish()
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.ExecuteFunc = func(ctx context.Context, req runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		token := req.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]
		select {
		case starts <- launched{req, token}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "synthetic only"}}, nil
	}
	o.runners = runner.NewRegistry()
	if err := o.runners.Register(&launchInfoStub{MockRunner: mock, args: []string{"--synthetic-in-process"}, runtimeVersion: "synthetic-in-process/1"}); err != nil {
		t.Fatal(err)
	}
	expiry := time.Now().Add(time.Hour)
	owner := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "fixture-owner", Scopes: []string{"agent-manager:write", "agent-manager:read", "agent-manager:orchestrate"}, ExpiresAt: expiry}
	ownerCtx := coreidentity.WithPrincipal(ctx, owner)
	profile, err := o.CreateProfile(ctx, &domain.AgentProfile{Name: "synthetic parent child", ProfileKey: "keyring-inprocess", RoleRef: "code.default", DeclaredScopes: owner.Scopes, MaxTurns: 5, Timeout: time.Minute, NetworkAccess: domain.NetworkAccessNone, SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}})
	if err != nil {
		t.Fatal(err)
	}
	newTask := func(title string) *domain.Task {
		t.Helper()
		created, err := o.CreateTask(ctx, &domain.Task{Title: title, Description: "synthetic in-process only", ScopePath: root, ProjectRoot: root})
		if err != nil {
			t.Fatal(err)
		}
		return created
	}
	parentTask := newTask("isolated native parent")

	wait := func() launched {
		select {
		case got := <-starts:
			return got
		case <-time.After(10 * time.Second):
			for _, key := range []string{"keyring-real-parent", "keyring-real-child"} {
				if saved, err := o.runs.GetByIdempotencyKey(ctx, key); err == nil && saved != nil {
					t.Logf("synthetic run %s status=%s failure=%s", key, saved.Status, saved.ErrorMsg)
				}
			}
			t.Fatal("synthetic native lifecycle did not launch")
			return launched{}
		}
	}

	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "delivery.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = coredb.EnsureSchemas(ctx, db, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	service := supervision.NewEffortService(supervision.NewRepository(db), nil, supervision.EffortDiscoveryConfig{})
	service.SetRunRegistry(o.runs)
	o.SetDeliveryEfforts(service)
	_, err = service.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:parallel", TargetRevision: "selected-1", Delivery: &pb.EffortDelivery{CommissionId: "disposable-native", BudgetSeconds: 3600, MaxWorkers: 2, State: "running"}}, IdempotencyKey: "native-start"}, supervision.EffortActor{ID: "fixture-owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	ref := &eventpb.WorkReference{Kind: "effort", Id: "effort:parallel", Revision: "selected-1", Relationship: "orchestrator", Verified: true, Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC, State: eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE}
	parent, err := o.CreateRun(ownerCtx, CreateRunRequest{TaskID: parentTask.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", OwnerToken: "disposable-native-owner-fixture", IdempotencyKey: "native-delivery-parent", WorkReferences: []*eventpb.WorkReference{ref}})
	if err != nil {
		t.Fatal(err)
	}
	launch := wait()
	children := []*domain.Run{}
	for i := 0; i < 3; i++ {
		task := newTask("independent disposable surface")
		child, err := o.CreateRun(ctx, CreateRunRequest{TaskID: task.ID, AgentProfileID: &profile.ID, PreferredRunner: "claude-code", ParentRunID: &parent.ID, RunIdentityToken: launch.token, RequestedScopes: []string{"agent-manager:read"}, IdempotencyKey: task.ID.String()})
		if i == 2 {
			if err == nil {
				t.Fatal("third native worker exceeded commission limit")
			}
			if saved, _ := o.runs.GetByIdempotencyKey(ctx, task.ID.String()); saved != nil {
				t.Fatal("rejected child persisted")
			}
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		wait()
		children = append(children, child)
	}
	for _, child := range children {
		current, err := o.runs.Get(ctx, child.ID)
		if err != nil || current.Status != domain.RunStatusRunning || current.ResolvedConfig.DeliveryCommissions[ref.Id] != "disposable-native" {
			t.Fatalf("worker did not overlap under native commission: %+v %v", current, err)
		}
	}
	current, err := o.runs.Get(ctx, parent.ID)
	if err != nil || current.Status != domain.RunStatusRunning {
		t.Fatal("parent did not retain independent orchestration", err)
	}
}
