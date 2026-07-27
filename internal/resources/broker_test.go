package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type testCredentialIssuer struct {
	called bool
}

func (i *testCredentialIssuer) IssueScopedCredential(instance ManagedInstance, lease Lease) (ScopedCredential, error) {
	i.called = true
	return ScopedCredential{LeaseID: lease.ID, Resource: instance.Resource, Scope: lease.Scope, ExpiresAt: lease.ExpiresAt, Credential: fmt.Sprintf("token-for-%s", lease.Scope)}, nil
}

func TestBrokerSharesOnlyVerifiedAuthorizedInstance(t *testing.T) {
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	broker := NewBroker(func() time.Time { return now })
	if err := broker.Register(ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200", AuthorizedScopes: []string{"app:one", "app:two"}}); err != nil {
		t.Fatal(err)
	}
	lease, err := broker.Acquire("vault", "app:one", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.AuthorizeUse(lease.ID, "vault", "app:two"); err == nil {
		t.Fatal("lease must not cross application scopes")
	}
	if _, err := broker.AuthorizeManagement("vault-user", "app:one"); err == nil {
		t.Fatal("lease holder must not manage shared service")
	}
	if _, err := broker.AuthorizeUse(lease.ID, "vault", "app:one"); err != nil {
		t.Fatalf("authorized use: %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := broker.AuthorizeUse(lease.ID, "vault", "app:one"); err == nil {
		t.Fatal("expired lease must be rejected")
	}
}

func TestBrokerRejectsAttachOnlyRegistration(t *testing.T) {
	broker := NewBroker(nil)
	err := broker.Register(ManagedInstance{ID: "external", Resource: "vault", Provider: resourcedeployment.ProviderAttachOnly, OwnerScope: "org", CapabilityVersion: "v1", Endpoint: "https://vault.example"})
	if err == nil {
		t.Fatal("external endpoint must remain attach-only")
	}
}

func TestBrokerIssuesCredentialOnlyForAuthorizedLeaseScope(t *testing.T) {
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	broker := NewBroker(func() time.Time { return now })
	if err := broker.Register(ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200", AuthorizedScopes: []string{"app:one"}}); err != nil {
		t.Fatal(err)
	}
	lease, err := broker.Acquire("vault", "app:one", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	issuer := &testCredentialIssuer{}
	credential, err := broker.IssueScopedCredential(lease.ID, "vault", "app:one", issuer)
	if err != nil || !issuer.called || credential.Credential == "" {
		t.Fatalf("IssueScopedCredential() = %#v, %v", credential, err)
	}
	issuer.called = false
	if _, err := broker.IssueScopedCredential(lease.ID, "vault", "app:two", issuer); err == nil || issuer.called {
		t.Fatalf("cross-scope credential issue = %v, issuer called=%v", err, issuer.called)
	}
}

func TestPersistentBrokerRestoresVerifiedOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "broker.json")
	broker, err := NewPersistentBroker(nil, FileBrokerStore{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if err := broker.Register(ManagedInstance{ID: "vault-private", Resource: "vault", Provider: resourcedeployment.ProviderManagedPrivate, OwnerScope: "app:one", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200"}); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewPersistentBroker(nil, FileBrokerStore{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.AuthorizeManagement("vault-private", "app:one"); err != nil {
		t.Fatalf("restored ownership: %v", err)
	}
}

func TestBrokerControlTransportBindsScopeToPerAppCredential(t *testing.T) {
	broker := NewBroker(nil)
	if err := broker.Register(ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200", AuthorizedScopes: []string{"app:one", "app:two"}}); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := StartBrokerControlServer(listener, broker, map[string]string{"app:one": "token-one", "app:two": "token-two", "user:alice": "owner-token"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close(context.Background()) })
	endpoint, err := BrokerControlEndpoint(listener)
	if err != nil {
		t.Fatal(err)
	}
	appOne, err := NewBrokerControlClient(BrokerControlCredential{Endpoint: endpoint, Scope: "app:one", Token: "token-one"})
	if err != nil {
		t.Fatal(err)
	}
	appTwo, err := NewBrokerControlClient(BrokerControlCredential{Endpoint: endpoint, Scope: "app:two", Token: "token-two"})
	if err != nil {
		t.Fatal(err)
	}
	lease, err := appOne.Acquire(context.Background(), "vault", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := appOne.AuthorizeUse(context.Background(), lease.ID, "vault"); err != nil {
		t.Fatalf("authorized app use: %v", err)
	}
	if err := server.RegisterCredentialIssuer("vault", &testCredentialIssuer{}); err != nil {
		t.Fatal(err)
	}
	var managedAction string
	if err := server.RegisterOwnerLifecycle("vault", OwnerLifecycleFunc(func(_ context.Context, instance ManagedInstance, action string) (any, error) {
		managedAction = action
		return map[string]string{"instance": instance.ID, "action": action}, nil
	})); err != nil {
		t.Fatal(err)
	}
	credential, err := appOne.IssueScopedCredential(context.Background(), lease.ID, "vault")
	if err != nil || credential.Scope != "app:one" || credential.Credential == "" {
		t.Fatalf("scoped credential = %#v, %v", credential, err)
	}
	if _, err := appTwo.AuthorizeUse(context.Background(), lease.ID, "vault"); err == nil {
		t.Fatal("a second app credential must not claim app one's lease")
	}
	if _, err := appTwo.IssueScopedCredential(context.Background(), lease.ID, "vault"); err == nil {
		t.Fatal("a second app credential must not receive app one's scoped credential")
	}
	if _, err := appOne.AuthorizeManagement(context.Background(), "vault-user"); err == nil {
		t.Fatal("lease credential must not gain management authority")
	}
	if _, err := appOne.Manage(context.Background(), "vault-user", "stop"); err == nil {
		t.Fatal("lease credential must not execute lifecycle action")
	}
	owner, err := NewBrokerControlClient(BrokerControlCredential{Endpoint: endpoint, Scope: "user:alice", Token: "owner-token"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.AuthorizeManagement(context.Background(), "vault-user"); err != nil {
		t.Fatalf("registered owner management authorization: %v", err)
	}
	result, err := owner.Manage(context.Background(), "vault-user", "restart")
	if err != nil || managedAction != "restart" || result["instance"] != "vault-user" {
		t.Fatalf("owner lifecycle action = %#v, %v; action=%q", result, err, managedAction)
	}
}

func TestBrokerControlClientRejectsNonLoopbackEndpoint(t *testing.T) {
	if _, err := NewBrokerControlClient(BrokerControlCredential{Endpoint: "http://broker.example:8080", Scope: "app:one", Token: "token"}); err == nil {
		t.Fatal("non-loopback control endpoint must be rejected")
	}
}

func TestBrokerRejectsNonLoopbackManagedRegistration(t *testing.T) {
	broker := NewBroker(nil)
	err := broker.Register(ManagedInstance{ID: "remote", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "https://vault.example", AuthorizedScopes: []string{"app:one"}})
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("non-loopback managed registration = %v", err)
	}
}

func TestBrokerRequiresCurrentOwnershipProofWhenVerifierIsConfigured(t *testing.T) {
	broker := NewBroker(nil)
	instance := ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200", AuthorizedScopes: []string{"app:one"}}
	broker.SetOwnershipVerifier(func(ManagedInstance) error { return fmt.Errorf("stale attestation") })
	if err := broker.Register(instance); err == nil || !strings.Contains(err.Error(), "attestation") {
		t.Fatalf("registration without current ownership proof = %v", err)
	}

	broker.SetOwnershipVerifier(nil)
	if err := broker.Register(instance); err != nil {
		t.Fatal(err)
	}
	broker.SetOwnershipVerifier(func(ManagedInstance) error { return fmt.Errorf("replayed attestation") })
	if _, err := broker.Acquire("vault", "app:one", time.Minute); err == nil || !strings.Contains(err.Error(), "verified shared") {
		t.Fatalf("lease issued from replayed ownership proof = %v", err)
	}
}

func TestBrokerRefreshesAttestationAfterOwnerRestart(t *testing.T) {
	broker := NewBroker(nil)
	instance := ManagedInstance{
		ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared,
		OwnerScope: "user:alice", CapabilityVersion: "v1", Endpoint: "http://127.0.0.1:8200",
		AuthorizedScopes: []string{"app:one"}, Attestation: OwnershipAttestation{Proof: "old"},
	}
	broker.SetOwnershipVerifier(func(candidate ManagedInstance) error {
		if candidate.Attestation.Proof != "new" {
			return fmt.Errorf("attestation belongs to stopped process")
		}
		return nil
	})
	if err := broker.Register(instance); err == nil {
		t.Fatal("stale first-process attestation was accepted")
	}

	// Simulate the record created while the original process was alive, then
	// require a fresh proof when its replacement starts.
	broker.SetOwnershipVerifier(nil)
	if err := broker.Register(instance); err != nil {
		t.Fatal(err)
	}
	broker.SetOwnershipVerifier(func(candidate ManagedInstance) error {
		if candidate.Attestation.Proof != "new" {
			return fmt.Errorf("attestation belongs to stopped process")
		}
		return nil
	})
	instance.Attestation.Proof = "new"
	refreshed, err := broker.RegisterOrGrantScope(instance, "app:one")
	if err != nil {
		t.Fatalf("refresh owner attestation: %v", err)
	}
	if refreshed.Attestation.Proof != "new" {
		t.Fatalf("broker retained stale attestation: %#v", refreshed.Attestation)
	}
	if _, err := broker.Acquire("vault", "app:one", time.Minute); err != nil {
		t.Fatalf("freshly attested restarted service was not leasable: %v", err)
	}
}

func TestVaultCredentialIssuerCreatesLeaseBoundScopedToken(t *testing.T) {
	var policy string
	var tokenRequest struct {
		Policies []string `json:"policies"`
		TTL      string   `json:"ttl"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Vault-Token") != "management-token" {
			t.Errorf("management token header = %q", request.Header.Get("X-Vault-Token"))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch request.URL.Path {
		case "/v1/sys/policies/acl/vrooli-app-e50bfda9a5ca5ac46cd52ddbbb53906d":
			var input map[string]string
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			policy = input["policy"]
			w.WriteHeader(http.StatusNoContent)
		case "/v1/auth/token/create":
			if err := json.NewDecoder(request.Body).Decode(&tokenRequest); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"auth": map[string]string{"client_token": "app-one-token"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	issuer := VaultCredentialIssuer{Now: func() time.Time { return now }, ManagementToken: func(ManagedInstance) (string, error) { return "management-token", nil }}
	lease := Lease{ID: "lease-1", Scope: "app:one", ExpiresAt: now.Add(2 * time.Minute)}
	credential, err := issuer.IssueScopedCredential(ManagedInstance{Resource: "vault", Endpoint: server.URL}, lease)
	if err != nil {
		t.Fatal(err)
	}
	if credential.Credential != "app-one-token" || !credential.ExpiresAt.Equal(lease.ExpiresAt) {
		t.Fatalf("credential = %#v", credential)
	}
	if !strings.Contains(policy, "secret/data/apps/e50bfda9a5ca5ac46cd52ddbbb53906d/*") || strings.Contains(policy, "app:one") {
		t.Fatalf("scoped policy = %q", policy)
	}
	if len(tokenRequest.Policies) != 1 || tokenRequest.Policies[0] != "vrooli-app-e50bfda9a5ca5ac46cd52ddbbb53906d" || tokenRequest.TTL != "2m0s" {
		t.Fatalf("token request = %#v", tokenRequest)
	}
}
