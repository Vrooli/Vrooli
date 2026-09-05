package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

type writeOnlySecureStore struct{ memorySecureStore }

func (s *writeOnlySecureStore) Get(service, key string) (string, error) {
	if service == vaultbootstrap.Service {
		return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
	}
	return s.memorySecureStore.Get(service, key)
}

func TestStoreVaultBootstrapMaterialRequiresSecureStoreReadback(t *testing.T) {
	store := &writeOnlySecureStore{memorySecureStore: memorySecureStore{values: map[string]string{}}}
	if err := storeVaultBootstrapMaterial(store, "vault-user", VaultBootstrapMaterial{RootToken: "root", UnsealKey: "unseal"}); err == nil {
		t.Fatal("storeVaultBootstrapMaterial accepted a write that cannot be read back")
	}
}

func TestBootstrapPrivateVaultInitializesThenRecoversWithScopedReadiness(t *testing.T) {
	useFakeUnsealKeys(t)
	store := &memorySecureStore{values: map[string]string{}}
	previousStore := privateVaultSecureStore
	privateVaultSecureStore = func() securestore.Store { return store }
	t.Cleanup(func() { privateVaultSecureStore = previousStore })

	initialized, sealed, initCalls := false, true, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		write := func(value any, status int) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(value)
		}
		switch {
		case request.URL.Path == "/v1/sys/seal-status":
			write(map[string]bool{"initialized": initialized, "sealed": sealed}, http.StatusOK)
		case request.URL.Path == "/v1/sys/init":
			initCalls++
			initialized = true
			write(map[string]any{"keys": []string{"unseal"}, "root_token": "root"}, http.StatusOK)
		case request.URL.Path == "/v1/sys/unseal":
			sealed = false
			write(map[string]bool{"sealed": false}, http.StatusOK)
		case request.URL.Path == "/v1/sys/mounts/secret":
			write(map[string]any{}, http.StatusNoContent)
		case strings.HasPrefix(request.URL.Path, "/v1/sys/policies/acl/"):
			write(map[string]any{}, http.StatusNoContent)
		case request.URL.Path == "/v1/auth/token/create":
			write(map[string]any{"auth": map[string]string{"client_token": "scoped"}}, http.StatusOK)
		case request.URL.Path == "/v1/auth/token/lookup-self":
			if request.Header.Get("X-Vault-Token") != "scoped" {
				write(map[string]string{"error": "wrong token"}, http.StatusForbidden)
				return
			}
			write(map[string]any{}, http.StatusOK)
		default:
			write(map[string]string{"path": request.URL.Path}, http.StatusNotFound)
		}
	}))
	defer server.Close()
	state := ManagedServiceState{InstanceID: "private-vault"}
	if err := bootstrapPrivateVault(context.Background(), state, server.URL); err != nil {
		t.Fatalf("first private bootstrap: %v", err)
	}
	if initCalls != 1 || sealed {
		t.Fatalf("first bootstrap init/sealed = %d/%v, want 1/false", initCalls, sealed)
	}
	sealed = true
	if err := bootstrapPrivateVault(context.Background(), state, server.URL); err != nil {
		t.Fatalf("private recovery: %v", err)
	}
	if initCalls != 1 || sealed {
		t.Fatalf("recovery init/sealed = %d/%v, want 1/false", initCalls, sealed)
	}
	if _, err := store.Get(vaultbootstrap.Service, state.InstanceID); err != nil {
		t.Fatalf("private recovery material missing from secure storage: %v", err)
	}
}

func TestClassifyVaultSealStatus(t *testing.T) {
	tests := []struct {
		name                string
		initialized, sealed bool
		want                VaultLifecycleState
	}{
		{name: "uninitialized is never usable", want: VaultStateUninitialized},
		{name: "initialized sealed", initialized: true, sealed: true, want: VaultStateSealed},
		{name: "initialized unsealed", initialized: true, want: VaultStateUnsealed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ClassifyVaultSealStatus(test.initialized, test.sealed); got != test.want {
				t.Fatalf("ClassifyVaultSealStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestHTTPVaultBootstrapperScopedReadinessRejectsUnscopedAndAcceptsScopedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/auth/token/lookup-self" || request.Header.Get("X-Vault-Token") != "scoped" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bootstrapper := HTTPVaultBootstrapper{}
	if err := bootstrapper.VerifyScopedOperation(context.Background(), server.URL, ""); err == nil {
		t.Fatal("empty scoped credential was accepted")
	}
	if err := bootstrapper.VerifyScopedOperation(context.Background(), server.URL, "scoped"); err != nil {
		t.Fatalf("scoped operation = %v", err)
	}
}

// fakeUnsealKeys stands in for the credential authority so a test never reaches
// this host's real keyring. Without it these tests pay the full Secret Service
// timeout per call and fail for a reason that has nothing to do with Vault.
type fakeUnsealKeys struct{ values map[string]string }

func (f *fakeUnsealKeys) Put(identity credentialauthority.Identity, field, value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[string(identity)+":"+field] = value
	return nil
}

func (f *fakeUnsealKeys) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	value, ok := f.values[string(identity)+":"+field]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

// useFakeUnsealKeys points the unseal-key sink at an in-memory stand-in and
// returns it, so a test can assert the key was recorded where recovery looks.
func useFakeUnsealKeys(t *testing.T) *fakeUnsealKeys {
	t.Helper()
	keys := &fakeUnsealKeys{}
	previous := unsealKeyStore
	unsealKeyStore = func() vaultbootstrap.UnsealKeyStore { return keys }
	t.Cleanup(func() { unsealKeyStore = previous })
	return keys
}
