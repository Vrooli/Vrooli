package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestServiceSupervisorStartsOnlyVerifiedPlanService(t *testing.T) {
	if testing.Short() {
		t.Skip("uses a helper service process")
	}
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(resourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(resourceDir, "fixture")
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	plan := &Plan{Resources: []Item{{
		Resource: "fixture", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service",
		Service: &Service{
			ProviderPolicy: resourcedeployment.ProviderPolicy{TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{resourcedeployment.ProviderTargetControlPlane: resourcedeployment.ProviderManagedShared, resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate}, AllowedModes: []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared}, SharedReuseRequiresConsent: true, ExternalManagement: "forbidden"},
			Artifact:       "fixture", Version: "1.0.0", SHA256: hex.EncodeToString(sum[:]),
			Arguments:   []string{"-test.run=TestBundledServiceFixtureProcess", "--"},
			Environment: map[string]string{"VROOLI_BUNDLED_SERVICE_FIXTURE": "1"},
			Config:      &resourcedeployment.ServiceConfig{Path: "fixture.conf", Content: "port=${RESOURCE_PORT_HTTP}\n"},
			Ports:       []ServicePort{{Name: "http", Host: 8200}},
			Files:       []Artifact{{Name: "fixture", SHA256: hex.EncodeToString(sum[:])}},
		},
	}}}
	supervisor := NewServiceSupervisor(root, filepath.Join(root, "app-data"))
	if err := supervisor.Start(context.Background(), plan); err != nil {
		t.Fatalf("Start: %v", err)
	}
	statuses := supervisor.Statuses()
	if !statuses["fixture"].Running || statuses["fixture"].PID <= 0 {
		t.Fatalf("status after start = %#v", statuses["fixture"])
	}
	configPath := filepath.Join(root, "app-data", "resources", "fixture", "config", "fixture.conf")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	port := statuses["fixture"].Ports["http"]
	if port <= 0 || port == 8200 {
		t.Fatalf("private service port = %d, want dynamically allocated value", port)
	}
	if want := fmt.Sprintf("port=%d\n", port); string(config) != want {
		t.Fatalf("config = %q, want %q", config, want)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := supervisor.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if statuses := supervisor.Statuses(); statuses["fixture"].Running {
		t.Fatalf("status after stop = %#v", statuses["fixture"])
	}
}

func TestBundledServiceFixtureProcess(t *testing.T) {
	if os.Getenv("VROOLI_BUNDLED_SERVICE_FIXTURE") != "1" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	<-signals
}

func TestExpandServiceTemplatePreservesUnknownValues(t *testing.T) {
	values := map[string]string{"VROOLI_RESOURCE_CONFIG_DIR": "/private/config"}
	if got := expandServiceTemplate("-config=${VROOLI_RESOURCE_CONFIG_DIR}/vault.hcl", values); got != "-config=/private/config/vault.hcl" {
		t.Fatalf("expanded argument = %q", got)
	}
	if got := expandServiceTemplate("${MISSING_VALUE}", values); got != "${MISSING_VALUE}" {
		t.Fatalf("unknown variable = %q", got)
	}
}

type testSharedResolver struct {
	binding SharedServiceBinding
	err     error
	called  bool
}

func (r *testSharedResolver) ResolveSharedService(_ context.Context, _ Item) (SharedServiceBinding, error) {
	r.called = true
	return r.binding, r.err
}

func TestServiceSupervisorUsesConsentedSharedBinding(t *testing.T) {
	resolver := &testSharedResolver{binding: SharedServiceBinding{
		Endpoint:    "http://127.0.0.1:8200",
		Environment: map[string]string{"VAULT_ADDR": "http://127.0.0.1:8200", "VAULT_TOKEN": "scoped-token"},
		ExpiresAt:   time.Now().Add(time.Minute),
	}}
	plan := &Plan{Resources: []Item{{
		Resource: "vault", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service",
		Service: &Service{ProviderPolicy: resourcedeployment.ProviderPolicy{
			TargetDefaults:             map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{resourcedeployment.ProviderTargetControlPlane: resourcedeployment.ProviderManagedShared, resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate},
			AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared},
			SharedReuseRequiresConsent: true,
			ExternalManagement:         "forbidden",
		}},
	}}}
	supervisor := NewServiceSupervisor(t.TempDir(), t.TempDir(), resolver)
	if err := supervisor.Start(context.Background(), plan); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !resolver.called {
		t.Fatal("shared resolver was not used")
	}
	status := supervisor.Statuses()["vault"]
	if !status.Running || status.PID != 0 || status.Provider != "managed-shared" {
		t.Fatalf("shared status = %#v", status)
	}
	if environment := supervisor.Environment(); environment["VAULT_TOKEN"] != "scoped-token" || environment["VAULT_ADDR"] != "http://127.0.0.1:8200" {
		t.Fatalf("shared environment = %#v", environment)
	}
}

func TestServiceSupervisorFallsBackToPrivateServiceForExpiredSharedBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("uses a helper service process")
	}
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(resourceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(resourceDir, "fixture")
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	plan := &Plan{Resources: []Item{{
		Resource: "fixture", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service",
		Service: &Service{
			ProviderPolicy: resourcedeployment.ProviderPolicy{TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{resourcedeployment.ProviderTargetControlPlane: resourcedeployment.ProviderManagedShared, resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate}, AllowedModes: []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared}, SharedReuseRequiresConsent: true, ExternalManagement: "forbidden"},
			Artifact:       "fixture", Version: "1.0.0", SHA256: hex.EncodeToString(sum[:]),
			Arguments:   []string{"-test.run=TestBundledServiceFixtureProcess", "--"},
			Environment: map[string]string{"VROOLI_BUNDLED_SERVICE_FIXTURE": "1"},
			Ports:       []ServicePort{{Name: "http", Host: 8200}},
			Files:       []Artifact{{Name: "fixture", SHA256: hex.EncodeToString(sum[:])}},
		},
	}}}
	resolver := &testSharedResolver{binding: SharedServiceBinding{
		Endpoint: "http://127.0.0.1:8200", Environment: map[string]string{"VAULT_TOKEN": "expired-token"}, ExpiresAt: time.Now().Add(-time.Minute),
	}}
	supervisor := NewServiceSupervisor(root, filepath.Join(root, "app-data"), resolver)
	if err := supervisor.Start(context.Background(), plan); err != nil {
		t.Fatalf("Start: %v", err)
	}
	status := supervisor.Statuses()["fixture"]
	if !resolver.called || !status.Running || status.Provider != "managed-private" || status.PID <= 0 {
		t.Fatalf("expired shared binding did not fall back to private service: %#v", status)
	}
	if _, ok := supervisor.Environment()["VAULT_TOKEN"]; ok {
		t.Fatal("expired shared credential was exposed to the private fallback")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := supervisor.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestWaitForServiceHealthHonorsDeclaredHTTPStatus(t *testing.T) {
	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	if err := waitForServiceHealth(ctx, []HealthCheck{{Type: "http", Target: unhealthy.URL, ExpectedStatus: []int{http.StatusOK}}}, nil); err == nil {
		t.Fatal("unexpected healthy result for a declared status mismatch")
	}

	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()
	if err := waitForServiceHealth(context.Background(), []HealthCheck{{Type: "http", Target: healthy.URL, ExpectedStatus: []int{http.StatusOK}}}, nil); err != nil {
		t.Fatalf("expected declared healthy response: %v", err)
	}
}
