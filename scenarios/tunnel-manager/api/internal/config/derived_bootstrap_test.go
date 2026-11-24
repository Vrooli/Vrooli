package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// Exercise the real HTTP client and encrypted authority across an ambiguous
// provider response and a new service instance, without contacting Cloudflare.
func TestStartupReconcilesLostRotationResponseOverHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority.enc")
	authority, err := credentialauthority.OpenEncryptedFile(path, "test passphrase")
	if err != nil {
		t.Fatal(err)
	}
	id, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]string{"cloudflare-api-token": "fixture-api", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector"} {
		if err := authority.Put(id, field, value); err != nil {
			t.Fatal(err)
		}
	}
	connector := "old-connector"
	patches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-api" {
			t.Error("missing authority credential")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/user/tokens/verify":
			fmt.Fprint(w, `{"success":true}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-1/cfd_tunnel":
			fmt.Fprint(w, `{"success":true,"result":[{"id":"tunnel-1"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account-1/cfd_tunnel/tunnel-1/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": connector})
		case r.Method == http.MethodPatch && r.URL.Path == "/accounts/account-1/cfd_tunnel/tunnel-1":
			patches++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			payload, _ := json.Marshal(map[string]string{"a": "account-1", "t": "tunnel-1", "s": body["tunnel_secret"]})
			connector = base64.StdEncoding.EncodeToString(payload)
			// The mutation succeeds remotely but the response is lost.
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("server cannot close connection")
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	api := NewCloudflareBootstrapAPI(server.Client(), server.URL)
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true}); err == nil {
		t.Fatal("expected lost response")
	}
	reopened, err := credentialauthority.OpenEncryptedFile(path, "test passphrase")
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: reopened})
	if _, err := restarted.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored, err := reopened.Resolve(id, "cloudflare-connector-token")
	if err != nil || stored != connector {
		t.Fatal("replacement connector was not persisted")
	}
	if _, err := reopened.Resolve(id, rotationIntentField); !errors.Is(err, credentialauthority.ErrUnconfigured) {
		t.Fatal("rotation intent was not cleared")
	}
	if patches != 1 {
		t.Fatalf("provider received %d rotations, want 1", patches)
	}
}

type derivedBootstrapAuthority struct{ values map[string]string }

func (a *derivedBootstrapAuthority) Resolve(_ credentialauthority.Identity, field string) (string, error) {
	value := a.values[field]
	if value == "" {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (a *derivedBootstrapAuthority) Require(id credentialauthority.Identity, field string) (string, error) {
	return a.Resolve(id, field)
}

func (a *derivedBootstrapAuthority) Put(_ credentialauthority.Identity, field, value string) error {
	a.values[field] = value
	return nil
}

func (a *derivedBootstrapAuthority) Delete(_ credentialauthority.Identity, field string) error {
	delete(a.values, field)
	return nil
}
func (*derivedBootstrapAuthority) Status(credentialauthority.Identity, string) credentialauthority.Status {
	return credentialauthority.Status{}
}
func (*derivedBootstrapAuthority) Provider() string { return "test" }

type derivedBootstrapAPI struct {
	calls            int
	connectorErr     error
	onConnector      func()
	rotations        int
	rotationErr      error
	connector        string
	rotatedConnector string
	rotationSecrets  []string
}

func (a *derivedBootstrapAPI) VerifyToken(context.Context, string) error { a.calls++; return nil }
func (a *derivedBootstrapAPI) ListAccounts(context.Context, string) ([]CloudflareAccount, error) {
	a.calls++
	return []CloudflareAccount{{ID: "account-1"}}, nil
}

func (a *derivedBootstrapAPI) ListTunnels(context.Context, string, string) ([]CloudflareTunnel, error) {
	a.calls++
	return []CloudflareTunnel{{ID: "tunnel-1", Name: "vrooli"}}, nil
}

func (*derivedBootstrapAPI) CreateTunnel(context.Context, string, string, string) (CloudflareTunnel, error) {
	return CloudflareTunnel{}, errors.New("unexpected create")
}

func (a *derivedBootstrapAPI) ConnectorToken(context.Context, string, string, string) (string, error) {
	a.calls++
	if a.onConnector != nil {
		a.onConnector()
	}
	if a.connector != "" {
		return a.connector, a.connectorErr
	}
	return "connector-1", a.connectorErr
}
func (a *derivedBootstrapAPI) RotateConnectorSecret(_ context.Context, _, _, _, secret string) error {
	a.calls++
	a.rotations++
	a.rotationSecrets = append(a.rotationSecrets, secret)
	if a.rotationErr == nil && a.rotatedConnector != "" {
		payload, _ := json.Marshal(map[string]string{"a": "account-1", "t": "tunnel-1", "s": secret})
		a.connector = base64.StdEncoding.EncodeToString(payload)
	}
	return a.rotationErr
}

func TestRotationStoresReplacementAndClearsIntent(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector"}}
	api := &derivedBootstrapAPI{rotatedConnector: "connector-2"}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	result, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true})
	if err != nil || !result.Written || api.rotations != 1 || authority.values["cloudflare-connector-token"] != api.connector || authority.values[rotationIntentField] != "" {
		t.Fatalf("rotation failed: %v", err)
	}
}

func TestRotationIntentSurvivesAuthorityReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority.enc")
	authority, err := credentialauthority.OpenEncryptedFile(path, "test passphrase")
	if err != nil {
		t.Fatal(err)
	}
	id, err := credentialauthority.ParseIdentity(cloudflareCredentialID)
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector"} {
		if err := authority.Put(id, field, value); err != nil {
			t.Fatal(err)
		}
	}
	api := &derivedBootstrapAPI{rotationErr: errors.New("lost response")}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true}); err == nil {
		t.Fatal("expected ambiguous response")
	}
	reopened, err := credentialauthority.OpenEncryptedFile(path, "test passphrase")
	if err != nil {
		t.Fatal(err)
	}
	api.rotatedConnector = "yes"
	api.rotationErr = nil
	restarted := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: reopened})
	if _, err := restarted.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background()); err != nil {
		t.Fatal(err)
	}
	value, err := reopened.Resolve(id, "cloudflare-connector-token")
	if err != nil || value != api.connector {
		t.Fatal("restart did not recover the replacement connector")
	}
	if _, err := reopened.Resolve(id, rotationIntentField); !errors.Is(err, credentialauthority.ErrUnconfigured) {
		t.Fatal("restart did not clear intent")
	}
}

func TestPendingRotationPermitsOnlyAPITokenReplacement(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{rotationIntentField: "pending"}}
	store, err := NewCloudflareCredentialStore(CredentialStoreOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(Deps{BootstrapAuthority: authority, CredentialStore: store})
	_, _ = svc.SetCloudflareCredentials(context.Background(), CredentialUpdate{APIToken: "new-api-token"})
	if authority.values["cloudflare-api-token"] != "new-api-token" {
		t.Fatal("pending intent prevented API token replacement")
	}
	if _, err := svc.SetCloudflareCredentials(context.Background(), CredentialUpdate{TunnelID: "other"}); err == nil {
		t.Fatal("pending intent permitted tunnel replacement")
	}
}

func TestCredentialStatusExposesPendingRotation(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{rotationIntentField: "pending"}}
	store, err := NewCloudflareCredentialStore(CredentialStoreOptions{Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(Deps{BootstrapAuthority: authority, CredentialStore: store})
	status, err := svc.GetCredentialStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, field := range status.MissingFields {
		if field == "CLOUDFLARE_CONNECTOR_ROTATION_PENDING" {
			found = true
		}
	}
	if status.Ready || !found {
		t.Fatal("pending rotation was not visible in credential readiness")
	}
}

func TestStartupReconcilesIntentPersistedBeforeDispatch(t *testing.T) {
	intent, err := json.Marshal(rotationIntent{AccountID: "account-1", TunnelID: "tunnel-1", PreviousDigest: connectorDigest("connector-1"), Secret: "previously-generated-secret"})
	if err != nil {
		t.Fatal(err)
	}
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "connector-1", rotationIntentField: string(intent)}}
	api := &derivedBootstrapAPI{rotatedConnector: "connector-2"}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	_, err = svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil || api.rotations != 1 || api.rotationSecrets[0] != "previously-generated-secret" || authority.values["cloudflare-connector-token"] != api.connector || authority.values[rotationIntentField] != "" {
		t.Fatal("startup failed to reconcile pre-dispatch intent")
	}
}

func TestPendingRecoveryRefusesIndependentCloudflareChange(t *testing.T) {
	intent, _ := json.Marshal(rotationIntent{AccountID: "account-1", TunnelID: "tunnel-1", PreviousDigest: connectorDigest("connector-1"), Secret: "desired-secret"})
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "connector-1", rotationIntentField: string(intent)}}
	api := &derivedBootstrapAPI{connector: "external-change"}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true}); err == nil || api.rotations != 0 {
		t.Fatal("independent provider change was overwritten")
	}
	if authority.values[rotationIntentField] == "" {
		t.Fatal("independent provider change cleared intent")
	}
}

func TestRotationDryRunAndUncertainOutcomeDoNotOverwriteConnector(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector"}}
	api := &derivedBootstrapAPI{rotationErr: errors.New("connection lost")}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	_, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true, DryRun: true})
	if err != nil || api.rotations != 0 {
		t.Fatalf("dry-run rotated or failed: %v", err)
	}
	_, err = svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true})
	if err == nil || api.rotations != 1 || authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatal("uncertain rotation failed to preserve stored connector")
	}
	api.rotationErr = nil
	if _, err = svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true}); err == nil {
		t.Fatal("pending intent permitted another rotation")
	}
	api.rotatedConnector = "yes"
	_, err = svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil || api.rotations != 2 || authority.values["cloudflare-connector-token"] != api.connector || authority.values[rotationIntentField] != "" {
		t.Fatal("refresh did not recover without another rotation")
	}
	if api.rotationSecrets[0] != api.rotationSecrets[1] {
		t.Fatal("recovery did not reuse the same desired secret")
	}
}

func TestRotationRejectsUnchangedConnector(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1", "cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector"}}
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	_, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, RotateConnectorToken: true})
	if err == nil || authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatal("unchanged provider token was accepted")
	}
}

func TestStoredCredentialRefreshRejectsChangedIdentity(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{
		"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1",
		"cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector",
	}}
	api := &derivedBootstrapAPI{onConnector: func() { authority.values["cloudflare-tunnel-id"] = "different" }}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true}); err == nil {
		t.Fatal("refresh accepted credentials changed during fetch")
	}
	if authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatal("refresh replaced connector for stale identity")
	}
}

func TestStoredCredentialRefreshPreservesConnectorWhenFetchFails(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{
		"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1",
		"cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector",
	}}
	svc := NewService(Deps{BootstrapAPI: &derivedBootstrapAPI{connectorErr: errors.New("unavailable")}, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true}); err == nil {
		t.Fatal("refresh accepted a failed fetch")
	}
	if authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatal("failed fetch changed the stored connector")
	}
}

func TestBootstrapConfiguredCloudflareIsOwnerLifecycleSeam(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token"}}
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})

	result, err := svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil {
		t.Fatalf("BootstrapConfiguredCloudflare() error = %v", err)
	}
	if !result.Written || result.AccountID != "account-1" || result.TunnelID != "tunnel-1" {
		t.Fatalf("bootstrap result = %+v", result)
	}
	for field, want := range map[string]string{
		"cloudflare-api-token":       "api-token",
		"cloudflare-account-id":      "account-1",
		"cloudflare-tunnel-id":       "tunnel-1",
		"cloudflare-connector-token": "connector-1",
	} {
		if authority.values[field] != want {
			t.Fatalf("authority[%q] = %q, want %q", field, authority.values[field], want)
		}
	}
}

func TestBootstrapConfiguredCloudflareMissingOperatorCredentialIsNoop(t *testing.T) {
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: &derivedBootstrapAuthority{values: map[string]string{}}})
	result, err := svc.(DerivedCredentialBootstrapper).BootstrapConfiguredCloudflare(context.Background())
	if err != nil {
		t.Fatalf("BootstrapConfiguredCloudflare() error = %v", err)
	}
	if result.Written || api.calls != 0 {
		t.Fatalf("missing operator credential caused bootstrap: result=%+v calls=%d", result, api.calls)
	}
}

func TestStoredCredentialRefreshTargetsOnlyConfiguredTunnel(t *testing.T) {
	authority := &derivedBootstrapAuthority{values: map[string]string{
		"cloudflare-api-token": "api-token", "cloudflare-account-id": "account-1",
		"cloudflare-tunnel-id": "tunnel-1", "cloudflare-connector-token": "old-connector",
	}}
	api := &derivedBootstrapAPI{}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, TunnelID: "other"}); err == nil {
		t.Fatal("stored credential refresh accepted a tunnel override")
	}
	if api.calls != 0 || authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatal("rejected refresh made a remote call or changed the connector")
	}
	preview, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true, DryRun: true})
	if err != nil || preview.Written || authority.values["cloudflare-connector-token"] != "old-connector" {
		t.Fatalf("dry-run result = %+v, error = %v", preview, err)
	}
	result, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true})
	if err != nil || !result.Written || result.TunnelID != "tunnel-1" {
		t.Fatalf("refresh result = %+v, error = %v", result, err)
	}
	if authority.values["cloudflare-connector-token"] != "connector-1" {
		t.Fatal("refresh did not store the returned connector")
	}
}

func TestStoredCredentialRefreshRequiresConfiguredIdentity(t *testing.T) {
	api := &derivedBootstrapAPI{}
	authority := &derivedBootstrapAuthority{values: map[string]string{"cloudflare-api-token": "api-token"}}
	svc := NewService(Deps{BootstrapAPI: api, BootstrapAuthority: authority})
	if _, err := svc.BootstrapCloudflare(context.Background(), BootstrapRequest{UseStoredAPIToken: true}); err == nil {
		t.Fatal("refresh accepted missing tunnel coordinates")
	}
	if api.calls != 0 {
		t.Fatal("refresh contacted Cloudflare without configured coordinates")
	}
}
