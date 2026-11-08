package installgateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func cloudflaredResourceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "resources", "cloudflared", "resource.json")
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "cloudflared", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	service := m["managed_service"].(map[string]any)
	service["artifact"] = map[string]any{"path": "server/cloudflared", "version": "2026.10.7", "sha256": strings.Repeat("a", 64), "bundle_artifact": "cloudflared_${os}_${arch}"}
	service["acquisition"] = map[string]any{"kind": "url", "license": "Apache-2.0", "targets": []any{map[string]any{"when": map[string]string{"os": runtime.GOOS, "arch": runtime.GOARCH}, "url": "https://example.invalid/cloudflared", "sha256": strings.Repeat("a", 64), "archive": "none", "layout": "file"}}}
	body, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestManagedServiceReleaseRequiresExactCloudflaredResource(t *testing.T) {
	root := cloudflaredResourceFixture(t)
	r, err := Resolve(root, "cloudflared", "host-tool", "source", "cloudflared", "2026.10.7")
	if err != nil {
		t.Fatal(err)
	}
	if r.Profile.LifecycleMode != "managed-service-artifact" || !r.Profile.FrozenLockfile || !r.Profile.ScriptsDisabled || !r.ExactVersion || r.HostTool || r.HostSource || r.Command() != "native-acquire cloudflared 2026.10.7" {
		t.Fatalf("unsafe staging: %+v", r)
	}
	if err := ApplyExactVersion(&r, "2026.10.7"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyExactVersion(&r, "2025.10.0"); err == nil {
		t.Fatal("changed release admitted")
	}
	for _, x := range []struct{ owner, ecosystem, pkg, version string }{{"cloudflared", "npm", "cloudflared", "2026.10.7"}, {"cloudflared", "source", "other", "2026.10.7"}, {"cloudflared", "source", "cloudflared", "latest"}, {"cloudflared", "source", "cloudflared", "2025.10.0"}, {"../cloudflared", "source", "cloudflared", "2026.10.7"}} {
		if _, err := Resolve(root, x.owner, "host-tool", x.ecosystem, x.pkg, x.version); err == nil {
			t.Fatalf("unreviewed acquisition admitted: %+v", x)
		}
	}
}

func TestManagedServiceReleaseRevalidatesBeforeAcquisition(t *testing.T) {
	root := cloudflaredResourceFixture(t)
	base, err := Resolve(root, "cloudflared", "host-tool", "source", "cloudflared", "2026.10.7")
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Resolution){
		"command":    func(r *Resolution) { r.Argv = []string{"sh", "unsafe"} },
		"path":       func(r *Resolution) { r.SurfaceRoot = root },
		"manifest":   func(r *Resolution) { r.ManifestPath += ".other" },
		"policy":     func(r *Resolution) { r.Profile.ScriptsDisabled = false },
		"exact":      func(r *Resolution) { r.ExactVersion = false },
		"host tool":  func(r *Resolution) { r.HostTool = true },
		"refresh":    func(r *Resolution) { r.HostToolRefresh = true },
		"extra lock": func(r *Resolution) { r.PythonLockfiles = []string{"unowned.lock"} },
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			change(&r)
			if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil || !strings.Contains(err.Error(), "resolution changed") {
				t.Fatalf("changed staging admitted: %v", err)
			}
		})
	}
	if err := os.WriteFile(base.ManifestPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), base); err == nil {
		t.Fatal("changed source admitted")
	}
}
