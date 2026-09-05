package bundles

import (
	"context"
	"net/http"
	"net/http/httptest"
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
