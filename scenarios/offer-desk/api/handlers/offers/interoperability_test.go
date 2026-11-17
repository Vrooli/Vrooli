package offers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"
	ledger "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	swarmapi "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	swarmconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type positionServiceStub struct {
	ledgerconnect.UnimplementedPositionServiceHandler
	getCalls int
}

func (s *positionServiceStub) GetPosition(context.Context, *connect.Request[ledger.PositionRequest]) (*connect.Response[ledger.PositionResponse], error) {
	s.getCalls++
	return connect.NewResponse(&ledger.PositionResponse{}), nil
}

func (s *positionServiceStub) ListGoals(context.Context, *connect.Request[ledger.ListGoalsRequest]) (*connect.Response[ledger.ListGoalsResponse], error) {
	return connect.NewResponse(&ledger.ListGoalsResponse{}), nil
}

func ledgerPositionServer(t *testing.T) (*httptest.Server, *positionServiceStub) {
	t.Helper()
	stub := &positionServiceStub{}
	_, handler := ledgerconnect.NewPositionServiceHandler(stub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, stub
}

func TestBoardDoesNotReuseLedgerClientWhenResolutionFailsAndRecoversAfterRestart(t *testing.T) {
	s, ctx := testOfferService(t, schedule.NewFake(time.Now().UTC()))
	s.bookID = "book"
	first, firstStub := ledgerPositionServer(t)
	second, secondStub := ledgerPositionServer(t)
	currentURL := first.URL
	var resolutionErr error
	s.resolveLedgerURL = func(context.Context) (string, error) { return currentURL, resolutionErr }

	_, err := s.GetBoard(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, firstStub.getCalls)

	first.Close()
	currentURL = ""
	resolutionErr = errors.New("money-ledger is restarting")
	degraded, err := s.GetBoard(ctx, nil)
	require.NoError(t, err, "optional ledger failure degrades the board instead of failing the operation")
	require.Empty(t, degraded.Msg.PostureSource)
	require.Contains(t, degraded.Msg.Availability[0].Reason, "could not be resolved")

	currentURL = second.URL
	resolutionErr = nil
	recovered, err := s.GetBoard(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 1, secondStub.getCalls, "the next request must use the restarted peer's address")
	require.Equal(t, "money-ledger.position", recovered.Msg.PostureSource)
}

type goalServiceStub struct {
	swarmconnect.UnimplementedGoalServiceHandler
	calls int
}

func (s *goalServiceStub) ListGoals(context.Context, *connect.Request[swarmapi.ListGoalsRequest]) (*connect.Response[swarmapi.ListGoalsResponse], error) {
	s.calls++
	return connect.NewResponse(&swarmapi.ListGoalsResponse{}), nil
}

func goalServer(t *testing.T) (*httptest.Server, *goalServiceStub) {
	t.Helper()
	stub := &goalServiceStub{}
	_, handler := swarmconnect.NewGoalServiceHandler(stub)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, stub
}

func TestReleaseLadderResolvesSwarmManagerForEachRequestAndDegradesExplicitly(t *testing.T) {
	s, _ := testOfferService(t, schedule.NewFake(time.Now().UTC()))
	first, firstStub := goalServer(t)
	second, secondStub := goalServer(t)
	currentURL := first.URL
	var resolutionErr error
	s.resolveGoalsURL = func(context.Context) (string, error) { return currentURL, resolutionErr }

	response := &offerspb.ReleaseLadderResponse{}
	s.attachGoalImpacts(context.Background(), response)
	require.Empty(t, response.Availability)
	require.Equal(t, 1, firstStub.calls)

	first.Close()
	currentURL = ""
	resolutionErr = errors.New("swarm-manager is restarting")
	response = &offerspb.ReleaseLadderResponse{}
	s.attachGoalImpacts(context.Background(), response)
	require.Len(t, response.Availability, 1)
	require.Contains(t, response.Availability[0].Reason, "restarting")

	currentURL = second.URL
	resolutionErr = nil
	response = &offerspb.ReleaseLadderResponse{}
	s.attachGoalImpacts(context.Background(), response)
	require.Empty(t, response.Availability)
	require.Equal(t, 1, secondStub.calls, "the next ladder request must reach the restarted peer")
}

func TestDeploymentReadinessProviderResolvesPeerPerRead(t *testing.T) {
	makeServer := func(goalExists bool) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if goalExists {
				_, _ = w.Write([]byte(`{"goal_exists":true,"goal_closed":false,"approved_commit":"a"}`))
				return
			}
			_, _ = w.Write([]byte(`{"goal_exists":false,"goal_closed":true,"approved_commit":"b"}`))
		}))
	}
	first, second := makeServer(true), makeServer(false)
	t.Cleanup(first.Close)
	t.Cleanup(second.Close)
	currentURL := first.URL
	provider := deploymentReadinessProvider(&http.Client{}, func(context.Context) (string, error) { return currentURL, nil })

	state, err := provider(context.Background(), &offerspb.Node{Name: "storefront"})
	require.NoError(t, err)
	require.True(t, state.GoalExists)
	first.Close()
	currentURL = second.URL
	state, err = provider(context.Background(), &offerspb.Node{Name: "storefront"})
	require.NoError(t, err)
	require.False(t, state.GoalExists)
	require.Equal(t, "b", state.ApprovedCommit)
}

func TestReadinessReporterIsConstructedWithoutResolvingDeploymentManager(t *testing.T) {
	for key, value := range map[string]string{
		"_TOKEN": "token", "_POLICY_VERSION": "1", "_SCENARIO": "offer-desk",
		"_PROFILE_ID": "profile", "_CANDIDATE_COMMIT": "commit", "_ARTIFACT_DIGEST": "digest",
		"_TARGETS": "linux", "_CHANNEL": "local",
	} {
		t.Setenv(readinessPrefix+key, value)
	}
	t.Setenv("DEPLOYMENT_MANAGER_API_BASE_URL", "")
	reporter, err := newReadinessReporter(context.Background(), "catalog-ready", "offer-desk.catalog")
	require.NoError(t, err)
	require.NotNil(t, reporter, "a local discovery miss must not prevent Offer Desk from starting")
}
