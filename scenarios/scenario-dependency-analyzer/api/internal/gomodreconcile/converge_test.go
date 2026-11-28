package gomodreconcile

import (
	"context"
	"errors"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The convergence tests run the real go toolchain offline against
// golang.org/x/mod, which every SDA build already holds in the module cache
// (SDA itself requires it).
const fixtureThirdParty = "golang.org/x/mod"

func fixtureThirdPartyVersion(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", fixtureThirdParty).Output()
	if err != nil {
		t.Skipf("resolve %s from the SDA module: %v", fixtureThirdParty, err)
	}
	return strings.TrimSpace(string(out))
}

func offlineGo(t *testing.T) {
	t.Helper()
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOWORK", "off")
}

func mustConverge(t *testing.T, goModPath string, topo Topology) *Convergence {
	t.Helper()
	conv, err := Converge(context.Background(), goModPath, topo)
	if err != nil {
		t.Fatalf("Converge: %v", err)
	}
	return conv
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return string(data)
}

func assertTidy(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy -diff not clean after Write: %v\n%s", err, out)
	}
}

// sharedLeafFixture is the incident shape: a shared in-repo module imports a
// third-party package, and a consumer that reaches it through a local replace
// lacks the requirement and checksums.
func sharedLeafFixture(t *testing.T, version string) (consumerDir string, topo Topology) {
	t.Helper()
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	consumerDir = filepath.Join(root, "scenarios", "demo", "api")
	writeModule(t, leafDir,
		"module example.com/leaf\n\ngo 1.25.0\n\nrequire "+fixtureThirdParty+" "+version+"\n",
		map[string]string{
			"leaf.go": "package leaf\n\nimport \"golang.org/x/mod/module\"\n\ntype V = module.Version\n",
		})
	writeModule(t, consumerDir,
		"module demo/api\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n\nreplace example.com/leaf => ../../../packages/leaf\n",
		map[string]string{
			"main.go": "package main\n\nimport \"example.com/leaf\"\n\nfunc main() { _ = leaf.V{} }\n",
		})
	return consumerDir, Topology{"example.com/leaf": leafDir}
}

func TestConvergeAddsSharedModulesThirdPartyRequirementWithoutTouchingSurface(t *testing.T) {
	offlineGo(t)
	version := fixtureThirdPartyVersion(t)
	consumerDir, topo := sharedLeafFixture(t, version)
	goModPath := filepath.Join(consumerDir, "go.mod")
	before := readFile(t, goModPath)

	conv := mustConverge(t, goModPath, topo)

	if readFile(t, goModPath) != before || readFile(t, filepath.Join(consumerDir, "go.sum")) != "" {
		t.Fatal("Converge must not modify the surface")
	}
	if len(conv.Missing) != 0 {
		t.Fatalf("local replaces were complete, got missing %#v", conv.Missing)
	}
	want := RequirementChange{Path: fixtureThirdParty, To: version}
	if len(conv.Requirements) != 1 || conv.Requirements[0] != want {
		t.Fatalf("requirements = %#v, want [%#v]", conv.Requirements, want)
	}
	if recorded := conv.Recorded(); len(recorded) != 1 || recorded[0].Direct {
		t.Fatalf("recorded = %#v, want one indirect requirement", recorded)
	}
	if !conv.Breaking() || conv.SumAdded == 0 {
		t.Fatalf("missing requirement and checksums must be breaking: %#v", conv)
	}
	if !strings.Contains(conv.GoModAfter, "replace example.com/leaf => ../../../packages/leaf") {
		t.Fatalf("local replace must survive convergence:\n%s", conv.GoModAfter)
	}
	if !strings.Contains(conv.GoSumAfter, fixtureThirdParty+" "+version+" h1:") {
		t.Fatalf("go.sum must gain the requirement's checksums:\n%s", conv.GoSumAfter)
	}
	if strings.Contains(conv.GoSumAfter, "example.com/leaf") {
		t.Fatalf("go.sum must not carry the replaced in-repo module:\n%s", conv.GoSumAfter)
	}
}

func TestConvergenceWriteInstallsTidyStateAndIsIdempotent(t *testing.T) {
	offlineGo(t)
	consumerDir, topo := sharedLeafFixture(t, fixtureThirdPartyVersion(t))
	goModPath := filepath.Join(consumerDir, "go.mod")

	conv := mustConverge(t, goModPath, topo)
	if err := conv.Write(); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if readFile(t, goModPath) != conv.GoModAfter || readFile(t, filepath.Join(consumerDir, "go.sum")) != conv.GoSumAfter {
		t.Fatal("Write must install exactly the previewed files")
	}
	assertTidy(t, consumerDir)

	if again := mustConverge(t, goModPath, topo); again.Changed() {
		t.Fatalf("a converged surface must report no change, got %#v", again)
	}
}

func TestConvergenceWriteRefusesConcurrentEdit(t *testing.T) {
	offlineGo(t)
	consumerDir, topo := sharedLeafFixture(t, fixtureThirdPartyVersion(t))
	goModPath := filepath.Join(consumerDir, "go.mod")
	conv := mustConverge(t, goModPath, topo)

	edited := conv.GoModBefore + "\n// edited concurrently\n"
	if err := os.WriteFile(goModPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := conv.Write(); !errors.Is(err, ErrConcurrentEdit) {
		t.Fatalf("Write = %v, want ErrConcurrentEdit", err)
	}
	if readFile(t, goModPath) != edited {
		t.Fatal("a refused Write must leave the concurrent edit in place")
	}
}

func TestConvergeMarksConsumersOwnImportDirect(t *testing.T) {
	offlineGo(t)
	version := fixtureThirdPartyVersion(t)
	consumerDir, topo := sharedLeafFixture(t, version)
	writeModule(t, consumerDir, readFile(t, filepath.Join(consumerDir, "go.mod")), map[string]string{
		"direct.go": "package main\n\nimport \"golang.org/x/mod/semver\"\n\nvar _ = semver.IsValid\n",
	})

	conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
	want := RequirementChange{Path: fixtureThirdParty, To: version, Direct: true}
	if len(conv.Requirements) != 1 || conv.Requirements[0] != want {
		t.Fatalf("requirements = %#v, want [%#v]", conv.Requirements, want)
	}
}

func TestConvergeAddsReplaceAndRequireForImportedInRepoModule(t *testing.T) {
	offlineGo(t)
	root := t.TempDir()
	apiDir := filepath.Join(root, "scenarios", "demo", "api")
	cliDir := filepath.Join(root, "scenarios", "demo", "cli")
	writeModule(t, apiDir, "module demo\n\ngo 1.25.0\n", map[string]string{
		filepath.Join("models", "model.go"): "package models\n\ntype Model struct{}\n",
	})
	writeModule(t, cliDir, "module demo/cli\n\ngo 1.25.0\n", map[string]string{
		"main.go": "package main\n\nimport \"demo/models\"\n\nfunc main() { _ = models.Model{} }\n",
	})
	goModPath := filepath.Join(cliDir, "go.mod")

	conv := mustConverge(t, goModPath, Topology{"demo": apiDir, "demo/cli": cliDir})
	if len(conv.Missing) != 1 || !conv.Missing[0].AddRequire || conv.Missing[0].RelPath != "../api" {
		t.Fatalf("missing = %#v", conv.Missing)
	}
	if len(conv.Requirements) != 0 {
		t.Fatalf("in-repo wiring is not a third-party requirement change: %#v", conv.Requirements)
	}
	if !strings.Contains(conv.GoModAfter, "demo v0.0.0") || !strings.Contains(conv.GoModAfter, "replace demo => ../api") {
		t.Fatalf("converged go.mod lacks require/replace:\n%s", conv.GoModAfter)
	}
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	assertTidy(t, cliDir)
}

func TestConvergeCanonicalizesGoSumOrderAsHousekeeping(t *testing.T) {
	offlineGo(t)
	consumerDir, topo := sharedLeafFixture(t, fixtureThirdPartyVersion(t))
	goModPath := filepath.Join(consumerDir, "go.mod")
	if err := mustConverge(t, goModPath, topo).Write(); err != nil {
		t.Fatal(err)
	}
	sumPath := filepath.Join(consumerDir, "go.sum")
	lines := strings.Split(strings.TrimSpace(readFile(t, sumPath)), "\n")
	if len(lines) < 2 {
		t.Fatalf("fixture go.sum too small to reorder: %q", lines)
	}
	reversed := make([]string, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		reversed = append(reversed, lines[i])
	}
	if err := os.WriteFile(sumPath, []byte(strings.Join(reversed, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	conv := mustConverge(t, goModPath, topo)
	if conv.ModChanged() || !conv.SumChanged() {
		t.Fatalf("a reordered go.sum must converge go.sum only: %#v", conv)
	}
	if conv.Breaking() || conv.SumAdded != 0 {
		t.Fatalf("reordering is housekeeping, not breaking: %#v", conv)
	}
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	assertTidy(t, consumerDir)
}

func TestRequirementChangeRecordedOnlyForNewVersionsOrDirectUse(t *testing.T) {
	for _, tc := range []struct {
		change RequirementChange
		want   bool
	}{
		{RequirementChange{Path: "m", To: "v1.0.0"}, true},
		{RequirementChange{Path: "m", From: "v1.0.0", To: "v1.1.0"}, true},
		{RequirementChange{Path: "m", From: "v1.0.0", To: "v1.0.0", Direct: true}, true},
		{RequirementChange{Path: "m", From: "v1.0.0", To: "v1.0.0", FromDirect: true}, false},
		{RequirementChange{Path: "m", From: "v1.0.0"}, false},
	} {
		if got := tc.change.Recorded(); got != tc.want {
			t.Errorf("%s: Recorded() = %t, want %t", tc.change, got, tc.want)
		}
	}
}

// inRepoChainFixture: the consumer requires packages/parent, whose go.mod
// requires packages/leaf. useLeaf decides whether the consumer's import closure
// actually reaches leaf's packages.
func inRepoChainFixture(t *testing.T, useLeaf bool, consumerExtra string) (consumerDir string, topo Topology) {
	t.Helper()
	root := t.TempDir()
	leafDir := filepath.Join(root, "packages", "leaf")
	parentDir := filepath.Join(root, "packages", "parent")
	consumerDir = filepath.Join(root, "scenarios", "demo", "api")
	writeModule(t, leafDir, "module example.com/leaf\n\ngo 1.25.0\n", map[string]string{"leaf.go": "package leaf\n\nfunc F() {}\n"})
	parentFiles := map[string]string{"parent.go": "package parent\n\nfunc G() {}\n"}
	if useLeaf {
		parentFiles["parent.go"] = "package parent\n\nimport \"example.com/leaf\"\n\nfunc G() { leaf.F() }\n"
	} else {
		// leaf is used only by a parent package the consumer never imports.
		parentFiles["uses/uses.go"] = "package uses\n\nimport \"example.com/leaf\"\n\nfunc H() { leaf.F() }\n"
	}
	writeModule(t, parentDir, "module example.com/parent\n\ngo 1.25.0\n\nrequire example.com/leaf v0.0.0\n\nreplace example.com/leaf => ../leaf\n", parentFiles)
	writeModule(t, consumerDir,
		"module demo/api\n\ngo 1.25.0\n\nrequire example.com/parent v0.0.0\n\nreplace example.com/parent => ../../../packages/parent\n"+consumerExtra,
		map[string]string{"main.go": "package main\n\nimport \"example.com/parent\"\n\nfunc main() { parent.G() }\n"})
	return consumerDir, Topology{"example.com/parent": parentDir, "example.com/leaf": leafDir}
}

func TestConvergeReplacesOnlyInRepoModulesTheImportClosureLoads(t *testing.T) {
	offlineGo(t)
	t.Run("leaf in the import closure", func(t *testing.T) {
		consumerDir, topo := inRepoChainFixture(t, true, "")
		conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
		if !strings.Contains(conv.GoModAfter, "example.com/leaf v0.0.0 // indirect") || !strings.Contains(conv.GoModAfter, "replace example.com/leaf => ../../../packages/leaf") {
			t.Fatalf("leaf provides a package in the closure; it needs a requirement and replace:\n%s", conv.GoModAfter)
		}
		if err := conv.Write(); err != nil {
			t.Fatal(err)
		}
		assertTidy(t, consumerDir)
	})
	t.Run("leaf only in the pruned graph", func(t *testing.T) {
		consumerDir, topo := inRepoChainFixture(t, false, "")
		conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
		if conv.Changed() {
			t.Fatalf("leaf is never loaded; the consumer is already converged:\n%s", conv.GoModAfter)
		}
	})
}

func TestConvergeDropsStaleInRepoReplace(t *testing.T) {
	offlineGo(t)
	consumerDir, topo := inRepoChainFixture(t, false, "\nreplace example.com/leaf => ../../../packages/leaf\n")
	conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
	if len(conv.Dropped) != 1 || conv.Dropped[0] != "example.com/leaf" {
		t.Fatalf("dropped = %#v, want the unused leaf replace", conv.Dropped)
	}
	if strings.Contains(conv.GoModAfter, "example.com/leaf") || !strings.Contains(conv.GoModAfter, "replace example.com/parent => ../../../packages/parent") {
		t.Fatalf("only the stale replace may go:\n%s", conv.GoModAfter)
	}
	if conv.Breaking() {
		t.Fatal("dropping a dead replace is housekeeping")
	}
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	assertTidy(t, consumerDir)
}

// A retained source copy under docs/ without its own go.mod compiles as part of
// the enclosing module; its imports must not leak into that module's go.mod.
func TestConvergeRefusesModuleWithUnfencedDocsEvidenceAndHonorsFence(t *testing.T) {
	offlineGo(t)
	consumerDir, topo := inRepoChainFixture(t, false, "")
	evidence := filepath.Join(consumerDir, "docs", "evidence", "E1", "copy")
	if err := os.MkdirAll(evidence, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumerDir, "docs", "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "copy.go"), []byte("package copy\n\nimport \"example.com/leaf\"\n\nvar _ = leaf.F\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	goModPath := filepath.Join(consumerDir, "go.mod")

	conv := mustConverge(t, goModPath, topo)
	if len(conv.Evidence) != 1 || conv.Evidence[0] != "docs/evidence/E1/copy" || conv.Changed() {
		t.Fatalf("unfenced evidence must be reported and nothing converged: %#v", conv)
	}

	// The docs-evidence fence: a go.mod at the top of docs/.
	if err := os.WriteFile(filepath.Join(consumerDir, "docs", "go.mod"), []byte("module demo/api/docs-evidence\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fenced := mustConverge(t, goModPath, topo)
	if len(fenced.Evidence) != 0 || fenced.Changed() {
		t.Fatalf("a fenced evidence tree is its own module and leaves the parent converged: %#v", fenced)
	}
}

func TestModuleSourceWalkFollowsGoPackageRules(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/root\n")
	write("cmd/main.go", "package main\n")
	write("internal/linux.go", "//go:build linux\n\npackage internal\n")
	write("internal/not_windows.go", "//go:build !windows\n\npackage internal\n")
	for _, ignored := range []string{
		".vrooli/work/evidence/copy.go", "_scratch/x.go", "pkg/testdata/x.go", "vendor/x/x.go",
		"nested/x.go", "tools/_gen.go", "tools/.hidden.go",
	} {
		write(ignored, "package x\n")
	}
	write("nested/go.mod", "module example.com/nested\n")
	write("tools/gen.go", "//go:build ignore\n\npackage main\n")

	var got []string
	if err := walkModuleGoFiles(root, func(path string, _ *ast.File) {
		rel, _ := filepath.Rel(root, path)
		got = append(got, filepath.ToSlash(rel))
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"cmd/main.go", "internal/linux.go", "internal/not_windows.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("module Go files = %v, want %v", got, want)
	}
}

func TestUnfencedEvidenceIgnoresProductionPackagesNamedDocs(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "module example.com/cli\n", map[string]string{
		"domains/docs/register.go":   "package docs\n",
		"handlers/docs/docs.go":      "package docs\n",
		"docs/README.md":             "# docs\n",
		"docs/evidence/copy/copy.go": "package copy\n",
	})
	got, err := UnfencedEvidence(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "docs/evidence/copy" {
		t.Fatalf("evidence = %v, want only the documentation tree copy", got)
	}
}

// vendoredFixture is a converged consumer of the shared leaf with a committed
// vendor/ tree that also holds a non-module input (like packages/proto's
// vendored googleapis sources).
func vendoredFixture(t *testing.T) (consumerDir string, topo Topology) {
	t.Helper()
	offlineGo(t)
	consumerDir, topo = sharedLeafFixture(t, fixtureThirdPartyVersion(t))
	conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	conv.Close()
	runIn(t, consumerDir, "go", "mod", "vendor")
	writeVendorInput(t, consumerDir)
	return consumerDir, topo
}

// writeVendorInput adds a vendor/ entry the go command does not own.
func writeVendorInput(t *testing.T, consumerDir string) {
	t.Helper()
	dir := filepath.Join(consumerDir, "vendor", "googleapis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "annotations.proto"), []byte("syntax = \"proto3\";\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runIn(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func assertVendorConsistent(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "list", "-mod=vendor", "-buildvcs=false", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=vendor -buildvcs=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("vendor tree inconsistent with go.mod: %v\n%s", err, out)
	}
}

func TestConvergeVendoredModuleIsAlreadyConverged(t *testing.T) {
	consumerDir, topo := vendoredFixture(t)
	conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
	defer conv.Close()
	if conv.Changed() {
		t.Fatalf("a freshly vendored converged module must not change: vendor=%t", conv.VendorChanged())
	}
	assertVendorConsistent(t, consumerDir)
}

// Dropping a dead replace changes what vendor/modules.txt must record; the
// vendor tree must move with go.mod in the same Write.
func TestConvergeRevendorsWhenDroppingDeadReplaceAndPreservesNonModuleInputs(t *testing.T) {
	consumerDir, topo := vendoredFixture(t)
	goModPath := filepath.Join(consumerDir, "go.mod")
	deadDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(consumerDir))), "packages", "dead")
	writeModule(t, deadDir, "module example.com/dead\n\ngo 1.25.0\n", nil)
	topo["example.com/dead"] = deadDir
	runIn(t, consumerDir, "go", "mod", "edit", "-replace=example.com/dead=../../../packages/dead")
	runIn(t, consumerDir, "go", "mod", "vendor")
	writeVendorInput(t, consumerDir)

	conv := mustConverge(t, goModPath, topo)
	defer conv.Close()
	if len(conv.Dropped) != 1 || !conv.VendorChanged() {
		t.Fatalf("dropping the dead replace must re-vendor: dropped=%v vendor=%t", conv.Dropped, conv.VendorChanged())
	}
	if before, after := conv.VendorModules(); !strings.Contains(before, "example.com/dead") || strings.Contains(after, "example.com/dead") {
		t.Fatalf("modules.txt must drop the replace:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if got := conv.VendorPreserved(); len(got) != 1 || got[0] != "googleapis" {
		t.Fatalf("preserved = %v, want the non-module googleapis input", got)
	}
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	assertVendorConsistent(t, consumerDir)
	assertTidy(t, consumerDir)
	if _, err := os.Stat(filepath.Join(consumerDir, "vendor", "googleapis", "annotations.proto")); err != nil {
		t.Fatalf("non-module vendor input must survive re-vendoring: %v", err)
	}
	if again := mustConverge(t, goModPath, topo); again.Changed() {
		again.Close()
		t.Fatal("re-vendored module must be converged")
	}
	if entries, _ := filepath.Glob(filepath.Join(consumerDir, ".vendor-reconcile-*")); len(entries) != 0 {
		t.Fatalf("staging directories must be cleaned up: %v", entries)
	}
}

// A vendored copy of a locally replaced in-repo module goes stale when that
// module changes, even though go.mod does not.
func TestConvergeRevendorsStaleCopyOfInRepoModule(t *testing.T) {
	consumerDir, topo := vendoredFixture(t)
	leafDir := topo["example.com/leaf"]
	if err := os.WriteFile(filepath.Join(leafDir, "extra.go"), []byte("package leaf\n\n// Extra is new shared code.\nfunc Extra() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conv := mustConverge(t, filepath.Join(consumerDir, "go.mod"), topo)
	defer conv.Close()
	if conv.ModChanged() || conv.SumChanged() || !conv.VendorChanged() {
		t.Fatalf("only the vendor tree is stale: mod=%t sum=%t vendor=%t", conv.ModChanged(), conv.SumChanged(), conv.VendorChanged())
	}
	if err := conv.Write(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(consumerDir, "vendor", "example.com", "leaf", "extra.go")); err != nil {
		t.Fatalf("vendored leaf must carry the new file: %v", err)
	}
}
