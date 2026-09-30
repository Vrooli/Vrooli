package monetization

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type sessionClient struct{ value string }

func (c *sessionClient) Provision(_ context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	c.value = request.Value
	return credentialclient.ProvisionResponse{Identity: request.Identity, Field: request.Field}, nil
}
func (c *sessionClient) Resolve(context.Context, string, string) (string, error) { return c.value, nil }
func (c *sessionClient) Delete(context.Context, string, string) error            { c.value = ""; return nil }
func (c *sessionClient) Status(context.Context, string, string) (credentialclient.CredentialStatus, error) {
	return credentialclient.CredentialStatus{Configured: c.value != ""}, nil
}

func (c *sessionClient) List(context.Context) ([]credentialclient.CredentialRef, error) {
	return nil, nil
}

func (c *sessionClient) Doctor(context.Context) (credentialclient.DoctorResponse, error) {
	return credentialclient.DoctorResponse{}, nil
}

func (c *sessionClient) KeyringInspect(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}

func (c *sessionClient) KeyringRepair(context.Context, string) (credentialclient.KeyringReport, error) {
	return credentialclient.KeyringReport{}, nil
}

func (c *sessionClient) RecoveryExport(context.Context, credentialclient.RecoveryExportRequest) (credentialclient.RecoveryExportResponse, error) {
	return credentialclient.RecoveryExportResponse{}, nil
}

func (c *sessionClient) RecoveryVerify(context.Context, credentialclient.RecoveryVerifyRequest) (credentialclient.RecoveryVerifyResponse, error) {
	return credentialclient.RecoveryVerifyResponse{}, nil
}

func (c *sessionClient) RecoveryRestore(context.Context, credentialclient.RecoveryRestoreRequest) error {
	return nil
}

func (c *sessionClient) StoreStatus(context.Context) (credentialclient.StoreStatus, error) {
	return credentialclient.StoreStatus{}, nil
}

func TestSessionModuleStoresRefreshTokenWithoutReturningIt(t *testing.T) {
	module := NewSessionModule(&sessionClient{}, "vrooli/account", "refresh-token")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"refresh_token":"secret"}`))
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	module.Provision(rec, req)
	body, _ := io.ReadAll(rec.Result().Body)
	if rec.Code != http.StatusCreated || strings.Contains(string(body), "secret") {
		t.Fatalf("status=%d body=%s", rec.Code, body)
	}
}
