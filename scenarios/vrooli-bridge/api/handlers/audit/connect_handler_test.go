package audit

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/audit"
	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/auth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

// [REQ:BRG-P0-008] ListAuditRecords is owner-gated.
func TestAuditHandler_RequiresOwner(t *testing.T) {
	h := NewConnectHandler(Deps{Reader: &auditmocks.FakeReader{}})
	_, err := h.ListAuditRecords(context.Background(), connect.NewRequest(&auditv1.ListAuditRecordsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-008] The handler passes the filter through and translates domain
// records to the wire shape (action/outcome enums included).
func TestAuditHandler_ListTranslatesAndFilters(t *testing.T) {
	reader := &auditmocks.FakeReader{ListOut: []audit.Record{
		{
			ID: "a1", Action: audit.ActionDispatch, Actor: "owner-1", NodeID: "n1",
			Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"},
			Outcome: audit.OutcomeAccepted, RunID: "run-1", RecordedAt: time.Unix(100, 0).UTC(),
		},
		{
			ID: "a2", Action: audit.ActionDispatch, Actor: "owner-1", NodeID: "n1",
			Verb: "scenario deploy", Outcome: audit.OutcomeRejected, Detail: "verb not allowlisted",
			RecordedAt: time.Unix(90, 0).UTC(),
		},
	}}
	h := NewConnectHandler(Deps{Reader: reader})

	resp, err := h.ListAuditRecords(ownerCtx(), connect.NewRequest(&auditv1.ListAuditRecordsRequest{NodeId: "n1", RunId: "run-1", Limit: 10}))
	require.NoError(t, err)
	require.Equal(t, "n1", reader.LastFilter.NodeID)
	require.Equal(t, "run-1", reader.LastFilter.RunID)
	require.Equal(t, 10, reader.LastFilter.Limit)

	require.Len(t, resp.Msg.Records, 2)
	require.Equal(t, auditv1.AuditAction_AUDIT_ACTION_DISPATCH, resp.Msg.Records[0].Action)
	require.Equal(t, auditv1.AuditOutcome_AUDIT_OUTCOME_ACCEPTED, resp.Msg.Records[0].Outcome)
	require.Equal(t, auditv1.AuditOutcome_AUDIT_OUTCOME_REJECTED, resp.Msg.Records[1].Outcome)
	require.Equal(t, "verb not allowlisted", resp.Msg.Records[1].Detail)
}
