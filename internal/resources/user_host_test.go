package resources

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type memorySecureStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *memorySecureStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[service+"/"+key] = value
	return nil
}

func (s *memorySecureStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[service+"/"+key]
	if !ok {
		// Every real adapter reports a clean "no value" as ErrNotFound. The
		// bootstrap depends on that distinction: a generic error must mean the
		// store is broken, not that the instance is new, or one transient read
		// failure would re-initialize a live Vault over its own data.
		return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
	}
	return value, nil
}

func (s *memorySecureStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, service+"/"+key)
	return nil
}

type bootstrapStub struct {
	calls, recoveryCalls int
	readinessErr         error
}

func (b *bootstrapStub) Bootstrap(context.Context, string) (VaultBootstrapMaterial, error) {
	b.calls++
	return VaultBootstrapMaterial{RootToken: "root", UnsealKey: "unseal"}, nil
}

func (b *bootstrapStub) Unseal(context.Context, string, VaultBootstrapMaterial) error {
	b.recoveryCalls++
	return nil
}

func (b *bootstrapStub) LifecycleState(context.Context, string) (VaultLifecycleState, error) {
	return VaultStateUsable, nil
}

func (b *bootstrapStub) VerifyScopedOperation(context.Context, string, string) error {
	return b.readinessErr
}

func TestUserResourceHostBootstrapsOnceAndScopesEachApplication(t *testing.T) {
	useFakeUnsealKeys(t)
	now := time.Now()
	broker := NewBroker(func() time.Time { return now })
	store := &memorySecureStore{values: map[string]string{}}
	host, err := newUserResourceHost(broker, store, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := &bootstrapStub{}
	vault := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/sys/seal-status":
			_, _ = w.Write([]byte(`{"initialized":true,"sealed":false}`))
		case "/v1/sys/policies/acl/vrooli-app-":
			w.WriteHeader(http.StatusNotFound)
		case "/v1/auth/token/create":
			_, _ = w.Write([]byte(`{"auth":{"client_token":"scoped"}}`))
		case "/v1/sys/mounts/secret":
			w.WriteHeader(http.StatusNoContent)
		default:
			if len(request.URL.Path) > len("/v1/sys/policies/acl/") && request.URL.Path[:len("/v1/sys/policies/acl/")] == "/v1/sys/policies/acl/" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer vault.Close()
	instance := ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:test", CapabilityVersion: "1", Endpoint: vault.URL}
	if _, err := host.EnsureVault(context.Background(), instance, "app:one", bootstrap); err != nil {
		t.Fatal(err)
	}
	if _, err := host.EnsureVault(context.Background(), instance, "app:two", bootstrap); err != nil {
		t.Fatal(err)
	}
	if bootstrap.calls != 1 || bootstrap.recoveryCalls != 1 {
		t.Fatalf("bootstrap/recovery = %d/%d, want 1/1", bootstrap.calls, bootstrap.recoveryCalls)
	}
	lease, err := broker.Acquire("vault", "app:one", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.AuthorizeUse(lease.ID, "vault", "app:two"); err == nil {
		t.Fatal("lease crossed application scope")
	}
	if token, err := host.VaultManagementToken(instance); err != nil || token != "root" {
		t.Fatalf("management token = %q, %v", token, err)
	}
}

func TestUserResourceHostDoesNotPublishVaultBeforeScopedReadiness(t *testing.T) {
	useFakeUnsealKeys(t)
	broker := NewBroker(nil)
	host, err := newUserResourceHost(broker, &memorySecureStore{values: map[string]string{}}, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	vault := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/v1/sys/seal-status":
			_, _ = w.Write([]byte(`{"initialized":true,"sealed":false}`))
		case request.URL.Path == "/v1/auth/token/create":
			_, _ = w.Write([]byte(`{"auth":{"client_token":"scoped"}}`))
		case request.URL.Path == "/v1/sys/mounts/secret":
			w.WriteHeader(http.StatusNoContent)
		case len(request.URL.Path) > len("/v1/sys/policies/acl/") && request.URL.Path[:len("/v1/sys/policies/acl/")] == "/v1/sys/policies/acl/":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer vault.Close()
	instance := ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:test", CapabilityVersion: "1", Endpoint: vault.URL}
	_, err = host.EnsureVault(context.Background(), instance, "app:one", &bootstrapStub{readinessErr: fmt.Errorf("scoped lookup denied")})
	if err == nil || !strings.Contains(err.Error(), "scoped operation") {
		t.Fatalf("EnsureVault() error = %v, want scoped readiness failure", err)
	}
	if _, err := broker.Acquire("vault", "app:one", time.Minute); err == nil {
		t.Fatal("Vault was published before scoped readiness succeeded")
	}
}

func TestUserResourceHostRejectsUnverifiedVault(t *testing.T) {
	host, err := newUserResourceHost(NewBroker(nil), &memorySecureStore{values: map[string]string{}}, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = host.EnsureVault(context.Background(), ManagedInstance{ID: "external", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:test", CapabilityVersion: "1", Endpoint: "https://vault.example"}, "app:one", &bootstrapStub{})
	if err == nil {
		t.Fatal("external Vault was accepted as a user-hosted resource")
	}
}

func TestUserResourceHostBrokerUsesScopedVaultIssuer(t *testing.T) {
	host, err := newUserResourceHost(NewBroker(nil), &memorySecureStore{values: map[string]string{}}, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server, err := host.StartVaultBrokerControl(listener, map[string]string{"app:one": "app-token", "user:test": "owner-token"})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close(context.Background())
	if _, err := NewBrokerControlClient(BrokerControlCredential{Endpoint: "http://" + listener.Addr().String(), Scope: "app:one", Token: "app-token"}); err != nil {
		t.Fatalf("scoped broker client: %v", err)
	}
}

func TestStartVaultBrokerControlWithLifecycleRejectsIncompleteOwnerRuntime(t *testing.T) {
	host, err := newUserResourceHost(NewBroker(nil), &memorySecureStore{values: map[string]string{}}, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.StartVaultBrokerControlWithLifecycle(listener, map[string]string{"user:test": "owner-token"}, nil, ResourceManifest{Name: "vault"}); err == nil {
		t.Fatal("broker control accepted a lifecycle without a fixed Controller")
	}
}

func TestUserResourceHostRefusesBootstrapWhenSecureStorageIsUnavailable(t *testing.T) {
	host, err := newUserResourceHost(NewBroker(nil), failingSecureStore{}, "user:test")
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := &bootstrapStub{}
	_, err = host.EnsureVault(context.Background(), ManagedInstance{ID: "vault-user", Resource: "vault", Provider: resourcedeployment.ProviderManagedShared, OwnerScope: "user:test", CapabilityVersion: "1", Endpoint: "http://127.0.0.1:8200"}, "app:one", bootstrap)
	if err == nil || bootstrap.calls != 0 {
		t.Fatalf("bootstrap proceeded without secure storage: err=%v calls=%d", err, bootstrap.calls)
	}
}

type failingSecureStore struct{}

func (failingSecureStore) Put(string, string, string) error   { return fmt.Errorf("unavailable") }
func (failingSecureStore) Get(string, string) (string, error) { return "", fmt.Errorf("unavailable") }
func (failingSecureStore) Delete(string, string) error        { return fmt.Errorf("unavailable") }
