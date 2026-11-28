package audit_test

import (
	"context"
	"errors"
	"testing"

	"tunnel-manager/handlers/audit"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/audit/audit_v1connect"

	internalaudit "tunnel-manager/internal/audit"
)

// fakeService implements internalaudit.Service for handler tests.
type fakeService struct {
	out []internalaudit.PortAuditResult
	err error
}

func (f *fakeService) RunAudit(_ context.Context) ([]internalaudit.PortAuditResult, error) {
	return f.out, f.err
}

func newClient(t *testing.T, svc internalaudit.Service) auditconnect.AuditServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := auditconnect.NewAuditServiceHandler(audit.NewConnectHandler(audit.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return auditconnect.NewAuditServiceClient(server.Client(), server.URL)
}

func TestHandlerRunAuditMapsResultsAndCount(t *testing.T) {
	client := newClient(t, &fakeService{out: []internalaudit.PortAuditResult{
		{Subdomain: "web-console", Scenario: "web-console", ExpectedPort: 21233, ActualPort: 21233, Status: internalaudit.StatusCompliant},
		{Subdomain: "agent-manager", Scenario: "agent-manager", ExpectedPort: 99999, ActualPort: 21238, Status: internalaudit.StatusMismatch, Detail: "drift"},
		{Subdomain: "ghost", Scenario: "ghost", ExpectedPort: 21001, Status: internalaudit.StatusMissingScenario, Detail: "no service.json"},
		{Subdomain: "ranged", Scenario: "ranged", ExpectedPort: 21000, Status: internalaudit.StatusMissingPort},
	}})

	resp, err := client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 4)
	require.Equal(t, int32(3), resp.Msg.ViolationCount, "all non-compliant findings are counted")

	require.Equal(t, auditv1.AuditStatus_AUDIT_STATUS_COMPLIANT, resp.Msg.Results[0].Status)
	require.Equal(t, int32(21233), resp.Msg.Results[0].ExpectedPort)
	require.Equal(t, int32(21233), resp.Msg.Results[0].ActualPort)

	require.Equal(t, auditv1.AuditStatus_AUDIT_STATUS_MISMATCH, resp.Msg.Results[1].Status)
	require.Equal(t, int32(21238), resp.Msg.Results[1].ActualPort)
	require.Equal(t, "drift", resp.Msg.Results[1].Detail)

	require.Equal(t, auditv1.AuditStatus_AUDIT_STATUS_MISSING_SCENARIO, resp.Msg.Results[2].Status)
	require.Equal(t, auditv1.AuditStatus_AUDIT_STATUS_MISSING_PORT, resp.Msg.Results[3].Status)
}

func TestHandlerRunAuditEmpty(t *testing.T) {
	client := newClient(t, &fakeService{out: nil})
	resp, err := client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.Results)
	require.Equal(t, int32(0), resp.Msg.ViolationCount)
}

func TestHandlerRunAuditInternalError(t *testing.T) {
	client := newClient(t, &fakeService{err: errors.New("manifest read failed")})
	_, err := client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
