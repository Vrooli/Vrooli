//go:build linux

package target

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	pipelineconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	release "github.com/vrooli/vrooli/packages/resource-deployment"
	bundleruntime "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime"
	runtimecomponent "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/components"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"google.golang.org/protobuf/encoding/protojson"
	"scenario-to-desktop-api/bundle"
)

// This producer exports the actual Audio Tools build and signs it with a
// fixture-owned ephemeral release key. It never substitutes component code.
func TestJ11PublishAudioToolsFixture(t *testing.T) {
	root := os.Getenv("J11_PRODUCER_ROOT")
	if root == "" {
		t.Skip("J11_PRODUCER_ROOT selects the isolated release producer")
	}
	source := os.Getenv("J11_AUDIO_BUNDLE")
	if source == "" {
		t.Fatal("real Audio Tools bundle required")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(source, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var audio manifest.Manifest
	if err = json.Unmarshal(raw, &audio); err != nil {
		t.Fatal(err)
	}
	// The isolated LPBS fixture binds port 0 and signs its actual URL before
	// the host is generated. This placeholder never serves or downloads code.
	entry := component.Entry{SchemaVersion: 1, ID: "audio-tools", Owner: audio.App.Scenario, Version: audio.App.Version, OS: "linux", Arch: "amd64", HostRange: ">=0.0.1 <1.0.0", StateClass: "stateless", License: "Vrooli source license", ArtifactURL: "http://127.0.0.1:1/api/v1/components?id=audio-tools", Provides: []component.Feature{{ID: "audio.processing", Service: "api", Port: "api"}}}
	entry, err = bundle.PackageComponent(source, filepath.Join(root, "audio-tools.tar.gz"), entry)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})
	if err = os.WriteFile(filepath.Join(root, "release.key"), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		t.Fatal(err)
	}
	canonical, err := entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	signed := component.SignedEntry{Entry: entry, Signature: release.ReleaseSignature{SchemaVersion: "v1", KeyID: "j11-isolated-release", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}}
	if err = signed.Verify(string(publicPEM)); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.MarshalIndent(signed, "", "  ")
	if err = os.WriteFile(filepath.Join(root, "entry.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "release.pub"), publicPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	base := os.Getenv("J11_HOST_MANIFEST_BASE")
	raw, err = os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	host.Services = func() []manifest.Service {
		var services []manifest.Service
		for _, svc := range host.Services {
			if !strings.HasPrefix(svc.ID, "audio-tools--") {
				services = append(services, svc)
			}
		}
		return services
	}()
	for i := range host.Peers {
		if host.Peers[i].Scenario == "audio-tools" {
			host.Peers[i].BundlePolicy = "discover"
			host.Peers[i].EmbeddedVersion = ""
			host.Peers[i].Features = nil
		}
	}
	host.Components = &component.Source{CatalogURL: "http://127.0.0.1:1/api/v1/components", PublicKeyPEM: string(publicPEM)}
	encoded, _ = json.MarshalIndent(host, "", "  ")
	if err = os.WriteFile(filepath.Join(root, "host-input.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("real owner=%s version=%s artifact_bytes=%d archive_sha256=%s tree_sha256=%s", entry.Owner, entry.Version, entry.Size, entry.SHA256, entry.TreeSHA256)
}

func TestJ11PublishFullAudioFixture(t *testing.T) {
	output := os.Getenv("J11_FULL_AUDIO_OUTPUT")
	if output == "" {
		t.Skip("J11_FULL_AUDIO_OUTPUT selects isolated full-app producer")
	}
	source := os.Getenv("J11_FULL_OWNER_OUTPUT")
	if source == "" {
		t.Fatal("J11_FULL_OWNER_OUTPUT selects the fresh owner bundle directory")
	}
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(source, "owner-input.json")
	if out, err := exec.CommandContext(t.Context(), "deployment-manager", "bundle", "export", "audio-tools", "--include-secrets=false", "--output", fresh).CombinedOutput(); err != nil {
		t.Fatalf("fresh owner export: %v\n%s", err, out)
	}
	owner, err := filepath.Abs("../../../audio-tools")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir("..")
	packaged, err := bundle.NewPackager().Package(owner, fresh, "electron", []string{"linux-amd64"}, filepath.Join(source, "build"))
	if err != nil {
		t.Fatal(err)
	}
	source = packaged.BundleDir
	raw, err := os.ReadFile(filepath.Join(source, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ownerManifest manifest.Manifest
	if err = json.Unmarshal(raw, &ownerManifest); err != nil {
		t.Fatal(err)
	}
	if ownerManifest.Features["audio.processing"].Service != "api" {
		t.Fatal("fresh owner lost explicit Audio feature")
	}
	artifact := buildJ11Artifact(t, "audio-tools", filepath.Join(source, "bundle.json"))
	digest := copyJ10Artifact(t, artifact, output)
	t.Logf("real full Audio Tools AppImage=%s sha256=%s owner_bundle=%s", output, digest, source)
}

// startJ11Release runs the real LPBS delivery composition on native PostgreSQL
// and MinIO. Its port-0 address is signed into the generated host manifest.
func startJ11Release(t *testing.T) (string, string) {
	t.Helper()
	source := os.Getenv("J11_RELEASE_ROOT")
	return startJ11ReleaseFrom(t, source)
}

func startJ11ReleaseFrom(t *testing.T, source string) (string, string) {
	t.Helper()
	if source == "" {
		t.Skip("J11_RELEASE_ROOT selects the real signed Audio Tools release")
	}
	root := t.TempDir()
	entryRaw, err := os.ReadFile(filepath.Join(source, "entry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var release component.SignedEntry
	if err = json.Unmarshal(entryRaw, &release); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"entry.json", "release.key", "release.pub", release.Entry.ID + ".tar.gz"} {
		raw, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Obtain today's UI hashes and source build contracts through the owning
	// exporter. A retained release manifest can outlive its source UI chunks.
	fresh := filepath.Join(root, "fresh-host.json")
	output, err := exec.CommandContext(t.Context(), "deployment-manager", "bundle", "export", "web-console", "--include-secrets=false", "--output", fresh).CombinedOutput()
	if err != nil {
		t.Fatalf("real Deployment Manager host export: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(fresh)
	if err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	services := make([]manifest.Service, 0, len(host.Services))
	for _, svc := range host.Services {
		if !strings.HasPrefix(svc.ID, "audio-tools--") {
			services = append(services, svc)
		}
	}
	host.Services = services
	for i := range host.Peers {
		if host.Peers[i].Scenario == "audio-tools" {
			host.Peers[i].BundlePolicy = "discover"
			host.Peers[i].EmbeddedVersion = ""
			host.Peers[i].Features = nil
		}
	}
	raw, err = json.MarshalIndent(host, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "host-input.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(root, "ready")
	log, err := os.Create(filepath.Join(root, "lpbs.log"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, "go", "test", "-tags=integration", ".", "-run", "^TestDesktopComponentJourneyServer$", "-count=1", "-v", "-timeout=20m")
	command.Dir = "../../../landing-page-business-suite/api"
	command.Env = append(os.Environ(), "J11_RELEASE_ROOT="+root, "J11_SERVER_READY="+ready)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Stdout, command.Stderr = log, log
	if err = command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan struct{})
	var serverErr error
	go func() { serverErr = command.Wait(); close(done) }()
	base := ""
	t.Cleanup(func() {
		if base != "" {
			response, err := http.Post(base+"/__journey/stop", "text/plain", nil)
			if err == nil {
				response.Body.Close()
			}
		}
		select {
		case <-done:
			if serverErr != nil {
				data, _ := os.ReadFile(log.Name())
				t.Errorf("real LPBS helper: %v\n%s", serverErr, data)
			}
		case <-time.After(10 * time.Second):
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			cancel()
			t.Error("LPBS helper did not stop")
		}
		cancel()
		log.Close()
		if evidence := os.Getenv("J11_EVIDENCE"); evidence != "" {
			data, _ := os.ReadFile(log.Name())
			_ = os.WriteFile(filepath.Join(evidence, t.Name()+"-lpbs.log"), data, 0o600)
		}
	})
	deadline := time.NewTimer(3 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-done:
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("LPBS exited before ready: %v\n%s", serverErr, data)
		case <-deadline.C:
			t.Fatal("LPBS component readiness timeout")
		case <-tick.C:
			if raw, err := os.ReadFile(ready); err == nil {
				base = strings.TrimSpace(string(raw))
				return root, base
			}
		}
	}
}

func buildJ11Host(t *testing.T, manifestPath string) string {
	return buildJ11Artifact(t, "web-console", manifestPath)
}

func buildJ11Artifact(t *testing.T, scenario, manifestPath string) string {
	t.Helper()
	base, err := discovery.ResolveScenarioURLDefault(t.Context(), "scenario-to-desktop")
	if err != nil {
		t.Fatal(err)
	}
	client := pipelineconnect.NewPipelineServiceClient(http.DefaultClient, base)
	request := &pipelinev1.PipelineRunRequest{}
	config := fmt.Sprintf(`{"config":{"scenario_name":%q,"platforms":["PLATFORM_LINUX"],"deployment_mode":"DEPLOYMENT_MODE_BUNDLED","framework":"FRAMEWORK_ELECTRON","template_type":"TEMPLATE_TYPE_BASIC","location_mode":"staging","version":"0.0.1","stages":["STAGE_NAME_BUNDLE","STAGE_NAME_PREFLIGHT","STAGE_NAME_GENERATE","STAGE_NAME_BUILD"],"artifact_trust_mode":"development-local","platform_targets":["linux-amd64"],"bundle_manifest_path":%q}}`, scenario, manifestPath)
	if err = protojson.Unmarshal([]byte(config), request); err != nil {
		t.Fatal(err)
	}
	admit, err := client.Run(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("real port-0 component host pipeline %s", admit.Msg.GetPipelineId())
	result, err := client.Wait(t.Context(), connect.NewRequest(&pipelinev1.PipelineWaitRequest{PipelineId: admit.Msg.GetPipelineId(), TimeoutSeconds: 900}))
	if err != nil {
		t.Fatal(err)
	}
	if evidence := os.Getenv("J11_EVIDENCE"); evidence != "" {
		raw, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(result.Msg)
		_ = os.WriteFile(filepath.Join(evidence, t.Name()+"-pipeline.json"), raw, 0o600)
	}
	if result.Msg.GetStatus() != sharedv1.StageStatus_STAGE_STATUS_COMPLETED {
		t.Fatalf("component host pipeline %s (%s): %s", result.Msg.GetPipelineId(), result.Msg.GetStatus(), result.Msg.GetError())
	}
	packaged := result.Msg.GetStages()["bundle"].GetDetails().GetBundle().GetBundleDir()
	raw, err := os.ReadFile(filepath.Join(packaged, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	for _, service := range host.Services {
		if strings.HasPrefix(service.ID, "audio-tools--") {
			t.Fatal("optional Audio Tools was included in the host download")
		}
	}
	if _, err = os.Stat(filepath.Join(packaged, "peers", "audio-tools")); !os.IsNotExist(err) {
		t.Fatal("optional Audio Tools bytes were included in the host download")
	}
	for _, artifact := range result.Msg.GetFinalArtifacts() {
		if strings.HasSuffix(artifact, ".AppImage") {
			return artifact
		}
	}
	t.Fatalf("component host build failed: %s", result.Msg)
	return ""
}

func TestJ11_InstalledFromIntegrationsPageOnCleanHome(t *testing.T) {
	root, _ := startJ11Release(t)
	artifact := buildJ11Host(t, filepath.Join(root, "host-input.json"))
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, artifact, installed)
	session := startJ11Desktop(t, installed, home, digest)
	state := runJ11Browser(t, session, "install")
	if len(state["before"].(map[string]any)["installed"].([]any)) != 0 {
		t.Fatal("clean home already contains optional code")
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, installed, home, digest)
	port := runtimePort(t, home, "audio-tools")
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var health map[string]any
	if err = json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || health["status"] != "healthy" {
		t.Fatalf("real installed Audio Tools health: %d %+v", response.StatusCode, health)
	}
	state = runJ11Browser(t, session, "installed")
	rows := state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected one installed component, got %d", len(rows))
	}
	entry := rows[0].(map[string]any)["signed"].(map[string]any)["entry"].(map[string]any)
	t.Logf("real clean-home UI install, same-home restart and Audio Tools health on %d: %+v; version=%s archive_sha256=%s", port, health, entry["version"], entry["sha256"])
}

func installedJ11Manager(t *testing.T) (*runtimecomponent.Manager, component.SignedEntry, string) {
	t.Helper()
	root, _ := startJ11Release(t)
	raw, err := os.ReadFile(filepath.Join(root, "host-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	m := runtimecomponent.New(t.TempDir(), host.App.Version, host.Components)
	rows, err := m.Catalog(t.Context())
	if err != nil || len(rows) != 1 {
		t.Fatalf("real LPBS catalog: rows=%d err=%v", len(rows), err)
	}
	if _, err = m.Install(t.Context(), rows[0]); err != nil {
		t.Fatal(err)
	}
	return m, rows[0], root
}

func TestJ11_EmbeddedAndInstalledEntriesIdentical(t *testing.T) {
	m, signed, _ := installedJ11Manager(t)
	source := os.Getenv("J11_OWNER_BUNDLE")
	if source == "" {
		t.Fatal("J11_OWNER_BUNDLE must identify the real Audio Tools owner bundle used by the release producer")
	}
	raw, err := os.ReadFile(filepath.Join(source, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var owner manifest.Manifest
	if err = json.Unmarshal(raw, &owner); err != nil {
		t.Fatal(err)
	}
	if owner.App.Scenario != signed.Entry.Owner || owner.App.Version != signed.Entry.Version {
		t.Fatal("owner bundle identity differs")
	}
	embedded, err := manifest.ProjectComponent(owner.App.Scenario, owner.Services)
	if err != nil {
		t.Fatal(err)
	}
	host := manifest.Manifest{}
	if err = m.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	// Physical roots/data roots are per installation; the declarative service
	// entries must be byte-for-byte identical to the embedded owner projection.
	a, _ := json.Marshal(embedded)
	b, _ := json.Marshal(host.Services)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("embedded/installed service entries differ\nembedded=%s\ninstalled=%s", a, b)
	}
	if len(host.Services) == 0 {
		t.Fatal("real component has no services")
	}
	for _, svc := range host.Services {
		if svc.ComponentID != signed.Entry.ID || svc.BundleRoot == "" {
			t.Fatal("installed service has no verified generation root")
		}
	}
	// Select the same explicit owner capability from the actual embedded
	// graph. Its files are the identical verified owner archive just installed.
	embeddedHost := owner
	embeddedHost.App.Scenario = "web-console"
	embeddedHost.Features = nil
	embeddedHost.Authentication = nil
	embeddedHost.Components = nil
	embeddedHost.Services = embedded
	features := map[string]manifest.FeatureExport{}
	for id, feature := range owner.Features {
		feature.Service = owner.App.Scenario + "--" + feature.Service
		features[id] = feature
	}
	embeddedHost.Peers = []manifest.Peer{{Scenario: owner.App.Scenario, BundlePolicy: "embed", EmbeddedVersion: owner.App.Version, VersionRange: ">=0.0.1 <1.0.0", Features: features}}
	if len(features) == 0 {
		t.Fatal("owner export has no explicit feature bindings")
	}
	embeddedRoot := filepath.Join(m.Root, signed.Entry.ID, "generations", signed.Entry.SHA256)
	embeddedRuntime := startJ11Native(t, embeddedRoot, t.TempDir(), t.TempDir(), &embeddedHost)
	selected, err := embeddedRuntime.ResolveFeature(t.Context(), owner.App.Scenario, "audio.processing", "")
	if err != nil || selected.Source != "embedded" || selected.OwnerVersion != signed.Entry.Version {
		t.Fatalf("embedded owner feature selection: %+v %v", selected, err)
	}
	if _, err := embeddedRuntime.ResolveFeature(t.Context(), owner.App.Scenario, "audio.processing", ">=1.0.0"); err == nil {
		t.Fatal("embedded feature ignored consumer version range")
	}
	embeddedHost.Peers = append(embeddedHost.Peers, embeddedHost.Peers[0])
	if _, err := embeddedRuntime.ResolveFeature(t.Context(), owner.App.Scenario, "audio.processing", ""); err == nil {
		t.Fatal("ambiguous embedded owner bindings were accepted")
	}
	t.Logf("real owner %s/%s: %d identical service entries; archive=%s", owner.App.Scenario, owner.App.Version, len(host.Services), signed.Entry.SHA256)
}

func TestJ11_TamperedEntryRefused(t *testing.T) {
	m, signed, releaseRoot := installedJ11Manager(t)
	// Even a genuine publisher signature cannot relabel host/other-component
	// ports as this owner. Exercise admission against the real installed release.
	keyPEM, err := os.ReadFile(filepath.Join(releaseRoot, "release.key"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(keyPEM)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := signed.Entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"api", "web-console--api"} {
		wrong := signed
		wrong.Entry.Provides = append([]component.Feature(nil), signed.Entry.Provides...)
		original := wrong.Entry.Provides[0].Service
		wrong.Entry.Provides[0].Service = target
		badCanonical := bytes.Replace(canonical, []byte(fmt.Sprintf(`"service":%q`, original)), []byte(fmt.Sprintf(`"service":%q`, target)), 1)
		if bytes.Equal(canonical, badCanonical) {
			t.Fatal("feature signature fixture did not change signed tuple")
		}
		sum := sha256.Sum256(badCanonical)
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if err != nil {
			t.Fatal(err)
		}
		if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, sum[:], sig); err != nil {
			t.Fatal(err)
		}
		wrong.Signature.Signature = base64.StdEncoding.EncodeToString(sig)
		if _, err := m.Install(t.Context(), wrong); err == nil {
			t.Fatalf("valid signature claimed outside component endpoint %s", target)
		}
	}
	tampered := signed
	tampered.Entry.Version = "0.0.2"
	if _, err := m.Install(t.Context(), tampered); err == nil {
		t.Fatal("tampered signed entry was admitted")
	}
	rows, err := m.Installed(t.Context())
	if err != nil || len(rows) != 1 || rows[0].Signed.Entry.Version != signed.Entry.Version {
		t.Fatalf("tamper altered old installation: %v", err)
	}
	root := filepath.Join(m.Root, signed.Entry.ID, "generations", signed.Entry.SHA256)
	if err = os.WriteFile(filepath.Join(root, "tampered"), []byte("unexpected executable content"), 0o600); err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = m.Merge(t.Context(), &host); err != nil {
		t.Fatal(err)
	}
	if len(host.Services) != 0 {
		t.Fatal("tampered extracted generation entered the startup service graph")
	}
	rows, err = m.Installed(t.Context())
	if err != nil || rows[0].State != runtimecomponent.NeedsUpdate || rows[0].Reason != "installed tree failed verification" {
		t.Fatalf("tampered generation did not fail closed: %v %+v", err, rows)
	}
	t.Log("real LPBS release installed; modified signed metadata refused before replacement; modified generation excluded at boot")
}

func newJ11InstalledDesktop(t *testing.T) (*ElectronSession, string, string, string, string) {
	t.Helper()
	root, base := startJ11Release(t)
	artifact := buildJ11Host(t, filepath.Join(root, "host-input.json"))
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, artifact, installed)
	session := startJ11Desktop(t, installed, home, digest)
	runJ11Browser(t, session, "install")
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	return startJ11Desktop(t, installed, home, digest), home, installed, digest, base
}

func TestJ11_UpdateFailureKeepsOldVersion(t *testing.T) {
	if os.Getenv("J11_RELEASE_ROOT") == "" {
		t.Skip("J11_RELEASE_ROOT selects the real signed Audio Tools release")
	}
	// Keep the owner's executable and assets intact. Only its release identity
	// and startup configuration change, so verification succeeds before the real
	// Audio Tools binary fails to serve its allocated API port.
	owner := os.Getenv("J11_OWNER_BUNDLE")
	if owner == "" {
		t.Fatal("J11_OWNER_BUNDLE must identify the actual Audio Tools owner bundle")
	}
	variant := t.TempDir()
	variantOwner := filepath.Join(variant, "owner")
	if err := os.CopyFS(variantOwner, os.DirFS(owner)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(variantOwner, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var next manifest.Manifest
	if err = json.Unmarshal(raw, &next); err != nil {
		t.Fatal(err)
	}
	next.App.Version = "0.0.3"
	foundAPI := false
	for i := range next.Services {
		if next.Services[i].ID == "api" {
			if next.Services[i].Env == nil {
				next.Services[i].Env = map[string]string{}
			}
			next.Services[i].Env["API_PORT"] = "invalid"
			next.Services[i].Readiness.TimeoutMs = 1000
			foundAPI = true
		}
	}
	if !foundAPI {
		t.Fatal("actual owner has no Audio Tools API service")
	}
	raw, err = json.MarshalIndent(next, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(variantOwner, "bundle.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(os.Getenv("J11_RELEASE_ROOT"), "entry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var original component.SignedEntry
	if err = json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	entry := original.Entry
	entry.Version = next.App.Version
	entry, err = bundle.PackageComponent(variantOwner, filepath.Join(variant, "audio-tools.tar.gz"), entry)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(variant, "entry.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("J11_STARTUP_FAILURE_RELEASE_ROOT", variant)
	session, home, artifact, digest, base := newJ11InstalledDesktop(t)
	port := runtimePort(t, home, "audio-tools")
	response, err := http.Post(base+"/__journey/broken-update", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("publish failing update: HTTP %d", response.StatusCode)
	}
	state := runJ11Browser(t, session, "update-failure")
	rows := state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["signed"].(map[string]any)["entry"].(map[string]any)["version"] != "0.0.1" {
		t.Fatal("failed update changed installed generation")
	}
	response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("old generation stopped serving after failed update")
	}
	if err = session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, artifact, home, digest)
	port = runtimePort(t, home, "audio-tools")
	response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("old generation not healthy after restart")
	}
	oldGeneration := rows[0].(map[string]any)["generation"]
	var componentRoot string
	if err = filepath.WalkDir(home, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Name() == "installations.json" && filepath.Base(filepath.Dir(path)) == "components" {
			componentRoot = filepath.Dir(path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if componentRoot == "" {
		t.Fatal("installed component state missing")
	}
	data := filepath.Join(componentRoot, "audio-tools", "data", "saved-setting")
	if err = os.MkdirAll(filepath.Dir(data), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(data, []byte("before-startup-failure"), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err = http.Post(base+"/__journey/startup-failure-update", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("publish checksum-valid startup-failing update: HTTP %d", response.StatusCode)
	}
	state = runJ11Browser(t, session, "update-startup-failure")
	rows = state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 {
		t.Fatalf("staged candidate count=%d", len(rows))
	}
	staged := rows[0].(map[string]any)
	if staged["state"] != runtimecomponent.Pending || staged["generation"] != entry.SHA256 || staged["signed"].(map[string]any)["entry"].(map[string]any)["version"] != "0.0.3" {
		t.Fatalf("valid update did not stage the actual candidate: %+v", staged)
	}
	previous, ok := staged["previous"].(map[string]any)
	if !ok || previous["version"] != "0.0.1" || previous["generation"] != oldGeneration {
		t.Fatalf("staging did not retain the previous verified generation: %+v", staged)
	}
	if err = session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, artifact, home, digest)
	port = runtimePort(t, home, "audio-tools")
	response, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatal(err)
	}
	var health map[string]any
	err = json.NewDecoder(response.Body).Decode(&health)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || health["status"] != "healthy" {
		t.Fatalf("old Audio Tools not healthy after candidate startup failure: HTTP %d health=%+v err=%v", response.StatusCode, health, err)
	}
	state = runJ11Browser(t, session, "installed")
	rows = state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["state"] != runtimecomponent.Installed || rows[0].(map[string]any)["generation"] != oldGeneration || rows[0].(map[string]any)["signed"].(map[string]any)["entry"].(map[string]any)["version"] != "0.0.1" {
		t.Fatalf("startup failure did not restore the old installed generation: %+v", rows)
	}
	if restored := rows[0].(map[string]any); restored["previous"] != nil || restored["reason"] != "The update could not start; the previous version was restored." {
		t.Fatalf("startup failure did not finish rollback: %+v", restored)
	}
	if preserved, err := os.ReadFile(data); err != nil || string(preserved) != "before-startup-failure" {
		t.Fatalf("old component data lost after rollback: %q %v", preserved, err)
	}
	t.Logf("real LPBS checksum failure retained 0.0.1; signed/checksum-valid 0.0.3 archive=%s staged through UI, real Audio API startup failed, same-home restart restored healthy installed 0.0.1 generation=%s and preserved data", entry.SHA256, oldGeneration)
	// Stage the identical, verified candidate again. The retained old tree is
	// deliberately changed only after shutdown, so it cannot be trusted as a
	// fallback when the real candidate fails startup on the next launch.
	state = runJ11Browser(t, session, "update-startup-failure")
	rows = state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 {
		t.Fatalf("restaged candidate count=%d", len(rows))
	}
	staged = rows[0].(map[string]any)
	previous, ok = staged["previous"].(map[string]any)
	if staged["state"] != runtimecomponent.Pending || staged["generation"] != entry.SHA256 || !ok || previous["version"] != "0.0.1" || previous["generation"] != oldGeneration {
		t.Fatalf("second update did not stage the same candidate and retain the old generation: %+v", staged)
	}
	if err = session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	oldManifest := filepath.Join(componentRoot, "audio-tools", "generations", oldGeneration.(string), "bundle.json")
	oldRaw, err := os.ReadFile(oldManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(oldManifest, append(oldRaw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, artifact, home, digest)
	state = runJ11Browser(t, session, "activation-disabled")
	rows = state["after"].(map[string]any)["installed"].([]any)
	if len(rows) != 1 {
		t.Fatalf("disabled component count=%d", len(rows))
	}
	disabled := rows[0].(map[string]any)
	const disabledReason = "The dependency could not start and no healthy compatible version is available. Update or remove it."
	if disabled["state"] != runtimecomponent.NeedsUpdate || disabled["reason"] != disabledReason || disabled["activation_disabled"] != true || disabled["previous"] != nil || disabled["generation"] != entry.SHA256 {
		t.Fatalf("failed candidate without a trusted fallback did not stay off with a reason: %+v", disabled)
	}
	for _, route := range []string{"/scenario-port?scenario=audio-tools&port_name=API_PORT", "/features?scenario=audio-tools&feature=audio.processing"} {
		if code, body := j11RuntimeRead(t, home, route); code != http.StatusNotFound {
			t.Fatalf("disabled Audio Tools remained discoverable at %s: HTTP%d %+v", route, code, body)
		}
	}
	client := &http.Client{Timeout: time.Second}
	if response, err = client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port)); err == nil {
		response.Body.Close()
		t.Fatal("previous Audio Tools API is still serving after activation was disabled")
	}
	if preserved, err := os.ReadFile(data); err != nil || string(preserved) != "before-startup-failure" {
		t.Fatalf("old component data lost while leaving failed dependency off: %q %v", preserved, err)
	}
	t.Log("restaged identical real 0.0.3, changed retained old tree while closed, restarted healthy Web Console; no compatible trusted Audio fallback was admitted, UI explains Needs update, bare/feature lookup return404 and saved data remains")
}

func TestJ11_UninstallOffersDataChoice(t *testing.T) {
	session, home, artifact, digest, _ := newJ11InstalledDesktop(t)
	var componentRoot string
	if err := filepath.WalkDir(home, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "installations.json" && filepath.Base(filepath.Dir(path)) == "components" {
			componentRoot = filepath.Dir(path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if componentRoot == "" {
		t.Fatal("installed component state missing")
	}
	data := filepath.Join(componentRoot, "audio-tools", "data", "saved-setting")
	if err := os.MkdirAll(filepath.Dir(data), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("user-choice"), 0o600); err != nil {
		t.Fatal(err)
	}
	port := runtimePort(t, home, "audio-tools")
	runJ11Browser(t, session, "remove-keep")
	if _, err := os.Stat(data); err != nil {
		t.Fatal("keep-data choice deleted data")
	}
	if _, err := os.Stat(filepath.Join(componentRoot, "audio-tools", "generations")); !os.IsNotExist(err) {
		t.Fatal("removed executable generation remains")
	}
	client := &http.Client{Timeout: time.Second}
	if response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port)); err == nil {
		response.Body.Close()
		t.Fatal("removed component still serving")
	}
	runJ11Browser(t, session, "install")
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, artifact, home, digest)
	t.Setenv("J11_VIEWPORT", "phone")
	runJ11Browser(t, session, "remove-delete")
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("delete-data choice retained data")
	}
	t.Log("real UI offered keep/delete/cancel; cancel retained installation; keep removed running code and retained data; reinstall/delete removed both")
}

func TestJ11_AddHereOrGetFullAppBothOffered(t *testing.T) {
	// The release root selects this live journey; once selected, the full
	// AppImage is a required input like the journey's other artifacts.
	root, _ := startJ11Release(t)
	full := os.Getenv("J11_FULL_AUDIO_APPIMAGE")
	if full == "" {
		t.Fatal("real published full Audio AppImage required")
	}
	artifact := buildJ11Host(t, filepath.Join(root, "host-input.json"))
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, artifact, installed)
	session := startJ11Desktop(t, installed, home, digest)
	state := runJ11Browser(t, session, "both-offered")
	t.Setenv("J11_VIEWPORT", "phone")
	runJ11Browser(t, session, "both-offered")
	url, _ := state["full_app_url"].(string)
	if url == "" {
		t.Fatal("full-app control has no published URL")
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("real full-app delivery HTTP%d", response.StatusCode)
	}
	want, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	got := sha256.New()
	if _, err = io.Copy(got, response.Body); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", got.Sum(nil)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
		t.Fatal("full-app link did not deliver actual owner AppImage")
	}
	t.Log("real clean-home Integrations offered Add here and Get full app; signed full-app link delivered actual Audio AppImage from LPBS/MinIO")
}

func j11RuntimeRead(t *testing.T, home, route string) (int, map[string]any) {
	t.Helper()
	var stateDir string
	if err := filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "ipc_port" && filepath.Base(filepath.Dir(path)) == "runtime" {
			stateDir = filepath.Dir(path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	portRaw, err := os.ReadFile(filepath.Join(stateDir, "ipc_port"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(portRaw)))
	if err != nil {
		t.Fatal(err)
	}
	token, err := os.ReadFile(filepath.Join(stateDir, "auth_token"))
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, route), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body map[string]any
	if response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
	}
	return response.StatusCode, body
}

type j11IsolatedEnv map[string]string

func (e j11IsolatedEnv) Getenv(key string) string { return e[key] }
func (e j11IsolatedEnv) Environ() []string {
	values := make([]string, 0, len(e))
	for key, value := range e {
		values = append(values, key+"="+value)
	}
	return values
}

func startJ11Native(t *testing.T, source, home, data string, input *manifest.Manifest) *bundleruntime.Supervisor {
	t.Helper()
	s, err := bundleruntime.NewSupervisor(bundleruntime.Options{Manifest: input, BundlePath: source, AppDataDir: data, PeerHomeDir: home, EnvReader: j11IsolatedEnv{"HOME": home, "PATH": "/usr/bin:/bin", "USERPROFILE": home, "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local/share")}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	return s
}

func j11PeerLifecycle(t *testing.T, closeFull bool) {
	t.Helper()
	m, signed, root := installedJ11Manager(t)
	raw, err := os.ReadFile(filepath.Join(root, "host-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var host manifest.Manifest
	if err = json.Unmarshal(raw, &host); err != nil {
		t.Fatal(err)
	}
	// Run the production native supervisor with this optional service graph.
	// The renderer/install control path is independently operated by the other
	// J11 checks; both providers here execute real Audio Tools owner binaries.
	host.Services = nil
	host.Authentication = nil
	peerHome := t.TempDir()
	local := startJ11Native(t, t.TempDir(), peerHome, filepath.Dir(m.Root), &host)
	first, err := local.ResolveFeature(t.Context(), "audio-tools", "audio.processing", "")
	if err != nil || first.Source != "component" {
		t.Fatalf("local component: %+v %v", first, err)
	}
	source := os.Getenv("J11_OWNER_BUNDLE")
	if source == "" {
		t.Fatal("real Audio owner bundle required")
	}
	raw, err = os.ReadFile(filepath.Join(source, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var full manifest.Manifest
	if err = json.Unmarshal(raw, &full); err != nil {
		t.Fatal(err)
	}
	standalone := startJ11Native(t, source, peerHome, t.TempDir(), &full)
	fullURL := fmt.Sprintf("http://127.0.0.1:%d", standalone.PortMap()["api"]["api"])
	deadline := time.Now().Add(30 * time.Second)
	for {
		response, err := http.Get(fullURL + "/health")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("real full app unhealthy: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	selected, err := local.ResolveFeature(t.Context(), "audio-tools", "audio.processing", "")
	if err != nil || selected.Source != "full_app" || selected.URL != fullURL {
		t.Fatalf("live full app must win: %+v %v", selected, err)
	}
	if response, err := http.Get(first.URL + "/health"); err == nil {
		response.Body.Close()
		t.Fatal("installed copy remains running beside full app")
	}
	if closeFull {
		if err := standalone.Shutdown(t.Context()); err != nil {
			t.Fatal(err)
		}
		back, err := local.ResolveFeature(t.Context(), "audio-tools", "audio.processing", "")
		if err != nil || back.Source != "component" || back.ComponentID != signed.Entry.ID {
			t.Fatalf("component did not return: %+v %v", back, err)
		}
		response, err := http.Get(back.URL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatal("returned component unhealthy")
		}
		t.Log("real full Audio app closed; call-time resolver restarted the installed verified owner generation and returned its healthy API")
	} else {
		t.Log("real standalone Audio supervisor/API won; installed component process stopped before resolving to full app")
	}
}

func TestJ11_LiveFullAppWinsOverSlice(t *testing.T)      { j11PeerLifecycle(t, false) }
func TestJ11_SliceReturnsWhenFullAppCloses(t *testing.T) { j11PeerLifecycle(t, true) }

func startJ11Desktop(t *testing.T, artifact, home, digest string) *ElectronSession {
	t.Helper()
	session, err := StartElectronSession(t.Context(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         ElectronLaunchOptions{ContextID: "agent-connect-j11", ScenarioName: "web-console", ArtifactDigest: "sha256:" + digest, TargetID: "local-linux-xvfb", JourneyID: "J11", ProfileID: "optional-component-home", IsolationLeaseID: "j11-component-home", BundledRuntime: true, UserDataDir: filepath.Join(home, "electron")},
		Environment:    map[string]string{"HOME": home, "USERPROFILE": home, "PATH": filepath.Join(home, ".local/bin") + ":/usr/bin:/bin", "VROOLI_ROOT": "", "WC_SESSION_STATE_ROOT": filepath.Join(home, "sessions"), "WC_CODEX_SESSIONS_DIR": filepath.Join(home, "sessions", "codex"), "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local/share"), "XDG_STATE_HOME": filepath.Join(home, ".local/state"), "XDG_CACHE_HOME": filepath.Join(home, ".cache"), "LIBGL_ALWAYS_SOFTWARE": "1"},
		StartupTimeout: 75 * time.Second, Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(context.Background()); err != nil {
			t.Error(err)
		}
		if process, ok := session.process.(*execElectronProcess); ok {
			if evidence := os.Getenv("J11_EVIDENCE"); evidence != "" {
				_ = os.WriteFile(filepath.Join(evidence, fmt.Sprintf("%s-desktop-%d.log", t.Name(), process.PID())), []byte(process.output.String()), 0o600)
			}
		}
	})
	return session
}

func runJ11Browser(t *testing.T, session *ElectronSession, mode string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 75*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "python3", "desktop_component_journey.py", session.endpoint, mode).CombinedOutput()
	if err != nil {
		t.Fatalf("real component UI %s: %v\n%s", mode, err, out)
	}
	var state map[string]any
	if err = json.Unmarshal(out, &state); err != nil {
		t.Fatalf("component UI output: %v\n%s", err, out)
	}
	return state
}

// Diagnostic only: operate a previously generated real host, without publishing
// another catalog or building another shell merely to inspect its empty-home UI.
func TestJ11InspectHostRenderer(t *testing.T) {
	artifact := os.Getenv("J11_INSPECT_APPIMAGE")
	if artifact == "" {
		t.Skip("J11_INSPECT_APPIMAGE selects an existing real host for renderer diagnosis")
	}
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, artifact, installed)
	session := startJ11Desktop(t, installed, home, digest)
	mode := os.Getenv("J11_INSPECT_MODE")
	if mode == "" {
		mode = "inspect"
	}
	t.Logf("real packaged renderer: %+v", runJ11Browser(t, session, mode))
}
