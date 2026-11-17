package buildinfo

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

// The whole-source policy owner must include every package-local function it
// calls; otherwise moving an encoder helper outside the stamped file silently
// drops that helper from the contract.
func TestCapacityPolicyStorageLocalCallsRemainStamped(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	sources, err := loadGoSources(fset, filepath.Join(root, "internal/capacity"), "")
	if err != nil {
		t.Fatal(err)
	}
	funcFiles := map[string]map[string]bool{}
	varFiles := map[string]string{}
	for _, source := range sources {
		for _, decl := range source.file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
				for _, spec := range gen.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						varFiles[name.Name] = filepath.Base(source.path)
					}
				}
			}
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if funcFiles[fn.Name.Name] == nil {
					funcFiles[fn.Name.Name] = map[string]bool{}
				}
				funcFiles[fn.Name.Name][filepath.Base(source.path)] = true
			}
		}
	}
	for _, source := range sources {
		if filepath.Base(source.path) != "policy_storage.go" {
			continue
		}
		for _, decl := range source.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if _, ok := node.(*ast.SelectorExpr); ok {
					return false
				}
				if id, ok := node.(*ast.Ident); ok && id.Name != "_" && id.Name != "ErrInvalidClaim" {
					if file := varFiles[id.Name]; file != "" && file != "policy_storage.go" {
						t.Errorf("policy codec %s references variable %s outside stamped source %s", fn.Name.Name, id.Name, file)
					}
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch target := call.Fun.(type) {
				case *ast.Ident:
					name = target.Name
				case *ast.SelectorExpr:
					if receiver, ok := target.X.(*ast.Ident); ok && receiver.Name == "p" {
						name = target.Sel.Name
					}
				}
				if files := funcFiles[name]; len(files) > 0 && !files["policy_storage.go"] && name != "formatTime" {
					t.Errorf("policy codec %s calls %s outside its stamped source: %v", fn.Name.Name, name, files)
				}
				return true
			})
		}
	}
}

func TestBreakGlassStoredFormatChangesItsOwnerDigest(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	paths := []string{"packages/api-core/trustposture/breakglass.go", "packages/proto/privilegedops/operations.go"}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, fixture, path, string(data))
	}
	digest := func() string {
		t.Helper()
		value, err := wholeSourceDigest(fixture, statecontract.WholeSources["break-glass"])
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest()
	writeTestFile(t, fixture, "packages/api-core/trustposture/posture.go", "package trustposture\nconst unrelated = 1\n")
	if digest() != before {
		t.Fatal("unrelated posture implementation changed the break-glass contract")
	}
	data, err := os.ReadFile(filepath.Join(fixture, paths[0]))
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), "json:\"ciphertext\"", "json:\"encrypted\"", 1)
	if changed == string(data) {
		t.Fatal("wrapped envelope fixture no longer contains ciphertext field")
	}
	writeTestFile(t, fixture, paths[0], changed)
	if digest() == before {
		t.Fatal("wrapped key format change escaped break-glass owner")
	}
}

func TestCapacityPolicyStoredVocabularyIsStamped(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "internal/capacity/policy_storage.go"))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	path := "internal/capacity/policy_storage.go"
	writeTestFile(t, fixture, path, string(data))
	writeTestFile(t, fixture, "internal/storagetime/storagetime.go", "package storagetime\n")
	for _, path := range []string{"packages/capacity/broker.go", "packages/capacity/disk_job.go", "internal/app/capacity/service.go", "internal/app/capacity/placement_capture.go", "internal/app/capacity/residency.go"} {
		writeTestFile(t, fixture, path, "package fixture\n")
	}
	digest := func() string {
		t.Helper()
		value, err := wholeSourceDigest(fixture, statecontract.WholeSources["capacity"])
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest()
	behavior, err := os.ReadFile(filepath.Join(root, "internal/capacity/policy.go"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, fixture, "internal/capacity/policy.go", string(behavior))
	constantsBefore, err := packageConstantsDigest(fixture, []string{"internal/capacity"}, statecontract.NotStoredConstants)
	if err != nil {
		t.Fatal(err)
	}
	changedBehavior := strings.Replace(string(behavior), "256 * 1024 * bytesPerKiB", "300 * 1024 * bytesPerKiB", 1)
	if changedBehavior == string(behavior) {
		t.Fatal("policy default fixture missing")
	}
	writeTestFile(t, fixture, "internal/capacity/policy.go", changedBehavior)
	if digest() != before {
		t.Fatal("policy behavior edit changed policy format digest")
	}
	constantsAfter, err := packageConstantsDigest(fixture, []string{"internal/capacity"}, statecontract.NotStoredConstants)
	if err != nil || constantsAfter != constantsBefore {
		t.Fatalf("policy default edit changed constants digest: %v", err)
	}
	changed := strings.Replace(string(data), `"tracking_threshold"`, `"tracking_threshold_v2"`, 1)
	if changed == string(data) {
		t.Fatal("policy key fixture missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("stored policy key escaped the contract")
	}
	changed = strings.Replace(string(data), "strconv.FormatInt(p.TrackingThreshold, 10)", "strconv.FormatInt(p.TrackingThreshold, 16)", 1)
	if changed == string(data) {
		t.Fatal("policy value encoder fixture missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("stored policy value format escaped the contract")
	}
	changed = strings.Replace(string(data), "[]string{AccelReprobeOff, AccelReprobeReport, AccelReprobeRestart}", "[]string{AccelReprobeOff, AccelReprobeReport}", 1)
	if changed == string(data) {
		t.Fatal("accepted mode initializer fixture missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("accepted policy mode escaped the contract")
	}
}

func TestCapacityLegacyMigrationDefaultsAreStamped(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "internal/capacity/schema.go"))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	path := "internal/capacity/schema.go"
	writeTestFile(t, fixture, path, string(data))
	digest := func() string {
		t.Helper()
		value, err := packageConstantsDigest(fixture, []string{"internal/capacity"}, statecontract.NotStoredConstants)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest()
	changed := strings.Replace(string(data), "ADD COLUMN claim_generation INTEGER NOT NULL DEFAULT 0", "ADD COLUMN claim_generation INTEGER NOT NULL DEFAULT 1", 1)
	if changed == string(data) {
		t.Fatal("migration fixture missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("legacy migration default escaped the contract")
	}
}

func TestCapacitySQLWriterTimestampFramingIsStamped(t *testing.T) {
	root := t.TempDir()
	path := "internal/capacity/fixture.go"
	base := `package capacity
func save(db DB, now Time) { db.ExecContext("insert", formatTime(now)) }
func decision() int { return 1 }
func formatTime(t Time) string { return t.String() }
`
	digest := func(source string) string {
		t.Helper()
		writeTestFile(t, root, path, source)
		value, err := capacitySQLWriterDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest(base)
	if digest(strings.Replace(base, "return 1", "return 2", 1)) != before {
		t.Fatal("unrelated decision changed writer digest")
	}
	if digest(strings.Replace(base, "formatTime(now)", "formatTime(now) + \"Z\"", 1)) == before {
		t.Fatal("SQL timestamp framing escaped writer digest")
	}
	if digest(strings.Replace(base, "return t.String()", "return t.String() + \"Z\"", 1)) == before {
		t.Fatal("timestamp helper escaped writer digest")
	}
}

func TestCapacityReservationStoredStatusIsStamped(t *testing.T) {
	source, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	entries, err := os.ReadDir(filepath.Join(source, "internal/capacity"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join("internal/capacity", entry.Name())
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, root, path, string(data))
	}
	for _, path := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, root, path, string(data))
	}
	before, err := capacitySQLWriterDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	path := "internal/capacity/operation_reservation.go"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), "c.Status, c.ActivityState = StatusReserved, ActivityActive", "c.Status, c.ActivityState = StatusGranted, ActivityActive", 1)
	if changed == string(data) {
		t.Fatal("reservation status fixture missing")
	}
	writeTestFile(t, root, path, changed)
	after, err := capacitySQLWriterDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("reservation status change escaped writer contract")
	}
}

func TestCapacityBrokerClaimProducerIsStamped(t *testing.T) {
	source, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := "packages/capacity/broker.go"
	if !slices.Contains(statecontract.WholeSources["capacity"], path+"*") {
		t.Fatal("capacity owner omits broker claim producer")
	}
	data, err := os.ReadFile(filepath.Join(source, path))
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, path, string(data))
	digest := func() string {
		t.Helper()
		value, err := wholeSourceDigest(root, []string{path + "*"})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := digest()
	changed := strings.Replace(string(data), "Status:         internalcapacity.StatusGranted", "Status:         internalcapacity.StatusReserved", 1)
	if changed == string(data) {
		t.Fatal("broker claim fixture missing")
	}
	writeTestFile(t, root, path, changed)
	if digest() == before {
		t.Fatal("broker claim status escaped contract")
	}
}

func TestCapacityWriterDigestFollowsHelpersAndRefusesUnknownImports(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "internal/capacity/writer.go", `package capacity
func save(db DB, value Value) { db.ExecContext("insert", marshalProfile(value), boolToInt(true), diskFilesystem("/")) }
func prepare(db DB, value Value) { save(db, transform(value)) }
`)
	helper := `package capacity
func marshalProfile(v Value) string { return encode(v) }
func encode(v Value) string { return "json-v1" }
func boolToInt(v bool) int { if v { return 1 }; return 0 }
func diskFilesystem(path string) string { return "dev-v1" }
func transform(v Value) Value { return v }
`
	digest := func(source string) string {
		t.Helper()
		writeTestFile(t, root, "internal/capacity/helpers.go", source)
		d, err := capacitySQLWriterDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	before := digest(helper)
	for _, change := range []struct{ old, new string }{
		{`"json-v1"`, `"json-v2"`}, {"return 1", "return 2"}, {`"dev-v1"`, `"dev-v2"`}, {"return v }", "return alter(v) }"},
	} {
		if digest(strings.Replace(helper, change.old, change.new, 1)) == before {
			t.Fatalf("writer dependency %s escaped digest", change.old)
		}
	}
	writeTestFile(t, root, "internal/capacity/variant.go", "package capacity\nfunc diskFilesystem(path string) string { return \"other-v1\" }\n")
	withVariant := digest(helper)
	writeTestFile(t, root, "internal/capacity/variant.go", "package capacity\nfunc diskFilesystem(path string) string { return \"other-v2\" }\n")
	if digest(helper) == withVariant {
		t.Fatal("alternate platform writer helper escaped digest")
	}
	writeTestFile(t, root, "internal/capacity/helpers.go", "package capacity\nimport ext \"example.com/unknown\"\nfunc marshalProfile(v Value) string { return ext.Encode(v) }\n")
	if _, err := capacitySQLWriterDigest(root); err == nil || !strings.Contains(err.Error(), "dependency not covered") {
		t.Fatalf("unknown dependency qualified: %v", err)
	}
}

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
	writeTestFile(t, root, harnessStateContractAsset, `[{"resource":"fixture","id":"fixture","home_isolation":{"kind":"none"},"features":{}}]`)
	writeTestFile(t, root, "packages/binaryfetch/cmd/acquisition-schema/main.go", "package main\n")
	writeTestFile(t, root, "internal/capacity/writer_fixture.go", "package capacity\nfunc save(db any) { db.ExecContext() }\n")
	// Preserve the real lifecycle whole-source selection shape: its exact
	// scenario locator exclusion must match a selected file in source fixtures.
	writeTestFile(t, root, "internal/lifecycle/scenario_locator.go", "package lifecycle\n")
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

func TestCapacityConstantsCoverPersistedResidencyVocabulary(t *testing.T) {
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
const halfLifeDecayBase = 0.5
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
	if digest(residency) == digest(baseline) {
		t.Fatal("persisted residency vocabulary did not change the stored capacity contract")
	}
	before := digest(residency)
	if digest(strings.Replace(residency, "2 * time.Minute", "3 * time.Minute", 1)) != before {
		t.Fatal("nonpersistent freshness tuning changed the stored capacity contract")
	}
	if digest(strings.Replace(residency, "0.5", "0.6", 1)) != before {
		t.Fatal("sampling decay tuning changed the stored capacity contract")
	}
	for name, source := range map[string]string{
		"persisted residency backend":      strings.Replace(residency, `"cuda"`, `"cuda-v2"`, 1),
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
