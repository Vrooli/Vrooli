package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestCheckUpstreamRequiresTwoFailuresAndResetsOnSuccess(t *testing.T) {
	dead := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dead {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	root := t.TempDir()
	writeLivenessManifest(t, root, server.URL)
	state := filepath.Join(t.TempDir(), "state.json")
	first, err := CheckUpstream(context.Background(), root, state, "", server.Client(), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if first.Findings[0].Stale {
		t.Fatal("one failure must not be stale")
	}
	second, err := CheckUpstream(context.Background(), root, state, "", server.Client(), time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !second.Findings[0].Stale || second.Findings[0].FirstFailedAt.IsZero() {
		t.Fatalf("second failure = %+v", second.Findings[0])
	}
	dead = false
	third, err := CheckUpstream(context.Background(), root, state, "", server.Client(), time.Unix(3, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !third.Findings[0].Reachable || third.Findings[0].Stale {
		t.Fatalf("reachable result = %+v", third.Findings[0])
	}
}

func TestCheckReferenceUsesOCIAndNPMRegistryEndpoints(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	oci := server.URL + "/org/image@sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err := checkReference(context.Background(), server.Client(), "oci-image", oci); err != nil {
		t.Fatal(err)
	}
	// npmVersionURL is registry-shaped by contract; use the helper's result
	// through a transport rewrite so the test remains fully local.
	client := server.Client()
	client.Transport = rewriteRegistryTransport{base: server.URL, next: http.DefaultTransport}
	if _, err := checkReference(context.Background(), client, "npm", "@scope/pkg@1.2.3"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/v2/org/image/manifests/sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" || paths[1] != "/@scope/pkg/1.2.3" {
		t.Fatalf("probe paths = %v", paths)
	}
}

func TestCheckUpstreamReportsBrokenResourceManifestsWithoutAborting(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "resources", "missing"), 0o750); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(root, "resources", "invalid")
	if err := os.MkdirAll(invalid, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalid, "resource.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := CheckUpstream(context.Background(), root, filepath.Join(t.TempDir(), "state.json"), "", nil, time.Unix(4, 0))
	if err != nil {
		t.Fatalf("CheckUpstream() error = %v", err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("findings = %+v, want one finding per broken manifest", report.Findings)
	}
	for _, finding := range report.Findings {
		if finding.Kind != "manifest" || finding.Reachable || finding.Note == "" {
			t.Fatalf("broken manifest finding = %+v", finding)
		}
	}
}

func TestBearerChallengeParsesRegistryClaims(t *testing.T) {
	realm, service, scope := bearerChallenge(`Bearer realm="https://auth.example/token",service="registry.example",scope="repository:org/image:pull"`)
	if realm != "https://auth.example/token" || service != "registry.example" || scope != "repository:org/image:pull" {
		t.Fatalf("bearerChallenge() = %q, %q, %q", realm, service, scope)
	}
}

type rewriteRegistryTransport struct {
	base string
	next http.RoundTripper
}

func (t rewriteRegistryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	base, _ := url.Parse(t.base)
	clone.URL.Scheme, clone.URL.Host = base.Scheme, base.Host
	return t.next.RoundTrip(clone)
}

func writeLivenessManifest(t *testing.T, root, endpoint string) {
	t.Helper()
	dir := filepath.Join(root, "resources", "fixture")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	manifest := manifestpkg.ResourceManifest{Name: "fixture", Driver: "external-cli", Binary: "bash", Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: endpoint, SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}}}
	b, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}
