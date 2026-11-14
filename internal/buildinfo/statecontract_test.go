package buildinfo

import (
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestPersistentStateBuildFlagStampsTheWholeCodecInventory(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	flag, err := PersistentStateBuildFlag(root)
	if err != nil {
		t.Fatal(err)
	}
	stamp, ok := strings.CutPrefix(flag, "-X github.com/vrooli/vrooli/internal/statecontract.CodecStamp=")
	if !ok || strings.ContainsAny(stamp, " \t\n'\"") {
		t.Fatalf("build flag is not a single linker assignment: %q", flag)
	}
	again, err := PersistentStateCodecStamp(root)
	if err != nil || again != stamp {
		t.Fatal("codec stamp is not deterministic")
	}
	for _, dir := range statecontract.CodecPackages {
		importPath, err := sourceImportPath(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stamp, importPath) {
			t.Fatalf("codec inventory package %s (%s) is missing from the stamp", dir, importPath)
		}
	}
	for owner := range statecontract.WholeSources {
		if !strings.Contains(stamp, owner+":") {
			t.Fatalf("whole-source owner %s is missing from the stamp", owner)
		}
	}
	for owner := range statecontract.ConstantSources {
		if !strings.Contains(stamp, owner+":") {
			t.Fatalf("constants-stamped owner %s is missing from the stamp", owner)
		}
	}
	if _, err := PersistentStateBuildFlag("relative/root"); err == nil {
		t.Fatal("a non-canonical source root was accepted")
	}
}

func TestSourceImportPathFollowsTheNearestModule(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/root\n\ngo 1.26\n")
	writeTestFile(t, root, "packages/nested/go.mod", "module example.com/nested\n")
	writeTestFile(t, root, "packages/nested/sub/x.go", "package sub\n")
	writeTestFile(t, root, "packages/plain/x.go", "package plain\n")
	for dir, want := range map[string]string{
		"packages/nested":     "example.com/nested",
		"packages/nested/sub": "example.com/nested/sub",
		"packages/plain":      "example.com/root/packages/plain",
	} {
		got, err := sourceImportPath(root, dir)
		if err != nil || got != want {
			t.Fatalf("sourceImportPath(%s) = %q, %v; want %q", dir, got, err, want)
		}
	}
	if _, err := sourceImportPath(filepath.Join(root, "packages", "plain"), "."); err == nil {
		t.Fatal("a directory outside any module resolved")
	}
}

// writeStateContractFixture gives a fixture checkout one source per codec
// inventory package so governed build paths can stamp it.
func writeStateContractFixture(t *testing.T, root string) {
	t.Helper()
	for _, dir := range statecontract.CodecPackages {
		writeTestFile(t, root, dir+"/fixture.go", "package fixture\n\ntype Phase string\n\nconst Ready Phase = \"ready\"\n")
	}
	for _, dirs := range statecontract.PreciseSources {
		for _, dir := range dirs {
			writeTestFile(t, root, dir+"/fixture.go", "package fixture\n\ntype Phase string\n\nconst Ready Phase = \"ready\"\n")
		}
	}
	for _, patterns := range statecontract.WholeSources {
		for _, pattern := range patterns {
			path := pattern + "/fixture.go"
			if strings.HasSuffix(pattern, "*") {
				path = strings.TrimSuffix(pattern, "*") + "_fixture.go"
			}
			writeTestFile(t, root, path, "package fixture\n")
		}
	}
}

func mustStateBuildFlag(t *testing.T, root string) string {
	t.Helper()
	flag, err := PersistentStateBuildFlag(root)
	if err != nil {
		t.Fatal(err)
	}
	return flag
}

func TestWholeSourceDigestCoversBodiesButNotComments(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pkg/operation.go", "package pkg\n\nfunc slice() string { return \"vrooli-operations.slice\" }\n")
	writeTestFile(t, root, "pkg/other.go", "package pkg\n\nfunc other() int { return 1 }\n")
	digest := func() string {
		t.Helper()
		d, err := wholeSourceDigest(root, []string{"pkg/operation*"})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	base := digest()
	writeTestFile(t, root, "pkg/operation.go", "package pkg\n\n// slice names the native slice.\nfunc slice() string {\n\treturn \"vrooli-operations.slice\"\n}\n")
	writeTestFile(t, root, "pkg/other.go", "package pkg\n\nfunc other() int { return 2 }\n")
	writeTestFile(t, root, "pkg/operation_test.go", "package pkg\n")
	if digest() != base {
		t.Fatal("comments, layout, tests or unselected files changed a whole-source digest")
	}
	writeTestFile(t, root, "pkg/operation.go", "package pkg\n\nfunc slice() string { return \"vrooli-svc.slice\" }\n")
	if digest() == base {
		t.Fatal("a function-body literal change kept the whole-source digest")
	}
	if _, err := wholeSourceDigest(root, []string{"pkg/missing*"}); err == nil {
		t.Fatal("a whole-source pattern matching nothing was accepted")
	}
}

func TestPackageConstantsDigestCoversUntypedConstantsButNotBodies(t *testing.T) {
	root := t.TempDir()
	write := func(src string) string {
		t.Helper()
		writeTestFile(t, root, "pkg/types.go", src)
		d, err := packageConstantsDigest(root, []string{"pkg"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	base := write("package pkg\n\nconst (\n\tStatusRunning = \"running\"\n\tStatusStopping = \"stopping\"\n)\n\nfunc f() string { return \"a\" }\n")
	if write("package pkg\n\n// statuses\nconst (\n\tStatusRunning  = \"running\" // live\n\tStatusStopping = \"stopping\"\n)\n\nfunc f() string { return \"b\" }\n") != base {
		t.Fatal("comments, layout or a function body changed the constants digest")
	}
	for name, src := range map[string]string{
		"changed value": "package pkg\n\nconst (\n\tStatusRunning = \"running\"\n\tStatusStopping = \"draining\"\n)\n",
		"added value":   "package pkg\n\nconst (\n\tStatusRunning = \"running\"\n\tStatusStopping = \"stopping\"\n\tStatusPaused = \"paused\"\n)\n",
	} {
		if write(src) == base {
			t.Fatalf("%s kept the constants digest", name)
		}
	}
}

func TestCapacityConstantsExcludeOnlyNonpersistentResidencyEvidence(t *testing.T) {
	root := t.TempDir()
	const dir = "internal/capacity"
	const baseline = `package capacity
import "time"
const (
 FootprintSourceManifestDefault = "manifest-default"
 FootprintSourceMeasured = "measured"
)
const StatusGranted = "granted"
`
	const residency = `package capacity
import "time"
const (
 FootprintSourceManifestDefault = "manifest-default"
 FootprintSourceMeasured = "measured"
 ResidencyBackendCPU = "cpu"
 ResidencyBackendCUDA = "cuda"
 ResidencyBackendMetal = "metal"
 ResidencyBackendROCm = "rocm"
 ResidencyBackendVulkan = "vulkan"
 ResidencyBackendUnloaded = "unloaded"
)
const StatusGranted = "granted"
const maxResidencyFootprintAge = 2 * time.Minute
`
	digest := func(source string) string {
		t.Helper()
		writeTestFile(t, root, dir+"/types.go", source)
		value, err := packageConstantsDigest(root, []string{dir}, statecontract.NotStoredConstants)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest(baseline)
	if digest(residency) != before {
		t.Fatal("in-memory residency vocabulary or freshness tunable changed the stored capacity contract")
	}
	if digest(strings.Replace(residency, "2 * time.Minute", "3 * time.Minute", 1)) != before {
		t.Fatal("nonpersistent freshness tuning changed the stored capacity contract")
	}
	for name, source := range map[string]string{
		"existing stored footprint source": strings.Replace(residency, `"measured"`, `"measured-v2"`, 1),
		"existing stored status":           strings.Replace(residency, `"granted"`, `"granted-v2"`, 1),
		"new unclassified stored enum":     residency + `const StatusPaused = "paused"` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if digest(source) == before {
				t.Fatal("stored or unclassified vocabulary change escaped the contract")
			}
		})
	}
}

func TestOwnerAnalysisClassifiesStoredValuesAndGatesVersionsAndLayouts(t *testing.T) {
	root := t.TempDir()
	const base = "package pkg\n\nimport \"time\"\n\nconst recordVersion = 2\n\nconst (\n\tStatusRunning = \"running\"\n\tenvName = \"VROOLI_X\"\n)\n\ntype Record struct {\n\tVersion int `json:\"version\"`\n\tSchemaVersion string `json:\"schema_version\"`\n\tArtifactVersion string `json:\"artifact_version\"`\n\tAt string `json:\"at\"`\n}\n\nfunc write(t time.Time, v string) Record {\n\treturn Record{Version: recordVersion, SchemaVersion: \"\", ArtifactVersion: v, At: t.Format(time.RFC3339)}\n}\n"
	writeTestFile(t, root, "go.mod", "module example.test/owner\n\ngo 1.22\n")
	analyse := func(source string) ownerAnalysis {
		t.Helper()
		writeTestFile(t, root, "pkg/record.go", source)
		analysis, err := analyzeOwnerPackages(root, []string{"pkg"}, map[string][]string{"pkg": {"envName"}}, true)
		if err != nil {
			t.Fatal(err)
		}
		return analysis
	}
	got := analyse(base)
	if len(got.violations) != 0 || strings.Join(got.versions, ",") != "recordVersion" || len(got.values) != 1 || got.values["StatusRunning"] == "" {
		t.Fatalf("base analysis = %+v", got)
	}
	for name, tc := range map[string]struct {
		source    string
		violation bool
	}{
		"version through a variable": {strings.Replace(base, "Version: recordVersion", "Version: storedVersion", 1) + "\nvar storedVersion = recordVersion + 1\n", true},
		"version expression":         {strings.Replace(base, "Version: recordVersion", "Version: recordVersion + 1", 1), true},
		"version literal":            {strings.Replace(base, "Version: recordVersion", "Version: 3", 1), true},
		"positional literal":         {base + "\nvar positional = Record{4, \"\", \"\", \"\"}\n", true},
		"artifact version data":      {strings.Replace(base, "ArtifactVersion: v", "ArtifactVersion: v + \"-1\"", 1), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := analyse(tc.source); (len(got.violations) > 0) != tc.violation {
				t.Fatalf("violations = %v, want violation=%t", got.violations, tc.violation)
			}
		})
	}
	layout := analyse(base).layoutDigest()
	if analyse(strings.Replace(base, "time.RFC3339", "\"2006-01-02 15:04:05 MST\"", 1)).layoutDigest() == layout {
		t.Fatal("a changed stored time layout kept the layout digest")
	}
	if analyse(base+"\nfunc helper() int { return 1 }\n").layoutDigest() != layout {
		t.Fatal("a behaviour-only edit changed the layout digest")
	}
	added := analyse(strings.Replace(base, "\tenvName = \"VROOLI_X\"\n", "\tenvName = \"VROOLI_X\"\n\tStatusPaused = \"paused\"\n", 1))
	if added.values["StatusPaused"] == "" {
		t.Fatal("a new unclassified untyped string constant was not treated as stored")
	}
}

func TestEveryPersistenceOwnerPackagePassesItsSourceChecks(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, inventory := range []map[string][]string{statecontract.PreciseSources, statecontract.ConstantSources} {
		for _, dirs := range inventory {
			for _, dir := range dirs {
				findings, err := PersistentStateSourceFindings(root, dir)
				if err != nil {
					t.Fatal(err)
				}
				for _, finding := range findings {
					t.Errorf("%s: %s", dir, finding)
				}
			}
		}
	}
}

// A stored vocabulary word (a state an older reader branches on) written as a
// literal is a candidate; a free-text field, a field of another package's type
// and a named constant are not.
func TestOwnerAnalysisFindsStoredVocabularyWrittenAsLiterals(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/owner\n\ngo 1.22\n")
	writeTestFile(t, root, "other/other.go", "package other\n\ntype Step struct{ Kind string }\n")
	const base = `package pkg

import "example.test/owner/other"

const fencePrepared = "prepared"

type Fence struct {
	State  string ` + "`json:\"state\"`" + `
	Reason string ` + "`json:\"reason\"`" + `
}

func (f Fence) Validate() bool {
	switch f.State {
	case fencePrepared:
		return true
	}
	return false
}

func prepare() Fence { return Fence{State: fencePrepared, Reason: "operator asked"} }

func steps(target []other.Step) bool {
	for _, step := range target {
		if step.Kind == "url" {
			return true
		}
	}
	return false
}
`
	candidates := func(source string) []string {
		t.Helper()
		writeTestFile(t, root, "pkg/fence.go", source)
		analysis, err := analyzeOwnerPackages(root, []string{"pkg"}, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		return analysis.literals
	}
	if got := candidates(base); len(got) != 0 {
		t.Fatalf("named constants, free text and another package's field were candidates: %v", got)
	}
	for name, source := range map[string]string{
		"composite literal": strings.Replace(base, "Fence{State: fencePrepared", `Fence{State: "prepared"`, 1),
		"assignment":        base + "\nfunc claim(f *Fence) { f.State = \"claimed\" }\n",
		"comparison":        base + "\nfunc claimed(f Fence) bool { return f.State == \"claimed\" }\n",
		"switch case":       strings.Replace(base, "case fencePrepared:", `case "prepared":`, 1),
		"local variable":    base + "\nfunc claimLocal(f *Fence) {\n\tstate := \"claimed\"\n\tf.State = state\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			got := candidates(source)
			if len(got) != 1 || !strings.HasPrefix(got[0], "example.test/owner/pkg.Fence.State@pkg/fence.go:") {
				t.Fatalf("candidates = %v, want the Fence.State literal", got)
			}
		})
	}
}
