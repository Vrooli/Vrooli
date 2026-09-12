package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/release"
	"scenario-to-cloud/releasesvc"
)

func releaseTestServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	writeFileBytes(t, root, ".vrooli/service.json", []byte(`{"version":"2.0.0"}`))
	writeFileBytes(t, root, "scenarios/app/README.md", []byte("app\n"))
	native, err := filepath.Abs(filepath.Join("release", "testdata", "nativecli"))
	if err != nil {
		t.Fatal(err)
	}
	srv := newTestServer()
	srv.releaseSvc = releasesvc.New(releasesvc.Config{RepoRoot: root, StoreDir: t.TempDir(), NativeCLI: release.NativeCLIOptions{ModuleDir: native, Package: "."}, TrustMode: release.TrustDevelopmentLocal})
	srv.registerReleaseRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	return srv
}

// TestReleaseRoutesBuildGetVerify [REQ:STC-P0-026] proves the REST surface
// builds, reads and verifies a release and writes the typed error envelope.
func TestReleaseRoutesBuildGetVerify(t *testing.T) {
	srv := releaseTestServer(t)
	manifest := domain.CloudManifest{
		Version:      "1.0.0",
		Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10"}},
		Scenario:     domain.ManifestScenario{ID: "app"},
		Dependencies: domain.ManifestDependencies{Scenarios: []string{"app"}, ClosureDigest: "sha256:" + strings.Repeat("c", 64)},
		Bundle:       domain.ManifestBundle{Scenarios: []string{"app"}},
		Edge:         domain.ManifestEdge{Domain: "example.com"},
	}
	body, _ := json.Marshal(releasesvc.BuildRequest{Manifest: manifest, GOOS: "linux", GOARCH: "amd64"})
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/build", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("build: %d %s", rec.Code, rec.Body.String())
	}
	var built struct {
		SchemaVersion string          `json:"schema_version"`
		Release       release.Release `json:"release"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &built); err != nil {
		t.Fatal(err)
	}
	digest := built.Release.Digest
	if built.SchemaVersion != "1" || len(digest) != 64 {
		t.Fatalf("unexpected build body: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/releases/"+digest, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), digest) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+digest+"/verify", strings.NewReader(`{"goos":"linux","goarch":"amd64"}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"verified":true`) {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}

	// Tamper the stored native binary: verification must fail with the
	// typed code and name the artifact.
	if err := os.WriteFile(filepath.Join(built.Release.Dir, "vrooli-linux-amd64"), []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/"+digest+"/verify", nil))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("tampered verify: %d %s", rec.Code, rec.Body.String())
	}
	typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes())
	if typed.Code != apierrors.CodeReleaseVerificationFailed || typed.Details["check"] != "native_cli" {
		t.Fatalf("tampered verify error: %+v", typed)
	}

	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/releases/"+strings.Repeat("0", 64), nil))
	if typed := apierrors.FromHTTP(rec.Code, rec.Body.Bytes()); typed.Code != apierrors.CodeReleaseVerificationFailed || typed.Details["reason"] != release.ReasonNotFound {
		t.Fatalf("missing release: %d %s", rec.Code, rec.Body.String())
	}
}
