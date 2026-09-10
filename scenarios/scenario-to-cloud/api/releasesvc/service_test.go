package releasesvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/release"

	"github.com/vrooli/vrooli/packages/cloudrelease"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/releases"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/releases/releasesv1connect"
)

func writeFile(t *testing.T, root, rel string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), contents, 0o644); err != nil {
		t.Fatal(err)
	}
}

func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if data, err := os.ReadFile(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("live repo contract not found")
		}
		dir = parent
	}
}

func fixtureService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/repo-contract.json", liveRepoContract(t))
	writeFile(t, root, ".vrooli/service.json", []byte(`{"version":"2.0.0"}`))
	writeFile(t, root, "go.mod", []byte("module fixture\n\ngo 1.24.0\n"))
	writeFile(t, root, "scenarios/app/README.md", []byte("app\n"))
	native, _ := filepath.Abs(filepath.Join("..", "release", "testdata", "nativecli"))
	return New(Config{RepoRoot: root, StoreDir: t.TempDir(), NativeCLI: release.NativeCLIOptions{ModuleDir: native, Package: "."}, TrustMode: release.TrustDevelopmentLocal})
}

func fixtureManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Version:      "1.0.0",
		Target:       domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10"}},
		Scenario:     domain.ManifestScenario{ID: "app"},
		Dependencies: domain.ManifestDependencies{Scenarios: []string{"app"}, ClosureDigest: "sha256:" + strings.Repeat("c", 64)},
		Bundle:       domain.ManifestBundle{Scenarios: []string{"app"}},
		Edge:         domain.ManifestEdge{Domain: "example.com"},
	}
}

// TestConnectReleasesServiceBuildsGetsAndVerifies [REQ:STC-P0-026] drives
// the typed surface end to end and proves failures carry the stable codes.
func TestConnectReleasesServiceBuildsGetsAndVerifies(t *testing.T) {
	svc := fixtureService(t)
	mux := http.NewServeMux()
	mux.Handle(svc.Handler())
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := releasesv1connect.NewReleasesServiceClient(http.DefaultClient, ts.URL)
	ctx := context.Background()

	manifest, err := ManifestStruct(fixtureManifest())
	if err != nil {
		t.Fatal(err)
	}
	built, err := client.BuildRelease(ctx, connect.NewRequest(&releasesv1.BuildReleaseRequest{Manifest: manifest, Goos: "linux", Goarch: "amd64"}))
	if err != nil {
		t.Fatalf("BuildRelease: %v", err)
	}
	rel := built.Msg.GetRelease()
	if built.Msg.GetSchemaVersion() != SchemaVersion || !cloudrelease.IsDigest(rel.GetReleaseDigest()) || !rel.GetComplete() {
		t.Fatalf("unexpected build response: %+v", built.Msg)
	}
	if rel.GetManifest().GetNativeCli().GetGoarch() != "amd64" || rel.GetInputs().GetProvenance().GetPolicy() != cloudrelease.PolicyDevelopmentLocalUnsigned {
		t.Fatalf("release projection incomplete: %+v", rel)
	}

	got, err := client.GetRelease(ctx, connect.NewRequest(&releasesv1.GetReleaseRequest{ReleaseDigest: rel.GetReleaseDigest()}))
	if err != nil || got.Msg.GetRelease().GetReleaseDigest() != rel.GetReleaseDigest() {
		t.Fatalf("GetRelease: %v %+v", err, got)
	}

	verified, err := client.VerifyRelease(ctx, connect.NewRequest(&releasesv1.VerifyReleaseRequest{ReleaseDigest: rel.GetReleaseDigest(), Goos: "linux", Goarch: "amd64"}))
	if err != nil || !verified.Msg.GetVerified() || len(verified.Msg.GetChecks()) == 0 {
		t.Fatalf("VerifyRelease: %v %+v", err, verified)
	}

	_, err = client.VerifyRelease(ctx, connect.NewRequest(&releasesv1.VerifyReleaseRequest{ReleaseDigest: rel.GetReleaseDigest(), Goos: "linux", Goarch: "arm64"}))
	assertConnectCode(t, err, apierrors.CodeUnsupportedCapability)

	_, err = client.GetRelease(ctx, connect.NewRequest(&releasesv1.GetReleaseRequest{ReleaseDigest: strings.Repeat("0", 64)}))
	assertConnectCode(t, err, apierrors.CodeReleaseVerificationFailed)

	_, err = client.BuildRelease(ctx, connect.NewRequest(&releasesv1.BuildReleaseRequest{Manifest: manifest, Goos: "windows", Goarch: "amd64"}))
	assertConnectCode(t, err, apierrors.CodeUnsupportedCapability)

	_, err = client.BuildRelease(ctx, connect.NewRequest(&releasesv1.BuildReleaseRequest{Goos: "linux", Goarch: "amd64"}))
	assertConnectCode(t, err, apierrors.CodeManifestInvalid)

	_, err = client.BuildRelease(ctx, connect.NewRequest(&releasesv1.BuildReleaseRequest{Manifest: manifest, Goos: "linux", Goarch: "amd64", TrustMode: "production"}))
	assertConnectCode(t, err, apierrors.CodeReleaseVerificationFailed)
}

func assertConnectCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got success", code)
	}
	cerr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect error, got %T %v", err, err)
	}
	for _, detail := range cerr.Details() {
		msg, err := detail.Value()
		if err != nil {
			continue
		}
		if typed, ok := msg.(interface{ GetCode() string }); ok {
			if typed.GetCode() != code {
				t.Fatalf("code = %s, want %s", typed.GetCode(), code)
			}
			return
		}
	}
	t.Fatalf("no typed error detail on %v", err)
}
