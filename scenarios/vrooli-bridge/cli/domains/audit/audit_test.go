package audit

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit/audit_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

type fakeAudit struct {
	lastReq *auditv1.ListAuditRecordsRequest
}

func (f *fakeAudit) ListAuditRecords(_ context.Context, req *connect.Request[auditv1.ListAuditRecordsRequest]) (*connect.Response[auditv1.ListAuditRecordsResponse], error) {
	f.lastReq = req.Msg
	return connect.NewResponse(&auditv1.ListAuditRecordsResponse{Records: []*auditv1.AuditRecord{
		{
			Id: "a1", Action: auditv1.AuditAction_AUDIT_ACTION_DISPATCH, Actor: "owner-1", NodeId: "n1",
			Scenario: "web-search", Verb: "scenario test", Outcome: auditv1.AuditOutcome_AUDIT_OUTCOME_ACCEPTED,
			RunId: "run-1", RecordedAt: timestamppb.Now(),
		},
	}}), nil
}

func connectAPI(svc auditconnect.AuditServiceHandler) http.Handler {
	path, handler := auditconnect.NewAuditServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

// [REQ:BRG-P0-008] The audit CLI lists records through the generated client and
// passes its filters through.
func TestAudit_ListRoundTrip(t *testing.T) {
	svc := &fakeAudit{}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "node"}, {Name: "run"}, {Name: "limit"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"node": "n1", "limit": "10"},
	})
	require.NoError(t, h.list(ctx))

	require.Equal(t, "n1", svc.lastReq.NodeId)
	require.Equal(t, int32(10), svc.lastReq.Limit)
	require.Contains(t, out.String(), "scenario test")
	require.Contains(t, out.String(), "accepted")
}
