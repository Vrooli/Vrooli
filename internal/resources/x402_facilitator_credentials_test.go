package resources

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/process"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/securestore"
	"github.com/vrooli/vrooli/internal/testenv"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestX402FacilitatorSignerIsResolvedFromCAOnlyIntoManagedChild(t *testing.T) {
	manifestBytes, err := os.ReadFile(filepath.Join("..", "..", "resources", "x402-facilitator", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest manifestpkg.ResourceManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "x402-facilitator" || len(manifest.Credentials.Descriptors) != 1 {
		t.Fatalf("unexpected facilitator credential declaration: %+v", manifest.Credentials.Descriptors)
	}
	descriptor := manifest.Credentials.Descriptors[0]
	if descriptor.LogicalID != "vrooli/x402-facilitator" || descriptor.Field != "facilitator-private-key" || descriptor.Env != "FACILITATOR_PRIVATE_KEY" {
		t.Fatalf("unexpected facilitator credential descriptor: %+v", descriptor)
	}

	store := testenv.NewCredentialStore(securestore.ErrNotFound)
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	previousAuthority := credentialauthority.DefaultAuthority
	credentialauthority.DefaultAuthority = func() (*credentialauthority.Authority, error) { return authority, nil }
	t.Cleanup(func() { credentialauthority.DefaultAuthority = previousAuthority })

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	secret := "0x" + hex.EncodeToString(key)
	if err := authority.Put(credentialauthority.Identity(descriptor.LogicalID), descriptor.ResolvedField(), secret); err != nil {
		t.Fatal(err)
	}
	resolution, err := resourceenv.ResolveCredentialValues(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Values[descriptor.Env] != secret || len(resolution.Missing) != 0 {
		t.Fatalf("credential resolution did not inject the CA value into the child environment: %+v", resolution.Missing)
	}
	childEnv, err := managedServiceRuntimeEnvironment(context.Background(), NewController(projectRoot(t), t.TempDir()), manifest, t.TempDir(), resourcedeployment.ServiceArtifact{Layout: "dir"})
	if err != nil {
		t.Fatalf("resolve managed-service child environment: %v", err)
	}
	if got := managedServiceEnvValues(childEnv)[descriptor.Env]; got != secret {
		t.Fatal("managed service runtime environment did not carry the resolved CA signer into the child")
	}

	configBytes, err := os.ReadFile(filepath.Join("..", "..", "resources", "x402-facilitator", "config", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifestBytes), secret) || strings.Contains(string(configBytes), secret) {
		t.Fatal("facilitator test signer entered the repository manifest or config")
	}
	var safeConfig struct {
		Chains  map[string]json.RawMessage `json:"chains"`
		Schemes []json.RawMessage          `json:"schemes"`
	}
	if err := json.Unmarshal(configBytes, &safeConfig); err != nil {
		t.Fatal(err)
	}
	if len(safeConfig.Chains) != 0 || len(safeConfig.Schemes) != 0 {
		t.Fatal("shared facilitator config is no longer the empty-chain bootstrap")
	}

	if testing.Short() {
		t.Skip("managed child boundary uses a helper process")
	}
	stateDir := t.TempDir()
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256Bytes(binary))
	supervisor := newManagedServiceSupervisor(filepath.Join(stateDir, "managed.json"), filepath.Join(stateDir, "managed.log"))
	args := []string{"-test.run=^TestManagedServiceFixtureProcess$", "--"}
	environment := append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1", descriptor.Env+"="+resolution.Values[descriptor.Env])
	state, err := supervisor.Start(os.Args[0], resourcedeployment.ServiceArtifact{Path: "test-fixture", Version: "1", SHA256: digest}, args, environment, stateDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := supervisor.Stop(ctx); err != nil {
			t.Errorf("stop managed child: %v", err)
		}
		stopped = true
	}
	t.Cleanup(stop)

	if runtime.GOOS == "linux" {
		liveEnvironment, err := process.ReadEnvironment(state.PID)
		if err != nil {
			t.Fatal(err)
		}
		if liveEnvironment[descriptor.Env] != secret {
			t.Fatal("managed child did not receive the exact Credential Authority value")
		}
	}
	argv := managedServiceRunAsArgs("runuser", "test-user", os.Args[0], args, environment)
	for _, argument := range argv {
		if strings.Contains(argument, secret) {
			t.Fatal("signer credential entered managed-service wrapper argv")
		}
	}

	var serializedState ManagedServiceState
	stateBytes, err := os.ReadFile(filepath.Join(stateDir, "managed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stateBytes), secret) {
		t.Fatal("signer credential entered managed-service state/status data")
	}
	if err := json.Unmarshal(stateBytes, &serializedState); err != nil || serializedState.PID != state.PID {
		t.Fatalf("managed-service state did not persist valid metadata: state=%+v err=%v", serializedState, err)
	}
	status, running, err := supervisor.Status()
	if err != nil || !running || status.PID != state.PID {
		t.Fatalf("managed status = (%+v, running=%t, err=%v)", status, running, err)
	}
	statusBytes, err := json.Marshal(status)
	if err != nil || strings.Contains(string(statusBytes), secret) {
		t.Fatalf("managed status leaked the signer value: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "managed.log")); err != nil {
		t.Fatal(err)
	}
	logBytes, err := os.ReadFile(filepath.Join(stateDir, "managed.log"))
	if err != nil || strings.Contains(string(logBytes), secret) {
		t.Fatalf("managed-service logs leaked the signer value: err=%v", err)
	}
	stop()
}

func TestX402LocalTestConfigKeepsSignerAsChildEnvironmentReference(t *testing.T) {
	root := projectRoot(t)
	manifest, err := manifestpkg.Load(filepath.Join(root, "resources", "x402-facilitator-local-test", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	secret := "0x" + strings.Repeat("a", 64)
	configDir := t.TempDir()
	env := []string{
		"RESOURCE_CONFIG_DIR=" + configDir,
		"RESOURCE_PORT_HTTP=14021",
		"FACILITATOR_PRIVATE_KEY=" + secret,
	}
	if err := writeManagedServiceConfig(&Controller{Root: root, Home: t.TempDir()}, manifest, env); err != nil {
		t.Fatalf("write managed-service config: %v", err)
	}
	configPath := filepath.Join(configDir, filepath.FromSlash(manifest.ManagedService.Config.Path))
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("Credential Authority signer was persisted in managed-service config")
	}
	if !strings.Contains(string(data), "${FACILITATOR_PRIVATE_KEY}") {
		t.Fatal("managed-service config lost its child-only signer environment reference")
	}
}

func sha256Bytes(contents []byte) [32]byte {
	return sha256.Sum256(contents)
}

func TestX402LocalTestFacilitatorIsPinnedToLoopbackBaseFork(t *testing.T) {
	root := filepath.Join("..", "..", "resources", "x402-facilitator-local-test")
	manifestBytes, err := os.ReadFile(filepath.Join(root, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest manifestpkg.ResourceManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "x402-facilitator-local-test" || len(manifest.Credentials.Descriptors) != 1 {
		t.Fatalf("unexpected local test manifest identity or credential descriptors: %s %+v", manifest.Name, manifest.Credentials.Descriptors)
	}
	d := manifest.Credentials.Descriptors[0]
	if d.LogicalID != "vrooli/x402-facilitator-local-test" || d.Field != "facilitator-private-key" || d.Env != "FACILITATOR_PRIVATE_KEY" {
		t.Fatalf("unexpected local test signer descriptor: %+v", d)
	}

	var shared manifestpkg.ResourceManifest
	sharedBytes, err := os.ReadFile(filepath.Join("..", "..", "resources", "x402-facilitator", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(sharedBytes, &shared); err != nil {
		t.Fatal(err)
	}
	if shared.ManagedService == nil || manifest.ManagedService == nil || !reflect.DeepEqual(shared.ManagedService.Acquisition, manifest.ManagedService.Acquisition) {
		t.Fatal("local test facilitator must use the exact same pinned OCI acquisition target as the shared resource")
	}
	sharedArtifact := shared.ManagedService.Artifact
	localArtifact := manifest.ManagedService.Artifact
	sharedArtifact.SHA256ByPlatform = localArtifact.SHA256ByPlatform
	if !reflect.DeepEqual(sharedArtifact, localArtifact) {
		t.Fatalf("local test artifact metadata differs beyond its independently pinned extracted-tree checksum: shared=%+v local=%+v", sharedArtifact, localArtifact)
	}

	var config struct {
		Port   int    `json:"port"`
		Host   string `json:"host"`
		Chains map[string]struct {
			RPC []struct {
				HTTP string `json:"http"`
			} `json:"rpc"`
			Signers []string `json:"signers"`
		} `json:"chains"`
		Schemes []struct {
			ID string `json:"id"`
		} `json:"schemes"`
	}
	configBytes, err := os.ReadFile(filepath.Join(root, "config", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatal(err)
	}
	chain, ok := config.Chains["eip155:8453"]
	if config.Host != "127.0.0.1" || config.Port != 14021 || len(config.Chains) != 1 || !ok || len(chain.RPC) != 1 || chain.RPC[0].HTTP != "http://127.0.0.1:18545" || len(chain.Signers) != 1 || chain.Signers[0] != "$FACILITATOR_PRIVATE_KEY" {
		t.Fatalf("local test facilitator escaped the loopback Base-fork profile: host=%q port=%d chains=%+v", config.Host, config.Port, config.Chains)
	}
	if strings.Contains(string(manifestBytes), "0x") || strings.Contains(string(configBytes), "0x") {
		t.Fatal("local test signer value must only be injected from Credential Authority")
	}
}
