package treasuryadmin_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	approvalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/approval"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/approval"
	"treasury/internal/operatorauth"
)

type approvalResolver struct{ gotResolver string }

func (r *approvalResolver) Resolve(_ context.Context, id string, status approval.Status, resolver string) (approval.Request, error) {
	r.gotResolver = resolver
	return approval.Request{ID: id, AuthorizationID: "auth-1", MandateID: "mandate-1", Status: status, ResolverIdentity: resolver, CreatedAt: time.Unix(1, 0).UTC(), ResolvedAt: time.Unix(2, 0).UTC()}, nil
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
