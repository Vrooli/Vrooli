package orchestration

import (
	"agent-manager/internal/adapters/database"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/supervision"
	"agent-manager/internal/workflowruntime"
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	watchpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionpb "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
	familyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families/families_v1connect"
	"google.golang.org/protobuf/proto"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/workflowcatalog"
	familypb "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
)

func TestFamilyCompilerPreservesOwnerParallelAndDependentWaves(t *testing.T) {
	family := &familypb.PlanFamily{FamilyId: "family-proof", Graph: &familypb.GraphRevision{Revision: 7}, Policy: &familypb.FamilyPolicy{MaximumParallelPlans: 2}}
	for _, id := range []string{"independent-a", "independent-b", "integration"} {
		family.Members = append(family.Members, &familypb.FamilyMember{PlanId: id, ExecutionId: "execution-" + id, State: familypb.MemberState_MEMBER_STATE_PENDING})
	}
	frontier := &familypb.GetFrontierResponse{Launchable: true, GraphRevision: 7, ProjectedBatches: []*familypb.FrontierBatch{{PlanIds: []string{"independent-a", "independent-b"}}, {PlanIds: []string{"integration"}}}}
	definition, input, err := compilePlanFamilyWorkflow(family, frontier, "policy-v1", "", "code.default")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(definition)
	parsed, err := workflowcatalog.Parse(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if domain.HasBlockingDiagnostic(parsed.Diagnostics) {
		t.Fatalf("invalid workflow: %v", parsed.Diagnostics)
	}
	if parsed.Definition.Budgets.MaxChildren != 3 || parsed.Definition.Budgets.MaxConcurrency != 2 {
		t.Fatal("owner concurrency was not preserved")
	}
	joins := 0
	children := 0
	for _, node := range parsed.Definition.Nodes {
		if node.Join != nil {
			joins++
		}
		if node.Run != nil {
			children++
			if node.Run.PromptRef == nil || node.Run.PromptRef.SkillID != "plan-manager-family-child" {
				t.Fatal("child judgment must be scenario-owned")
			}
		}
	}
	if children != 3 || joins != 2 {
		t.Fatalf("children=%d joins=%d", children, joins)
	}
	var decoded map[string]any
	if json.Unmarshal(input, &decoded) != nil || len(decoded["children"].(map[string]any)) != 3 {
		t.Fatal("missing child identities")
	}
	frontier.GraphRevision = 8
	if _, _, err = compilePlanFamilyWorkflow(family, frontier, "policy-v1", "", "code.default"); err == nil {
		t.Fatal("stale graph accepted")
	}
	frontier.GraphRevision = 7
	family.Members[0].State = familypb.MemberState_MEMBER_STATE_RUNNING
	if _, _, err = compilePlanFamilyWorkflow(family, frontier, "policy-v1", "", "code.default"); err == nil {
		t.Fatal("second executor accepted an occupied family")
	}
}

// The fake owners use the same typed client seam as production. Run state and
// workflow/watch stores survive rebuilding the parent orchestrator.
type familyProofOwner struct {
	familyconnect.FamiliesServiceClient
	family *familypb.PlanFamily
}

func (f *familyProofOwner) GetFamily(context.Context, *connect.Request[familypb.GetFamilyRequest]) (*connect.Response[familypb.GetFamilyResponse], error) {
	return connect.NewResponse(&familypb.GetFamilyResponse{Family: proto.Clone(f.family).(*familypb.PlanFamily)}), nil
}
func (f *familyProofOwner) PutMember(_ context.Context, r *connect.Request[familypb.PutMemberRequest]) (*connect.Response[familypb.PutMemberResponse], error) {
	if r.Msg.GetExpectedRevision() != f.family.GetRevision() {
		return nil, fmt.Errorf("stale family revision")
	}
	for i, m := range f.family.Members {
		if m.PlanId == r.Msg.Member.PlanId {
			f.family.Members[i] = proto.Clone(r.Msg.Member).(*familypb.FamilyMember)
			f.family.Revision++
			return connect.NewResponse(&familypb.PutMemberResponse{Family: proto.Clone(f.family).(*familypb.PlanFamily)}), nil
		}
	}
	return nil, fmt.Errorf("member missing")
}

type familyProofExecutions struct {
	executionconnect.ExecutionServiceClient
	accepted bool
}

func (f familyProofExecutions) GetStatus(_ context.Context, r *connect.Request[executionpb.GetStatusRequest]) (*connect.Response[executionpb.GetStatusResponse], error) {
	id := strings.TrimPrefix(r.Msg.ExecutionId, "execution-")
	return connect.NewResponse(&executionpb.GetStatusResponse{Execution: &executionpb.Execution{Id: r.Msg.ExecutionId, PlanId: id, Complete: f.accepted}}), nil
}

type admittedProofLauncher struct {
	*fakeRunLauncher
	o *Orchestrator
}

func (l admittedProofLauncher) StartFresh(ctx context.Context, r workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	if err := l.o.admitPlanFamilyChild(ctx, r); err != nil {
		return workflowruntime.ChildState{}, err
	}
	return l.fakeRunLauncher.StartFresh(ctx, r)
}

func TestFamilyExecutorDurableParallelAdmissionAndOwnerAcceptance(t *testing.T) {
	for _, mode := range []string{"accepted", "unaccepted", "cancelled"} {
		accepted := mode != "unaccepted"
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos := database.NewRepositories(db, logrus.New())
			family := &familypb.PlanFamily{FamilyId: uuid.NewString(), Revision: 1, Graph: &familypb.GraphRevision{Revision: 7}, Policy: &familypb.FamilyPolicy{MaximumParallelPlans: 2}}
			for _, id := range []string{"a", "b", "integration"} {
				family.Members = append(family.Members, &familypb.FamilyMember{PlanId: id, ExecutionId: "execution-" + id, State: familypb.MemberState_MEMBER_STATE_PENDING})
			}
			owner := &familyProofOwner{family: family}
			frontier := &familypb.GetFrontierResponse{Launchable: true, GraphRevision: 7, ProjectedBatches: []*familypb.FrontierBatch{{PlanIds: []string{"a", "b"}}, {PlanIds: []string{"integration"}}}}
			def, input, err := compilePlanFamilyWorkflow(family, frontier, "proof-policy", "", "code.default")
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(def)
			parsed, err := workflowcatalog.Parse(raw, nil)
			if err != nil || domain.HasBlockingDiagnostic(parsed.Diagnostics) {
				t.Fatalf("definition %v %v", parsed.Diagnostics, err)
			}
			// Prompt resolution is tested by the catalog tests; this proof controls runs.
			for i := range parsed.Definition.Nodes {
				if r := parsed.Definition.Nodes[i].Run; r != nil {
					r.PromptRef = nil
					r.PromptTemplate = "Complete the bound child: {{.child}}"
				}
			}
			revision := &domain.WorkflowRevision{ID: uuid.New(), Owner: "plan-manager", Key: parsed.Definition.Key, SemanticVersion: parsed.Definition.Version, Digest: parsed.Digest, Definition: parsed.Definition, Active: true, SourcePath: "proof", SourceHash: parsed.Digest, CreatedAt: time.Now(), SourceUpdatedAt: time.Now()}
			if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
				t.Fatal(err)
			}
			runs := newFakeRunLauncher()
			buildParent := func() *Orchestrator {
				o := New(repos.Profiles, repos.Tasks, repos.Runs, WithWorkflowRepository(repos.Workflows), WithWorkflowExecutionRepository(repos.WorkflowExecutions))
				t.Cleanup(o.dispatcher.Close)
				o.familyOwners = &familyOwnerClients{families: owner, executions: familyProofExecutions{accepted: accepted}}
				o.SetFamilySupervision(supervision.NewService(supervision.NewRepository(db), eventlog.NewSQLiteRepository(db)))
				o.workflowEngine.Children = admittedProofLauncher{fakeRunLauncher: runs, o: o}
				return o
			}
			o := buildParent()
			x, err := o.workflowEngine.Start(ctx, revision, input, "proof-family")
			if err != nil {
				t.Fatal(err)
			}
			x, err = o.driveWorkflowExecution(ctx, x.ID)
			if err != nil {
				t.Fatal(err)
			}
			a, b := runIDForNode(t, repos.WorkflowExecutions, x.ID, "child1"), runIDForNode(t, repos.WorkflowExecutions, x.ID, "child2")
			if a == uuid.Nil || b == uuid.Nil || runIDForNode(t, repos.WorkflowExecutions, x.ID, "child3") != uuid.Nil {
				t.Fatal("initial parallel frontier incorrect")
			}
			if owner.family.Members[0].AdmissionKey == "" || owner.family.Members[1].AdmissionKey == "" {
				t.Fatal("children launched without durable admission")
			}
			if mode == "cancelled" {
				cancelled, _, err := o.workflowEngine.Cancel(ctx, x.ID, "cancel-proof", "operator cancelled", 0)
				if err != nil {
					t.Fatal(err)
				}
				if err := o.cleanupPlanFamilyClaims(ctx, cancelled); err == nil {
					t.Fatal("live child claims released during cancellation")
				}
				_ = runs.Stop(ctx, a)
				_ = runs.Stop(ctx, b)
				if err := o.cleanupPlanFamilyClaims(ctx, cancelled); err != nil {
					t.Fatal(err)
				}
				revision := owner.family.Revision
				if err := o.cleanupPlanFamilyClaims(ctx, cancelled); err != nil || owner.family.Revision != revision {
					t.Fatalf("cleanup was not idempotent: %v", err)
				}
				if owner.family.Members[0].State != familypb.MemberState_MEMBER_STATE_BLOCKED || owner.family.Members[1].State != familypb.MemberState_MEMBER_STATE_BLOCKED {
					t.Fatal("terminated child claims retained")
				}
				return
			}
			// One child remains productive while the other is not finished. Restart the
			// parent and repeat wake delivery; neither operation may duplicate a child.
			runs.complete(a, map[string]any{"outcome": "completed", "evidence": []string{"plan:a"}})
			o = buildParent()
			for i := 0; i < 2; i++ {
				x, err = o.driveWorkflowExecution(ctx, x.ID)
				if err != nil {
					t.Fatal(err)
				}
			}
			if runIDForNode(t, repos.WorkflowExecutions, x.ID, "child2") != b || runIDForNode(t, repos.WorkflowExecutions, x.ID, "child3") != uuid.Nil {
				t.Fatal("restart duplicated or prematurely advanced children")
			}
			runs.complete(b, map[string]any{"outcome": "completed", "evidence": []string{"plan:b"}})
			x, err = o.driveWorkflowExecution(ctx, x.ID)
			if err != nil {
				t.Fatal(err)
			}
			integration := runIDForNode(t, repos.WorkflowExecutions, x.ID, "child3")
			if !accepted {
				if integration != uuid.Nil || x.Status != domain.WorkflowExecutionFailed {
					t.Fatalf("unaccepted plans advanced: %+v", x)
				}
				return
			}
			if integration == uuid.Nil {
				t.Fatalf("accepted dependency did not advance: %+v", x)
			}
			runs.complete(integration, map[string]any{"outcome": "completed", "evidence": []string{"plan:integration"}})
			x, err = o.driveWorkflowExecution(ctx, x.ID)
			if err != nil {
				t.Fatal(err)
			}
			if x.Status != domain.WorkflowExecutionSucceeded {
				t.Fatalf("family did not complete: %+v", x)
			}
			watches, err := o.familySupervision.List(ctx, &watchpb.ListCohortWatchesRequest{FamilyExecutionId: x.ID.String()})
			if err != nil || len(watches.GetWatches()) != 2 {
				t.Fatalf("watch cohorts not durable: %v %v", watches, err)
			}
		})
	}
}
