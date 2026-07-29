package channelmanager

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	channelmanagerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager"
	channelmanagerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager/channelmanager_v1connect"
)

type staticResolver struct{ url string }

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type releaseHandler struct {
	channelmanagerconnect.UnimplementedChannelManagerServiceHandler
}

func (releaseHandler) GetEligibility(_ context.Context, request *connect.Request[channelmanagerv1.GetEligibilityRequest]) (*connect.Response[channelmanagerv1.GetEligibilityResponse], error) {
	if request.Msg.IdentityId != "identity-1" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&channelmanagerv1.GetEligibilityResponse{Eligibility: "eligible"}), nil
}

func (releaseHandler) SubmitRelease(_ context.Context, request *connect.Request[channelmanagerv1.SubmitReleaseRequest]) (*connect.Response[channelmanagerv1.SubmitReleaseResponse], error) {
	if request.Msg.IdempotencyKey != "release-1" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&channelmanagerv1.SubmitReleaseResponse{Receipt: &channelmanagerv1.ReleaseReceipt{Id: "release-1", ActionId: "action-1", Status: "scheduled"}}), nil
}
func TestClientUsesGeneratedContractAndResolvedURL(t *testing.T) {
	_, handler := channelmanagerconnect.NewChannelManagerServiceHandler(releaseHandler{})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &Client{resolver: staticResolver{url: server.URL}, http: server.Client()}
	receipt, err := client.SubmitRelease(context.Background(), Submission{IdentityID: "identity-1", Lane: "main", DraftID: "draft-1", IdempotencyKey: "release-1"})
	require.NoError(t, err)
	require.Equal(t, Receipt{ID: "release-1", ActionID: "action-1", Status: "scheduled"}, receipt)
	eligibility, err := client.CheckEligibility(context.Background(), "identity-1", "main")
	require.NoError(t, err)
	require.Equal(t, "eligible", eligibility)
}
