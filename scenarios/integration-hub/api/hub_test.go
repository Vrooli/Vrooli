package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

type connectorManifest struct {
	ID   string `json:"id"`
	Auth struct {
		Kind  string `json:"kind"`
		Field string `json:"field"`
	} `json:"auth"`
	Handler      string `json:"handler"`
	MetadataOnly bool   `json:"metadata_only"`
}

func TestOpenRouterConnectorManifestMatchesPilot(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("..", "connectors", "openrouter", "connector.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest connectorManifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ID != openRouterConnector || manifest.Auth.Kind != "api_key" || manifest.Auth.Field != credentialField || manifest.Handler != "openrouter_api_key" || !manifest.MetadataOnly {
		t.Fatalf("manifest does not describe the pilot: %+v", manifest)
	}
}

type fakeCredentials struct {
	values      map[string]string
	unavailable bool
	provisions  int
}

func (f *fakeCredentials) Provision(_ context.Context, req CredentialProvisionRequest) error {
	if f.unavailable {
		return errors.New("unavailable")
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[req.Identity+":"+req.Field] = req.Value
	f.provisions++
	return nil
}
func (f *fakeCredentials) Status(_ context.Context, identity, field string) (CredentialStatus, error) {
	if f.unavailable {
		return CredentialStatus{ProviderState: "unavailable"}, nil
	}
	_, ok := f.values[identity+":"+field]
	return CredentialStatus{Configured: ok, ProviderState: "available"}, nil
}
func (f *fakeCredentials) Delete(_ context.Context, identity, field string) error {
	if f.unavailable {
		return errors.New("unavailable")
	}
	delete(f.values, identity+":"+field)
	return nil
}

func testHub(t *testing.T) (*Hub, *fakeCredentials) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	creds := &fakeCredentials{values: map[string]string{}}
	hub := NewHub(store, creds)
	hub.now = func() time.Time { return time.Date(2026, 9, 3, 5, 0, 0, 0, time.UTC) }
	return hub, creds
}
func authed[T any](msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set(identityHeader, "user@example.test")
	return req
}

func TestConnectionLifecycleIsMetadataOnlyAndIdempotent(t *testing.T) {
	hub, creds := testHub(t)
	created, err := hub.CreateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", ConnectorId: openRouterConnector, DisplayName: "Team OpenRouter", CredentialValue: "super-secret", RequestId: "create-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if created.Msg.GetConnection().GetCredentialAuthorityRef() == "" || created.Msg.GetConnection().GetDisplayName() != "Team OpenRouter" {
		t.Fatalf("unexpected metadata: %v", created.Msg.GetConnection())
	}
	if created.Msg.GetConnection().GetAccountIdentity() != "user@example.test" {
		t.Fatal("account identity missing")
	}
	encoded := created.Msg.String()
	if strings.Contains(encoded, "super-secret") {
		t.Fatal("secret leaked into response")
	}
	repeated, err := hub.CreateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", ConnectorId: openRouterConnector, CredentialValue: "different-secret", RequestId: "create-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Msg.GetConnection().GetId() != "openrouter-main" || creds.provisions != 1 {
		t.Fatal("request id was not idempotent")
	}
	probed, err := hub.ProbeConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main"}))
	if err != nil {
		t.Fatal(err)
	}
	if probed.Msg.GetConnection().GetStatus() != commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED {
		t.Fatal("probe did not report connected")
	}
	bound, err := hub.BindConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", BindingScenarioSlug: "web-console", BindingContext: "default"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(bound.Msg.GetConnection().GetBindings()) != 1 {
		t.Fatal("binding not persisted")
	}
	if _, err := hub.BindConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", BindingScenarioSlug: "blocked-scenario", RequiredScopes: []string{"write-models"}})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing scope code = %v", connect.CodeOf(err))
	}
	if _, err := hub.GetConnection(context.Background(), authed(&commonv1.GetConnectionRequest{ConnectionId: "openrouter-main"})); err != nil {
		t.Fatal(err)
	}
	rotated, err := hub.RotateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", CredentialValue: "rotated-secret", RequestId: "rotate-1"}))
	if err != nil || rotated.Msg.GetConnection().GetStatus() != commonv1.ConnectionStatus_CONNECTION_STATUS_CONNECTED || creds.provisions != 2 {
		t.Fatalf("rotation failed: err=%v response=%v provisions=%d", err, rotated, creds.provisions)
	}
	repeatedRotate, err := hub.RotateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", CredentialValue: "third-secret", RequestId: "rotate-1"}))
	if err != nil || repeatedRotate.Msg.GetConnection().GetId() != "openrouter-main" || creds.provisions != 2 {
		t.Fatalf("rotation replay was not idempotent: err=%v provisions=%d", err, creds.provisions)
	}
	revoked, err := hub.RevokeConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", RequestId: "revoke-1"}))
	if err != nil || revoked.Msg.GetConnection().GetStatus() != commonv1.ConnectionStatus_CONNECTION_STATUS_REVOKED {
		t.Fatalf("revoke failed: err=%v response=%v", err, revoked)
	}
	repeatedRevoke, err := hub.RevokeConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", RequestId: "revoke-1"}))
	if err != nil || repeatedRevoke.Msg.GetConnection().GetStatus() != commonv1.ConnectionStatus_CONNECTION_STATUS_REVOKED {
		t.Fatalf("revoke replay was not idempotent: err=%v response=%v", err, repeatedRevoke)
	}
	deleted, err := hub.DeleteConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", RequestId: "delete-1"}))
	if err != nil || deleted.Msg.GetConnection().GetId() != "openrouter-main" {
		t.Fatalf("delete failed: err=%v response=%v", err, deleted)
	}
	repeatedDelete, err := hub.DeleteConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", RequestId: "delete-1"}))
	if err != nil || repeatedDelete.Msg.GetConnection().GetId() != "openrouter-main" {
		t.Fatalf("delete replay was not idempotent: err=%v response=%v", err, repeatedDelete)
	}
}

func TestConnectionAuthorizationAndProviderFailure(t *testing.T) {
	hub, creds := testHub(t)
	_, err := hub.ListConnections(context.Background(), connect.NewRequest(&commonv1.ListConnectionsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated code = %v", connect.CodeOf(err))
	}
	_, err = hub.CreateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", ConnectorId: openRouterConnector, CredentialValue: "secret"}))
	if err != nil {
		t.Fatal(err)
	}
	other := connect.NewRequest(&commonv1.GetConnectionRequest{ConnectionId: "openrouter-main"})
	other.Header().Set(identityHeader, "other@example.test")
	if _, err := hub.GetConnection(context.Background(), other); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-owner code = %v", connect.CodeOf(err))
	}
	creds.unavailable = true
	result, err := hub.ProbeConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main"}))
	if err != nil {
		t.Fatal(err)
	}
	if result.Msg.GetConnection().GetStatus() != commonv1.ConnectionStatus_CONNECTION_STATUS_PROVIDER_UNAVAILABLE {
		t.Fatal("provider failure was not mapped safely")
	}
}

func TestDurableStateContainsNoCredentialValue(t *testing.T) {
	hub, _ := testHub(t)
	_, err := hub.CreateConnection(context.Background(), authed(&commonv1.ConnectionMutationRequest{ConnectionId: "openrouter-main", ConnectorId: openRouterConnector, CredentialValue: "secret-not-persisted"}))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(hub.store.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-not-persisted") {
		t.Fatal("secret persisted in hub state")
	}
}

func TestGeneratedConnectClientUsesMetadataOnlyWireContract(t *testing.T) {
	hub, _ := testHub(t)
	path, serviceHandler := commonv1connect.NewConnectionServiceHandler(hub)
	mux := http.NewServeMux()
	mux.Handle(path, serviceHandler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := commonv1connect.NewConnectionServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: "wire", ConnectorId: openRouterConnector, CredentialValue: "wire-secret"})
	request.Header().Set(identityHeader, "wire-user@example.test")
	response, err := client.CreateConnection(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetConnection().GetCredentialAuthorityRef() == "" || strings.Contains(response.Msg.String(), "wire-secret") {
		t.Fatalf("wire response contains unsafe or missing metadata: %s", response.Msg)
	}
}

func TestBearerIdentityIsHashedBeforeDurableMetadata(t *testing.T) {
	hub, _ := testHub(t)
	request := connect.NewRequest(&commonv1.ConnectionMutationRequest{ConnectionId: "bearer", ConnectorId: openRouterConnector, CredentialValue: "secret"})
	request.Header().Set("Authorization", "Bearer bearer-token-that-must-not-persist")
	response, err := hub.CreateConnection(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetConnection().GetAccountIdentity() != "authenticated-user" || strings.Contains(response.Msg.String(), "bearer-token-that-must-not-persist") {
		t.Fatalf("bearer identity leaked: %s", response.Msg)
	}
	raw, err := os.ReadFile(hub.store.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "bearer-token-that-must-not-persist") {
		t.Fatal("bearer token persisted in hub metadata")
	}
}
