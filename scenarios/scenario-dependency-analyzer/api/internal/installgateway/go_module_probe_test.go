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

	"golang.org/x/mod/modfile"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
)

func TestProbeBoundsEveryProxyResponse(t *testing.T) {
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

func TestProbeGoModuleGrowthRequiresRepositoryContext(t *testing.T) {
	_, err := probeGoModuleGrowth(context.Background(), Resolution{
		PackageName: "example.org/remote/package", PackageManager: "go",
	})
	if err == nil || !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("Go probe without a repository = %v, want a refusal naming the missing context", err)
	}
}

// goProxyFixture serves modules to the external-module probe and records the
// paths requested.
type goProxyFixture struct {
	server    *httptest.Server
	mu        sync.Mutex
	requested []string
}

func newGoProxy(t *testing.T, modules map[string]string, latest map[string]string) *goProxyFixture {
	t.Helper()
	fixture := &goProxyFixture{}
	zips := map[string][]byte{}
	for key := range modules {
		path, version, _ := strings.Cut(key, "@")
		zips[key] = moduleZip(t, path, version, "pkg.go")
	}
	fixture.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.mu.Lock()
		fixture.requested = append(fixture.requested, r.URL.Path)
		fixture.mu.Unlock()
		trimmed := strings.TrimPrefix(r.URL.Path, "/")
		if path, ok := strings.CutSuffix(trimmed, "/@latest"); ok {
			if version, found := latest[path]; found {
				_, _ = fmt.Fprintf(w, `{"Version":%q}`, version)
				return
			}
		}
		path, rest, ok := strings.Cut(trimmed, "/@v/")
		if !ok {
			http.NotFound(w, r)
			return
		}
		switch {
		case strings.HasSuffix(rest, ".info"):
			version := strings.TrimSuffix(rest, ".info")
			if _, found := modules[path+"@"+version]; found {
				_, _ = fmt.Fprintf(w, `{"Version":%q}`, version)
				return
			}
		case strings.HasSuffix(rest, ".mod"):
			key := path + "@" + strings.TrimSuffix(rest, ".mod")
			if body, found := modules[key]; found {
				_, _ = w.Write([]byte(body))
				return
			}
		case strings.HasSuffix(rest, ".zip"):
			if body, found := zips[path+"@"+strings.TrimSuffix(rest, ".zip")]; found {
				_, _ = w.Write(body)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (f *goProxyFixture) asked(suffix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, path := range f.requested {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func externalGoManifest(t *testing.T, requires string) *modfile.File {
	t.Helper()
	manifest, err := modfile.Parse("go.mod", []byte("module example.org/target\n\ngo 1.22\n\nrequire (\n"+requires+")\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func externalGoResolution(pkg, version string) Resolution {
	spec := pkg
	if version != "" {
		spec += "@" + version
	}
	return Resolution{PackageName: pkg, PackageManager: "go", Argv: []string{"go", "get", spec}}
}

func TestProbeExternalGoModuleTreatsAlreadyRequiredCachedModuleAsCovered(t *testing.T) {
	// notification-hub/cli already required gopkg.in/yaml.v3 as an indirect
	// dependency; making it direct needs no new bytes.
	modCache := t.TempDir()
	t.Setenv("GOMODCACHE", modCache)
	if err := os.MkdirAll(filepath.Join(modCache, "gopkg.in", "yaml.v3@v3.0.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	proxy := newGoProxy(t, map[string]string{"gopkg.in/yaml.v3@v3.0.1": "module gopkg.in/yaml.v3\n"}, nil)
	manifest := externalGoManifest(t, "\tgopkg.in/yaml.v3 v3.0.1 // indirect\n")
	probe, err := probeExternalGoModuleGrowth(context.Background(), externalGoResolution("gopkg.in/yaml.v3", "v3.0.1"), manifest, proxy.server.URL, proxy.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if probe.ModuleCount != 1 || probe.ArchiveBytes != 0 || probe.ExpandedBytes != 0 || probe.MetadataBytes <= 0 {
		t.Fatalf("covered module probe = %+v, want metadata only", probe)
	}
	if proxy.asked(".zip") {
		t.Fatal("a module the cache already holds was downloaded to be sized")
	}
}

func TestProbeExternalGoModuleCountsOnlyWhatTheManifestLacks(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	proxy := newGoProxy(t, map[string]string{
		"example.org/new@v1.0.0":    "module example.org/new\n\nrequire (\n\texample.org/shared v1.1.0\n\texample.org/other v1.0.0\n)\n",
		"example.org/other@v1.0.0":  "module example.org/other\n",
		"example.org/shared@v1.1.0": "module example.org/shared\n",
	}, nil)
	modCache := os.Getenv("GOMODCACHE")
	if err := os.MkdirAll(filepath.Join(modCache, "example.org", "shared@v1.2.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := externalGoManifest(t, "\texample.org/shared v1.2.0\n")
	probe, err := probeExternalGoModuleGrowth(context.Background(), externalGoResolution("example.org/new/sub/pkg", "v1.0.0"), manifest, proxy.server.URL, proxy.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if probe.ModuleCount != 3 || probe.ArchiveBytes <= 0 {
		t.Fatalf("closure probe = %+v", probe)
	}
	if !proxy.asked("/example.org/new/@v/v1.0.0.zip") || !proxy.asked("/example.org/other/@v/v1.0.0.zip") {
		t.Fatalf("new module or its missing dependency was not sized: %v", proxy.requested)
	}
	if proxy.asked("/example.org/shared/@v/v1.1.0.zip") {
		t.Fatalf("shared v1.1.0 is satisfied by the manifest's v1.2.0 and must not be downloaded: %v", proxy.requested)
	}
}

func TestProbeExternalGoModuleNamesTheLatestVersionWhenNoneIsGiven(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	proxy := newGoProxy(t, nil, map[string]string{"example.org/new": "v1.4.0"})
	manifest := externalGoManifest(t, "")
	_, err := probeExternalGoModuleGrowth(context.Background(), externalGoResolution("example.org/new/sub", ""), manifest, proxy.server.URL, proxy.server.Client())
	if err == nil || !strings.Contains(err.Error(), "repeat with example.org/new/sub@v1.4.0") {
		t.Fatalf("unversioned external install = %v, want an actionable exact-version instruction", err)
	}
}

func TestProbeExternalGoModuleRejectsNonSemverQueries(t *testing.T) {
	t.Setenv("GOMODCACHE", t.TempDir())
	proxy := newGoProxy(t, nil, nil)
	manifest := externalGoManifest(t, "\texample.org/new v1.0.0\n")
	_, err := probeExternalGoModuleGrowth(context.Background(), externalGoResolution("example.org/new", "master"), manifest, proxy.server.URL, proxy.server.Client())
	if err == nil || !strings.Contains(err.Error(), "exact semantic version") {
		t.Fatalf("branch query = %v", err)
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

// Module zips name their entries with the module path as written. Upper-case
// letters are escaped only in proxy URLs and cache directories, so a path such
// as github.com/BurntSushi/toml must be accepted.
func TestZipExpandedBytesAcceptsModulePathsWithUpperCaseLetters(t *testing.T) {
	zipped := moduleZip(t, "github.com/BurntSushi/toml", "v1.4.0", "decode.go")
	got, err := zipExpandedBytes("github.com/BurntSushi/toml", "v1.4.0", zipped)
	if err != nil || got != int64(len("package fixture\n")) {
		t.Fatalf("zipExpandedBytes = %d, %v", got, err)
	}
	if _, err := zipExpandedBytes("github.com/BurntSushi/toml", "v1.5.0", zipped); err == nil || !strings.Contains(err.Error(), "not under") {
		t.Fatalf("an entry under another version must be refused, got %v", err)
	}
}

// repoPackageFixture is a repository module whose package imports external
// modules. goList plays the offline `go list -deps` the owner module answers.
type repoPackageFixture struct {
	ownerDir, cache string
	owner, consumer *modfile.File
	listed          string
	args            []string
	dir             string
	env             []string
}

func newRepoPackageFixture(t *testing.T, ownerRequires string, listed ...string) *repoPackageFixture {
	t.Helper()
	f := &repoPackageFixture{ownerDir: t.TempDir(), cache: t.TempDir(), listed: strings.Join(listed, "\n")}
	owner, err := modfile.Parse("go.mod", []byte("module example.org/vrooli\n\ngo 1.22\n\nrequire (\n"+ownerRequires+")\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	f.owner, f.consumer = owner, externalGoManifest(t, "")
	previous := runGoList
	runGoList = func(_ context.Context, dir string, env []string, args ...string) ([]byte, error) {
		f.dir, f.env, f.args = dir, env, args
		return []byte(f.listed), nil
	}
	t.Cleanup(func() { runGoList = previous })
	return f
}

func (f *repoPackageFixture) cacheModule(t *testing.T, path, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(f.cache, path+"@"+version), 0o755); err != nil {
		t.Fatal(err)
	}
	download := filepath.Join(f.cache, "cache", "download", path, "@v")
	if err := os.MkdirAll(download, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(download, version+".mod"), []byte("module "+path+"\n\ngo 1.22\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f *repoPackageFixture) probe(t *testing.T, proxy *goProxyFixture) (goModuleProbeResult, error) {
	t.Helper()
	r := Resolution{PackageName: "example.org/vrooli/packages/auth", PackageManager: "go"}
	return probeRepositoryPackageGrowth(context.Background(), r, f.ownerDir, f.owner, f.consumer, proxy.server.URL, f.cache, proxy.server.Client())
}

// The evidence case: installing a package of the repository-root module used to
// walk the whole checkout and the module's entire requirement graph. It now
// costs what the package's import closure needs beyond the cache.
func TestRepositoryPackageCostsOnlyWhatItsImportClosureNeedsBeyondTheCache(t *testing.T) {
	f := newRepoPackageFixture(t, "\texample.org/cached v1.0.0\n\texample.org/fresh v1.2.0\n\texample.org/unused v1.0.0\n",
		`{"ImportPath":"fmt","Standard":true}`,
		`{"ImportPath":"example.org/vrooli/packages/auth","Module":{"Path":"example.org/vrooli","Main":true}}`,
		`{"ImportPath":"example.org/cached/sub","Module":{"Path":"example.org/cached","Version":"v1.0.0"}}`,
		`{"ImportPath":"example.org/fresh/pkg","Error":{"Err":"module lookup disabled by GOPROXY=off"}}`,
	)
	f.cacheModule(t, "example.org/cached", "v1.0.0")
	proxy := newGoProxy(t, map[string]string{"example.org/fresh@v1.2.0": "module example.org/fresh\n\ngo 1.22\n"}, nil)

	probe, err := f.probe(t, proxy)
	if err != nil {
		t.Fatal(err)
	}
	if probe.ModuleCount != 2 || probe.ArchiveBytes <= 0 || probe.ExpandedBytes <= 0 {
		t.Fatalf("probe = %+v; the cached module is free and only the uncached one is downloaded", probe)
	}
	if proxy.asked("example.org/unused/@v/v1.0.0.mod") || proxy.asked("example.org/cached/@v/v1.0.0.zip") {
		t.Fatalf("the proxy was asked about a module the package does not need: %v", proxy.requested)
	}
	if f.dir != f.ownerDir || strings.Join(f.args, " ") != "list -e -deps -json=ImportPath,Standard,Module,Error example.org/vrooli/packages/auth" {
		t.Fatalf("go list ran as %q in %s", f.args, f.dir)
	}
	env := strings.Join(f.env, "\n")
	for _, want := range []string{"GOPROXY=off", "GOFLAGS=-mod=readonly", "GOWORK=off", "GOTOOLCHAIN=local", "GOMODCACHE=" + f.cache} {
		if !strings.Contains(env, want) {
			t.Fatalf("go list must run offline and read-only; missing %s", want)
		}
	}
}

func TestRepositoryPackageWhoseModulesAreAllCachedStillGetsAFloorAllowance(t *testing.T) {
	f := newRepoPackageFixture(t, "\texample.org/cached v1.0.0\n",
		`{"ImportPath":"example.org/cached","Module":{"Path":"example.org/cached","Version":"v1.0.0"}}`)
	f.cacheModule(t, "example.org/cached", "v1.0.0")
	probe, err := f.probe(t, newGoProxy(t, nil, nil))
	if err != nil || probe.ArchiveBytes != 0 || probe.MetadataBytes < goRepositoryInstallAllowance {
		t.Fatalf("probe = %+v, %v", probe, err)
	}
}

func TestRepositoryPackageFollowsFirstPartyModulesInPlaceAndNeverAsksTheProxyForThem(t *testing.T) {
	f := newRepoPackageFixture(t, "\texample.org/platform v0.0.0\n",
		`{"ImportPath":"example.org/platform/x","Module":{"Path":"example.org/platform","Version":"v0.0.0","Replace":{"Path":"../platform"}}}`)
	proxy := newGoProxy(t, nil, nil)
	probe, err := f.probe(t, proxy)
	if err != nil || probe.ModuleCount != 0 || len(proxy.requested) != 0 {
		t.Fatalf("probe = %+v, %v, proxy asked %v", probe, err, proxy.requested)
	}
}

func TestRepositoryPackageRefusesAnImportNoRequiredModuleProvides(t *testing.T) {
	f := newRepoPackageFixture(t, "\texample.org/cached v1.0.0\n",
		`{"ImportPath":"example.org/missing/pkg","Error":{"Err":"no required module provides package"}}`)
	_, err := f.probe(t, newGoProxy(t, nil, nil))
	if err == nil || !strings.Contains(err.Error(), "example.org/missing/pkg") || !strings.Contains(err.Error(), "no module in example.org/vrooli provides it") {
		t.Fatalf("err = %v", err)
	}
}

func TestRepositoryPackageExplainsAGoListFailure(t *testing.T) {
	f := newRepoPackageFixture(t, "")
	runGoList = func(context.Context, string, []string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 1: go.mod needs updating")
	}
	_, err := f.probe(t, newGoProxy(t, nil, nil))
	if err == nil || !strings.Contains(err.Error(), "could not load the import closure") || !strings.Contains(err.Error(), "go.mod needs updating") {
		t.Fatalf("err = %v", err)
	}
}

// Naming a module itself (its root need not be a package) lists every package
// the module holds instead of failing on a root that has no Go files.
func TestRepositoryModuleRootIsListedWithAPackagePattern(t *testing.T) {
	f := newRepoPackageFixture(t, "", `{"ImportPath":"example.org/vrooli/sub","Module":{"Path":"example.org/vrooli","Main":true}}`)
	r := Resolution{PackageName: "example.org/vrooli", PackageManager: "go"}
	if _, err := probeRepositoryPackageGrowth(context.Background(), r, f.ownerDir, f.owner, f.consumer, "http://127.0.0.1:1", f.cache, http.DefaultClient); err != nil {
		t.Fatal(err)
	}
	if got := f.args[len(f.args)-1]; got != "example.org/vrooli/..." {
		t.Fatalf("pattern = %q", got)
	}
}

// A first-party Go package resolves through a local replace and its import
// closure is already cached, so the install rewrites go.mod and go.sum and
// fetches nothing: the plan says so, with a size of kilobytes rather than the
// build-cache reservation of an install that compiles.
func TestPlanForAFirstPartyGoPackageThatFetchesNothingIsInputsOnly(t *testing.T) {
	repo := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.org/vrooli\n\ngo 1.22\n\nrequire example.org/cached v1.0.0\n")
	write("packages/auth/auth.go", "package auth\n")
	write("scenarios/demo/cli/go.mod", "module example.org/demo/cli\n\ngo 1.22\n\nrequire example.org/vrooli v0.0.0\n\nreplace example.org/vrooli => ../../..\n")
	previousTopology := loadGoTopology
	loadGoTopology = func(string) (gomodreconcile.Topology, error) {
		return gomodreconcile.Topology{
			"example.org/vrooli":   repo,
			"example.org/demo/cli": filepath.Join(repo, "scenarios", "demo", "cli"),
		}, nil
	}
	t.Cleanup(func() { loadGoTopology = previousTopology })
	modCache := t.TempDir()
	t.Setenv("GOMODCACHE", modCache)
	t.Setenv("HOME", t.TempDir())
	f := &repoPackageFixture{cache: modCache}
	f.cacheModule(t, "example.org/cached", "v1.0.0")
	previous := runGoList
	runGoList = func(context.Context, string, []string, ...string) ([]byte, error) {
		return []byte(`{"ImportPath":"example.org/vrooli/packages/auth","Module":{"Path":"example.org/vrooli","Main":true}}` + "\n" +
			`{"ImportPath":"example.org/cached","Module":{"Path":"example.org/cached","Version":"v1.0.0"}}`), nil
	}
	t.Cleanup(func() { runGoList = previous })

	plan, err := PlanDiskAdmission(context.Background(), Resolution{
		RepositoryRoot: repo, SurfaceRoot: filepath.Join(repo, "scenarios", "demo", "cli"),
		ManifestPath: filepath.Join(repo, "scenarios", "demo", "cli", "go.mod"),
		PackageName:  "example.org/vrooli/packages/auth", PackageManager: "go",
		Argv: []string{"go", "get", "example.org/vrooli/packages/auth"},
	})
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if !plan.WritesInputsOnly || plan.GoArchiveBytes != 0 {
		t.Fatalf("an install that fetches nothing must be inputs-only: %+v", plan)
	}
	var total int64
	for _, p := range plan.Paths {
		total += p.PeakBytes
	}
	if total > 32<<20 {
		t.Fatalf("an install that fetches nothing is reserved at its true size, got %d bytes: %+v", total, plan.Paths)
	}
}
