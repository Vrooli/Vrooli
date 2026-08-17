package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type artifactFinderStub struct{ path string }

func (s artifactFinderStub) FindArtifact(string) (string, error) { return s.path, nil }

func TestDesktopRampBuilderCarriesMonetizationIdentityAndDigest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.AppImage")
	if err := os.WriteFile(path, []byte("desktop artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	rootWithScenario := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootWithScenario, "scenarios", "paid-app", ".vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"bundle_key":"business_suite","app_key":"paid-app"}`
	if err := os.WriteFile(filepath.Join(rootWithScenario, "scenarios", "paid-app", ".vrooli", "monetization.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", rootWithScenario)
	artifact, err := (desktopRampBuilder{finder: artifactFinderStub{path: path}}).Build(context.Background(), deliveryramp.BuildRequest{SourceRef: "paid-app"})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Metadata["bundle_key"] != "business_suite" || artifact.Metadata["app_key"] != "paid-app" || artifact.Checksum == "" {
		t.Fatalf("artifact metadata = %#v, checksum=%q", artifact.Metadata, artifact.Checksum)
	}
}

func TestDesktopRampDistributorRegistersImmutableArtifact(t *testing.T) {
	var gotPath string
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode registration: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("S2D_LPBS_CATALOG_URL", server.URL)
	t.Setenv("S2D_DESKTOP_ARTIFACT_URL", "https://downloads.example.test/paid.AppImage")
	t.Setenv("S2D_LPBS_ADMIN_TOKEN", "operator-token")

	result, err := (desktopRampDistributor{client: server.Client()}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{
		ImmutableRef: "artifact:sha256",
		Checksum:     "sha256:abc",
		Metadata:     map[string]string{"bundle_key": "business_suite", "app_key": "paid-app", "platform": "linux"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || gotPath != "/api/v1/admin/download-apps/paid-app" {
		t.Fatalf("registration result=%+v path=%q", result, gotPath)
	}
	if got["app_key"] != "paid-app" {
		t.Fatalf("registration payload = %#v", got)
	}
}
