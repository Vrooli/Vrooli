package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"github.com/vrooli/vrooli/packages/resource-deployment/securestore"
)

// TestDesktopVaultArtifactIntegration exercises a signed, real Vault binary
// through the exact bundle supervisor path. It is opt-in because release
// artifacts are intentionally not checked into the runtime module.
func TestDesktopVaultArtifactIntegration(t *testing.T) {
	source := os.Getenv("VROOLI_VAULT_INTEGRATION_BINARY")
	if source == "" {
		t.Skip("set VROOLI_VAULT_INTEGRATION_BINARY to run the signed Vault desktop integration")
	}
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	bundleRoot, appData := t.TempDir(), t.TempDir()
	artifactDir := filepath.Join(bundleRoot, "resources", "vault")
	if err := os.MkdirAll(artifactDir, 0o700); err != nil {
		t.Fatal(err)
	}
	artifact := "vault"
	if err := os.WriteFile(filepath.Join(artifactDir, artifact), body, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	store := &memoryCredentialStore{}
	previousStore := desktopVaultStore
	desktopVaultStore = func() securestore.Store { return store }
	t.Cleanup(func() { desktopVaultStore = previousStore })
	service := &Service{
		ProviderPolicy: resourcedeployment.ProviderPolicy{
			TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{resourcedeployment.ProviderTargetControlPlane: resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate},
			AllowedModes:   []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate},
		},
		Artifact: artifact, Version: "1.17.6", SHA256: fmt.Sprintf("%x", sum), Files: []Artifact{{Name: artifact, SHA256: fmt.Sprintf("%x", sum)}},
		Arguments:    []string{"server", "-config=${RESOURCE_CONFIG_DIR}/vault.hcl"},
		Config:       &resourcedeployment.ServiceConfig{Path: "vault.hcl", Content: "storage \"file\" { path = \"${RESOURCE_DATA_DIR}\" }\nlistener \"tcp\" { address = \"127.0.0.1:${RESOURCE_PORT_HTTP}\" tls_disable = true }\napi_addr = \"http://127.0.0.1:${RESOURCE_PORT_HTTP}\"\ndisable_mlock = true\n"},
		Ports:        []ServicePort{{Name: "http", Host: 8200}},
		HealthChecks: []HealthCheck{{Type: "http", Target: "http://127.0.0.1:${RESOURCE_PORT_HTTP}/v1/sys/health", ExpectedStatus: []int{200, 429, 472, 473}, TimeoutSeconds: 20}},
	}
	plan := &Plan{SchemaVersion: "v2", Resources: []Item{{RequestedResource: "vault", Resource: "vault", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service", Support: "conditional", Service: service}}}
	start := func(appDataDir string) (*ServiceSupervisor, map[string]string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		t.Cleanup(cancel)
		supervisor := NewServiceSupervisor(bundleRoot, appDataDir)
		if err := supervisor.Start(ctx, plan); err != nil {
			t.Fatalf("start desktop private Vault: %v", err)
		}
		t.Cleanup(func() { _ = supervisor.Stop(context.Background()) })
		environment := supervisor.Environment()
		if environment["VAULT_TOKEN"] == "" || environment["VAULT_ADDR"] == "" || environment["VAULT_TOKEN"] == "root" {
			t.Fatalf("desktop environment leaked or omitted scoped settings: %#v", environment)
		}
		return supervisor, environment
	}
	first, environment := start(appData)
	instanceID := desktopVaultInstanceID(appData)
	if err := desktopVaultRequest(context.Background(), environment["VAULT_ADDR"], http.MethodPost, "/v1/secret/data/apps/"+instanceID+"/persistence", environment["VAULT_TOKEN"], map[string]any{"data": map[string]string{"value": "survives"}}, nil); err != nil {
		t.Fatalf("write scoped desktop secret: %v", err)
	}
	secondAppData := t.TempDir()
	second, secondEnvironment := start(secondAppData)
	secondID := desktopVaultInstanceID(secondAppData)
	if secondID == instanceID {
		t.Fatal("separate desktop installs share a Vault identity")
	}
	if err := desktopVaultRequest(context.Background(), secondEnvironment["VAULT_ADDR"], http.MethodPost, "/v1/secret/data/apps/"+instanceID+"/denied", secondEnvironment["VAULT_TOKEN"], map[string]any{"data": map[string]string{"value": "no"}}, nil); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("second desktop app wrote first app path: %v", err)
	}
	if err := second.Stop(context.Background()); err != nil {
		t.Fatalf("stop second desktop private Vault: %v", err)
	}
	if err := first.Stop(context.Background()); err != nil {
		t.Fatalf("stop desktop private Vault: %v", err)
	}
	_, recovered := start(appData)
	var secret struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := desktopVaultRequest(context.Background(), recovered["VAULT_ADDR"], http.MethodGet, "/v1/secret/data/apps/"+instanceID+"/persistence", recovered["VAULT_TOKEN"], nil, &secret); err != nil || secret.Data.Data["value"] != "survives" {
		t.Fatalf("read persisted desktop secret = %#v, %v", secret, err)
	}
}

func TestDesktopVaultWaitReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/sys/seal-status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	if err := desktopVaultWaitReachable(context.Background(), server.URL); err != nil {
		t.Fatalf("wait for reachable Vault: %v", err)
	}
}

func TestDesktopVaultWaitReachableHonorsContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := desktopVaultWaitReachable(ctx, "http://127.0.0.1:1"); err == nil {
		t.Fatal("unreachable Vault was accepted")
	}
}

func TestDesktopVaultInstanceIDIsStableAndAppPrivate(t *testing.T) {
	first := desktopVaultInstanceID("/tmp/one")
	if first == "" || first != desktopVaultInstanceID("/tmp/one") {
		t.Fatalf("instance identity is not stable: %q", first)
	}
	if first == desktopVaultInstanceID("/tmp/two") {
		t.Fatal("two desktop app data homes share a Vault identity")
	}
}

type memoryCredentialStore struct{ values map[string]string }

func (s *memoryCredentialStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+":"+key] = value
	return nil
}

func (s *memoryCredentialStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+":"+key]
	if !ok {
		return "", fmt.Errorf("missing")
	}
	return value, nil
}

func (s *memoryCredentialStore) Delete(service, key string) error {
	delete(s.values, service+":"+key)
	return nil
}

func TestBootstrapPrivateVaultPersistsRecoveryAndExportsOnlyScopedCredential(t *testing.T) {
	store := &memoryCredentialStore{}
	previous := desktopVaultStore
	desktopVaultStore = func() securestore.Store { return store }
	t.Cleanup(func() { desktopVaultStore = previous })
	initialized, sealed := false, true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		write := func(value any, status int) { w.WriteHeader(status); _ = json.NewEncoder(w).Encode(value) }
		switch request.URL.Path {
		case "/v1/sys/seal-status":
			write(map[string]bool{"initialized": initialized, "sealed": sealed}, http.StatusOK)
		case "/v1/sys/init":
			initialized = true
			write(map[string]any{"keys": []string{"unseal"}, "root_token": "root"}, http.StatusOK)
		case "/v1/sys/unseal":
			sealed = false
			write(map[string]bool{"sealed": false}, http.StatusOK)
		case "/v1/sys/mounts/secret", "/v1/sys/policies/acl/vrooli-desktop-" + desktopVaultInstanceID("app"):
			write(map[string]any{}, http.StatusNoContent)
		case "/v1/auth/token/create":
			write(map[string]any{"auth": map[string]string{"client_token": "scoped"}}, http.StatusOK)
		case "/v1/auth/token/lookup-self":
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
	port, _ := strconv.Atoi(strings.TrimPrefix(server.URL[strings.LastIndex(server.URL, ":")+1:], "/"))
	if port == 0 {
		t.Fatalf("test server URL %q has no port", server.URL)
	}
	environment, err := bootstrapPrivateVault(context.Background(), Item{}, map[string]int{"http": port}, "app")
	if err != nil {
		t.Fatalf("bootstrap private Vault: %v", err)
	}
	if environment["VAULT_TOKEN"] != "scoped" || environment["VAULT_ADDR"] == "" {
		t.Fatalf("environment = %#v", environment)
	}
	for _, value := range environment {
		if value == "root" || value == "unseal" {
			t.Fatalf("bootstrap material leaked in environment: %#v", environment)
		}
	}
	if len(store.values) == 0 {
		t.Fatal("recovery material was not persisted in credential store")
	}
}
