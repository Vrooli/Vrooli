package installgateway

import (
	"archive/zip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
)

func TestProbeGoModuleClosureCountsLocalAndFullPinnedProxyClosure(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "packages", "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	writeModuleFixture(t, root, "example.org/root", "require example.org/dep v1.0.0\n")
	if err := os.WriteFile(filepath.Join(root, "root.go"), []byte("package root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	localTwo := []byte("module example.org/dep2\n\ngo 1.22\n")
	localOne := []byte("module example.org/dep\n\ngo 1.22\n\nrequire example.org/dep2 v1.0.0\n")
	depZip := moduleZip(t, "example.org/dep", "v1.0.0", "dep.go")
	dep2Zip := moduleZip(t, "example.org/dep2", "v1.0.0", "dep2.go")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/example.org/dep/@v/v1.0.0.mod":
			_, _ = w.Write(localOne)
		case "/example.org/dep/@v/v1.0.0.zip":
			_, _ = w.Write(depZip)
		case "/example.org/dep2/@v/v1.0.0.mod":
			_, _ = w.Write(localTwo)
		case "/example.org/dep2/@v/v1.0.0.zip":
			_, _ = w.Write(dep2Zip)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	topology := gomodreconcile.Topology{"example.org/root": root}
	targetManifest, err := readGoModuleFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	probe, err := probeGoModuleClosure(context.Background(), repo, "example.org/root", "example.org/root", targetManifest, topology, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if probe.ModuleCount != 3 || probe.SourceBytes <= 0 || probe.ArchiveBytes <= 0 || probe.ExpandedBytes <= 0 || probe.MetadataBytes <= 0 {
		t.Fatalf("incomplete Go module closure probe: %+v", probe)
	}
}

func TestProbeGoModuleClosureIncludesExistingTargetManifestDependencies(t *testing.T) {
	repo := t.TempDir()
	target := filepath.Join(repo, "scenarios", "notes", "api")
	newModule := filepath.Join(repo, "packages", "new")
	for _, dir := range []string{target, newModule} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeModuleFixture(t, target, "example.org/target", "require example.org/existing v1.0.0\n")
	writeModuleFixture(t, newModule, "example.org/new", "require example.org/newdep v1.0.0\n")
	for _, file := range []string{filepath.Join(target, "target.go"), filepath.Join(newModule, "new.go")} {
		if err := os.WriteFile(file, []byte("package fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	requested := map[string]bool{}
	var requestedMu sync.Mutex
	existingZip := moduleZip(t, "example.org/existing", "v1.0.0", "dep.go")
	newDepZip := moduleZip(t, "example.org/newdep", "v1.0.0", "dep.go")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedMu.Lock()
		requested[r.URL.Path] = true
		requestedMu.Unlock()
		modulePath := ""
		switch r.URL.Path {
		case "/example.org/existing/@v/v1.0.0.mod", "/example.org/existing/@v/v1.0.0.zip":
			modulePath = "example.org/existing"
		case "/example.org/newdep/@v/v1.0.0.mod", "/example.org/newdep/@v/v1.0.0.zip":
			modulePath = "example.org/newdep"
		default:
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".mod") {
			_, _ = fmt.Fprintf(w, "module %s\n\ngo 1.22\n", modulePath)
			return
		}
		if modulePath == "example.org/existing" {
			_, _ = w.Write(existingZip)
		} else {
			_, _ = w.Write(newDepZip)
		}
	}))
	defer server.Close()
	targetManifest, err := readGoModuleFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	topology := gomodreconcile.Topology{"example.org/target": target, "example.org/new": newModule}
	probe, err := probeGoModuleClosure(context.Background(), repo, "example.org/new", "example.org/target", targetManifest, topology, server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	requestedMu.Lock()
	defer requestedMu.Unlock()
	if probe.ModuleCount != 4 || !requested["/example.org/existing/@v/v1.0.0.zip"] || !requested["/example.org/newdep/@v/v1.0.0.zip"] {
		t.Fatalf("closure omitted target or new-module dependencies: probe=%+v requests=%v", probe, requested)
	}
}

func TestProbeGoModuleClosureFailsClosedOnUnboundedOrUnsupportedGraph(t *testing.T) {
	t.Run("local replace", func(t *testing.T) {
		repo := t.TempDir()
		root := filepath.Join(repo, "root")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(repo, "dep"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeModuleFixture(t, filepath.Join(repo, "dep"), "example.org/wrong", "")
		writeModuleFixture(t, root, "example.org/root", "replace example.org/dep => ../dep\n")
		targetManifest, err := readGoModuleFile(filepath.Join(root, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = probeGoModuleClosure(context.Background(), repo, "example.org/root", "example.org/root", targetManifest, gomodreconcile.Topology{"example.org/root": root}, "https://proxy.golang.org", http.DefaultClient)
		if err == nil || !strings.Contains(err.Error(), "does not resolve to an in-repository module") {
			t.Fatalf("replace graph = %v, want fail-closed unsupported reason", err)
		}
	})

	t.Run("oversized response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", fmt.Sprint(goModuleProbeResponseLimit+1))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		_, err := boundedHTTPGet(context.Background(), server.Client(), server.URL, 32)
		if err == nil || !strings.Contains(err.Error(), "Content-Length") {
			t.Fatalf("oversized response = %v, want bounded refusal", err)
		}
	})
}

func TestProbeGoModuleGrowthRefusesRemotePackageWithoutLocalModuleEvidence(t *testing.T) {
	_, err := probeGoModuleGrowth(context.Background(), Resolution{
		PackageName: "example.org/remote/package", PackageManager: "go",
	})
	if err == nil || !strings.Contains(err.Error(), "exact repository-local module target") {
		t.Fatalf("remote Go package probe = %v, want fail-closed local-source requirement", err)
	}
}

func TestGoModuleProxyPolicyPinsCoveredPublicConfiguration(t *testing.T) {
	t.Setenv("GOPROXY", "https://proxy.golang.org,direct")
	t.Setenv("GOSUMDB", "sum.golang.org")
	t.Setenv("GOWORK", "off")
	if got, err := goModuleProxyURL(); err != nil || got != "https://proxy.golang.org" {
		t.Fatalf("approved Go module source = %q, %v", got, err)
	}
	t.Setenv("GOPROXY", "https://private.example/proxy")
	if _, err := goModuleProxyURL(); err == nil || !strings.Contains(err.Error(), "custom GOPROXY") {
		t.Fatalf("custom Go module proxy was not rejected: %v", err)
	}
}

func TestGoModuleExecutorPinsProbeEnvironment(t *testing.T) {
	env := managerCacheEnvironment("go", "/tmp/go-mod-cache", "/tmp/go-build-cache")
	want := []string{
		"GOMODCACHE=/tmp/go-mod-cache",
		"GOCACHE=/tmp/go-build-cache",
		"GOPROXY=https://proxy.golang.org",
		"GOSUMDB=sum.golang.org",
		"GONOPROXY=",
		"GONOSUMDB=",
		"GOPRIVATE=",
		"GOINSECURE=",
		"GOWORK=off",
		"GOTOOLCHAIN=local",
		"GOFLAGS=",
		"GOCACHEPROG=",
	}
	for _, expected := range want {
		found := false
		for _, actual := range env {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Go executor environment omits %q: %#v", expected, env)
		}
	}
}

func writeModuleFixture(t *testing.T, dir, modulePath, extra string) {
	t.Helper()
	text := "module " + modulePath + "\n\ngo 1.22\n"
	if extra != "" {
		text += "\n" + extra
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func moduleZip(t *testing.T, modulePath, version, file string) []byte {
	t.Helper()
	var output strings.Builder
	writer := zip.NewWriter(&stringWriter{Builder: &output})
	entry, err := writer.Create(modulePath + "@" + version + "/" + file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("package fixture\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return []byte(output.String())
}

type stringWriter struct{ *strings.Builder }

func (w *stringWriter) Write(p []byte) (int, error) { return w.Builder.Write(p) }
