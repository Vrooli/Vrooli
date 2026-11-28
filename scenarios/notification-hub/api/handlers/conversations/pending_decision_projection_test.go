package conversations_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestPendingDecisionProjectionIsOwnerAuthenticatedAndAllowlisted(t *testing.T) {
	f := newReaderFixture(t)
	decisionID := f.eventAsk(t, "autoheal", "projection-safe")
	f.ask(t, "owner", "status-not-projected")

	unauthenticated, err := f.client.ListPendingDecisionReferences(context.Background(), connect.NewRequest(&v1.ListPendingDecisionReferencesRequest{}))
	require.Nil(t, unauthenticated)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	wrongOwnerReq := connect.NewRequest(&v1.ListPendingDecisionReferencesRequest{})
	wrongOwnerReq.Header().Set("X-Vrooli-Identity-Subject", "bob")
	wrongOwner, err := f.client.ListPendingDecisionReferences(context.Background(), wrongOwnerReq)
	require.Nil(t, wrongOwner)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	page, err := f.client.ListPendingDecisionReferences(context.Background(), reading("owner", &v1.ListPendingDecisionReferencesRequest{Limit: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.GetDecisions(), 1)
	projected := page.Msg.GetDecisions()[0]
	require.Equal(t, decisionID, projected.GetDecisionRef())
	require.Equal(t, "alice", projected.GetOwnerId())
	require.Equal(t, "pending", projected.GetState())
	require.NotEmpty(t, projected.GetRevision())

	read, err := f.client.ReadPendingDecisionReference(context.Background(), reading("owner", &v1.ReadPendingDecisionReferenceRequest{DecisionRef: decisionID, ExpectedRevision: projected.GetRevision()}))
	require.NoError(t, err)
	require.Equal(t, projected, read.Msg.GetDecision())
	_, err = f.client.ReadPendingDecisionReference(context.Background(), reading("owner", &v1.ReadPendingDecisionReferenceRequest{DecisionRef: decisionID, ExpectedRevision: "stale"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
