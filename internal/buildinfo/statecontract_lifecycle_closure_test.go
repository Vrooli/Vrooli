package buildinfo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

// These are representative producers and consumers, not a complete proof of
// lifecycle persistence closure. A precise split must replace this conservative
// guard with compatibility evidence for all durable records and old readers.
func TestLifecyclePersistentSemanticsRemainCovered(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ path, function string }{
		{"freshness.go", "recordInstallDigest"},
		{"freshness.go", "installInputsDigest"},
		{"shared_packages.go", "readSharedPackageStamp"},
		{"shared_packages.go", "writeSharedPackageStampWithSnapshot"},
		{"runtime_registry.go", "beginRuntimeRegistryStart"},
		{"operation_record.go", "linkProviderFence"},
		{"provider_fence.go", "resume"},
		{"retained_runtime.go", "CaptureRuntimeRelease"},
		{"retained_runtime.go", "VerifyRuntimeRelease"},
		{"retained_runtime.go", "bindRetainedRuntime"},
	}
	for _, tc := range cases {
		t.Run(tc.path+"/"+tc.function, func(t *testing.T) {
			path := filepath.Join("internal/lifecycle", tc.path)
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, data, 0)
			if err != nil {
				t.Fatal(err)
			}
			offset := -1
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if ok && fn.Name.Name == tc.function && fn.Body != nil {
					offset = fset.Position(fn.Body.Lbrace).Offset + 1
					break
				}
			}
			if offset < 0 {
				t.Fatalf("function %s missing", tc.function)
			}
			fixture := t.TempDir()
			writeTestFile(t, fixture, path, string(data))
			writeTestFile(t, fixture, "internal/lifecycle/scenario_locator.go", "package lifecycle\n")
			digest := func() string {
				t.Helper()
				d, err := wholeSourceDigest(fixture, statecontract.WholeSources["lifecycle-cache"])
				if err != nil {
					t.Fatal(err)
				}
				return d
			}
			before := digest()
			changed := string(data[:offset]) + "\n_ = \"persistent semantics mutation\"\n" + string(data[offset:])
			writeTestFile(t, fixture, path, changed)
			if digest() == before {
				t.Fatalf("%s semantics escaped lifecycle contract", tc.function)
			}
		})
	}
}

// A newer writer's snapshot vocabulary must change the rollback contract.
func TestLifecycleSharedPackageSnapshotVersionAndPatternsRemainCovered(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := "internal/lifecycle/shared_packages.go"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, before, after string }{
		{"content version", "const sharedPackageOutputContentDigestVersion = 3", "const sharedPackageOutputContentDigestVersion = 4"},
		{"pattern field", `json:"output_patterns_digest,omitempty"`, `json:"output_pattern_identity,omitempty"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(string(data), tc.before, tc.after, 1)
			if changed == string(data) {
				t.Fatal("mutation anchor missing")
			}
			fixture := t.TempDir()
			writeTestFile(t, fixture, path, string(data))
			writeTestFile(t, fixture, "internal/lifecycle/scenario_locator.go", "package lifecycle\n")
			digest := func() string {
				t.Helper()
				got, err := wholeSourceDigest(fixture, statecontract.WholeSources["lifecycle-cache"])
				if err != nil {
					t.Fatal(err)
				}
				return got
			}
			before := digest()
			writeTestFile(t, fixture, path, changed)
			if digest() == before {
				t.Fatal("snapshot contract mutation escaped lifecycle owner")
			}
		})
	}
}
