package buildinfo

import (
	"github.com/vrooli/vrooli/internal/statecontract"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

type Stored struct{ Status string }

func TestProbes(t *testing.T) {
	const path = "github.com/vrooli/vrooli/internal/buildinfo"
	cases := []struct {
		name, src string
		caller    bool
	}{
		{"unresolved_read", `package buildinfo
 const ready = "ready"
 type Stored struct { Status string }
 type A struct{}; type B struct{}
 func (A) Get() Stored { return Stored{} }; func (B) Get() int { return 0 }
 func f(a A) { r := a.Get(); if r.Status == ready {} ; _ = Stored{Status:"new"} }`, false},
		{"unresolved_function_value", `package buildinfo
 const ready = "ready"
 type Stored struct { Status string }
 func f(get func() Stored) { r := get(); if r.Status == ready {}; _ = Stored{Status:"new"} }`, false},
		{"shadow", `package buildinfo
 type Stored struct { Status string }; type Other struct { Status string }
 func f(r Stored) { { r := Other{}; _ = r }; if r.Status == "new" {} }`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "probe.go", tc.src, 0)
			if err != nil {
				t.Fatal(err)
			}
			sources := []goSource{{path: "probe.go", file: file}}
			pkg := indexPackage(fset, ".", sources)
			ip := path
			if tc.caller {
				ip = "github.com/vrooli/vrooli/internal/caller"
			}
			uses := fieldUses(pkg, ip, sources)
			ev := newVocabularyEvidence()
			ev.add(ip, uses)
			var owned map[string]bool
			if tc.caller {
				owned = map[string]bool{path: true}
			} else {
				typedFields(pkg, ip, ev.typed)
			}
			candidates := literalCandidates(pkg, ip, uses, ev, owned)
			violations := statecontract.LiteralViolations([]statecontract.Owner{{Roots: []reflect.Type{reflect.TypeOf(Stored{})}}}, candidates)
			if len(violations) == 0 {
				t.Fatalf("stored literal escaped: uses=%+v candidates=%v", uses, candidates)
			}
		})
	}
}

func TestQualificationWithCopiedScanner(t *testing.T) {
	const path = "github.com/vrooli/vrooli/internal/buildinfo"
	roots := []statecontract.Owner{{Name: "probe", WholeSource: true, Roots: []reflect.Type{reflect.TypeOf(Stored{})}}}
	source := `package buildinfo
 const ready = "ready"
 type Stored struct {Status string}
 type A struct{}; type B struct{}
 func (A) Get() Stored{return Stored{}}; func (B) Get() int{return 0}
 func f(a A) { r := a.Get(); if r.Status == ready {}; _ = Stored{Status:"unsafe"} }`
	fset := token.NewFileSet()
	f, e := parser.ParseFile(fset, "probe.go", source, 0)
	if e != nil {
		t.Fatal(e)
	}
	src := []goSource{{path: "probe.go", file: f}}
	pkg := indexPackage(fset, ".", src)
	uses := fieldUses(pkg, path, src)
	evidence := newVocabularyEvidence()
	evidence.add(path, uses)
	typedFields(pkg, path, evidence.typed)
	candidates := literalCandidates(pkg, path, uses, evidence, nil)
	stamp, err := statecontract.EncodeStamp(statecontract.Stamp{Packages: []string{path}, Sources: map[string]string{"probe": strings.Repeat("a", 32)}, Literals: candidates})
	if err != nil {
		t.Fatal(err)
	}
	original := statecontract.CodecStamp
	defer func() { statecontract.CodecStamp = original }()
	statecontract.CodecStamp = stamp
	if _, err = statecontract.Compute(roots); err == nil || !strings.Contains(err.Error(), "stored vocabulary") {
		t.Fatalf("expected qualification refusal: %v", err)
	}
	stamp, err = statecontract.EncodeStamp(statecontract.Stamp{Packages: []string{path}, Sources: map[string]string{"probe": strings.Repeat("a", 32)}})
	if err != nil {
		t.Fatal(err)
	}
	statecontract.CodecStamp = stamp
	if _, err = statecontract.Compute(roots); err != nil {
		t.Fatalf("safe fixture rejected: %v", err)
	}
}

func TestURLParserResultIsNotAttributedToStoredOwner(t *testing.T) {
	const path = "github.com/vrooli/vrooli/internal/buildinfo"
	for _, fn := range []string{"Parse", "ParseRequestURI"} {
		t.Run(fn, func(t *testing.T) {
			source := `package buildinfo
import endpoint "net/url"
func check(raw string) { u, err := endpoint.` + fn + `(raw); if err == nil && u.Scheme != "http" {} }`
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "probe.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			sources := []goSource{{path: "probe.go", file: file}}
			pkg := indexPackage(fset, ".", sources)
			uses := fieldUses(pkg, path, sources)
			for _, use := range uses {
				if use.field == "Scheme" {
					if use.key != "net/url.URL" {
						t.Fatalf("URL field attributed to %q", use.key)
					}
					return
				}
			}
			t.Fatal("URL scheme comparison was not inspected")
		})
	}
}

func TestShadowedURLImportDoesNotHideStoredField(t *testing.T) {
	const path = "github.com/vrooli/vrooli/internal/buildinfo"
	source := `package buildinfo
import endpoint "net/url"
var _ = endpoint.URL{}
type Stored struct { Scheme string }
type localParser struct{}
func (localParser) Parse(string) (Stored, error) { return Stored{}, nil }
func check(endpoint localParser) { u, err := endpoint.Parse(""); if err == nil && u.Scheme != "http" {} }`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "probe.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	sources := []goSource{{path: "probe.go", file: file}}
	pkg := indexPackage(fset, ".", sources)
	for _, use := range fieldUses(pkg, path, sources) {
		if use.field == "Scheme" {
			if use.key == "net/url.URL" {
				t.Fatal("shadowed local parser was mistaken for net/url")
			}
			return
		}
	}
	t.Fatal("local scheme comparison was not inspected")
}

func TestCallerSelectorOnOwnedTypeIsChecked(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/process/status.go", `package process
const Ready = "ready"
type Stored struct { Status string }
func read(s Stored) bool { return s.Status == Ready }
`)
	writeTestFile(t, root, "caller/caller.go", `package caller
import owner "github.com/vrooli/vrooli/internal/process"
type Other struct { Status string }
func update(s *owner.Stored, other *Other) {
    s.Status = "unsafe"
    other.Status = "unrelated"
}
`)
	got, err := StoredFieldLiteralCandidates(root, []string{"internal/process"}, "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "github.com/vrooli/vrooli/internal/process.Stored.Status@") {
		t.Fatalf("expected only owned selector literal, got %v", got)
	}
}

func TestNestedCallerMutationRefusesQualification(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/buildinfo/owner.go", `package buildinfo
 type Status string
 type Stored struct { Status Status }
 `)
	writeTestFile(t, root, "internal/bridge/bridge.go", `package bridge
 import "github.com/vrooli/vrooli/internal/buildinfo"
 type Container struct { Item *buildinfo.Stored }
 func Get() Container { return Container{} }
 `)
	writeTestFile(t, root, "internal/caller/caller.go", `package caller
 import "github.com/vrooli/vrooli/internal/bridge"
 func write() { bridge.Get().Item.Status = "unsafe" }
 `)
	candidates, err := StoredFieldLiteralCandidates(root, []string{"internal/buildinfo"}, "internal/bridge", "internal/caller")
	if err != nil {
		t.Fatal(err)
	}
	roots := []statecontract.Owner{{Name: "probe", WholeSource: true, Roots: []reflect.Type{reflect.TypeOf(Stored{})}}}
	stamp, err := statecontract.EncodeStamp(statecontract.Stamp{Packages: []string{"github.com/vrooli/vrooli/internal/buildinfo"}, Sources: map[string]string{"probe": strings.Repeat("a", 32)}, Literals: candidates})
	if err != nil {
		t.Fatal(err)
	}
	old := statecontract.CodecStamp
	defer func() { statecontract.CodecStamp = old }()
	statecontract.CodecStamp = stamp
	if _, err := statecontract.Compute(roots); err == nil || !strings.Contains(err.Error(), "stored vocabulary") {
		t.Fatalf("nested mutation qualified: candidates=%v err=%v", candidates, err)
	}
}

func TestForeignCodecValueRefusesQualification(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "fixture.go", `package fixture
import foreign "example.test/foreign"
type Stored struct{}
func (Stored) MarshalJSON() ([]byte, error) { return []byte(foreign.Value), nil }
`)
	_, violations, err := scanCodecPackage(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	stamp, err := statecontract.EncodeStamp(statecontract.Stamp{Packages: []string{"github.com/vrooli/vrooli/internal/buildinfo"}, Sources: map[string]string{"probe": strings.Repeat("a", 32)}, Violations: violations})
	if err != nil {
		t.Fatal(err)
	}
	old := statecontract.CodecStamp
	defer func() { statecontract.CodecStamp = old }()
	statecontract.CodecStamp = stamp
	roots := []statecontract.Owner{{Name: "probe", WholeSource: true, Roots: []reflect.Type{reflect.TypeOf(Stored{})}}}
	if _, err := statecontract.Compute(roots); err == nil || !strings.Contains(err.Error(), "another module package") {
		t.Fatalf("unbound codec value qualified: %v", err)
	}
}
