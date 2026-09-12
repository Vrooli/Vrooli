package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/credentialsvc"
	"scenario-to-cloud/domain"

	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/credentials/credentialsv1connect"
)

func TestBridgeOwnerTokenUsesTheSharedBridgeAPIConfiguration(t *testing.T) {
	for _, name := range []string{"VROOLI_BRIDGE_API_TOKEN", "VROOLI_BRIDGE_TOKEN", "VROOLI_API_TOKEN"} {
		t.Setenv(name, "")
	}
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "bridge-api-token")

	got, err := bridgeOwnerToken(context.Background())
	if err != nil || got != "bridge-api-token" {
		t.Fatalf("bridge owner token = %q, %v", got, err)
	}
}

// fakeCredentialLifecycle records requests and answers with scripted
// operations, so the routes are tested for mapping and status codes.
type fakeCredentialLifecycle struct {
	seenValue      string
	seenPassphrase string
	rotation       *domain.CredentialRotation
	err            error
}

func (f *fakeCredentialLifecycle) ListBindings(_ context.Context, deploymentID string) ([]credentials.BindingView, error) {
	descriptor := domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "password"}
	return []credentials.BindingView{{Binding: domain.CredentialBinding{ID: credentials.BindingID(deploymentID, descriptor), DeploymentID: deploymentID, Descriptor: descriptor, Class: domain.CredentialClassGeneratedDatabasePassword, Version: domain.CredentialVersion{Number: 2, ContentRef: "cref_x", CreatedAt: time.Now()}, ConsumerRefs: []string{"scenario:app"}, State: domain.CredentialBindingMaterialized, CreatedAt: time.Now(), UpdatedAt: time.Now()}, Acks: []domain.CredentialAck{}}}, nil
}

func (f *fakeCredentialLifecycle) Rotate(_ context.Context, req credentialsvc.RotateRequest) (*domain.CredentialRotation, error) {
	f.seenValue = req.Value
	return f.rotation, f.err
}

func (f *fakeCredentialLifecycle) Revoke(context.Context, credentialsvc.RevokeRequest) (*domain.CredentialRotation, error) {
	return f.rotation, f.err
}

func (f *fakeCredentialLifecycle) Recover(_ context.Context, req credentialsvc.RecoverRequest) (*domain.CredentialRotation, error) {
	f.seenPassphrase = req.Passphrase
	return f.rotation, f.err
}

func (f *fakeCredentialLifecycle) GetRotation(context.Context, string, string) (*domain.CredentialRotation, error) {
	return f.rotation, f.err
}

func (f *fakeCredentialLifecycle) Resume(context.Context, credentialsvc.ResumeRequest) (*domain.CredentialRotation, error) {
	return f.rotation, f.err
}

func (f *fakeCredentialLifecycle) BreakGlass(context.Context, credentialsvc.BreakGlassRequest) (*domain.CredentialRotation, error) {
	return f.rotation, f.err
}

func credentialTestServer(t *testing.T, fake *fakeCredentialLifecycle) (*Server, *httptest.Server) {
	t.Helper()
	credentialLifecycleOverride = fake
	t.Cleanup(func() { credentialLifecycleOverride = nil })
	srv := newTestServer()
	srv.authz = newTestEnforcer(srv, testOperator())
	srv.setupRoutes()
	srv.registerCredentialRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return srv, ts
}

func sampleRotation(state domain.CredentialRotationState) *domain.CredentialRotation {
	now := time.Now().UTC()
	return &domain.CredentialRotation{ID: "op-1", DeploymentID: "dep-1", BindingID: "cb_1", Kind: domain.CredentialOperationRotate, FromVersion: 1, ToVersion: 2, State: state, Consumers: []domain.CredentialConsumerProgress{{Consumer: "scenario:app", State: domain.ConsumerAcknowledged, Version: 2, UpdatedAt: now}}, Receipts: []domain.CredentialReceipt{{Step: "new-version", State: "new_version_created", Outcome: "created", At: now, Details: map[string]any{"version": int64(2)}}}, CreatedAt: now, UpdatedAt: now}
}

// [REQ:STC-P0-032] REST and Connect share one lifecycle: a complete
// operation is 200, an incomplete one 202, a typed refusal carries the
// operation, and inbound values never appear in any response.
func TestCredentialRoutesShareLifecycleAndNeverEchoValues(t *testing.T) {
	fake := &fakeCredentialLifecycle{rotation: sampleRotation(domain.RotationComplete)}
	_, ts := credentialTestServer(t, fake)
	client := ts.Client()
	post := func(path, body string) (*http.Response, string) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp, string(data)
	}
	resp, body := post("/api/v1/deployments/dep-1/credentials/cb_1/rotate", `{"value":"canary-rest-9a8b7c"}`)
	if resp.StatusCode != http.StatusOK || strings.Contains(body, "canary-rest-9a8b7c") || !strings.Contains(body, `"state":"complete"`) {
		t.Fatalf("rotate = %d %s", resp.StatusCode, body)
	}
	if fake.seenValue != "canary-rest-9a8b7c" {
		t.Fatal("value did not reach the lifecycle")
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("credential route is cacheable: %q", resp.Header.Get("Cache-Control"))
	}

	fake.rotation = sampleRotation(domain.RotationConsumersUpdated)
	fake.rotation.Unreached = []string{"scenario:app"}
	resp, body = post("/api/v1/deployments/dep-1/credentials/cb_1/rotate", `{}`)
	if resp.StatusCode != http.StatusAccepted || !strings.Contains(body, `"unreached":["scenario:app"]`) {
		t.Fatalf("incomplete rotate = %d %s", resp.StatusCode, body)
	}

	fake.rotation = sampleRotation(domain.RotationRevocationIncomplete)
	fake.err = apierrors.New(credentials.CodeRevocationIncomplete, "target unreachable").WithDetail("unreached", []string{"host:203.0.113.10"})
	resp, body = post("/api/v1/deployments/dep-1/credentials/cb_1/revoke", `{}`)
	if resp.StatusCode != http.StatusAccepted || !strings.Contains(body, `"code":"revocation_incomplete"`) || !strings.Contains(body, `"operation"`) {
		t.Fatalf("incomplete revoke = %d %s", resp.StatusCode, body)
	}
	fake.err = nil

	fake.rotation = sampleRotation(domain.RotationComplete)
	fake.rotation.Kind = domain.CredentialOperationRecover
	resp, body = post("/api/v1/deployments/dep-1/credentials/recover", `{"bundle_ref":"/root/recovery.bundle","passphrase":"canary-pass-1122"}`)
	if resp.StatusCode != http.StatusOK || strings.Contains(body, "canary-pass-1122") || fake.seenPassphrase != "canary-pass-1122" {
		t.Fatalf("recover = %d %s", resp.StatusCode, body)
	}

	resp, err := client.Get(ts.URL + "/api/v1/deployments/dep-1/credentials")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		SchemaVersion string                    `json:"schema_version"`
		Bindings      []credentials.BindingView `json:"bindings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil || resp.StatusCode != http.StatusOK || len(list.Bindings) != 1 || list.Bindings[0].Binding.Descriptor.Field != "password" {
		t.Fatalf("list = %d %+v err=%v", resp.StatusCode, list, err)
	}
	resp.Body.Close()

	resp, err = client.Get(ts.URL + "/api/v1/deployments/dep-1/credentials/rotations/op-1")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get rotation = %v %d", err, resp.StatusCode)
	}
	resp.Body.Close()

	// Connect shares the lifecycle and the typed code.
	cc := credentialsv1connect.NewCredentialsServiceClient(client, ts.URL)
	got, err := cc.RotateCredential(context.Background(), connect.NewRequest(&credentialsv1.RotateCredentialRequest{DeploymentId: "dep-1", BindingId: "cb_1", Value: "canary-connect-3344"}))
	if err != nil || got.Msg.GetOperation().GetState() != "complete" || got.Msg.GetSchemaVersion() != credentialsvc.SchemaVersion {
		t.Fatalf("connect rotate = %v %v", got, err)
	}
	raw, _ := json.Marshal(got.Msg)
	if strings.Contains(string(raw), "canary-connect-3344") {
		t.Fatal("connect response echoes the value")
	}
	fake.err = apierrors.New(credentials.CodeStoreLocked, "locked")
	_, err = cc.RotateCredential(context.Background(), connect.NewRequest(&credentialsv1.RotateCredentialRequest{DeploymentId: "dep-1", BindingId: "cb_1"}))
	if err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("connect locked = %v", err)
	}
}
