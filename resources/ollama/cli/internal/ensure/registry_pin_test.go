package ensure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDigest(seed string) string {
	s := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(s[:])
}
func testManifest(config, layer string, configSize, layerSize int64) []byte {
	b, _ := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": map[string]any{"mediaType": "application/vnd.ollama.image.model", "digest": config, "size": configSize}, "layers": []any{map[string]any{"mediaType": "application/vnd.ollama.image.model", "digest": layer, "size": layerSize}}})
	return b
}

func TestRegistryPinFreezesManifestAndRedirectsKnownBlobs(t *testing.T) {
	cfg, layer := testDigest("config"), testDigest("layer")
	original := testManifest(cfg, layer, 10, 25)
	p, err := OpenRegistryPin("library/model", "latest", original)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(context.Background())
	original[0] = 'x'
	repo := "/v2/" + p.Repo
	request := func(method, path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Host = p.listener.Addr().String()
		p.server.Handler.ServeHTTP(rr, req)
		return rr
	}
	manifest := request(http.MethodGet, repo+"/manifests/latest")
	if manifest.Code != http.StatusOK || !strings.Contains(manifest.Body.String(), cfg) {
		t.Fatalf("manifest response: %d %s", manifest.Code, manifest.Body.String())
	}
	if p.PeakBytes != 35 {
		t.Fatalf("peak=%d want 35", p.PeakBytes)
	}
	redirect := request(http.MethodGet, repo+"/blobs/"+layer)
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "https://registry.ollama.ai/v2/library/model/blobs/"+layer {
		t.Fatalf("redirect: %d %q", redirect.Code, redirect.Header().Get("Location"))
	}
	head := request(http.MethodHead, repo+"/blobs/"+layer)
	if head.Code != http.StatusOK || head.Header().Get("Content-Length") != "25" || head.Body.Len() != 0 {
		t.Fatalf("blob HEAD response: %d headers=%v body=%q", head.Code, head.Header(), head.Body.String())
	}
	for _, path := range []string{repo + "/manifests/other", repo + "/blobs/" + testDigest("unknown"), "/v2/other/manifests/latest"} {
		if got := request(http.MethodGet, path).Code; got != http.StatusNotFound {
			t.Errorf("%s status=%d", path, got)
		}
	}
	if got := request(http.MethodPost, repo+"/manifests/latest").Code; got != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", got)
	}
	wrongHost := httptest.NewRecorder()
	wrongHostReq := httptest.NewRequest(http.MethodGet, repo+"/manifests/latest", nil)
	p.server.Handler.ServeHTTP(wrongHost, wrongHostReq)
	if wrongHost.Code != http.StatusMisdirectedRequest {
		t.Fatalf("wrong host status=%d", wrongHost.Code)
	}
	if got := request(http.MethodHead, repo+"/manifests/latest"); got.Code != http.StatusOK || got.Body.Len() != 0 {
		t.Fatalf("HEAD response: %#v", got)
	}
}

type preflightFunc func(context.Context, string, string) ([]byte, error)

func (f preflightFunc) FetchManifest(c context.Context, r, t string) ([]byte, error) {
	return f(c, r, t)
}
func (preflightFunc) ValidateBlobSize(context.Context, string, string, int64) error { return nil }

type preflightRejectBlobSize struct {
	PreflightClient
	err error
}

func (p preflightRejectBlobSize) ValidateBlobSize(context.Context, string, string, int64) error {
	return p.err
}

func TestPreparePinnedPullValidatesRefAndReportsCopySafeMetadata(t *testing.T) {
	cfg, layer := testDigest("c"), testDigest("l")
	bytes := testManifest(cfg, layer, 7, 11)
	var gotRepo, gotTag string
	client := preflightFunc(func(_ context.Context, repo, tag string) ([]byte, error) {
		gotRepo, gotTag = repo, tag
		return bytes, nil
	})
	p, err := PreparePinnedPull(context.Background(), "registry.ollama.ai/model:stable", client)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if gotRepo != "library/model" || gotTag != "stable" {
		t.Fatalf("preflight ref %s:%s", gotRepo, gotTag)
	}
	if p.ModelBytes() != 11 || p.PeakBytes() != 18 || len(p.LayerDigests()) != 2 || p.LayerDigests()[0] != cfg || p.LayerDigests()[1] != layer {
		t.Fatalf("metadata: model=%d peak=%d layers=%v", p.ModelBytes(), p.PeakBytes(), p.LayerDigests())
	}
	copyLayers := p.LayerDigests()
	copyLayers[0] = "changed"
	if p.LayerDigests()[0] != cfg {
		t.Fatal("layer metadata aliases internal slice")
	}
	if !strings.HasPrefix(p.PullReference(), "127.0.0.1:") || !strings.Contains(p.PullReference(), "/library/pin-") {
		t.Fatalf("pull ref=%s", p.PullReference())
	}
	for _, bad := range []string{"https://evil/model", "evil.example/model", "user:pass@model", "model@sha256:abc", "a/b/c", "model:bad/tag"} {
		if _, err := PreparePinnedPull(context.Background(), bad, client); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestPinnedManifestInstallUsesAtomicNoClobberCommit(t *testing.T) {
	manifestBytes := testManifest(testDigest("cfg"), testDigest("layer"), 7, 11)
	pull, err := PreparePinnedPull(context.Background(), "model:stable", preflightFunc(func(context.Context, string, string) ([]byte, error) {
		return manifestBytes, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer pull.Close()
	root := t.TempDir()
	target := filepath.Join(root, "manifests", RegistryHost, "library", "model", "stable")
	if err := pull.InstallTarget(context.Background(), root, "model:stable"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(manifestBytes) {
		t.Fatalf("published manifest mismatch: err=%v", err)
	}
	if err := os.WriteFile(target, []byte("preexisting model"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pull.InstallTarget(context.Background(), root, "model:stable"); err == nil {
		t.Fatal("existing model manifest was overwritten")
	}
	got, err = os.ReadFile(target)
	if err != nil || string(got) != "preexisting model" {
		t.Fatalf("existing model changed: got=%q err=%v", got, err)
	}
}

func TestRegistryPinRejectsBadDescriptorsAndOverflow(t *testing.T) {
	good := testDigest("g")
	cases := [][]byte{[]byte(`{`), []byte(`{"schemaVersion":1,"config":{"digest":"` + good + `","size":1},"layers":[{"digest":"` + good + `","size":1}]}`), testManifest("sha256:BAD", good, 1, 1), testManifest(good, good, 0, 1), testManifest(good, good, math.MaxInt64, 1)}
	for i, b := range cases {
		if _, err := OpenRegistryPin("library/model", "latest", b); err == nil {
			t.Errorf("case %d accepted", i)
		}
	}
}

func TestPreparePinnedPullRejectsBlobHeadSizeMismatch(t *testing.T) {
	want := errors.New("registry HEAD size does not match manifest descriptor")
	base := preflightFunc(func(context.Context, string, string) ([]byte, error) { return validTestManifest(), nil })
	if _, err := PreparePinnedPull(context.Background(), "model:latest", preflightRejectBlobSize{PreflightClient: base, err: want}); !errors.Is(err, want) {
		t.Fatalf("PreparePinnedPull error=%v, want blob-size refusal", err)
	}
}

func TestRegistryPinCloseStopsListener(t *testing.T) {
	p, err := OpenRegistryPin("library/model", "latest", testManifest(testDigest("c"), testDigest("l"), 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	addr := p.listener.Addr().String()
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	client := http.Client{Timeout: time.Second}
	if _, err := client.Get("http://" + addr + "/v2/"); err == nil {
		t.Fatal("listener remained open after Close")
	}
}
