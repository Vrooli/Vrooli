package buildinfo

import (
	"strings"
	"testing"
)

func TestCallerReadsOfOwnedVocabularyAreChecked(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", `package owner
type Stored struct { Status string }
`)
	writeTestFile(t, root, "caller/caller.go", `package caller
import "example.test/app/owner"
type Other struct { Status string }
func check(s owner.Stored, other Other) bool {
    if other.Status == "unrelated" { return true }
    if s.Status == "ready" { return true }
    switch s.Status { case "pending": return true }
    return false
}

`)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected comparison and switch on owned type only, got %v", got)
	}
	for _, candidate := range got {
		if !strings.Contains(candidate, "example.test/app/owner.Stored.Status@") {
			t.Fatalf("unexpected candidate: %s", candidate)
		}
	}
}

func TestCallerOwnedConstantDoesNotEscapeQualification(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/process/status.go", `package process
type State string
type Stored struct { Status State }
const Ready = "ready"
`)
	writeTestFile(t, root, "caller/caller.go", `package caller
import owner "github.com/vrooli/vrooli/internal/process"
func check(s owner.Stored) {
    if s.Status == local {}
    if s.Status == owner.Ready {}
}

func shadow(s owner.Stored) {
    local := owner.Ready
    if s.Status == local {}
}`)
	writeTestFile(t, root, "caller/constants.go", "package caller\nconst local = \"ready\"\n")
	got, err := StoredFieldLiteralCandidates(root, []string{"internal/process"}, "caller")
	if err != nil || len(got) != 1 || !strings.Contains(got[0], "process.Stored.Status@caller/caller.go:4") {
		t.Fatalf("expected only caller-owned constant, got %v, %v", got, err)
	}
}

func TestForeignSelectorsRequireAStampedBinding(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/process/process.go", `package process
import dep "github.com/vrooli/vrooli/packages/resource-deployment"
import third "github.com/vrooli/vrooli/third"
type Stable struct { Status dep.Mode }
const Ready = "ready"
const Indirect = third.Ready
var Variable dep.Mode
`)
	writeTestFile(t, root, "internal/process/record_linux.go", "package process\nimport dep \"github.com/vrooli/vrooli/packages/resource-deployment\"\ntype Record struct { Status dep.Mode }\ntype AliasRecord struct { Status dep.Mode }\n")
	writeTestFile(t, root, "internal/process/record_windows.go", "package process\ntype Record struct { Status interface{} }\ntype AliasRecord = Stable\n")
	writeTestFile(t, root, "packages/resource-deployment/value.go", `package dep
type Mode string
const Ready Mode = "ready"
`)
	writeTestFile(t, root, "packages/resource-deployment/value_linux.go", "package dep\nconst Variant Mode = \"ready\"\n")
	writeTestFile(t, root, "packages/resource-deployment/value_windows.go", "package dep\nvar Variant Mode = \"ready\"\n")
	writeTestFile(t, root, "caller/caller.go", `package caller
import "github.com/vrooli/vrooli/internal/process"
import dep "github.com/vrooli/vrooli/packages/resource-deployment"
import third "github.com/vrooli/vrooli/third"
func check() {
    if (process.Record{}).Status == process.Ready {}; if (process.Stable{}).Status == dep.Ready {}; if (process.AliasRecord{}).Status == dep.Ready {}
    _ = process.Record{Status: third.Ready}
    _ = process.Record{Status: third.Variable}
    _ = process.Record{Status: process.Variable}
    _ = process.Record{Status: process.Ready}
    _ = process.Record{Status: dep.Ready}
    _ = process.Stable{Status: dep.Ready}
    _ = process.Record{Status: dep.Variant}
    _ = process.Record{Status: process.Indirect}
    _ = process.AliasRecord{Status: dep.Ready}
}
`)
	writeTestFile(t, root, "third/value.go", "package third\nimport dep \"github.com/vrooli/vrooli/packages/resource-deployment\"\nconst Ready = \"ready\"\nvar Variable dep.Mode = \"ready\"\n")
	got, err := StoredFieldLiteralCandidates(root, []string{"internal/process", "packages/resource-deployment"}, "caller", "third")
	joined := strings.Join(got, "\n")
	if err != nil || len(got) != 8 || !strings.Contains(joined, "caller/caller.go:6") || !strings.Contains(joined, "caller/caller.go:7") || !strings.Contains(joined, "caller/caller.go:8") || !strings.Contains(joined, "caller/caller.go:9") || !strings.Contains(joined, "caller/caller.go:11") || !strings.Contains(joined, "caller/caller.go:13") || !strings.Contains(joined, "caller/caller.go:14") || !strings.Contains(joined, "caller/caller.go:15") {
		t.Fatalf("expected only unstamped third-party values, got %v, %v", got, err)
	}
}

func TestCallerOpenIdentifierExceptionDoesNotHideWrites(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/process/process.go", `package process
type Record struct { Scenario string }
`)
	writeTestFile(t, root, "caller/caller.go", `package caller
import "github.com/vrooli/vrooli/internal/process"
func check(r *process.Record) { if r.Scenario == "specific" {}; r.Scenario = "written" }
`)
	got, err := StoredFieldLiteralCandidates(root, []string{"internal/process"}, "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "process.Record.Scenario@") {
		t.Fatalf("expected the write but not the identity comparison, got %v", got)
	}
}

func TestCallerScanFailsOnMalformedSource(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype Stored struct { Status string }\n")
	writeTestFile(t, root, "caller/bad.go", "package caller\nfunc broken(\n")
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller")
	if err == nil || got != nil {
		t.Fatalf("malformed caller source must fail the scan without partial results: got %v, err %v", got, err)
	}
}

func TestCallerScanFailsOnMalformedModule(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype Stored struct { Status string }\n")
	writeTestFile(t, root, "caller/go.mod", "go 1.25\n")
	writeTestFile(t, root, "caller/caller.go", "package caller\n")
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller")
	if err == nil || got != nil {
		t.Fatalf("malformed caller module must fail the scan without partial results: got %v, err %v", got, err)
	}
}

func TestCallerDirectOwnerFunctionResults(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", `package owner
type Status string
type Stored struct { Status Status; Nested Other }
type Other struct { Status string }
func Get() *Stored { return nil }
func GetOther() Other { return Other{} }
func Load() (*Stored, error) { return nil, nil }
`)
	writeTestFile(t, root, "caller/caller.go", `package caller
import "example.test/app/owner"
import "example.test/app/caller/bridge"
func check() {
    owner.Get().Status = "direct"
    s := owner.Get(); s.Status = "inferred"
    loaded, err := owner.Load(); _ = err; loaded.Status = "multiple"
    bridge.Get().Status = "through-bridge"
    bridge.Variant().Status = "variant"
    bridge.Get().Nested.Status = "nested-unrelated"
    other := owner.GetOther(); other.Status = "unrelated"
}
`)
	writeTestFile(t, root, "caller/bridge/bridge.go", `package bridge
import "example.test/app/owner"
func Get() *owner.Stored { return nil }
`)
	writeTestFile(t, root, "caller/bridge/variant_linux.go", `package bridge
import "example.test/app/owner"
type Alias = owner.Stored
func Variant() Alias { return Alias{} }
`)
	writeTestFile(t, root, "caller/bridge/variant_windows.go", `package bridge
import "example.test/app/owner"
func Variant() owner.Other { return owner.Other{} }
`)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("expected five stored writes, got %v", got)
	}
	for _, candidate := range got {
		if !strings.Contains(candidate, "owner.Stored.Status@") {
			t.Fatalf("unexpected candidate: %s", candidate)
		}
	}
}

func TestCallerNestedForeignFieldResolvesToStoredType(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", `package owner
 type Status string
 type Stored struct { Status Status }
 `)
	writeTestFile(t, root, "bridge/bridge.go", `package bridge
 import "example.test/app/owner"
 type Container struct { Item *owner.Stored; Other Other }
 type Other struct { Status string }
 func Get() Container { return Container{} }
 `)
	writeTestFile(t, root, "caller/caller.go", `package caller
 import "example.test/app/bridge"
 func check() { b := bridge.Get(); b.Item.Status = "unsafe"; b.Other.Status = "unrelated" }
 `)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "bridge", "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "owner.Stored.Status@") {
		t.Fatalf("expected stored nested write only, got %v", got)
	}
}

func TestCallerExternalMethodResultsAndShadowedImport(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", `package owner
 type Status string
 type Stored struct { Status Status }
 `)
	writeTestFile(t, root, "bridge/bridge.go", `package bridge
 import "example.test/app/owner"
 type Client struct{}
 func (Client) Get() *owner.Stored { return nil }
 func (Client) Load() (*owner.Stored, error) { return nil, nil }
 `)
	writeTestFile(t, root, "caller/caller.go", `package caller
 import "example.test/app/bridge"
 type Other struct { Status string }
 type Local struct{}
 func (Local) Get() Other { return Other{} }
 func check(client bridge.Client, bridge Local) {
   client.Get().Status = "direct"
   s, err := client.Load(); _ = err; s.Status = "multi"
   bridge.Get().Status = "unrelated"
 }
 `)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "bridge", "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two stored method results only, got %v", got)
	}
	for _, candidate := range got {
		if !strings.Contains(candidate, "owner.Stored.Status@") {
			t.Fatalf("unrelated result: %s", candidate)
		}
	}
}

func TestCallerInterfaceMethodResult(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype Status string\ntype Stored struct { Status Status }\n")
	writeTestFile(t, root, "bridge/bridge.go", `package bridge
 import "example.test/app/owner"
 type Reader interface { Get() (*owner.Stored, error) }
 `)
	writeTestFile(t, root, "caller/caller.go", `package caller
 import "example.test/app/bridge"
 func check(r bridge.Reader) { s, err := r.Get(); _ = err; s.Status = "unsafe" }
 `)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "bridge", "caller")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "owner.Stored.Status@") {
		t.Fatalf("expected stored interface result, got %v", got)
	}
}

func TestCallerValueAliasesAndStringExpressionsAreChecked(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype State string\ntype Stored struct { Status State }\n")
	writeTestFile(t, root, "caller/caller.go", `package caller
import "example.test/app/owner"
import third "example.test/app/third"
func check(s owner.Stored) {
    v := third.Ready
    const c = third.Ready
    s.Status = v
    s.Status = c
    s.Status = "run" + "ning"
}`)
	writeTestFile(t, root, "third/value.go", "package third\nconst Ready = \"ready\"\n")
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller", "third")
	if err != nil || len(got) != 3 {
		t.Fatalf("expected two foreign aliases and the literal expression, got %v, %v", got, err)
	}
}

func TestCallerLocalAliasChainResolvesOwnedType(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype State string\ntype Stored struct { Status State }\n")
	writeTestFile(t, root, "caller/caller.go", `package caller
import "example.test/app/owner"
type A = owner.Stored
type B = A
type P = *B
func check(s P) { s.Status = "unsafe"; _ = B{Status: "unsafe"} }
`)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller")
	if err != nil || len(got) != 1 || !strings.Contains(got[0], "owner.Stored.Status@") {
		t.Fatalf("alias chain did not resolve to the owner: %v, %v", got, err)
	}
}

func TestCallerValueScopesAndAssignmentOrder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module github.com/vrooli/vrooli\n\ngo 1.25\n")
	writeTestFile(t, root, "internal/process/status.go", "package process\ntype Stored struct { Status string }\nconst Ready = \"ready\"\nfunc New() Stored { return Stored{} }\nfunc Load() (Stored, error) { return Stored{}, nil }\n")
	writeTestFile(t, root, "third/value.go", "package third\nconst Ready = \"ready\"\n")
	writeTestFile(t, root, "caller/caller.go", `package caller
import "github.com/vrooli/vrooli/internal/process"
import vocab "github.com/vrooli/vrooli/third"
func check(s process.Stored) {
    v := "running"; { v := "unrelated"; s := 1; _, _ = v, s }; s.Status = v
    if s.Status == vocab.Ready {}; { vocab := 1; _ = vocab }
    safe := process.Ready; s.Status = safe; safe = "unrelated"; _ = safe
}
func loop(s process.Stored) { v := process.Ready; for { s.Status = v; v = "next" } }
func closure(s process.Stored) { v := process.Ready; f := func() { s.Status = v }; v = "later"; f() }
func again(s process.Stored, repeat bool) { v := process.Ready; again: s.Status = v; v = "again"; if repeat { goto again } }
func overwritten(s process.Stored) { v := "unrelated"; v = process.Ready; s.Status = v }
func self(s process.Stored) { v := "unsafe"; v = v + ""; s.Status = v }
func compound(s process.Stored) { v := "unsafe"; v += ""; s.Status = v }
func nested() { f := func(s *process.Stored) { s.Status = "nested" }; _ = f }
func inferred() { var s = process.Stored{}; s.Status = "inferred" }
func called() { var s = process.New(); s.Status = "called" }
func multiple() { var s, err = process.Load(); _ = err; s.Status = "multiple" }
func converted(s process.Stored) { s.Status = string("converted"); s.Status = string(vocab.Ready) }
`)
	got, err := StoredFieldLiteralCandidates(root, []string{"internal/process"}, "caller", "third")
	joined := strings.Join(got, "\n")
	if err != nil || len(got) != 12 || !strings.Contains(joined, "caller/caller.go:5") || !strings.Contains(joined, "caller/caller.go:6") || !strings.Contains(joined, "caller/caller.go:9") || !strings.Contains(joined, "caller/caller.go:10") || !strings.Contains(joined, "caller/caller.go:11") || !strings.Contains(joined, "caller/caller.go:13") || !strings.Contains(joined, "caller/caller.go:14") || !strings.Contains(joined, "caller/caller.go:15") || !strings.Contains(joined, "caller/caller.go:16") || !strings.Contains(joined, "caller/caller.go:17") || !strings.Contains(joined, "caller/caller.go:18") || !strings.Contains(joined, "caller/caller.go:19") || strings.Contains(joined, "caller/caller.go:7") || strings.Contains(joined, "caller/caller.go:12") {
		t.Fatalf("scope or assignment order misclassified: %v, %v", got, err)
	}
}

func TestCallerGlobalReceiversAndNamedConversions(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype State string\ntype Stored struct { Status State }\n")
	writeTestFile(t, root, "third/value.go", "package third\nconst Ready = \"ready\"\n")
	writeTestFile(t, root, "caller/globals.go", "package caller\nimport o \"example.test/app/owner\"\nvar typed o.Stored\nvar inferred = o.Stored{}\n")
	writeTestFile(t, root, "caller/global_linux.go", "package caller\nimport o \"example.test/app/owner\"\nvar platform o.Stored\nfunc makeState() o.Stored { return o.Stored{} }\nfunc makePair() (o.Stored, error) { return o.Stored{}, nil }\nfunc all() []o.Stored { return nil }\n")
	writeTestFile(t, root, "caller/global_windows.go", "package caller\ntype Other struct { Status string }\nvar platform Other\nfunc makeState() Other { return Other{} }\nfunc makePair() (Other, error) { return Other{}, nil }\nfunc all() []Other { return nil }\n")
	writeTestFile(t, root, "caller/bridge/bridge_linux.go", "package bridge\nimport o \"example.test/app/owner\"\nfunc All() []o.Stored { return nil }\n")
	writeTestFile(t, root, "caller/bridge/bridge_windows.go", "package bridge\ntype Other struct { Status string }\nfunc All() []Other { return nil }\n")
	writeTestFile(t, root, "caller/caller.go", `package caller
import "example.test/app/owner"
import third "example.test/app/third"
import "example.test/app/caller/bridge"
func check() {
    typed.Status = "typed"
    inferred.Status = "inferred"
    typed.Status = owner.State("converted")
    typed.Status = owner.State(third.Ready)
    platform.Status = "variant"
    made := makeState(); made.Status = "result"
    made2, err := makePair(); _ = err; made2.Status = "pair"
    var made3, err2 = makePair(); _ = err2; made3.Status = "var pair"
    for _, item := range all() { item.Status = "range" }
    items := all(); for _, item := range items { item.Status = "stored range" }
    for _, item := range bridge.All() { item.Status = "foreign range" }
    foreignItems := bridge.All(); for _, item := range foreignItems { item.Status = "stored foreign range" }
}`)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"}, "caller", "third")
	if err != nil || len(got) != 12 {
		t.Fatalf("global receivers or named conversions escaped: %v, %v", got, err)
	}
}

func TestNonStoredReceiversDoNotMatchOwnerFieldNames(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/app\n\ngo 1.25\n")
	writeTestFile(t, root, "owner/status.go", "package owner\ntype V string\ntype Stored struct { Priority V; Reason V; Phase V }\n")
	writeTestFile(t, root, "owner/behavior.go", `package owner
const priority = 3
const reason = "rollback"
const phase = "disabled"
type Other struct { Priority int; Reason string; Phase string }
type Store interface { All() ([]Other, error) }
func inspect(store Store) (result Other) {
    list, _ := store.All(); c := list[0]
    if c.Priority == priority {}
    candidates := make([]Other, 0, 1)
    for _, candidate := range candidates { candidate.Reason = reason }
    result.Phase = phase
    return result
}`)
	got, err := StoredFieldLiteralCandidates(root, []string{"owner"})
	if err != nil || strings.Contains(strings.Join(got, "\n"), "example.test/app/owner.*.") {
		t.Fatalf("receiver type was not resolved: %v, %v", got, err)
	}
}
