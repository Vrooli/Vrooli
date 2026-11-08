package statecontract

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

const thisPackage = "github.com/vrooli/vrooli/internal/statecontract"

type storedPhase string

type storedChild struct {
	Value string `json:"value"`
}

type storedV1 struct {
	ID      string            `json:"id"`
	Phase   storedPhase       `json:"phase"`
	Child   storedChild       `json:"child"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
	cache   int
	Skip    string `json:"-"`
}

type storedV2Field struct {
	ID      string            `json:"id"`
	Phase   storedPhase       `json:"phase"`
	Child   storedChild       `json:"child"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
	Owner   string            `json:"owner,omitempty"`
}

type storedV2Tag struct {
	ID      string            `json:"identifier"`
	Phase   storedPhase       `json:"phase"`
	Child   storedChild       `json:"child"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// storedV1Renamed renames Go fields behind unchanged tags.
type storedV1Renamed struct {
	Identifier string            `json:"id"`
	State      storedPhase       `json:"phase"`
	Child      storedChild       `json:"child"`
	Created    time.Time         `json:"created"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// storedV1Reordered changes only field order, which re-encoding self-hashed
// envelopes depends on.
type storedV1Reordered struct {
	Phase   storedPhase       `json:"phase"`
	ID      string            `json:"id"`
	Child   storedChild       `json:"child"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// storedV1Internal differs from storedV1 only in fields the JSON codec never
// reads or writes.
type storedV1Internal struct {
	ID      string            `json:"id"`
	Phase   storedPhase       `json:"phase"`
	Child   storedChild       `json:"child"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
	cache   map[string]int
	lock    chan struct{}
	Skip    func() `json:"-"`
}

// Fixture owners are registered as precise or constants-stamped owners so the
// inventory requirement applies to them as to real owners.
func TestMain(m *testing.M) {
	for _, name := range []string{"fixture", "alpha", "beta"} {
		PreciseSources[name] = []string{"fixture"}
	}
	ConstantSources["registry"] = []string{"fixture"}
	os.Exit(m.Run())
}

func fixtureLayouts() map[string]string {
	return map[string]string{"fixture": strings.Repeat("e", 32), "alpha": strings.Repeat("e", 32), "beta": strings.Repeat("e", 32), "registry": strings.Repeat("e", 32)}
}

func testStamp(t *testing.T, entries map[string]string) string {
	t.Helper()
	stamp, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Entries: entries, Layouts: fixtureLayouts()})
	if err != nil {
		t.Fatal(err)
	}
	return stamp
}

func mustOwner(t *testing.T, stamp string, owner Owner) OwnerManifest {
	t.Helper()
	manifest, err := compute([]Owner{owner}, stamp, "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	return manifest.Owners[0]
}

// asSameType re-labels a fixture type as another fixture's identity so two
// compiled versions of "the same" stored type can be compared in one binary.
func asSameType(entry OwnerManifest, from, to string) OwnerManifest {
	from, to = thisPackage+"."+from, thisPackage+"."+to
	out := OwnerManifest{Owner: entry.Owner, Roots: []string{}, Types: map[string]TypeShape{}, Facts: entry.Facts}
	for _, root := range entry.Roots {
		out.Roots = append(out.Roots, strings.ReplaceAll(root, from, to))
	}
	for key, shape := range entry.Types {
		out.Types[strings.ReplaceAll(key, from, to)] = shape
	}
	out.Digest = ownerDigest(out)
	return out
}

func manifestOf(owners ...OwnerManifest) Manifest {
	return Manifest{Version: ManifestVersion, Toolchain: "go1.26", Owners: owners}
}

func TestStoredFieldChangesChangeTheOwnerAndAreNamedPrecisely(t *testing.T) {
	stamp := testStamp(t, nil)
	prior := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}})
	cases := map[string]struct {
		root reflect.Type
		name string
		want string
	}{
		"added field":    {reflect.TypeFor[storedV2Field](), "storedV2Field", `stored type internal/statecontract.storedV1 field owner added (json "owner,omitempty", type string)`},
		"retagged field": {reflect.TypeFor[storedV2Tag](), "storedV2Tag", "field id removed (was json \"id\", type string)\nowner fixture: stored type internal/statecontract.storedV1 field identifier added (json \"identifier\", type string)"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := asSameType(mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{tc.root}}), tc.name, "storedV1")
			if candidate.Digest == prior.Digest {
				t.Fatal("a stored shape change kept the owner digest")
			}
			changes := strings.Join(Diff(manifestOf(prior), manifestOf(candidate)), "\n")
			if !strings.Contains(changes, "owner fixture: ") || !strings.Contains(changes, tc.want) {
				t.Fatalf("diff does not name the owner and stored change:\n%s", changes)
			}
		})
	}
}

func TestFieldsTheCodecIgnoresDoNotChangeTheContract(t *testing.T) {
	stamp := testStamp(t, nil)
	prior := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}})
	candidate := asSameType(mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1Internal]()}}), "storedV1Internal", "storedV1")
	if candidate.Digest != prior.Digest {
		t.Fatalf("unexported or json:\"-\" fields changed the contract: %v", Diff(manifestOf(prior), manifestOf(candidate)))
	}
}

func TestGoRenamesKeepTheContractButFieldOrderParticipates(t *testing.T) {
	stamp := testStamp(t, nil)
	prior := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}})
	renamed := asSameType(mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1Renamed]()}}), "storedV1Renamed", "storedV1")
	if renamed.Digest != prior.Digest {
		t.Fatalf("a Go-only rename changed the stored contract: %v", Diff(manifestOf(prior), manifestOf(renamed)))
	}
	reordered := asSameType(mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1Reordered]()}}), "storedV1Reordered", "storedV1")
	if reordered.Digest == prior.Digest {
		t.Fatal("field order changed without changing the contract; self-hashed envelopes depend on it")
	}
}

type ownCodecType struct{ Value string }

func (ownCodecType) MarshalJSON() ([]byte, error) { return []byte(`{}`), nil }

type promotedCodecHolder struct {
	ownCodecType
}

func TestCustomCodecsFailClosedWithoutAStampEntry(t *testing.T) {
	owner := Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[ownCodecType]()}}
	if _, err := compute([]Owner{owner}, testStamp(t, nil), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "custom codec the build stamp does not cover") {
		t.Fatalf("an unstamped custom codec qualified: %v", err)
	}
	stamped := testStamp(t, map[string]string{thisPackage + ".ownCodecType": strings.Repeat("a", 32)})
	if _, err := compute([]Owner{owner}, stamped, "go1.26.1"); err != nil {
		t.Fatal(err)
	}
	holder := Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[promotedCodecHolder]()}}
	if _, err := compute([]Owner{holder}, stamped, "go1.26.1"); err != nil {
		t.Fatalf("a promoted codec required its own entry on the embedding type: %v", err)
	}
	if _, err := compute([]Owner{holder}, testStamp(t, nil), "go1.26.1"); err == nil {
		t.Fatal("the embedded codec type escaped coverage through promotion")
	}
}

func TestWholeSourceOwnersBindTheirSourceDigest(t *testing.T) {
	stamp := func(digest string) string {
		s, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Sources: map[string]string{"machinery": digest}, Layouts: fixtureLayouts()})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	owner := Owner{Name: "machinery", WholeSource: true, Facts: map[string]any{"schema_version": 1}}
	before, err := compute([]Owner{owner}, stamp(strings.Repeat("a", 32)), "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	after, err := compute([]Owner{owner}, stamp(strings.Repeat("b", 32)), "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	if got := Diff(before, after); before.Contract == after.Contract || len(got) != 1 || !strings.Contains(got[0], "owner machinery: source changed") {
		t.Fatalf("whole-source edit = %q", got)
	}
	if _, err := compute([]Owner{owner}, testStamp(t, nil), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "no source digest") {
		t.Fatalf("a whole-source owner qualified without its digest: %v", err)
	}
	if _, err := compute([]Owner{{Name: "fixture", Facts: map[string]any{"v": 1}}}, stamp(strings.Repeat("a", 32)), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "no owner declares") {
		t.Fatalf("a stamped whole source without its owner qualified: %v", err)
	}
}

func TestNestedSharedTypeChangeIsNamedAtTheChangedType(t *testing.T) {
	type parentV1 struct {
		Items []*storedChild `json:"items"`
	}
	type parentV2 struct {
		Items []*storedV2Tag `json:"items"`
	}
	stamp := testStamp(t, nil)
	prior := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[parentV1]()}})
	candidate := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[parentV2]()}})
	changes := strings.Join(Diff(manifestOf(prior), manifestOf(candidate)), "\n")
	if prior.Digest == candidate.Digest || !strings.Contains(changes, "stored type internal/statecontract.storedV2Tag added") {
		t.Fatalf("nested element change was not detected and named:\n%s", changes)
	}
}

type (
	promotedValue     struct{ Value string }
	privatePromotedV1 struct{ Value string }
	privatePromotedV2 struct{ ValueV2 string }
)

func TestJSONPromotionAndPrivateEmbeddingParticipate(t *testing.T) {
	stamp := testStamp(t, nil)
	field := reflect.StructField{Name: "PromotedValue", Type: reflect.TypeFor[promotedValue](), Anonymous: true}
	promoted := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.StructOf([]reflect.StructField{field})}})
	field.Anonymous = false
	nested := mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.StructOf([]reflect.StructField{field})}})
	if promoted.Digest == nested.Digest {
		t.Fatal("flattened JSON fields and a nested JSON object share a contract")
	}
	one, two := struct{ privatePromotedV1 }{}, struct{ privatePromotedV2 }{}
	first, _ := json.Marshal(one)
	second, _ := json.Marshal(two)
	if string(first) == string(second) {
		t.Fatal("fixture does not change serialized fields")
	}
	if mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeOf(one)}}).Digest == mustOwner(t, stamp, Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeOf(two)}}).Digest {
		t.Fatal("a private embedded type hid changed exported stored fields")
	}
}

func TestEnumAndCodecSourceComeFromTheBuildStamp(t *testing.T) {
	root := Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}}
	if _, err := compute([]Owner{root}, "", "go1.26.1"); err == nil || !strings.Contains(err.Error(), "codec stamp") {
		t.Fatalf("unstamped build qualified: %v", err)
	}
	other, err := EncodeStamp(Stamp{Packages: []string{"example.com/other"}, Layouts: fixtureLayouts()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compute([]Owner{root}, other, "go1.26.1"); err == nil || !strings.Contains(err.Error(), thisPackage+".storedPhase") || !strings.Contains(err.Error(), "CodecPackages") {
		t.Fatalf("stored enum outside the codec inventory qualified or was not named: %v", err)
	}
	before := mustOwner(t, testStamp(t, map[string]string{thisPackage + ".storedPhase": strings.Repeat("a", 32)}), root)
	after := mustOwner(t, testStamp(t, map[string]string{thisPackage + ".storedPhase": strings.Repeat("b", 32)}), root)
	changes := strings.Join(Diff(manifestOf(before), manifestOf(after)), "\n")
	if before.Digest == after.Digest || !strings.Contains(changes, "stored type internal/statecontract.storedPhase enum constants") {
		t.Fatalf("enum/codec stamp change was not detected and named:\n%s", changes)
	}
	unrelated := mustOwner(t, testStamp(t, map[string]string{thisPackage + ".storedPhase": strings.Repeat("a", 32), thisPackage + ".notStored": strings.Repeat("c", 32)}), root)
	if unrelated.Digest != before.Digest {
		t.Fatal("an enum of a type no owner stores changed the contract")
	}
}

func TestFormatFactsParticipateAndAreNamed(t *testing.T) {
	stamp := testStamp(t, nil)
	prior := mustOwner(t, stamp, Owner{Name: "fixture", Facts: map[string]any{"schema_version": 3, "ddl": "CREATE TABLE a (id TEXT)"}})
	candidate := mustOwner(t, stamp, Owner{Name: "fixture", Facts: map[string]any{"schema_version": 4, "ddl": "CREATE TABLE a (id TEXT)"}})
	if got := Diff(manifestOf(prior), manifestOf(candidate)); len(got) != 1 || got[0] != "owner fixture: format fact schema_version changed" {
		t.Fatalf("diff = %q", got)
	}
	if _, err := compute([]Owner{{Name: "fixture", Facts: map[string]any{"bad": func() {}}}}, stamp, "go1.26.1"); err == nil {
		t.Fatal("a non-canonical fact qualified")
	}
}

func TestContractIsDeterministicAndOrderIndependent(t *testing.T) {
	stamp := testStamp(t, map[string]string{thisPackage + ".storedPhase": strings.Repeat("a", 32)})
	a := Owner{Name: "alpha", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}}
	b := Owner{Name: "beta", Facts: map[string]any{"schema_version": 1, "witness": "v1\n"}}
	first, err := compute([]Owner{a, b}, stamp, "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := compute([]Owner{b, a}, stamp, "go1.26.9")
	if err != nil {
		t.Fatal(err)
	}
	if first.Contract != second.Contract || len(Diff(first, second)) != 0 {
		t.Fatal("declaration order or a toolchain patch release changed the contract")
	}
	third, err := compute([]Owner{a, b}, stamp, "go1.27.0")
	if err != nil {
		t.Fatal(err)
	}
	if third.Contract == first.Contract || !strings.Contains(strings.Join(Diff(first, third), "\n"), "go1.26 -> go1.27") {
		t.Fatal("a Go release change was not detected and named")
	}
	for _, owners := range [][]Owner{{a, a}, {{Name: "Bad Name", Facts: map[string]any{"x": 1}}}, {{Name: "empty"}}, nil} {
		if _, err := compute(owners, stamp, "go1.26.1"); err == nil {
			t.Fatalf("invalid owner inventory qualified: %+v", owners)
		}
	}
}

func TestDiffNamesOwnerInventoryAndFormatChanges(t *testing.T) {
	stamp := testStamp(t, nil)
	a := mustOwner(t, stamp, Owner{Name: "alpha", Facts: map[string]any{"v": 1}})
	b := mustOwner(t, stamp, Owner{Name: "beta", Facts: map[string]any{"v": 1}})
	got := strings.Join(Diff(manifestOf(a), manifestOf(b)), "\n")
	if got != "owner alpha: removed\nowner beta: added" {
		t.Fatalf("owner inventory diff = %q", got)
	}
	legacy := Manifest{Version: 1}
	if got := Diff(legacy, manifestOf(a)); len(got) != 1 || !strings.Contains(got[0], "v1 to v2") {
		t.Fatalf("format change diff = %q", got)
	}
}

func TestStampFormatIsStrict(t *testing.T) {
	valid := testStamp(t, map[string]string{thisPackage + ".storedPhase": strings.Repeat("a", 32)})
	if _, err := parseStamp(valid); err != nil {
		t.Fatal(err)
	}
	sections := ";srcs=;consts=;vals=;lays=;vers=;viol="
	for _, stamp := range []string{
		"",
		"v1;pkgs=" + thisPackage + sections,
		"v2;" + thisPackage,
		"v2;pkgs=" + thisPackage + ";srcs=",
		"v2;pkgs=" + thisPackage + sections + ";" + thisPackage + ".storedPhase=xyz",
		"v2;pkgs=" + thisPackage + sections + ";example.com/other.T=" + strings.Repeat("a", 32),
		"v2;pkgs=" + thisPackage + "," + thisPackage + sections,
		"v2;pkgs=" + thisPackage + ";srcs=owner;consts=;vals=;lays=;vers=;viol=",
		"v2;pkgs=" + thisPackage + ";srcs=owner:" + strings.Repeat("a", 32) + ",owner:" + strings.Repeat("b", 32) + ";consts=;vals=;lays=;vers=;viol=",
		"v2;pkgs=" + thisPackage + ";srcs=;consts=;vals=Owner/X:" + strings.Repeat("a", 32) + ";lays=;vers=;viol=",
		"v2;pkgs=" + thisPackage + ";srcs=;consts=;vals=;lays=;vers=;viol=not a violation",
		valid + ";" + thisPackage + ".storedPhase=" + strings.Repeat("b", 32),
	} {
		if _, err := parseStamp(stamp); err == nil {
			t.Fatalf("malformed stamp accepted: %q", stamp)
		}
	}
	if _, err := EncodeStamp(Stamp{Packages: []string{"bad path"}}); err == nil {
		t.Fatal("stamp encoder accepted an invalid package")
	}
}

func TestVerifyRejectsATamperedOrForeignManifest(t *testing.T) {
	manifest, err := compute([]Owner{
		{Name: "alpha", Roots: []reflect.Type{reflect.TypeFor[storedChild]()}},
		{Name: "beta", Facts: map[string]any{"schema_version": 1}},
	}, testStamp(t, nil), "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(manifest); err != nil {
		t.Fatalf("computed manifest failed verification: %v", err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil || Verify(decoded) != nil {
		t.Fatalf("manifest did not survive its own JSON encoding: %v", err)
	}
	tamper := map[string]func(*Manifest){
		"field edit": func(m *Manifest) {
			for key, shape := range m.Owners[0].Types {
				shape.Fields = append(shape.Fields, "Extra|extra|string|false")
				m.Owners[0].Types[key] = shape
			}
		},
		"fact edit":     func(m *Manifest) { m.Owners[1].Facts["schema_version"] = strings.Repeat("0", 64) },
		"contract edit": func(m *Manifest) { m.Contract = strings.Repeat("0", 64) },
		"owner order":   func(m *Manifest) { m.Owners[0], m.Owners[1] = m.Owners[1], m.Owners[0] },
		"old format":    func(m *Manifest) { m.Version = 1 },
	}
	for name, mutate := range tamper {
		t.Run(name, func(t *testing.T) {
			var copy Manifest
			if err := json.Unmarshal(data, &copy); err != nil {
				t.Fatal(err)
			}
			mutate(&copy)
			if Verify(copy) == nil {
				t.Fatal("tampered manifest verified")
			}
		})
	}
}

func TestChecksumPinnedDependenciesNeedNoSourceScan(t *testing.T) {
	old := readBuildInfo
	t.Cleanup(func() { readBuildInfo = old })
	root := Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}}
	other, err := EncodeStamp(Stamp{Packages: []string{"example.com/other"}, Layouts: fixtureLayouts()})
	if err != nil {
		t.Fatal(err)
	}
	pin := func(dep debug.Module) {
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{Deps: []*debug.Module{{Path: "github.com/vrooli", Version: "v0.1.0", Sum: "h1:outer"}, &dep}}, true
		}
	}
	pin(debug.Module{Path: "github.com/vrooli/vrooli/internal", Version: "v1.2.3", Sum: "h1:pinned"})
	first, err := compute([]Owner{root}, other, "go1.26.1")
	if err != nil {
		t.Fatalf("a checksum-pinned dependency enum was unqualified: %v", err)
	}
	pin(debug.Module{Path: "github.com/vrooli/vrooli/internal", Version: "v1.2.4", Sum: "h1:bumped"})
	second, err := compute([]Owner{root}, other, "go1.26.1")
	if err != nil || first.Contract == second.Contract {
		t.Fatalf("a dependency version change kept the contract: %v", err)
	}
	pin(debug.Module{Path: "github.com/vrooli/vrooli/internal", Version: "v1.2.3", Sum: "h1:pinned", Replace: &debug.Module{Path: "../local"}})
	if _, err := compute([]Owner{root}, other, "go1.26.1"); err == nil || !strings.Contains(err.Error(), "checksum-pinned") {
		t.Fatalf("a replaced (unpinned) dependency qualified: %v", err)
	}
}

func TestConstantsStampedOwnersBindTheirPackageConstants(t *testing.T) {
	stamp := func(digest string) string {
		s, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Consts: map[string]string{"registry": digest}, Layouts: fixtureLayouts()})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	owner := Owner{Name: "registry", StampConstants: true, Facts: map[string]any{"SchemaVersion": 15}}
	before, err := compute([]Owner{owner}, stamp(strings.Repeat("a", 32)), "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	after, err := compute([]Owner{owner}, stamp(strings.Repeat("b", 32)), "go1.26.1")
	if err != nil {
		t.Fatal(err)
	}
	if got := Diff(before, after); len(got) != 1 || !strings.Contains(got[0], "owner registry: package constants changed") {
		t.Fatalf("constants edit = %q", got)
	}
	if _, err := compute([]Owner{owner}, testStamp(t, nil), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "no constants digest") {
		t.Fatalf("a constants-stamped owner qualified without its digest: %v", err)
	}
	if _, err := compute([]Owner{{Name: "registry", Facts: map[string]any{"v": 1}}}, stamp(strings.Repeat("a", 32)), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "no owner declares") {
		t.Fatalf("stamped constants without their owner qualified: %v", err)
	}
}

func TestStampTimeViolationsMakeTheExecutableIneligible(t *testing.T) {
	stamp, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Layouts: fixtureLayouts(), Violations: []string{"version-value@internal/x/y.go:12"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = compute([]Owner{{Name: "fixture", Facts: map[string]any{"v": 1}}}, stamp, "go1.26.1")
	if err == nil || !strings.Contains(err.Error(), "internal/x/y.go:12: a format version field is written or compared with something other than a named package constant") {
		t.Fatalf("a violating build qualified or was not explained: %v", err)
	}
}

func TestFormatVersionConstantsMustBeFacts(t *testing.T) {
	stamp := func(versions ...string) string {
		s, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Layouts: fixtureLayouts(), Versions: versions})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	owner := Owner{Name: "fixture", Facts: map[string]any{"recordVersion": 2}}
	if _, err := compute([]Owner{owner}, stamp("fixture/recordVersion"), "go1.26.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := compute([]Owner{owner}, stamp("fixture/otherVersion"), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "otherVersion") {
		t.Fatalf("an undeclared format version qualified: %v", err)
	}
}

func TestStoredConstantsAndLayoutsParticipateAndAreNamed(t *testing.T) {
	stamp := func(values map[string]string, layout string) string {
		layouts := fixtureLayouts()
		layouts["fixture"] = layout
		s, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Layouts: layouts, Values: values})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	owner := Owner{Name: "fixture", Facts: map[string]any{"v": 1}}
	manifest := func(values map[string]string, layout string) Manifest {
		m, err := compute([]Owner{owner}, stamp(values, layout), "go1.26.1")
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	base := manifest(map[string]string{"fixture/StatusRunning": strings.Repeat("a", 32)}, strings.Repeat("e", 32))
	added := manifest(map[string]string{"fixture/StatusRunning": strings.Repeat("a", 32), "fixture/StatusPaused": strings.Repeat("b", 32)}, strings.Repeat("e", 32))
	if got := strings.Join(Diff(base, added), "\n"); !strings.Contains(got, "stored constant StatusPaused added (if it is never persisted, classify it in statecontract.NotStoredConstants)") {
		t.Fatalf("new stored constant diff = %q", got)
	}
	relaid := manifest(map[string]string{"fixture/StatusRunning": strings.Repeat("a", 32)}, strings.Repeat("f", 32))
	if got := strings.Join(Diff(base, relaid), "\n"); !strings.Contains(got, "time layouts") {
		t.Fatalf("layout diff = %q", got)
	}
	if _, err := compute([]Owner{owner}, testStampWithout(t, "fixture"), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "not analysed") {
		t.Fatalf("an unanalysed precise owner qualified: %v", err)
	}
	if _, err := compute([]Owner{{Name: "unlisted", Facts: map[string]any{"v": 1}}}, stamp(nil, strings.Repeat("e", 32)), "go1.26.1"); err == nil || !strings.Contains(err.Error(), "no source inventory") {
		t.Fatalf("an owner outside every inventory qualified: %v", err)
	}
}

func testStampWithout(t *testing.T, owner string) string {
	t.Helper()
	layouts := fixtureLayouts()
	delete(layouts, owner)
	s, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Layouts: layouts})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A literal written to a string field of a declared stored type makes the
// executable unqualified; the same literal on a type no owner stores, or on a
// non-string field, does not.
func TestLiteralsInStoredStringFieldsMakeTheExecutableIneligible(t *testing.T) {
	owner := Owner{Name: "fixture", Roots: []reflect.Type{reflect.TypeFor[storedV1]()}}
	qualify := func(literals ...string) error {
		t.Helper()
		stamp, err := EncodeStamp(Stamp{Packages: []string{thisPackage}, Entries: map[string]string{thisPackage + ".storedPhase": strings.Repeat("a", 32)}, Layouts: fixtureLayouts(), Literals: literals})
		if err != nil {
			t.Fatal(err)
		}
		_, err = compute([]Owner{owner}, stamp, "go1.26.1")
		return err
	}
	if err := qualify(thisPackage+".storedV2Field.Owner@internal/x/a.go:3", thisPackage+".storedV1.Created@internal/x/b.go:4"); err != nil {
		t.Fatalf("a literal on an unstored type or a non-string field disqualified the executable: %v", err)
	}
	for _, literal := range []string{
		thisPackage + ".storedV1.Phase@internal/x/c.go:5",    // named string type
		thisPackage + ".storedChild.Value@internal/x/d.go:6", // reachable nested type
		thisPackage + ".*.Value@internal/x/e.go:7",           // receiver type not resolved
	} {
		err := qualify(literal)
		location := literal[strings.Index(literal, "@")+1:]
		if err == nil || !strings.Contains(err.Error(), location+": a stored vocabulary value is written as a string literal") {
			t.Fatalf("%s qualified or was not explained: %v", literal, err)
		}
	}
	if got := LiteralViolations([]Owner{owner}, []string{thisPackage + ".storedV1.Skip@internal/x/f.go:8"}); len(got) != 0 {
		t.Fatalf("a field the codec skips was treated as stored: %v", got)
	}
}
