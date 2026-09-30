package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenario"
)

const goExecutablePath = "/usr/bin/go"

// cannedGoList builds a hostProbeDeps whose goListJSON seam returns a fixed
// payload (or error) and whose lookPath resolves "go". Other fields are nil; the
// adapter under test never touches them.
func cannedGoList(payload []byte, listErr error, goFound bool) hostProbeDeps {
	return hostProbeDeps{
		lookPath: func(name string) (string, error) {
			if name == "go" && goFound {
				return goExecutablePath, nil
			}
			return "", exec.ErrNotFound
		},
		goListJSON: func(string) ([]byte, error) { return payload, listErr },
	}
}

const goListFixture = `
{"Dir":"/repo/scenarios/x/api","Module":{"Dir":"/repo/scenarios/x/api","GoMod":"/repo/scenarios/x/api/go.mod"}}
{"Dir":"/repo/packages/api-core/foo","Module":{"Dir":"/repo/packages/api-core","GoMod":"/repo/packages/api-core/go.mod"}}
{"Dir":"/usr/lib/go/src/fmt","Standard":true}
{"Dir":"/home/u/go/pkg/mod/github.com/x@v1/bar","Module":{"GoMod":"/home/u/go/pkg/mod/github.com/x@v1/go.mod"}}
`

func TestGoListFreshnessInputs_PreciseClosure(t *testing.T) {
	got, ok := goListFreshnessInputs("/repo/scenarios/x/api", "/repo", cannedGoList([]byte(goListFixture), nil, true))
	if !ok {
		t.Fatal("expected adapter to succeed")
	}
	want := []string{
		"packages/api-core/foo",
		"packages/api-core/go.mod",
		"packages/api-core/go.sum",
		"scenarios/x/api",
		"scenarios/x/api/go.mod",
		"scenarios/x/api/go.sum",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inputs mismatch:\n got=%v\nwant=%v", got, want)
	}
}

func TestGoListFreshnessInputsContextCancelsResolver(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "x", "api")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"go.mod":  "module example.test/x\n",
		"go.sum":  "",
		"main.go": "package main\nfunc main() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	started := make(chan struct{})
	deps := cannedGoList([]byte(goListFixture), nil, true)
	deps.cache = &hostProbeCache{goToolchain: "go version test", goToolchainOK: true}
	deps.readFile = os.ReadFile
	deps.goListJSONContext = func(ctx context.Context, _ string) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan bool, 1)
	go func() {
		_, ok := goListFreshnessInputsContext(ctx, dir, root, deps)
		result <- ok
	}()
	select {
	case <-started:
		cancel()
	case <-time.After(time.Second):
		cancel()
		t.Fatal("go list resolver did not start")
	}
	select {
	case ok := <-result:
		if ok || !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("closure ok=%v context error=%v, want canceled/unavailable", ok, ctx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("go list resolver kept running after caller cancellation")
	}
}

func TestFreshnessReportByNameContextReturnsWhenAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (&Runner{}).FreshnessReportByNameContext(ctx, "browser-automation-studio", "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("freshness error = %v, want context.Canceled", err)
	}
}

func TestGoListFreshnessInputs_CachesByModuleAndSourceFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("sum-v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	deps := hostProbeDeps{
		cache:    &hostProbeCache{},
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		readFile: os.ReadFile,
		goListJSON: func(string) ([]byte, error) {
			calls++
			return []byte(goListFixture), nil
		},
	}
	// Exercise the cache adapter directly so the test is independent of the
	// repository-root filtering in goListFreshnessInputs.
	if _, err := cachedGoListJSONContext(context.TODO(), dir, deps); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedGoListJSONContext(context.TODO(), dir, deps); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("go list calls = %d, want one cached call", calls)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example/v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedGoListJSONContext(context.TODO(), dir, deps); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("go list calls after go.mod change = %d, want two", calls)
	}
	if err := os.WriteFile(sourcePath, []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"changed\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedGoListJSONContext(context.TODO(), dir, deps); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("go list calls after source change = %d, want three", calls)
	}
}

func TestClosureCache_HitSkipsGoList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("sum-v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := []byte(fmt.Sprintf("{\"Dir\":%q,\"Module\":{\"Dir\":%q,\"GoMod\":%q}}\n", dir, dir, filepath.Join(dir, "go.mod")))
	calls := 0
	deps := hostProbeDeps{
		readFile: os.ReadFile,
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		goListJSON: func(string) ([]byte, error) {
			calls++
			return payload, nil
		},
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("first closure lookup did not succeed")
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("cached closure lookup did not succeed")
	}
	if calls != 1 {
		t.Fatalf("go list calls = %d, want one durable-cache miss", calls)
	}
	if _, err := os.Stat(closureCachePath(dir)); err != nil {
		t.Fatalf("closure cache was not written: %v", err)
	}
}

func TestClosureCache_MissOnGoModChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	deps := hostProbeDeps{
		readFile: os.ReadFile,
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		goListJSON: func(string) ([]byte, error) {
			calls++
			return []byte(fmt.Sprintf("{\"Dir\":%q,\"Module\":{\"Dir\":%q,\"GoMod\":%q}}\n", dir, dir, filepath.Join(dir, "go.mod"))), nil
		},
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("initial closure lookup did not succeed")
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example/v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("post-change closure lookup did not succeed")
	}
	if calls != 2 {
		t.Fatalf("go list calls = %d, want cache miss after go.mod change", calls)
	}
}

func TestClosureCache_MissOnSourceImportChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	deps := hostProbeDeps{
		readFile: os.ReadFile,
		walkDir:  filepath.WalkDir,
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		goListJSON: func(string) ([]byte, error) {
			calls++
			return []byte(fmt.Sprintf("{\"Dir\":%q,\"Module\":{\"Dir\":%q,\"GoMod\":%q}}\n", dir, dir, filepath.Join(dir, "go.mod"))), nil
		},
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("initial closure lookup did not succeed")
	}
	if err := os.WriteFile(mainPath, []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"changed\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := goListFreshnessInputs(dir, dir, deps); !ok {
		t.Fatal("post-source-change closure lookup did not succeed")
	}
	if calls != 2 {
		t.Fatalf("go list calls after source import change = %d, want two", calls)
	}
}

func TestClosureCache_IgnoresUnrelatedRepositorySource(t *testing.T) {
	repoRoot := t.TempDir()
	moduleDir := filepath.Join(repoRoot, "scenarios", "demo", "api")
	localPackageDir := filepath.Join(repoRoot, "packages", "shared")
	unrelatedDir := filepath.Join(repoRoot, "scenarios", "other", "api")
	for _, dir := range []string{moduleDir, localPackageDir, unrelatedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module github.com/vrooli/vrooli\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	module := "module example\n\nrequire github.com/vrooli/vrooli v0.0.0\n\nreplace github.com/vrooli/vrooli => ../../..\n"
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(moduleDir, "main.go"),
		filepath.Join(localPackageDir, "shared.go"),
		filepath.Join(unrelatedDir, "other.go"),
	} {
		if err := os.WriteFile(file, []byte("package example\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	payload := []byte(fmt.Sprintf(
		"{\"Dir\":%q,\"Module\":{\"Dir\":%q,\"GoMod\":%q}}\n{\"Dir\":%q,\"Module\":{\"Dir\":%q,\"GoMod\":%q}}\n",
		moduleDir, moduleDir, filepath.Join(moduleDir, "go.mod"),
		localPackageDir, repoRoot, filepath.Join(repoRoot, "go.mod"),
	))
	calls := 0
	deps := hostProbeDeps{
		readFile: os.ReadFile,
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		goListJSON: func(string) ([]byte, error) {
			calls++
			return payload, nil
		},
	}
	if _, ok := goListFreshnessInputs(moduleDir, repoRoot, deps); !ok {
		t.Fatal("initial closure lookup did not succeed")
	}
	if err := os.WriteFile(filepath.Join(unrelatedDir, "other.go"), []byte("package example\n\nfunc changed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := goListFreshnessInputs(moduleDir, repoRoot, deps); !ok {
		t.Fatal("closure lookup after unrelated source change did not succeed")
	}
	if calls != 1 {
		t.Fatalf("go list calls after unrelated source change = %d, want cached closure hit", calls)
	}
}

func TestGoListFreshnessInputs_Fallbacks(t *testing.T) {
	tests := []struct {
		name string
		deps hostProbeDeps
	}{
		{"nil seam", hostProbeDeps{lookPath: func(string) (string, error) { return goExecutablePath, nil }}},
		{"go missing", cannedGoList([]byte(goListFixture), nil, false)},
		{"command error", cannedGoList(nil, errors.New("exit 1"), true)},
		{"empty output", cannedGoList([]byte("  \n"), nil, true)},
		{"malformed stream", cannedGoList([]byte(`{"Dir":"/repo/a"} {bogus`), nil, true)},
		{"no repo-local pkgs", cannedGoList([]byte(`{"Dir":"/usr/lib/go/src/fmt","Standard":true}`), nil, true)},
		{"empty repo root", cannedGoList([]byte(goListFixture), nil, true)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := "/repo"
			if tc.name == "empty repo root" {
				repo = ""
			}
			if inputs, ok := goListFreshnessInputs("/repo/scenarios/x/api", repo, tc.deps); ok {
				t.Fatalf("expected fallback (ok=false), got inputs=%v", inputs)
			}
		})
	}
}

func TestBuilderFreshnessInputsDoesNotRepeatFailedGoListBeforeFallback(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "scenarios", "demo", "api")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.test/demo\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	cache := &hostProbeCache{goToolchain: "go version go1.24 linux/amd64", goToolchainOK: true}
	deps := hostProbeDeps{
		cache:    cache,
		readFile: os.ReadFile,
		walkDir:  filepath.WalkDir,
		lookPath: func(string) (string, error) { return goExecutablePath, nil },
		goListJSON: func(string) ([]byte, error) {
			calls++
			return nil, errors.New("controlled go list failure")
		},
	}

	inputs, err := builderFreshnessInputs(context.Background(), root, moduleDir,
		BuilderSpec{ClosureResolver: closureResolverGoList}, scenario.Component{}, deps)
	if err != nil {
		t.Fatalf("builderFreshnessInputs returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("go list calls = %d, want one failed attempt before static fallback", calls)
	}
	want := []string{"scenarios/demo/api"}
	if !reflect.DeepEqual(inputs, want) {
		t.Fatalf("fallback inputs = %v, want %v", inputs, want)
	}
}

// A package directory beneath the repo root that the binary does NOT import must
// never appear; an out-of-repo (module cache / GOROOT) directory must be dropped.
func TestGoListFreshnessInputs_DropsOutOfRepo(t *testing.T) {
	got, ok := goListFreshnessInputs("/repo/scenarios/x/api", "/repo", cannedGoList([]byte(goListFixture), nil, true))
	if !ok {
		t.Fatal("expected success")
	}
	for _, in := range got {
		if in == "" || in[0] == '/' {
			t.Fatalf("input not repo-relative: %q", in)
		}
	}
}

// TestGoListFreshnessInputs_RealRepo runs the adapter end-to-end with the real
// Go toolchain against an in-repo scenario, proving the headline correctness
// claims on live data: (1) genuinely-imported repo-root-replace packages (under
// packages/) ARE in the input set — the false negative the static fallback has;
// (2) unrelated scenarios are NOT — the false positive the mtime walk had.
// tidiness-manager is used because it is a self-contained in-repo module with
// a complete go.sum; image-tools is an optional media module whose generated
// protobuf dependency is not part of this lifecycle contract.
func TestGoListFreshnessInputs_RealRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real go list in -short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs repo root: %v", err)
	}
	apiDir := filepath.Join(repoRoot, "scenarios", "tidiness-manager", "api")
	if _, err := os.Stat(filepath.Join(apiDir, "go.mod")); err != nil {
		t.Skipf("tidiness-manager api module not present: %v", err)
	}

	inputs, ok := goListFreshnessInputs(apiDir, repoRoot, defaultHostProbeDeps())
	if !ok {
		t.Fatal("expected real go list to resolve the import closure")
	}

	var hasOwnDir, hasPackage bool
	for _, in := range inputs {
		switch {
		case in == "scenarios/tidiness-manager/api":
			hasOwnDir = true
		case strings.HasPrefix(in, "packages/"):
			hasPackage = true
		case strings.HasPrefix(in, "scenarios/") && !strings.HasPrefix(in, "scenarios/tidiness-manager/"):
			t.Errorf("input set leaks an unrelated scenario: %q", in)
		}
	}
	if !hasOwnDir {
		t.Errorf("input set missing the binary's own package dir; got %v", inputs)
	}
	if !hasPackage {
		t.Errorf("input set missing imported packages/* (repo-root-replace false-negative not closed); got %v", inputs)
	}
}

func TestPathUnderRoot(t *testing.T) {
	cases := []struct {
		root, target string
		want         bool
	}{
		{"/repo", "/repo", true},
		{"/repo", "/repo/a/b", true},
		{"/repo", "/repository", false}, // prefix-but-not-subpath guard
		{"/repo", "/other", false},
	}
	for _, c := range cases {
		if got := pathUnderRoot(c.root, c.target); got != c.want {
			t.Errorf("pathUnderRoot(%q,%q)=%v want %v", c.root, c.target, got, c.want)
		}
	}
}
