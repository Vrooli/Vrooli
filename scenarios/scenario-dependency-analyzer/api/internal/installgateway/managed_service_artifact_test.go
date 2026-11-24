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

func composedCloudflaredResourceFixture(t *testing.T, bound bool) string {
	t.Helper()
	root := cloudflaredResourceFixture(t)
	path := filepath.Join(root, "resources", "cloudflared", "resource.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	service := m["managed_service"].(map[string]any)
	// The managed-update owner currently requires a single URL target. Keep this
	// fixture focused on source-to-disk admission and avoid inventing an update
	// path as part of the composed capacity test.
	delete(service, "update_policy")
	service["acquisition"] = map[string]any{
		"kind": "composed", "license": "Apache-2.0",
		"targets": []any{map[string]any{
			"when":   map[string]string{"os": runtime.GOOS, "arch": runtime.GOARCH},
			"layout": "dir", "artifact_sha256": strings.Repeat("a", 64),
			"compose": []any{
				map[string]any{"role": "runtime", "kind": "url", "dest": "runtime", "url": "https://example.invalid/runtime.tar.gz", "sha256": strings.Repeat("b", 64), "archive": "tar.gz", "download_bytes": 100, "max_expanded_bytes": 1000, "max_entries": 10},
				map[string]any{"role": "support", "kind": "url", "dest": "support", "url": "https://example.invalid/support.tar.gz", "sha256": strings.Repeat("c", 64), "archive": "tar.gz", "download_bytes": 200, "max_expanded_bytes": 2000, "max_entries": 20},
			},
		}},
	}
	if !bound {
		steps := service["acquisition"].(map[string]any)["targets"].([]any)[0].(map[string]any)["compose"].([]any)
		delete(steps[1].(map[string]any), "max_entries")
	}
	body, err = json.Marshal(m)
	if err != nil {
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

func TestManagedServiceArtifactRouteDoesNotAdmitPlaywrightBrowser(t *testing.T) {
	_, err := resolveManagedServiceResource(t.TempDir(), "playwright-chromium", "source", "playwright-chromium", "1169")
	if err == nil || err.Error() != "resource has no admitted managed artifact route" {
		t.Fatalf("unapproved browser artifact route must fail closed with its exact gate, got %v", err)
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

func TestManagedServiceComposedReleaseCarriesSummedCapacityToSDA(t *testing.T) {
	root := composedCloudflaredResourceFixture(t, true)
	artifacts := filepath.Join(root, "private-artifacts")
	t.Setenv("HOME", filepath.Join(root, "operator-home"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", artifacts)
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "resource-state"))

	resolution, err := Resolve(root, "cloudflared", "host-tool", "source", "cloudflared", "2026.10.7")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil {
		t.Fatal(err)
	}
	wantPeak := int64(3*(300+3000) + 1<<20)
	if plan.Unsupported != "" || plan.ProbeBytes != 300 || len(plan.Paths) == 0 || plan.Paths[0].PeakBytes != wantPeak {
		t.Fatalf("composed SDA capacity plan=%+v want download=300 peak=%d", plan, wantPeak)
	}
	if _, err := os.Stat(artifacts); !os.IsNotExist(err) {
		t.Fatalf("planning performed artifact acquisition or created storage: %v", err)
	}

	manifestPath := filepath.Join(root, "resources", "cloudflared", "resource.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	steps := manifest["managed_service"].(map[string]any)["acquisition"].(map[string]any)["targets"].([]any)[0].(map[string]any)["compose"].([]any)
	delete(steps[0].(map[string]any), "download_bytes")
	changed, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := provisionManagedServiceArtifact(context.Background(), resolution); err == nil {
		t.Fatal("mutated per-step bounds reached artifact acquisition")
	}
	if _, err := os.Stat(artifacts); !os.IsNotExist(err) {
		t.Fatalf("changed plan acquired artifact bytes or created storage: %v", err)
	}
}

func TestManagedServiceComposedReleaseRejectsUnboundedStepBeforeAdmission(t *testing.T) {
	root := composedCloudflaredResourceFixture(t, false)
	if _, err := Resolve(root, "cloudflared", "host-tool", "source", "cloudflared", "2026.10.7"); err == nil {
		t.Fatalf("composed release with missing step bound resolved: %v", err)
	}
}

func openRGBResourceFixture(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("exact OpenRGB envelope staging profile is linux-amd64")
	}
	root := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	// This fixture omits source-selection persistence so disk planning can be
	// tested independently of the operator's source enrollment.
	service := manifest["managed_service"].(map[string]any)
	service["source_selection_required"] = false
	delete(service, "host_tool_executable")
	delete(service, "host_tool_version")
	raw, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "resources", "openrgb", "resource.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestOpenRGBManagedSourceFreezesManifestAndExactRelease(t *testing.T) {
	root := openRGBResourceFixture(t)
	resolution, err := Resolve(root, "openrgb", "host-tool", "source", "openrgb", "1.1")
	if err != nil {
		t.Fatal(err)
	}
	if resolution.HostManifestSHA256 == "" || resolution.Profile.LifecycleMode != "managed-service-artifact" || !resolution.Profile.ScriptsDisabled || resolution.Command() != "native-acquire openrgb 1.1" {
		t.Fatalf("unexpected source resolution %+v", resolution)
	}
	for _, version := range []string{"", "latest", "1.0"} {
		if _, err := Resolve(root, "openrgb", "host-tool", "source", "openrgb", version); err == nil {
			t.Fatalf("release %q admitted", version)
		}
	}
	if _, err := Resolve(root, "openrgb", "host-tool", "source", "cloudflared", "1.1"); err == nil {
		t.Fatal("cross-owner source admitted")
	}
	raw, err := os.ReadFile(resolution.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// An edited digest for the same named release must not inherit admission.
	raw = []byte(strings.ReplaceAll(string(raw), "d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d", strings.Repeat("a", 64)))
	if err := os.WriteFile(resolution.ManifestPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), resolution); err == nil || !strings.Contains(err.Error(), "resolution changed") {
		t.Fatalf("changed manifest was not fenced before acquisition: %v", err)
	}
}

func TestOpenRGBManagedSourceReservesAllCanonicalStagePaths(t *testing.T) {
	root := openRGBResourceFixture(t)
	artifacts := filepath.Join(root, "private-artifacts")
	t.Setenv("HOME", filepath.Join(root, "operator-home"))
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", artifacts)
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(root, "resource-state"))
	resolution, err := Resolve(root, "openrgb", "host-tool", "source", "openrgb", "1.1")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Unsupported != "" || plan.ProbeBytes != 37528768 || len(plan.Paths) != 8 {
		t.Fatalf("unbounded or incomplete disk plan %+v", plan)
	}
	if plan.Paths[0].Path != filepath.Join(artifacts, "openrgb", "1.1") || plan.Paths[0].PeakBytes < 3*plan.ProbeBytes {
		t.Fatalf("artifact path escaped capacity plan %+v", plan)
	}
	for _, requirement := range plan.Paths {
		if !filepath.IsAbs(requirement.Path) || requirement.PeakBytes <= 0 {
			t.Fatalf("uncovered stage path %+v", requirement)
		}
	}
	if _, err := os.Stat(artifacts); !os.IsNotExist(err) {
		t.Fatalf("planning created artifact storage: %v", err)
	}
}
