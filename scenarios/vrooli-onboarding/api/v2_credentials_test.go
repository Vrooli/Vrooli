package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

func TestV2CredentialProvisionUsesMetadataOnlyResponse(t *testing.T) {
	prior := credentialProvisionCommand
	var received string
	credentialProvisionCommand = func(_ context.Context, logicalID, field, value string) error {
		received = logicalID + "/" + field + "/" + value
		return nil
	}
	t.Cleanup(func() { credentialProvisionCommand = prior })

	w := doPost(t, NewServer(), "/api/v2/credentials/provision", `{"logical_id":"vrooli/demo","field":"api-key","value":"test-value"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if received != "vrooli/demo/api-key/test-value" {
		t.Fatalf("provision input = %q", received)
	}
	if strings.Contains(w.Body.String(), "test-value") {
		t.Fatalf("credential value leaked in response: %s", w.Body.String())
	}
}

func TestV2CredentialProvisionRejectsMissingValue(t *testing.T) {
	w := doPost(t, NewServer(), "/api/v2/credentials/provision", `{"logical_id":"vrooli/demo","field":"api-key"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2CredentialDoctorRelaysMetadataOnly(t *testing.T) {
	prior := credentialDoctorCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return []byte(`{"backend":"libsecret","condition":"available"}`), nil
	}
	t.Cleanup(func() { credentialDoctorCommand = prior })

	w := doGet(t, NewServer(), "/api/v2/credentials/doctor")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"backend":"libsecret"`) {
		t.Fatalf("status/body = %d/%s", w.Code, w.Body.String())
	}
}

func TestV2CredentialKeyringRepairRequiresConfirmation(t *testing.T) {
	w := doPost(t, NewServer(), "/api/v2/credentials/keyring/repair", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2CredentialDoctorHidesRelayFailureDetails(t *testing.T) {
	prior := credentialDoctorCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return nil, errors.New("private host detail")
	}
	t.Cleanup(func() { credentialDoctorCommand = prior })

	w := doGet(t, NewServer(), "/api/v2/credentials/doctor")
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "private host detail") {
		t.Fatalf("status/body = %d/%s", w.Code, w.Body.String())
	}
}

func TestCredentialStoreRoutesRelayWithoutReturningSecrets(t *testing.T) {
	previousStatus := credentialStoreStatusCommand
	previousReselect := credentialStoreReselectCommand
	previousSelect := credentialStoreSelectCommand
	previousInit := credentialStoreInitCommand
	previousUnlock := credentialStoreUnlockCommand
	previousChange := credentialStoreChangePassphraseCommand
	previousRewrap := credentialStoreRewrapCommand
	t.Cleanup(func() {
		credentialStoreStatusCommand = previousStatus
		credentialStoreReselectCommand = previousReselect
		credentialStoreSelectCommand = previousSelect
		credentialStoreInitCommand = previousInit
		credentialStoreUnlockCommand = previousUnlock
		credentialStoreChangePassphraseCommand = previousChange
		credentialStoreRewrapCommand = previousRewrap
	})
	credentialStoreStatusCommand = func(context.Context) (securestore.StoreStatus, error) {
		return securestore.StoreStatus{Initialized: true, Active: true, Entries: 2, ActiveWrap: "native-wrap", ActiveKeyStore: "keychain"}, nil
	}
	credentialStoreReselectCommand = func(context.Context) (securestore.MigrationReceipt, error) {
		return securestore.MigrationReceipt{
			From: "encrypted-file", To: "native", Attempted: []string{"vrooli.credentials.v1/vrooli/demo:api-key"},
			Verified: []string{"vrooli.credentials.v1/vrooli/demo:api-key"}, Committed: true,
		}, nil
	}
	credentialStoreSelectCommand = func(_ context.Context, backend, reason string) error {
		if backend != securestore.BackendNative || reason != "operator chose native" {
			t.Fatalf("select = %q, %q", backend, reason)
		}
		return nil
	}
	credentialStoreInitCommand = func(_ context.Context, passphrase string) (securestore.StoreStatus, error) {
		if passphrase != "init-secret" {
			t.Fatalf("init passphrase = %q", passphrase)
		}
		return securestore.StoreStatus{Initialized: true}, nil
	}
	credentialStoreUnlockCommand = func(_ context.Context, passphrase string) (securestore.StoreStatus, error) {
		if passphrase != "unlock-secret" {
			t.Fatalf("unlock passphrase = %q", passphrase)
		}
		return securestore.StoreStatus{ActiveWrap: "passphrase"}, nil
	}
	credentialStoreChangePassphraseCommand = func(_ context.Context, current, next string) error {
		if current != "old-secret" || next != "new-secret" {
			t.Fatalf("passphrase change = %q, %q", current, next)
		}
		return nil
	}
	credentialStoreRewrapCommand = func(_ context.Context, passphrase string) (securestore.WrapInfo, error) {
		if passphrase != "rewrap-secret" {
			t.Fatalf("rewrap passphrase = %q", passphrase)
		}
		return securestore.WrapInfo{Provider: "native-wrap", KeyStore: "dpapi"}, nil
	}

	server := NewServer()
	if response := doGet(t, server, "/api/v2/credentials/store/status"); response.Code != http.StatusOK || response.Body.String() == "" {
		t.Fatalf("status response = %d %s", response.Code, response.Body.String())
	}
	if response := doPost(t, server, "/api/v2/credentials/store/select", `{"backend":"native","reason":"operator chose native"}`); response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "verified reselect") {
		t.Fatalf("initialized select response = %d %s", response.Code, response.Body.String())
	}
	if response := doPost(t, server, "/api/v2/credentials/store/reselect", `{}`); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"committed":true`) || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("reselect response = %d %s", response.Code, response.Body.String())
	}
	credentialStoreStatusCommand = func(context.Context) (securestore.StoreStatus, error) {
		return securestore.StoreStatus{Initialized: false, Active: false}, nil
	}
	if response := doPost(t, server, "/api/v2/credentials/store/select", `{"backend":"native","reason":"operator chose native"}`); response.Code != http.StatusOK {
		t.Fatalf("empty select response = %d %s", response.Code, response.Body.String())
	}
	for path, body := range map[string]string{
		"/api/v2/credentials/store/init":              `{"passphrase":"init-secret"}`,
		"/api/v2/credentials/store/unlock":            `{"passphrase":"unlock-secret"}`,
		"/api/v2/credentials/store/change-passphrase": `{"current_passphrase":"old-secret","new_passphrase":"new-secret"}`,
		"/api/v2/credentials/store/rewrap":            `{"passphrase":"rewrap-secret"}`,
	} {
		response := doPost(t, server, path, body)
		if response.Code < http.StatusOK || response.Code >= 300 {
			t.Fatalf("%s response = %d %s", path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "secret") {
			t.Fatalf("%s leaked a passphrase: %s", path, response.Body.String())
		}
	}
}

func TestOnboardingCredentialMigrationInventoryCoversWholeCatalog(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	for _, path := range []string{
		filepath.Join(root, "scenarios", "selected", ".vrooli"),
		filepath.Join(root, "scenarios", "not-selected", ".vrooli"),
		filepath.Join(root, "resources", "database"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	service := `{"credentials":{"descriptors":[{"logical_id":"owner/key","field":"token"}]}}`
	for _, name := range []string{"selected", "not-selected"} {
		if err := os.WriteFile(filepath.Join(root, "scenarios", name, ".vrooli", "service.json"), []byte(service), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "database", "resource.json"), []byte(service), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) { return []byte(`{"configured":false}`), nil }
	t.Cleanup(func() { credentialStatusCommand = previous })

	entries, err := onboardingCredentialMigrationEntries(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Service != "vrooli.credentials.v1" || entries[0].Key != "owner/key:token" {
		t.Fatalf("migration inventory = %#v, want one deduplicated all-catalog entry", entries)
	}
}
