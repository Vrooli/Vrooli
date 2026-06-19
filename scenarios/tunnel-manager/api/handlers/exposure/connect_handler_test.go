package exposure_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/handlers/exposure"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	exposurev1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"

	internalexposure "tunnel-manager/internal/exposure"
)

// fakeService implements internalexposure.Service for handler tests.
type fakeService struct {
	exposeLease internalexposure.Lease
	exposeURL   string
	exposeErr   error
	revoked     bool
	leases      []internalexposure.Lease
	exposures   []internalexposure.Exposure
	isExposed   bool
	isURL       string
	coreEnsured int
	reaped      int
	lastInput   internalexposure.ExposeInput
}

func (f *fakeService) Expose(_ context.Context, in internalexposure.ExposeInput) (internalexposure.Lease, string, error) {
	f.lastInput = in
	return f.exposeLease, f.exposeURL, f.exposeErr
}

func (f *fakeService) ExtendLease(_ context.Context, _ string, _ time.Duration) (internalexposure.Lease, error) {
	return f.exposeLease, nil
}

func (f *fakeService) RevokeLease(context.Context, string) (bool, error) {
	return f.revoked, nil
}

func (f *fakeService) ListLeases(context.Context, internalexposure.LeaseStatus) ([]internalexposure.Lease, error) {
	return f.leases, nil
}

func (f *fakeService) ListExposures(context.Context) ([]internalexposure.Exposure, error) {
	return f.exposures, nil
}

func (f *fakeService) IsExposed(context.Context, string) (bool, string, error) {
	return f.isExposed, f.isURL, nil
}

func (f *fakeService) Reconcile(context.Context) (int, int, error) {
	return f.coreEnsured, f.reaped, nil
}

func newClient(t *testing.T, svc internalexposure.Service) exposureconnect.ExposureServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := exposureconnect.NewExposureServiceHandler(exposure.NewConnectHandler(exposure.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return exposureconnect.NewExposureServiceClient(server.Client(), server.URL)
}

func TestHandlerExposeMapsTTLAndReturnsURL(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	fake := &fakeService{
		exposeLease: internalexposure.Lease{ID: "l1", Scenario: "web-console", Status: internalexposure.LeaseActive, ExpiresAt: now},
		exposeURL:   "https://web-console.itsagitime.com",
	}
	client := newClient(t, fake)

	resp, err := client.Expose(context.Background(), connect.NewRequest(&exposurev1.ExposeRequest{Scenario: "web-console", TtlSeconds: 3600}))
	require.NoError(t, err)
	require.Equal(t, "https://web-console.itsagitime.com", resp.Msg.PublicUrl)
	require.Equal(t, "l1", resp.Msg.Lease.Id)
	require.Equal(t, exposurev1.LeaseStatus_LEASE_STATUS_ACTIVE, resp.Msg.Lease.Status)
	require.Equal(t, time.Hour, fake.lastInput.TTL, "ttl_seconds converted to duration")
}

func TestHandlerExposeInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{exposeErr: internalexposure.ErrInvalidExposure{Field: "scenario", Reason: "required"}})
	_, err := client.Expose(context.Background(), connect.NewRequest(&exposurev1.ExposeRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestHandlerExposePortUnresolvedFailedPrecondition(t *testing.T) {
	client := newClient(t, &fakeService{exposeErr: internalexposure.ErrPortUnresolved{Scenario: "x", Reason: "ranged"}})
	_, err := client.Expose(context.Background(), connect.NewRequest(&exposurev1.ExposeRequest{Scenario: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestHandlerRevokeReportsRetracted(t *testing.T) {
	client := newClient(t, &fakeService{revoked: true})
	resp, err := client.RevokeLease(context.Background(), connect.NewRequest(&exposurev1.RevokeLeaseRequest{LeaseId: "l1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Retracted)
}

func TestHandlerIsExposed(t *testing.T) {
	client := newClient(t, &fakeService{isExposed: true, isURL: "https://x.itsagitime.com"})
	resp, err := client.IsExposed(context.Background(), connect.NewRequest(&exposurev1.IsExposedRequest{Scenario: "x"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Exposed)
	require.Equal(t, "https://x.itsagitime.com", resp.Msg.PublicUrl)
}

func TestHandlerReconcileCounts(t *testing.T) {
	client := newClient(t, &fakeService{coreEnsured: 3, reaped: 2})
	resp, err := client.Reconcile(context.Background(), connect.NewRequest(&exposurev1.ReconcileRequest{}))
	require.NoError(t, err)
	require.EqualValues(t, 3, resp.Msg.CoreEnsured)
	require.EqualValues(t, 2, resp.Msg.LeasesReaped)
}
