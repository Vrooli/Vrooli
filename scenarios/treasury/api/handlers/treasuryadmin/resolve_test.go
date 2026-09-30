package treasuryadmin_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"treasury/handlers/treasuryadmin"
	"treasury/internal/approval"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	approvalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
)

type approvalResolver struct{ gotResolver string }

func (r *approvalResolver) List(_ context.Context, status approval.Status, bookID ...string) ([]approval.Request, error) {
	book := "book-1"
	if len(bookID) > 0 && bookID[0] != "" {
		book = bookID[0]
	}
	return []approval.Request{{ID: "approval-1", AuthorizationID: "auth-1", BookID: book, MandateID: "mandate-1", RequestingAgent: "agent:1", AmountMinor: 1250, Currency: "USD", Counterparty: "vendor.example", Status: status, CreatedAt: time.Unix(1, 0).UTC(), ExpiresAt: time.Unix(3601, 0).UTC()}}, nil
}

func (r *approvalResolver) Resolve(_ context.Context, id string, status approval.Status, resolver string) (approval.Request, error) {
	r.gotResolver = resolver
	return approval.Request{ID: id, AuthorizationID: "auth-1", BookID: "book-1", MandateID: "mandate-1", Status: status, ResolverIdentity: resolver, CreatedAt: time.Unix(1, 0).UTC(), ResolvedAt: time.Unix(2, 0).UTC()}, nil
}

func TestListApprovalsUsesLocalQueue(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	resolver := &approvalResolver{}
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, resolver))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorizationv1.ListApprovalsRequest{Status: approvalv1.ApprovalStatus_APPROVAL_STATUS_QUEUED, BookId: "book-1"})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	response, err := client.ListApprovals(context.Background(), request)
	require.NoError(t, err)
	require.Len(t, response.Msg.GetApprovals(), 1)
	require.Equal(t, int64(1250), response.Msg.GetApprovals()[0].GetAmountMinor())
	require.Equal(t, "book-1", response.Msg.GetApprovals()[0].GetBookId())
	require.NotNil(t, response.Msg.GetApprovals()[0].GetExpiresAt())
}

// [REQ:TRS-P0-006] The operator-owned Connect path resolves locally; no relay
// client participates in resolution.
func TestResolveApprovalUsesLocalService(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	resolver := &approvalResolver{}
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer, resolver))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	request := connect.NewRequest(&authorizationv1.ResolveApprovalRequest{ApprovalId: "approval-1", Resolution: approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED})
	request.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	response, err := client.ResolveApproval(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, approvalv1.ApprovalStatus_APPROVAL_STATUS_DECLINED, response.Msg.GetApproval().GetStatus())
	require.Equal(t, "local-operator", resolver.gotResolver)
}
