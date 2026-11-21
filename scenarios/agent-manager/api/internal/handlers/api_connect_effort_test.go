package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEffortConnectBoardAndAuthenticatedEnrollment(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	service := supervision.NewService(repo, nil)
	service.Efforts = supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	handler := NewAgentManagerConnectHandler(nil, service)
	path, rpc := apiconnect.NewAgentManagerServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(path, rpc)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
	board, err := client.GetEffortBoard(ctx, connect.NewRequest(&pb.GetEffortBoardRequest{}))
	if err != nil || len(board.Msg.Rows) != 0 || !board.Msg.Partial {
		t.Fatal("fresh board must be empty and explicitly unscanned", board, err)
	}
	request := connect.NewRequest(&pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:arbitrary", DisplayName: "Arbitrary investigation", WorkShape: "investigation"}, IdempotencyKey: "rpc-enroll"})
	if _, err = client.EnrollEffort(ctx, request); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatal("missing authorizer admitted enrollment", err)
	}
	handler.SetWatchActionAuthorizer(allowWatchAction{})
	enrolled, err := client.EnrollEffort(ctx, request)
	if err != nil || enrolled.Msg.AuthorizedBy != "operator-1" {
		t.Fatal("verified actor was not retained", enrolled, err)
	}
	board, err = client.GetEffortBoard(ctx, connect.NewRequest(&pb.GetEffortBoardRequest{}))
	if err != nil || len(board.Msg.Rows) != 1 || board.Msg.ActiveCount != 1 {
		t.Fatal("board did not expose arbitrary enrollment", board, err)
	}
	if board.Msg.Rows[0].Usage.ReportedCostUsd != nil || board.Msg.Rows[0].OutcomeStanding.State != "unknown" {
		t.Fatal("wire projection invented cost/acceptance", board.Msg)
	}
	replay, err := client.EnrollEffort(ctx, request)
	if err != nil || replay.Msg.Revision != enrolled.Msg.Revision {
		t.Fatal("RPC retry not idempotent", replay, err)
	}
}

func TestEffortAgentIdentityHeaderNeverAcquiresOperatorAuthority(t *testing.T) {
	h := http.Header{}
	h.Set("X-Agent-Identity-Token", "actual-cli-run-token")
	h.Set("Authorization", "Bearer saved-api-token")
	if got := effortToken(h, pb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT); got != "actual-cli-run-token" {
		t.Fatal(got)
	}
	if got := effortToken(h, pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR); got != "saved-api-token" {
		t.Fatal("agent identity substituted for operator", got)
	}
}

func TestApprovedDeliveryOperationUsesExistingClockWithoutEnrollmentAuthority(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
		t.Fatal(err)
	}
	repo := supervision.NewRepository(db)
	efforts := supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	current, err := efforts.Enroll(ctx, &pb.EnrollEffortRequest{Enrollment: &pb.EffortEnrollment{
		EffortRef: "effort:hub", DestinationRef: "path:delivery/CHARTER.md", TargetRevision: "selected-1", AuthorityRef: "original-approval",
	}, IdempotencyKey: "owner-enrollment"}, supervision.EffortActor{ID: "owner", Operator: true})
	if err != nil {
		t.Fatal(err)
	}
	h := NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: efforts})
	// A supplied human authorizer cannot become an agent's fallback authority.
	h.SetWatchActionAuthorizer(allowWatchAction{})
	call := func(actor context.Context, request *pb.EnrollEffortRequest) (*pb.EffortEnrollment, error) {
		path, rpc := apiconnect.NewAgentManagerServiceHandler(h)
		mux := http.NewServeMux()
		mux.Handle(path, rpc)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AgentOperationMiddleware(mux).ServeHTTP(w, r.WithContext(actor))
		}))
		defer server.Close()
		client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
		response, err := client.EnrollEffort(ctx, connect.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
	approved := grantContext([]string{"effort.delivery"}, []string{EffortGrantTarget(current.EffortRef)})
	next := proto.Clone(current).(*pb.EffortEnrollment)
	next.Delivery = &pb.EffortDelivery{CommissionId: "approved-run", State: "running", BudgetSeconds: 86400, MaxWorkers: 2, StartedAt: timestamppb.New(time.Now().Add(-48 * time.Hour))}
	request := &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: "delegated-start"}
	current, err = call(approved, request)
	if err != nil {
		t.Fatalf("approved existing-effort delivery control failed: %v", err)
	}
	if current.AuthorizedBy != "owner" || current.AuthorityRef != "original-approval" || current.Delivery.StartedAt.AsTime().Before(time.Now().Add(-time.Minute)) {
		t.Fatal("delivery control changed authority or accepted a caller's clock")
	}
	replayed, err := call(approved, request)
	if err != nil || !proto.Equal(replayed, current) {
		t.Fatalf("delivery replay changed the clock or revision: %v", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*pb.EffortEnrollment)
	}{
		{"destination", func(e *pb.EffortEnrollment) { e.DestinationRef = "path:other/CHARTER.md" }},
		{"target", func(e *pb.EffortEnrollment) { e.TargetRevision = "different-goal" }},
		{"authority", func(e *pb.EffortEnrollment) { e.AuthorityRef = "invented" }},
		{"owner", func(e *pb.EffortEnrollment) { e.AuthorizedBy = "agent" }},
		{"withdrawal", func(e *pb.EffortEnrollment) { e.Withdrawn = true }},
		{"subjects", func(e *pb.EffortEnrollment) {
			e.Subjects = append(e.Subjects, &pb.EffortSubject{Owner: "prompt-manager", Kind: "team", Reference: "other"})
		}},
		{"budget reset", func(e *pb.EffortEnrollment) { e.Delivery.BudgetSeconds++ }},
		{"worker reset", func(e *pb.EffortEnrollment) { e.Delivery.MaxWorkers++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := proto.Clone(current).(*pb.EffortEnrollment)
			tc.edit(next)
			if _, err := call(approved, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: tc.name}); err == nil {
				t.Fatal("delivery-only consent changed protected enrollment or clock policy")
			}
			stored, _, err := repo.GetEffort(ctx, current.EffortRef)
			if err != nil || !proto.Equal(stored, current) {
				t.Fatal("refused request left enrollment effects", err)
			}
		})
	}
	for _, operations := range [][]string{{"effort.reconcile"}, {"agent.manage"}, nil} {
		if _, err := call(grantContext(operations, []string{"agent-manager:*"}), &pb.EnrollEffortRequest{Enrollment: proto.Clone(current).(*pb.EffortEnrollment), ExpectedRevision: current.Revision, IdempotencyKey: "wrong-operation"}); err == nil {
			t.Fatal("unrelated or missing grant acquired delivery authority")
		}
	}
	newEffort := proto.Clone(current).(*pb.EffortEnrollment)
	newEffort.EffortRef = "effort:new"
	newEffort.Revision = 0
	if _, err := call(grantContext([]string{"effort.delivery"}, []string{EffortGrantTarget(newEffort.EffortRef)}), &pb.EnrollEffortRequest{Enrollment: newEffort, IdempotencyKey: "cannot-create"}); err == nil {
		t.Fatal("delivery consent created a new effort")
	}
	if _, err := call(approved, &pb.EnrollEffortRequest{Enrollment: proto.Clone(current).(*pb.EffortEnrollment), ExpectedRevision: current.Revision - 1, IdempotencyKey: "stale-revision"}); err == nil {
		t.Fatal("delivery consent bypassed revision fencing")
	}
	for _, state := range []string{"paused", "running", "stopped"} {
		next := proto.Clone(current).(*pb.EffortEnrollment)
		next.Delivery.State, next.Delivery.Reason = state, "approved lifecycle change"
		current, err = call(approved, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: state})
		if err != nil {
			t.Fatalf("approved delivery %s: %v", state, err)
		}
	}
	next = proto.Clone(current).(*pb.EffortEnrollment)
	next.Delivery.State = "running"
	if _, err := call(approved, &pb.EnrollEffortRequest{Enrollment: next, ExpectedRevision: current.Revision, IdempotencyKey: "cannot-restart"}); err == nil {
		t.Fatal("a stopped commission restarted")
	}
}
