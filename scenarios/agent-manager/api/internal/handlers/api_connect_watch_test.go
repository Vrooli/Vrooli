package handlers

import (
	"context"
	"testing"

	"agent-manager/internal/eventlog"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	_ "modernc.org/sqlite"
)

type watchEventSource struct{ generation int64 }

type allowWatchAction struct{}

func (allowWatchAction) AuthorizeWatchAction(_ context.Context, _ string, request *domainpb.RequestCohortWatchActionRequest) error {
	request.Authority = domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	request.RequestedBy = "operator-1"
	return nil
}

func (s watchEventSource) RetentionState(context.Context) (eventlog.RetentionState, error) {
	return eventlog.RetentionState{Generation: s.generation}, nil
}

func (watchEventSource) ReadCohort(context.Context, []uuid.UUID, int64, int) ([]eventlog.CohortEvent, error) {
	return nil, nil
}

func TestCohortWatchConnectCreateGetAndInspect(t *testing.T) { // [REQ:REQ-P2-008]
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
		t.Fatal(err)
	}
	service := supervision.NewService(supervision.NewRepository(db), watchEventSource{generation: 1})
	policies := supervision.NewPolicyStore(db, nil)
	if _, err := policies.EnsureInitialActive(context.Background(), supervision.DefaultSupervisionPolicy(), "bootstrap"); err != nil {
		t.Fatal(err)
	}
	service.SetPolicyStore(policies)
	service.SetActionService(supervision.NewActionService(supervision.NewRepository(db), nil))
	handler := NewAgentManagerConnectHandler(nil, service)
	handler.SetWatchActionAuthorizer(allowWatchAction{})
	runID := uuid.NewString()
	created, err := handler.CreateCohortWatch(context.Background(), connect.NewRequest(&domainpb.CreateCohortWatchRequest{IdempotencyKey: "connect-watch", Spec: &domainpb.WatchSpec{FamilyExecutionId: "family-run", Subjects: []*domainpb.WatchSubject{{PlanId: "plan-a", RunId: runID}}}}))
	if err != nil {
		t.Fatal(err)
	}
	watchID := created.Msg.GetWatchId()
	got, err := handler.GetCohortWatch(context.Background(), connect.NewRequest(&domainpb.GetCohortWatchRequest{WatchId: watchID}))
	if err != nil || got.Msg.GetWatchId() != watchID {
		t.Fatalf("get = %+v err=%v", got, err)
	}
	inspection, err := handler.InspectCohortWatch(context.Background(), connect.NewRequest(&domainpb.InspectCohortWatchRequest{WatchId: watchID}))
	if err != nil || inspection.Msg.GetWatch().GetWatchId() != watchID {
		t.Fatalf("inspect = %+v err=%v", inspection, err)
	}
	action, err := handler.RequestCohortWatchAction(context.Background(), connect.NewRequest(&domainpb.RequestCohortWatchActionRequest{WatchId: watchID, ExpectedWatchRevision: created.Msg.GetRevision(), IdempotencyKey: "observe-connect", Kind: domainpb.WatchActionKind_WATCH_ACTION_KIND_OBSERVE}))
	if err != nil || action.Msg.GetAction().GetState() != domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED {
		t.Fatalf("action = %+v err=%v", action, err)
	}
	actions, err := handler.ListCohortWatchActions(context.Background(), connect.NewRequest(&domainpb.ListCohortWatchActionsRequest{WatchId: watchID}))
	if err != nil || len(actions.Msg.GetActions()) != 1 {
		t.Fatalf("actions = %+v err=%v", actions, err)
	}
	candidateRequest := connect.NewRequest(&domainpb.CreateSupervisionPolicyCandidateRequest{CreatedBy: "spoofed", Supersedes: "supervision-v1", Policy: &domainpb.SupervisionPolicyDefinition{Version: "supervision-v2", EventCount: 32, QuietSeconds: 30, FrictionThreshold: .8, Terminal: true, AllowedActions: []string{"observe", "park", "escalate", "wake_parent"}, ClassifierRevision: "classifier-v2"}})
	candidateRequest.Header().Set("Authorization", "Bearer owner-token")
	candidate, err := handler.CreateSupervisionPolicyCandidate(context.Background(), candidateRequest)
	if err != nil || candidate.Msg.GetCreatedBy() != "operator-1" || candidate.Msg.GetState() != domainpb.SupervisionPolicyState_SUPERVISION_POLICY_STATE_CANDIDATE {
		t.Fatalf("candidate = %+v err=%v", candidate, err)
	}
	active, err := handler.GetSupervisionPolicy(context.Background(), connect.NewRequest(&domainpb.GetSupervisionPolicyRequest{}))
	if err != nil || active.Msg.GetPolicy().GetVersion() != "supervision-v1" {
		t.Fatalf("active policy = %+v err=%v", active, err)
	}
}
