package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
	service.Efforts = supervision.NewEffortService(repo, nil, supervision.NewPolicyStore(db, nil), supervision.EffortDiscoveryConfig{Root: t.TempDir()})
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
