package recovery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"tunnel-manager/handlers/recovery"
	"tunnel-manager/internal/authz"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"
	recoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery/recovery_v1connect"

	internalrecovery "tunnel-manager/internal/recovery"
)

// fakeService implements internalrecovery.Service for handler tests.
type fakeService struct {
	state        internalrecovery.RecoveryState
	events       []internalrecovery.RecoveryEvent
	listErr      error
	recoverOut   internalrecovery.EventOutcome
	recoverEvt   internalrecovery.RecoveryEvent
	recoverErr   error
	recoverCalls int
}

func (f *fakeService) GetState(context.Context) (internalrecovery.RecoveryState, error) {
	return f.state, nil
}

func (f *fakeService) ListEvents(context.Context, int) ([]internalrecovery.RecoveryEvent, error) {
	return f.events, f.listErr
}

func (f *fakeService) Recover(context.Context, bool) (internalrecovery.EventOutcome, internalrecovery.RecoveryEvent, error) {
	f.recoverCalls++
	return f.recoverOut, f.recoverEvt, f.recoverErr
}

func (f *fakeService) Evaluate(context.Context) (internalrecovery.RecoveryEvent, bool, error) {
	return internalrecovery.RecoveryEvent{}, false, nil
}

func newClient(t *testing.T, svc internalrecovery.Service) recoveryconnect.RecoveryServiceClient {
	t.Helper()
	return newClientWithAuthorizer(t, svc, nil)
}

func newClientWithAuthorizer(t *testing.T, svc internalrecovery.Service, authorizer authz.Authorizer) recoveryconnect.RecoveryServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := recoveryconnect.NewRecoveryServiceHandler(recovery.NewConnectHandler(recovery.Deps{
		Service:    svc,
		Logger:     logger,
		Authorizer: authorizer,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return recoveryconnect.NewRecoveryServiceClient(server.Client(), server.URL)
}

func TestHandlerGetStateMapsEnumAndZeroTimes(t *testing.T) {
	zone := time.FixedZone("offset", -5*60*60)
	lastCheck := time.Date(2026, 5, 4, 12, 0, 0, 0, zone)
	nextRetry := time.Date(2026, 5, 4, 13, 0, 0, 0, zone)
	client := newClient(t, &fakeService{state: internalrecovery.RecoveryState{
		Status: internalrecovery.StatusCircuitOpen, ConsecFailures: 4,
		LastCheck: lastCheck, NextRetryAfter: nextRetry,
	}})
	resp, err := client.GetState(context.Background(), connect.NewRequest(&recoveryv1.GetStateRequest{}))
	require.NoError(t, err)
	require.Equal(t, recoveryv1.RecoveryStatus_RECOVERY_STATUS_CIRCUIT_OPEN, resp.Msg.State.Status)
	require.EqualValues(t, 4, resp.Msg.State.ConsecFailures)
	require.True(t, resp.Msg.State.CircuitOpen)
	require.NotNil(t, resp.Msg.State.LastCheck)
	require.Equal(t, lastCheck.UTC(), resp.Msg.State.LastCheck.AsTime(), "LastCheck is emitted in UTC")
	require.Nil(t, resp.Msg.State.LastRecovery, "zero LastRecovery stays unset")
	require.NotNil(t, resp.Msg.State.NextRetryAfter)
	require.Equal(t, nextRetry.UTC(), resp.Msg.State.NextRetryAfter.AsTime(), "NextRetryAfter is emitted in UTC")
}

func TestHandlerRecoverMapsOutcome(t *testing.T) {
	zone := time.FixedZone("offset", -5*60*60)
	createdAt := time.Date(2026, 5, 4, 12, 0, 0, 0, zone)
	client := newClient(t, &fakeService{
		recoverOut: internalrecovery.OutcomeSuccess,
		recoverEvt: internalrecovery.RecoveryEvent{ID: "e1", Trigger: internalrecovery.TriggerManual, Action: internalrecovery.ActionRestart, Outcome: internalrecovery.OutcomeSuccess, CreatedAt: createdAt},
	})
	resp, err := client.Recover(context.Background(), connect.NewRequest(&recoveryv1.RecoverRequest{Force: true}))
	require.NoError(t, err)
	require.Equal(t, recoveryv1.EventOutcome_EVENT_OUTCOME_SUCCESS, resp.Msg.Outcome)
	require.Equal(t, "e1", resp.Msg.Event.Id)
	require.NotNil(t, resp.Msg.Event.CreatedAt)
	require.Equal(t, createdAt.UTC(), resp.Msg.Event.CreatedAt.AsTime(), "event CreatedAt is emitted in UTC")
}

func TestHandlerRecoverRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})

	_, err := client.Recover(context.Background(), connect.NewRequest(&recoveryv1.RecoverRequest{Force: true}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.recoverCalls, "denied recovery must not reach the service")
}

func TestHandlerRecoverAcceptsOperatorBearer(t *testing.T) {
	fake := &fakeService{recoverOut: internalrecovery.OutcomeSkipped}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	req := connect.NewRequest(&recoveryv1.RecoverRequest{Force: true})
	req.Header().Set("Authorization", "Bearer secret")

	resp, err := client.Recover(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, recoveryv1.EventOutcome_EVENT_OUTCOME_SKIPPED, resp.Msg.Outcome)
	require.Equal(t, 1, fake.recoverCalls)
}

func TestHandlerListEventsInternalError(t *testing.T) {
	client := newClient(t, &fakeService{listErr: errors.New("db down")})
	_, err := client.ListEvents(context.Background(), connect.NewRequest(&recoveryv1.ListEventsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
