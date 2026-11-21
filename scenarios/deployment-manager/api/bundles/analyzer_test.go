package bundles

import (
	"context"
	"encoding/json"
	runtimeManifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFetchSkeletonBundleBoundsUnresponsiveAnalyzer(t *testing.T) {
	requestCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(requestCanceled)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", server.URL)

	previousTimeout := skeletonFetchTimeout
	skeletonFetchTimeout = 20 * time.Millisecond
	t.Cleanup(func() { skeletonFetchTimeout = previousTimeout })

	_, err := FetchSkeletonBundle(context.Background(), "secrets-manager")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected analyzer deadline error, got %v", err)
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("analyzer request was not canceled after the deadline")
	}
}

func TestExtractManifestBytesAndOnDemandCLIMetadata(t *testing.T) {
	if _, err := extractManifestBytes(nil); err == nil {
		t.Fatal("missing manifest should fail")
	}
	wrapped, err := extractManifestBytes([]byte(`{"skeleton":{"schema_version":"v0.1"}}`))
	if err != nil || string(wrapped) != `{"schema_version":"v0.1"}` {
		t.Fatalf("wrapped manifest: %s %v", wrapped, err)
	}
	direct, err := extractManifestBytes([]byte(`{"schema_version":"v0.1"}`))
	if err != nil || string(direct) != `{"schema_version":"v0.1"}` {
		t.Fatalf("direct manifest: %s %v", direct, err)
	}
	manifest := &Manifest{Services: []ServiceEntry{
		{ID: "my-cli", Type: "cli"},
		{ID: "worker", Build: &BuildConfig{SourceDir: "cli"}},
		{ID: "api", Type: "api", Metadata: map[string]interface{}{"run_mode": "always"}},
	}}
	applyOnDemandCLIMetadata(manifest)
	if manifest.Services[0].Metadata["run_mode"] != "on_demand" || manifest.Services[1].Metadata["skip_reason"] == nil {
		t.Fatalf("CLI metadata not applied: %+v", manifest.Services)
	}
	if manifest.Services[2].Metadata["run_mode"] != "always" {
		t.Fatal("existing metadata should be preserved")
	}
	if isCLIService(ServiceEntry{ID: "cli-tool"}) == false || isCLIService(ServiceEntry{ID: "worker", Build: &BuildConfig{SourceDir: "cli"}}) == false || isCLIService(ServiceEntry{ID: "api"}) {
		t.Fatal("unexpected CLI classification")
	}
	applyOnDemandCLIMetadata(nil)
}

func TestFetchSkeletonBundlePreservesEmbeddedPeerContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "manifests", "desktop-happy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	source["peers"] = []map[string]any{{"scenario": "helper", "bundle_policy": "embed", "embedded_version": "1.0.0"}}
	source["program_binding_peers"] = []map[string]any{{"scenario": "program-owner", "bundle_policy": "discover"}}
	source["catalog_requirements"] = []string{"resources/codex/resource.json"}
	source["features"] = map[string]any{"audio.processing": map[string]any{"owner_version": "0.0.1", "service": "api", "port": "http"}}
	source["app"].(map[string]any)["scenario"] = "example"
	service := source["services"].([]any)[0].(map[string]any)
	service["dist_root"] = "ui/dist"
	service["asset_dirs"] = []string{"assets"}
	ownerAPI := source["services"].([]any)[1].(map[string]any)
	ownerClone := make(map[string]any, len(ownerAPI))
	for key, value := range ownerAPI {
		ownerClone[key] = value
	}
	ownerClone["id"] = "helper--api"
	source["services"] = append(source["services"].([]any), ownerClone)
	source["peers"].([]map[string]any)[0]["features"] = map[string]any{"usage.live": map[string]any{"owner_version": "0.0.1", "service": "helper--api", "port": "http"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"manifest": map[string]any{"skeleton": source}})
	}))
	defer server.Close()
	t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_URL", server.URL)
	manifest, err := FetchSkeletonBundle(t.Context(), "example")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.App.Scenario != "example" || len(manifest.Peers) != 1 || manifest.Peers[0].Scenario != "helper" || manifest.Peers[0].BundlePolicy != "embed" || len(manifest.ProgramBindingPeers) != 1 || len(manifest.CatalogRequirements) != 1 || manifest.Services[0].DistRoot != "ui/dist" || len(manifest.Services[0].AssetDirs) != 1 {
		t.Fatalf("analyzer contract was lost during export: %+v", manifest)
	}
	if manifest.Features["audio.processing"].OwnerVersion != "0.0.1" {
		t.Fatalf("owner feature lost: %#v", manifest.Features)
	}
	if manifest.Peers[0].Features["usage.live"].OwnerVersion != "0.0.1" || manifest.Peers[0].Features["usage.live"].Service != "helper--api" {
		t.Fatalf("embedded owner feature lost: %#v", manifest.Peers[0])
	}
	if _, ok := manifest.Features["usage.live"]; ok {
		t.Fatal("embedded owner feature leaked into root")
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var native runtimeManifest.Manifest
	if err := json.Unmarshal(encoded, &native); err != nil {
		t.Fatal(err)
	}
	if native.Peers[0].Features["usage.live"].Service != "helper--api" || len(native.Features) != 1 {
		t.Fatalf("native feature roundtrip lost scope: %#v", native.Peers)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	if _, ok := result["features"]; !ok {
		t.Fatal("export dropped owner features")
	}
	if _, ok := result["peers"]; !ok {
		t.Fatal("export dropped required peer")
	}
}
