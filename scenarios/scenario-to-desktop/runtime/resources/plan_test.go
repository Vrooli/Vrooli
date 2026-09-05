package resources

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestLoadValidatesBundledClientArtifacts(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "resource-demo_" + runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOOS == "darwin" {
		name = "resource-demo_darwin_" + runtime.GOARCH
	}
	files := []string{name, name + ".manifest.json", name + ".build.json"}
	artifacts := make([]Artifact, 0, 3)
	for _, file := range files {
		body := []byte(`{"name":"demo"}`)
		if file == name {
			body = []byte("binary")
		}
		if file == name+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + name + `","os":"` + artifactOS(runtimeOS()) + `","arch":"` + runtime.GOARCH + `"}`)
		}
		if err := os.WriteFile(filepath.Join(resourceDir, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		artifacts = append(artifacts, Artifact{Name: file, SHA256: hex.EncodeToString(sum[:])})
	}
	plan := Plan{SchemaVersion: "v3", Resources: []Item{{RequestedResource: "demo", Resource: "demo", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-client", Support: "supported", Privilege: "user", Bundling: "vendorable", Artifact: name, Files: artifacts}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, name), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected artifact tampering to fail")
	}
}

func TestResolveDeferredOffersRequiresHostCapabilityAndExplicitOutcome(t *testing.T) {
	plan := &Plan{DeferredTargets: []DeferredResourceTarget{{
		Resource: "whisper", OS: runtimeOS(), Architecture: runtime.GOARCH,
		When: map[string]string{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "has:vulkan"},
		Kind: "oci-image", Image: "ghcr.io/example/whisper@sha256:" + strings.Repeat("a", 64), SizeBytes: 123456,
	}}}
	if got := ResolveDeferredOffers(plan, binaryfetch.Facts{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "cpu"}, nil); len(got) != 0 {
		t.Fatalf("CPU facts unexpectedly offered %v", got)
	}
	got := ResolveDeferredOffers(plan, binaryfetch.Facts{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "vulkan"}, nil)
	if len(got) != 1 || got[0].Resource != "whisper" || got[0].SizeBytes != 123456 {
		t.Fatalf("offers = %#v", got)
	}
	outcomes := map[string]UpgradeOutcome{"whisper": {Resource: "whisper", Decision: "declined"}}
	if got := ResolveDeferredOffers(plan, binaryfetch.Facts{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "vulkan"}, outcomes); len(got) != 0 {
		t.Fatalf("declined candidate re-offered: %#v", got)
	}
}

func TestUpgradeOutcomePersistsDeclineAndFailureWithoutTouchingBundle(t *testing.T) {
	appData := t.TempDir()
	for _, decision := range []string{"declined", "failed"} {
		if err := RecordUpgradeOutcome(appData, UpgradeOutcome{Resource: "whisper", Decision: decision}); err != nil {
			t.Fatal(err)
		}
		outcomes, err := LoadUpgradeOutcomes(appData)
		if err != nil {
			t.Fatal(err)
		}
		if outcomes["whisper"].Decision != decision {
			t.Fatalf("decision = %#v", outcomes)
		}
		info, err := os.Stat(filepath.Join(appData, "resource-upgrade-outcomes.json"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("outcome permissions = %o", info.Mode().Perm())
		}
	}
}

func TestApplyDeferredUpgradeFailureLeavesStagedArtifact(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "whisper", "fallback")
	if err := os.MkdirAll(filepath.Join(resourceDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "bin", "server"), []byte("fallback"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := binaryfetch.TreeDigest(filepath.Join(root, "resources", "whisper", "fallback"))
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{
		Resources:       []Item{{Resource: "whisper", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service", Support: "supported", Privilege: "user", Bundling: "vendorable", Service: &Service{Artifact: "fallback", Layout: "dir", EntryPath: "bin/server", Version: "1", SHA256: digest, Files: []Artifact{{Name: "fallback", SHA256: digest}}}}},
		DeferredTargets: []DeferredResourceTarget{{Resource: "whisper", OS: runtimeOS(), Architecture: runtime.GOARCH, When: map[string]string{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "has:vulkan"}, Kind: "oci-image", Image: "https://127.0.0.1:1/never@sha256:" + strings.Repeat("a", 64), Layout: "dir", BinPath: "bin/server"}},
	}
	err = ApplyDeferredUpgrade(context.Background(), root, plan, "whisper", binaryfetch.Facts{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "vulkan"})
	if err == nil {
		t.Fatal("expected acquisition failure")
	}
	got, err := os.ReadFile(filepath.Join(resourceDir, "bin", "server"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fallback" {
		t.Fatalf("fallback changed after failed upgrade: %q", got)
	}
}

func TestApplyDeferredUpgradeAtomicallyInstallsVerifiedTree(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "whisper", "fallback")
	if err := os.MkdirAll(filepath.Join(resourceDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "bin", "server"), []byte("fallback"), 0o755); err != nil {
		t.Fatal(err)
	}
	upgrade := t.TempDir()
	if err := os.MkdirAll(filepath.Join(upgrade, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	acceleratedBody := []byte(strings.Repeat("accelerated\n", 128))
	if err := os.WriteFile(filepath.Join(upgrade, "bin", "server"), acceleratedBody, 0o755); err != nil {
		t.Fatal(err)
	}
	artifactDigest, err := binaryfetch.TreeDigest(upgrade)
	if err != nil {
		t.Fatal(err)
	}
	var layer bytes.Buffer
	tw := tar.NewWriter(&layer)
	body := acceleratedBody
	if err := tw.WriteHeader(&tar.Header{Name: "bin/server", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	layerSum := sha256.Sum256(layer.Bytes())
	layerDigest := hex.EncodeToString(layerSum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/blobs/sha256:"+layerDigest) {
			_, _ = w.Write(layer.Bytes())
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	plan := &Plan{
		Resources:       []Item{{Resource: "whisper", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-service", Support: "supported", Privilege: "user", Bundling: "vendorable", Service: &Service{Artifact: "fallback", Layout: "dir", EntryPath: "bin/server", Version: "1", SHA256: "old", Files: []Artifact{{Name: "fallback", SHA256: "old"}}}}},
		DeferredTargets: []DeferredResourceTarget{{Resource: "whisper", OS: runtimeOS(), Architecture: runtime.GOARCH, When: map[string]string{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "has:vulkan"}, Kind: "oci-image", Image: srv.URL + "/example@sha256:" + layerDigest, ArtifactSHA256: artifactDigest, Layout: "dir", BinPath: "bin/server"}},
	}
	if err := ApplyDeferredUpgrade(context.Background(), root, plan, "whisper", binaryfetch.Facts{"os": runtimeOS(), "arch": runtime.GOARCH, "accel.backends": "vulkan"}); err != nil {
		t.Fatalf("apply upgrade: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "resources", "whisper", "fallback", "bin", "server"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(acceleratedBody) {
		t.Fatalf("installed tree = %q", got)
	}
	if plan.Resources[0].Service.SHA256 != artifactDigest {
		t.Fatalf("plan digest = %q", plan.Resources[0].Service.SHA256)
	}
}

func TestLoadAcceptsCurrentPipelinePlanVersion(t *testing.T) {
	root := t.TempDir()
	data, err := json.Marshal(Plan{SchemaVersion: "v6", Resources: []Item{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load current pipeline plan: %v", err)
	}
}

func TestLoadRejectsMismatchedBuildMetadata(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "resource-demo_" + artifactOS(runtimeOS()) + "_" + runtime.GOARCH
	files := []string{name, name + ".manifest.json", name + ".build.json"}
	artifacts := make([]Artifact, 0, len(files))
	for _, file := range files {
		body := []byte("binary")
		if file == name+".manifest.json" {
			body = []byte(`{"name":"demo"}`)
		}
		if file == name+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + name + `","os":"windows","arch":"amd64"}`)
		}
		if err := os.WriteFile(filepath.Join(resourceDir, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		artifacts = append(artifacts, Artifact{Name: file, SHA256: hex.EncodeToString(sum[:])})
	}
	plan := Plan{SchemaVersion: "v3", Resources: []Item{{RequestedResource: "demo", Resource: "demo", OS: runtimeOS(), Architecture: runtime.GOARCH, Mode: "bundled-client", Support: "supported", Privilege: "user", Bundling: "vendorable", Artifact: name, Files: artifacts}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected mismatched metadata to fail")
	}
}

func TestLoadReturnsActionableDockerPreflight(t *testing.T) {
	originalFindDocker := findDocker
	findDocker = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { findDocker = originalFindDocker })
	root := t.TempDir()
	plan := Plan{SchemaVersion: "v3", Resources: []Item{{
		RequestedResource: "redis",
		Resource:          "redis",
		OS:                runtimeOS(),
		Architecture:      runtime.GOARCH,
		Mode:              "docker-desktop",
		Support:           "conditional",
		Privilege:         "user",
		Bundling:          "host-required",
		Requires:          []string{"docker-desktop"},
		Limitations:       []string{"Docker must be running"},
	}}}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "Docker Desktop or Docker Engine") {
		t.Fatalf("expected actionable Docker preflight error, got %v", err)
	}
}

func TestLoadRejectsUnknownModeBeforeHostSelection(t *testing.T) {
	root := t.TempDir()
	plan := Plan{SchemaVersion: "v3", Resources: []Item{{RequestedResource: "demo", Resource: "demo", OS: "windows", Architecture: "arm64", Mode: "server-mode", Support: "supported", Privilege: "user", Bundling: "vendorable"}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown deployment mode") {
		t.Fatalf("expected unknown mode to fail, got %v", err)
	}
}

func TestLoadValidatesSeparatelyPinnedBundledService(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	controller := "resource-demo_" + artifactOS(runtimeOS()) + "_" + runtime.GOARCH
	files := []string{controller, controller + ".manifest.json", controller + ".build.json"}
	artifacts := make([]Artifact, 0, len(files))
	for _, file := range files {
		body := []byte("controller")
		if file == controller+".manifest.json" {
			body = []byte(`{"name":"demo"}`)
		}
		if file == controller+".build.json" {
			body = []byte(`{"resource":"demo","artifact":"` + controller + `","os":"` + artifactOS(runtimeOS()) + `","arch":"` + runtime.GOARCH + `"}`)
		}
		if err := os.WriteFile(filepath.Join(resourceDir, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		artifacts = append(artifacts, Artifact{Name: file, SHA256: hex.EncodeToString(sum[:])})
	}
	serverName, serverBody := "server-"+runtime.GOOS+"-"+runtime.GOARCH, []byte("server")
	if err := os.WriteFile(filepath.Join(resourceDir, serverName), serverBody, 0o755); err != nil {
		t.Fatal(err)
	}
	serverSum := sha256.Sum256(serverBody)
	plan := Plan{SchemaVersion: "v3", Resources: []Item{{
		RequestedResource: "demo", Resource: "demo", OS: runtimeOS(), Architecture: runtime.GOARCH,
		Mode: "bundled-service", Support: "supported", Privilege: "user", Bundling: "vendorable", Artifact: controller, Files: artifacts,
		Service: &Service{ProviderPolicy: resourcedeployment.ProviderPolicy{TargetDefaults: map[resourcedeployment.ProviderTarget]resourcedeployment.ProviderMode{resourcedeployment.ProviderTargetControlPlane: resourcedeployment.ProviderManagedShared, resourcedeployment.ProviderTargetDesktopBundle: resourcedeployment.ProviderManagedPrivate}, AllowedModes: []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate, resourcedeployment.ProviderManagedShared}, SharedReuseRequiresConsent: true, ExternalManagement: "forbidden"}, Artifact: serverName, Version: "1.0.0", SHA256: hex.EncodeToString(serverSum[:]), Files: []Artifact{{Name: serverName, SHA256: hex.EncodeToString(serverSum[:])}}},
	}}}
	data, _ := json.Marshal(plan)
	if err := os.WriteFile(filepath.Join(root, "resource-deployment-plan.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, serverName), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "bundled service artifact hash mismatch") {
		t.Fatalf("Load after server tamper = %v", err)
	}
}
