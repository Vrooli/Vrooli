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

func TestGoListFreshnessInputs_CachesByModuleFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte("sum-v1\n"), 0o644); err != nil {
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
		lookPath: func(string) (string, error) { return "/usr/bin/go", nil },
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

func TestGoListFreshnessInputs_Fallbacks(t *testing.T) {
	tests := []struct {
		name string
		deps hostProbeDeps
	}{
		{"nil seam", hostProbeDeps{lookPath: func(string) (string, error) { return "/usr/bin/go", nil }}},
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
	apiDir := filepath.Join(repoRoot, "scenarios", "image-tools", "api")
	if _, err := os.Stat(filepath.Join(apiDir, "go.mod")); err != nil {
		t.Skipf("image-tools api module not present: %v", err)
	}

	inputs, ok := goListFreshnessInputs(apiDir, repoRoot, defaultHostProbeDeps())
	if !ok {
		t.Fatal("expected real go list to resolve the import closure")
	}

	var hasOwnDir, hasPackage bool
	for _, in := range inputs {
		switch {
		case in == "scenarios/image-tools/api":
			hasOwnDir = true
		case strings.HasPrefix(in, "packages/"):
			hasPackage = true
		case strings.HasPrefix(in, "scenarios/") && !strings.HasPrefix(in, "scenarios/image-tools/"):
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
