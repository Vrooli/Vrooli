package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	vaultbootstrap "github.com/vrooli/vrooli/packages/vaultbootstrap-go"
)

func TestManagedServiceDriverRunsVerifiedPrivateLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("uses a helper process")
	}
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	t.Setenv("VROOLI_MANAGED_SERVICE_FIXTURE", "1")
	resourceRoot := filepath.Join(root, "resources", "fixture-service", "bin")
	if err := os.MkdirAll(resourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(resourceRoot, "service")
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, body, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	manifest := managedServiceTestManifest(fmt.Sprintf("%x", sum))
	controller := NewController(root, filepath.Join(root, "home"))
	item := Resource{Name: manifest.Name}
	driver := managedServiceDriver{}
	if err := driver.Run(context.Background(), controller, item, manifest, "start", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("start: %v", err)
	}
	// A second start must reconcile the supervisor-owned process before port
	// preflight. The listener is intentionally still occupied by the first
	// invocation, so a port-first implementation regresses with EADDRINUSE.
	if err := driver.Run(context.Background(), controller, item, manifest, "start", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("idempotent start: %v", err)
	}
	t.Cleanup(func() {
		_ = driver.Run(context.Background(), controller, item, manifest, "stop", nil, io.Discard, io.Discard)
	})
	status, err := driver.Status(context.Background(), controller, item, manifest, true)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.Installed || !status.Running {
		t.Fatalf("status = %+v, want installed running service", status)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "stop", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestManagedServiceInstallAcquiresDeclaredArtifactIdempotently(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	body, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()
	manifest := managedServiceTestManifest(fmt.Sprintf("%x", sum))
	manifest.Name = "acquisition-fixture"
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{
		Kind: "url",
		Targets: []binaryfetch.AcquisitionTarget{{
			URL: server.URL, SHA256: fmt.Sprintf("%x", sum), Layout: "file",
		}},
	}
	controller := NewController(root, home)
	item := Resource{Name: manifest.Name}
	driver := managedServiceDriver{}
	if err := driver.Run(context.Background(), controller, item, manifest, "install", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("first install: %v", err)
	}
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "install", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("second install: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("idempotent install changed verified artifact bytes")
	}
}

func TestNativeDataArtifactStagesVendoredBytesAndChecksum(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	resourceRoot := filepath.Join(root, "resources", "native-data-fixture", "artifacts")
	if err := os.MkdirAll(resourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("vendored-wasi-fixture")
	sum := sha256.Sum256(body)
	digest := fmt.Sprintf("%x", sum)
	sourcePath := filepath.Join(resourceRoot, "fixture.wasm")
	if err := os.WriteFile(sourcePath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := ResourceManifest{
		Name: "native-data-fixture",
		ManagedService: &resourcedeployment.ManagedService{DataArtifacts: []resourcedeployment.ManagedServiceDataArtifact{{
			Name: "fixture-wasi",
			Path: "fixture.wasm",
			Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{
				URL: "https://example.invalid/fixture.wasm", SHA256: digest, Layout: "file",
			}}},
		}}},
	}
	controller := NewController(root, filepath.Join(root, "home"))
	if err := ensureManagedServiceDataArtifacts(context.Background(), controller, manifest); err != nil {
		t.Fatalf("stage vendored data artifact: %v", err)
	}
	dataPath := filepath.Join(root, "runtime", "data", "vrooli", "resources", manifest.Name, "fixture.wasm")
	got, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("staged bytes = %q, want %q", got, body)
	}
	checksum, err := os.ReadFile(dataPath + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(checksum)) != digest {
		t.Fatalf("checksum sidecar = %q, want %q", strings.TrimSpace(string(checksum)), digest)
	}
}

func TestManagedServiceFileArchiveBinPathIsExtractionOnly(t *testing.T) {
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Artifact.Layout = "file"
	manifest.ManagedService.Artifact.EntryPath = ""

	target := binaryfetch.AcquisitionTarget{
		Layout:  "file",
		BinPath: "AdGuardHome/AdGuardHome",
	}
	artifact, err := managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		t.Fatalf("managedServiceArtifactForTarget: %v", err)
	}
	if artifact.EntryPath != "" {
		t.Fatalf("file-layout archive bin_path became launch entry path %q", artifact.EntryPath)
	}

	target.Layout = "dir"
	artifact, err = managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		t.Fatalf("managedServiceArtifactForTarget dir: %v", err)
	}
	if artifact.EntryPath != target.BinPath {
		t.Fatalf("directory bin_path = %q, want %q", artifact.EntryPath, target.BinPath)
	}
}

func TestManagedServiceHostToolDoesNotRequireReleaseChecksum(t *testing.T) {
	manifest := managedServiceTestManifest("")
	manifest.ManagedService.Artifact = resourcedeployment.ServiceArtifact{
		Path: "cloudflared", Version: "host-tool", Verification: "host-tool",
	}
	artifact, err := managedServiceArtifactForTarget(manifest, binaryfetch.AcquisitionTarget{Executable: "cloudflared"})
	if err != nil {
		t.Fatalf("managedServiceArtifactForTarget host tool: %v", err)
	}
	if artifact.Verification != "host-tool" || artifact.SHA256 != "" {
		t.Fatalf("host-tool artifact = %+v", artifact)
	}
}

func TestPruneManagedServiceArtifactsKeepsCurrentVersion(t *testing.T) {
	root := t.TempDir()
	artifactRoot := filepath.Join(root, "artifacts")
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", artifactRoot)
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.Name = "prune-fixture"
	manifest.ManagedService.Artifact.Version = "current"
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{
		Kind: "url",
		Targets: []binaryfetch.AcquisitionTarget{{
			URL:    "https://example.invalid/service",
			SHA256: strings.Repeat("b", 64),
		}},
	}
	manifestDir := filepath.Join(root, "resources", manifest.Name)
	if err := os.MkdirAll(manifestDir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "resource.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"current", "superseded"} {
		path := filepath.Join(artifactRoot, manifest.Name, version, "server")
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "entry"), []byte(version), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := NewController(root, filepath.Join(root, "home")).PruneManagedServiceArtifacts(manifest.Name)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1 superseded version", removed)
	}
	if _, err := os.Stat(filepath.Join(artifactRoot, manifest.Name, "current")); err != nil {
		t.Fatalf("current artifact version missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactRoot, manifest.Name, "superseded")); !os.IsNotExist(err) {
		t.Fatalf("superseded artifact version still exists: %v", err)
	}
}

func TestManagedServiceStartTimeoutDerivesQdrantCollectionBudget(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	collections := filepath.Join(root, "runtime", "data", "vrooli", "resources", "qdrant", "collections")
	if err := os.MkdirAll(collections, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if err := os.Mkdir(filepath.Join(collections, fmt.Sprintf("collection-%02d", i)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	manifest := ResourceManifest{
		Name:      "qdrant",
		Lifecycle: ResourceLifecycle{StartTimeoutSeconds: 90},
		Orchestration: ResourceOrchestration{StartupBudget: &ResourceStartupBudget{
			Kind: "qdrant_collection_count", BaseSeconds: 30, PerUnitSeconds: 2, MaxSeconds: 600,
		}},
	}
	got := managedServiceStartTimeoutSeconds(NewController(root, filepath.Join(root, "home")), manifest)
	if got != 110 {
		t.Fatalf("derived timeout = %d, want 110 seconds for 40 collections", got)
	}
}

// TestManagedVaultArtifactIntegration exercises the actual signed Vault server
// when a release staging job supplies a verified artifact compatible with the
// host that runs this test. It remains opt-in for ordinary unit test runs
// because the upstream artifact is large.
func TestManagedVaultArtifactIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("requires an explicitly staged Vault server artifact")
	}
	source := os.Getenv("VROOLI_VAULT_INTEGRATION_BINARY")
	if source == "" {
		t.Skip("set VROOLI_VAULT_INTEGRATION_BINARY to run against a staged Vault server")
	}
	file, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	store := &memorySecureStore{values: map[string]string{}}
	previousStore := privateVaultSecureStore
	privateVaultSecureStore = func() securestore.Store { return store }
	t.Cleanup(func() { privateVaultSecureStore = previousStore })
	resourceRoot := filepath.Join(root, "resources", "vault", "server")
	if err := os.MkdirAll(resourceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	artifactName := managedVaultIntegrationArtifactName(source)
	binary := filepath.Join(resourceRoot, artifactName)
	if err := os.Link(source, binary); err != nil {
		input, openErr := os.Open(source)
		if openErr != nil {
			t.Fatal(openErr)
		}
		output, createErr := os.OpenFile(binary, os.O_CREATE|os.O_WRONLY, 0o700)
		if createErr != nil {
			input.Close()
			t.Fatal(createErr)
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		input.Close()
		if copyErr != nil || closeErr != nil {
			t.Fatalf("stage Vault fixture: copy=%v close=%v", copyErr, closeErr)
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := managedServiceTestManifest(fmt.Sprintf("%x", hash.Sum(nil)))
	manifest.Name = "vault"
	manifest.ManagedService.Artifact.Path = filepath.ToSlash(filepath.Join("server", artifactName))
	manifest.ManagedService.Artifact.Version = "1.17.6"
	manifest.ManagedService.ProviderPolicy.AllowedModes = append(manifest.ManagedService.ProviderPolicy.AllowedModes, resourcedeployment.ProviderManagedDiscovered)
	manifest.ManagedService.Arguments = []string{"server", "-config=${RESOURCE_CONFIG_DIR}/vault.hcl"}
	manifest.ManagedService.Config = &resourcedeployment.ServiceConfig{Path: "vault.hcl", Content: "storage \"file\" { path = \"${RESOURCE_DATA_DIR}\" }\nlistener \"tcp\" { address = \"127.0.0.1:${RESOURCE_PORT_HTTP}\" tls_disable = true }\napi_addr = \"http://127.0.0.1:${RESOURCE_PORT_HTTP}\"\ndisable_mlock = true\n"}
	manifest.Ports = []ResourcePort{{Name: "http", Host: port}}
	manifest.HealthChecks = []ResourceHealthCheck{{Type: "http", Target: "http://127.0.0.1:${RESOURCE_PORT_HTTP}/v1/sys/health", ExpectedStatus: []int{200, 429, 472, 473}, TimeoutSeconds: 10}}
	manifest.Lifecycle.StartTimeoutSeconds = 15
	controller := NewController(root, filepath.Join(root, "home"))
	driver := managedServiceDriver{}
	item := Resource{Name: manifest.Name}
	if err := driver.Run(context.Background(), controller, item, manifest, "start", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("start managed Vault: %v", err)
	}
	t.Cleanup(func() {
		_ = driver.Run(context.Background(), controller, item, manifest, "stop", nil, io.Discard, io.Discard)
	})
	status, err := driver.Status(context.Background(), controller, item, manifest, false)
	if err != nil || !status.Running || status.Health != "healthy" {
		t.Fatalf("managed Vault status = %+v, %v", status, err)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "restart", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("restart managed Vault: %v", err)
	}
	status, err = driver.Status(context.Background(), controller, item, manifest, false)
	if err != nil || !status.Running || status.Health != "healthy" {
		t.Fatalf("managed Vault status after restart = %+v, %v", status, err)
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	var material VaultBootstrapMaterial
	supervisor, _, err := managedServiceSupervisorFor("vault")
	if err != nil {
		t.Fatal(err)
	}
	state, running, err := supervisor.Status()
	if err != nil || !running {
		t.Fatalf("native Vault supervisor state = %#v running=%v err=%v", state, running, err)
	}
	rawMaterial, err := store.Get(vaultbootstrap.Service, state.InstanceID)
	if err != nil || json.Unmarshal([]byte(rawMaterial), &material) != nil || material.RootToken == "" {
		t.Fatalf("native bootstrap did not persist recovery material: %v", err)
	}
	now := time.Now()
	lease := Lease{ID: "vault-fixture-lease", Scope: "app:one", ExpiresAt: now.Add(time.Minute)}
	issuer := VaultCredentialIssuer{Now: func() time.Time { return now }, ManagementToken: func(ManagedInstance) (string, error) { return material.RootToken, nil }}
	credential, err := issuer.IssueScopedCredential(ManagedInstance{Resource: "vault", Endpoint: endpoint}, lease)
	if err != nil || credential.Credential == "" {
		t.Fatalf("issue actual scoped Vault token = %#v, %v", credential, err)
	}
	_, policy := issuer.policyForScope(lease.Scope)
	policyPath := "apps/" + strings.Split(strings.Split(policy, "apps/")[1], "/*")[0] + "/allowed"
	if status, err := vaultFixtureRequest(endpoint, "/v1/secret/data/"+policyPath, credential.Credential, map[string]any{"data": map[string]string{"value": "ok"}}, nil); err != nil || (status != http.StatusNoContent && status != http.StatusOK) {
		t.Fatalf("scoped token allowed write = status %d, error %v", status, err)
	}
	if status, err := vaultFixtureRequest(endpoint, "/v1/secret/data/apps/another-app/denied", credential.Credential, map[string]any{"data": map[string]string{"value": "no"}}, nil); err != nil || status != http.StatusForbidden {
		t.Fatalf("scoped token cross-app write = status %d, error %v", status, err)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "restart", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("restart managed Vault after scoped write: %v", err)
	}
	if status, err := vaultFixtureReadStatus(endpoint, "/v1/secret/data/"+policyPath, credential.Credential); err != nil || status != http.StatusOK {
		t.Fatalf("scoped token reads persisted value after restart = status %d, error %v", status, err)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "stop", nil, io.Discard, io.Discard); err != nil {
		t.Fatalf("stop before discovered launch: %v", err)
	}
	if err := driver.Run(context.Background(), controller, item, manifest, "start", []string{"--provider=managed-discovered", "--executable=" + source}, io.Discard, io.Discard); err != nil {
		t.Fatalf("start verified discovered Vault: %v", err)
	}
	status, err = driver.Status(context.Background(), controller, item, manifest, false)
	if err != nil || !status.Running || status.Health != "healthy" {
		t.Fatalf("discovered Vault status = %+v, %v", status, err)
	}
}

func TestManagedVaultIntegrationArtifactName(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   string
	}{
		{name: "unix artifact", source: "/artifacts/vault_darwin_arm64", want: "vault"},
		{name: "windows artifact", source: `C:\\artifacts\\vault_windows_amd64.exe`, want: "vault.exe"},
		{name: "uppercase windows suffix", source: "/artifacts/vault.EXE", want: "vault.exe"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := managedVaultIntegrationArtifactName(test.source); got != test.want {
				t.Fatalf("managedVaultIntegrationArtifactName(%q) = %q, want %q", test.source, got, test.want)
			}
		})
	}
}

func TestManagedDiscoveredExecutableRequiresExplicitAbsoluteExecutable(t *testing.T) {
	if _, err := managedDiscoveredExecutable(nil); err == nil {
		t.Fatal("missing executable was accepted")
	}
	if _, err := managedDiscoveredExecutable([]string{"--executable", "vault"}); err == nil {
		t.Fatal("relative executable was accepted")
	}
	path := filepath.Join(t.TempDir(), "candidate")
	if err := os.WriteFile(path, []byte("candidate"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := managedDiscoveredExecutable([]string{"--executable=" + path})
	if err != nil || got != path {
		t.Fatalf("managedDiscoveredExecutable = %q, %v", got, err)
	}
}

func TestVerifyManagedDiscoveredVersionRejectsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate")
	if err := os.WriteFile(path, []byte(shelltest.POSIXShebang()+"echo vault 1.17.6\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyManagedDiscoveredVersion(context.Background(), path, "1.17.6"); err != nil {
		t.Fatalf("matching discovered version: %v", err)
	}
	if err := verifyManagedDiscoveredVersion(context.Background(), path, "9.9.9"); err == nil {
		t.Fatal("mismatched discovered version was accepted")
	}
}

func managedVaultIntegrationArtifactName(source string) string {
	if strings.EqualFold(filepath.Ext(source), ".exe") {
		return "vault.exe"
	}
	return "vault"
}

func vaultFixtureRequest(endpoint, path, token string, input, output any) (int, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequest(http.MethodPut, endpoint+path, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	if token != "" {
		request.Header.Set("X-Vault-Token", token)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if output != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		if err := json.NewDecoder(response.Body).Decode(output); err != nil {
			return response.StatusCode, err
		}
	}
	return response.StatusCode, nil
}

func vaultFixtureReadStatus(endpoint, path, token string) (int, error) {
	request, err := http.NewRequest(http.MethodGet, endpoint+path, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("X-Vault-Token", token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, response.Body)
	return response.StatusCode, err
}

func TestManagedServiceDriverRefusesAttachOnlyLifecycle(t *testing.T) {
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.ProviderPolicy.TargetDefaults = map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{
		resourcedeployment.ProviderTargetControlPlane:  resourcedeployment.ProviderAttachOnly,
		resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderAttachOnly,
	}
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderAttachOnly}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = []resourcedeployment.AccessCapability{resourcedeployment.AccessReadOnly}
	err := (managedServiceDriver{}).Run(context.Background(), NewController(t.TempDir(), t.TempDir()), Resource{Name: manifest.Name}, manifest, "start", nil, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "does not grant local lifecycle authority") {
		t.Fatalf("attach-only start error = %v", err)
	}
}

func TestManagedServiceDriverValidatesExplicitAttachOnlyEndpointWithoutLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = []resourcedeployment.AccessCapability{resourcedeployment.AccessReadOnly}
	manifest.ManagedService.AttachHealthPath = "/health"
	var output bytes.Buffer
	err := (managedServiceDriver{}).Run(context.Background(), NewController(t.TempDir(), t.TempDir()), Resource{Name: manifest.Name}, manifest, "status", []string{"--provider=attach-only", "--endpoint=" + server.URL}, &output, io.Discard)
	if err != nil || !strings.Contains(output.String(), "attach-only endpoint healthy") {
		t.Fatalf("attach-only status = %q, %v", output.String(), err)
	}
}

func TestManagedServiceControlPlaneUsesSharedDefaultWithoutDesktopConsent(t *testing.T) {
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.ProviderPolicy.TargetDefaults = map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{
		resourcedeployment.ProviderTargetControlPlane:  resourcedeployment.ProviderManagedShared,
		resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedShared,
	}
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedShared}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	mode, err := managedServiceProvider(manifest, nil)
	if err != nil || mode != resourcedeployment.ProviderManagedShared {
		t.Fatalf("control-plane shared provider = %q, %v", mode, err)
	}
}

func TestManagedServicePortEnvironmentName(t *testing.T) {
	if got := managedServicePortEnvName("http-admin"); got != "RESOURCE_PORT_HTTP_ADMIN" {
		t.Fatalf("port environment name = %q", got)
	}
}

func TestManagedServicePortPreflightRefusesAnOccupiedLoopbackPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.Ports = []ResourcePort{{Name: "http", Host: listener.Addr().(*net.TCPAddr).Port}}
	if err := ensureManagedServicePortsAvailable(manifest); err == nil || !strings.Contains(err.Error(), "already unavailable") {
		t.Fatalf("port preflight error = %v", err)
	}
}

func TestRenderManagedServiceEnvironmentUsesResolvedPort(t *testing.T) {
	env := []string{"RESOURCE_PORT_HTTP=47201", "VAULT_ADDR=http://127.0.0.1:${RESOURCE_PORT_HTTP}"}
	rendered := renderManagedServiceEnvironment(env, map[string]string{"VAULT_ADDR": "http://127.0.0.1:${RESOURCE_PORT_HTTP}"})
	if got := managedServiceEnvValues(rendered)["VAULT_ADDR"]; got != "http://127.0.0.1:47201" {
		t.Fatalf("VAULT_ADDR = %q", got)
	}
}

func TestReadManagedServiceEnvironmentFileIsResourceOwnedAndOverridesDefaults(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "active.env"), []byte("# selected model\nRERANKER_MODEL=small-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := readManagedServiceEnvironmentFile("active.env", []string{"RESOURCE_DATA_DIR=" + dataRoot})
	if err != nil {
		t.Fatal(err)
	}
	if values["RERANKER_MODEL"] != "small-model" {
		t.Fatalf("RERANKER_MODEL = %q, want small-model", values["RERANKER_MODEL"])
	}
	if _, err := readManagedServiceEnvironmentFile("../active.env", []string{"RESOURCE_DATA_DIR=" + dataRoot}); err == nil {
		t.Fatal("environment file traversal unexpectedly succeeded")
	}
}

func TestReadManagedServiceEnvironmentFileMissingUsesManifestDefaults(t *testing.T) {
	dataRoot := t.TempDir()
	values, err := readManagedServiceEnvironmentFile("active.env", []string{"RESOURCE_DATA_DIR=" + dataRoot})
	if err != nil {
		t.Fatalf("missing optional environment file: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("missing optional environment file = %#v, want empty override", values)
	}
}

func TestManagedServiceArtifactPathUsesInstalledSignedArtifactStore(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	controller := NewController(root, home)
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Artifact.BundleArtifact = "fixture_${os}_${arch}"
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		t.Fatal(err)
	}
	name, err := manifest.ManagedService.Artifact.BundleArtifactForPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".vrooli", "artifacts", manifest.Name, "1.0.0", name)
	if path != want {
		t.Fatalf("artifact path = %q, want %q", path, want)
	}
}

func TestWriteManagedServiceConfigRendersOnlyResourceRuntimePaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "runtime"))
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.ManagedService.Config = &resourcedeployment.ServiceConfig{
		Path:    "vault.hcl",
		Content: "path = \"${RESOURCE_DATA_DIR}\"\n",
	}
	env := resourceEnvForResource(root, filepath.Join(root, "home"), manifest.Name)
	if err := writeManagedServiceConfig(manifest, env); err != nil {
		t.Fatal(err)
	}
	values := managedServiceEnvValues(env)
	path := filepath.Join(values["RESOURCE_CONFIG_DIR"], "vault.hcl")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(body), "path = \""+values["RESOURCE_DATA_DIR"]+"\"\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func managedServiceTestManifest(checksum string) ResourceManifest {
	return ResourceManifest{
		Name:      "fixture-service",
		CLI:       &scenario.CLIConfig{Enabled: false},
		Driver:    "managed-service",
		Platforms: ResourcePlatforms{Linux: "supported", MacOS: "supported", Windows: "supported"},
		ManagedService: &resourcedeployment.ManagedService{
			ProviderPolicy: resourcedeployment.ProviderPolicy{
				TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{
					resourcedeployment.ProviderTargetControlPlane:  resourcedeployment.ProviderManagedPrivate,
					resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate,
				},
				AllowedModes:               []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared, resourcedeployment.ProviderAttachOnly},
				SharedReuseRequiresConsent: true,
				ExternalManagement:         "forbidden",
				ExternalAccessCapabilities: []resourcedeployment.AccessCapability{resourcedeployment.AccessReadOnly},
			},
			Artifact:  resourcedeployment.ServiceArtifact{Path: "bin/service", Version: "1.0.0", SHA256: checksum},
			Arguments: []string{"-test.run=TestManagedServiceFixtureProcess", "--"},
		},
	}
}
