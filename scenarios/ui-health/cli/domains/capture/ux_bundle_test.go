package capture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureUXBundleRequestsAndVerifiesStrictTokenArtifact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("BAS_CAPTURES_ROOT", root)
	dir := filepath.Join(root, "ux", "bundle-1")
	if err := os.MkdirAll(filepath.Join(dir, "diagnostics"), 0o700); err != nil {
		t.Fatal(err)
	}
	diagnostic := []byte(`{"schema":"bas-strict-token-diagnostic/v1","token_count":9,"sentinel_element_count":4,"non_sentinel_color_count":2}`)
	if err := os.WriteFile(filepath.Join(dir, "diagnostics", "strict-tokens.json"), diagnostic, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(diagnostic)
	manifest := map[string]any{
		"schema_version": "ux-capture.v1", "id": "bundle-1",
		"source": map[string]string{"scenario": "git-control-tower", "route": "/"},
		"lenses": map[string]any{
			"L0": map[string]string{"status": "absent", "reason": "fixture"}, "L1": map[string]string{"status": "absent", "reason": "fixture"},
			"L2": map[string]string{"status": "absent", "reason": "fixture"}, "L3": map[string]string{"status": "absent", "reason": "fixture"},
			"L4": map[string]string{"status": "absent", "reason": "fixture"}, "L5": map[string]string{"status": "absent", "reason": "fixture"}, "L6": map[string]string{"status": "absent", "reason": "fixture"},
		}, "privacy": map[string]string{"redaction_status": "complete"},
		"artifacts": []any{map[string]any{"kind": "strict-token-diagnostic", "path": "diagnostics/strict-tokens.json", "size_bytes": len(diagnostic), "sha256": hex.EncodeToString(sum[:])}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ux-capture/scenario" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request["strict_tokens"] != true {
			t.Errorf("strict_tokens request = %v", request["strict_tokens"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"bundle_dir": dir, "manifest": manifest})
	}))
	defer server.Close()
	reader := &handlers{http: server.Client()}
	bundle, files, err := reader.captureUXBundle(context.Background(), server.URL, "git-control-tower", "/", "light", "desktop", true)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.ID != "bundle-1" || string(files["diagnostics/strict-tokens.json"]) != string(diagnostic) {
		t.Fatalf("bundle = %#v, artifact=%s", bundle, files["diagnostics/strict-tokens.json"])
	}
}

func TestConfinedArtifactRejectsTraversalAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	if _, err := confinedArtifact(dir, "../../etc/passwd"); err == nil {
		t.Fatal("accepted traversal path")
	}
	target := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "pixels.png")); err != nil {
		t.Fatal(err)
	}
	if _, err := confinedArtifact(dir, "pixels.png"); err == nil {
		t.Fatal("accepted symlink artifact")
	}
}

func TestEX9LayoutPreservesOutlineActionsStyleAndViewport(t *testing.T) {
	manifest := uxManifest{}
	manifest.Viewport.Width, manifest.Viewport.Height = 1280, 720
	outline := []byte(`{"nodes":[{"kind":"action","tag":"button","role":"button","name":"Save","text":"Save","selector":"#save","rect":{"x":5,"y":7,"width":80,"height":30},"disabled":true,"actionType":"click"}]}`)
	style := []byte(`{"clusters":[{"category":"color","property":"foreground","value":"rgb(1, 2, 3)"}]}`)
	layout := ex9Layout(manifest, outline, style)
	for _, expected := range []string{`"width":1280`, `"height":720`, `"selector":"#save"`, `"text":"Save"`, `"interactive":true`, `foreground:rgb(1, 2, 3)`} {
		if !strings.Contains(layout, expected) {
			t.Fatalf("layout missing %s: %s", expected, layout)
		}
	}
}
