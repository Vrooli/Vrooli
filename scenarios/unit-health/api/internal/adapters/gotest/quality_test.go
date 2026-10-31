package gotest

import (
	"os"
	"path/filepath"
	"testing"
	"unit-health/internal/testquality"
)

func analyzeSource(t *testing.T, source string) []testquality.Result {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "case_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	rows, err := Analyze(Input{Root: root, Workspace: "api", Files: []string{"case_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
func TestSupportedGoAssertionsAreScopedByIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		status       testquality.Status
		reason       testquality.Reason
	}{
		{"idiomatic compare", `package p;import "testing";func TestValue(t *testing.T){if 2+3!=5{t.Errorf("sum")}}`, testquality.CheckedClean, testquality.ReasonNone},
		{"constant unreachable", `package p;import "testing";func TestValue(t *testing.T){if false{t.Fatal("unreachable")}}`, testquality.Violation, testquality.ReasonNone},
		{"after return", `package p;import "testing";func TestValue(t *testing.T){return;t.Fatal("unreachable")}`, testquality.Violation, testquality.ReasonNone},
		{"unreachable child", `package p;import "testing";func TestValue(t *testing.T){if false{t.Run("dead",func(t *testing.T){t.Fatal("unreachable")})}}`, testquality.Violation, testquality.ReasonNone},
		{"child after return", `package p;import "testing";func TestValue(t *testing.T){return;t.Run("dead",func(t *testing.T){t.Fatal("unreachable")})}`, testquality.Violation, testquality.ReasonNone},
		{"import alias", `package p;import("testing";check "github.com/stretchr/testify/require");func TestValue(t *testing.T){check.Equal(t,5,2+3)}`, testquality.CheckedClean, testquality.ReasonNone},
		{"unasserting constructor", `package p;import("testing";"github.com/stretchr/testify/require");func TestValue(t *testing.T){_ = require.New(t)}`, testquality.Unknown, testquality.ExternalHelperUnresolved},
		{"uncalled closure", `package p;import "testing";func TestValue(t *testing.T){_ = func(){t.Fatal("not invoked")}}`, testquality.Violation, testquality.ReasonNone},
		{"helper argument binding", `package p;import "testing";func helper(t *testing.T){t.Fatal("x")};func TestValue(t *testing.T){helper(nil)}`, testquality.Unknown, testquality.ExternalHelperUnresolved},
		{"two helpers", `package p;import "testing";func helper(t *testing.T){t.Fatal("x")};func bridge(t *testing.T){helper(t)};func TestValue(t *testing.T){bridge(t)}`, testquality.CheckedClean, testquality.ReasonNone},
		{"cycle", `package p;import "testing";func helper(t *testing.T){helper(t)};func TestValue(t *testing.T){helper(t)}`, testquality.Unknown, testquality.ResolutionLimit},
		{"dynamic child callback", `package p;import "testing";var callback func(*testing.T);func TestValue(t *testing.T){t.Run("dynamic",callback)}`, testquality.Unknown, testquality.ExternalHelperUnresolved},
		{"dynamic child name", `package p;import "testing";var name string;func TestValue(t *testing.T){t.Run(name,func(t *testing.T){t.Log("no assertion")})}`, testquality.Unknown, testquality.ExternalHelperUnresolved},
		{"cleanup callback", `package p;import "testing";func TestValue(t *testing.T){t.Cleanup(func(){t.Fatal("cleanup assertion")})}`, testquality.Unknown, testquality.ExternalHelperUnresolved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := analyzeSource(t, tc.source)
			if len(rows) != 1 || rows[0].Status != tc.status || rows[0].Reason != tc.reason {
				t.Fatalf("got %+v", rows)
			}
		})
	}
}
func TestSubtestsCannotBorrowSiblingAssertions(t *testing.T) {
	rows := analyzeSource(t, `package p;import "testing";func TestChildren(t *testing.T){t.Run("first",func(t *testing.T){t.Fatal("x")});t.Run("logging",func(t *testing.T){t.Log("text")})}`)
	if len(rows) != 2 || rows[0].Target.TestID != "TestChildren/first" || rows[0].Status != testquality.CheckedClean || rows[1].Target.TestID != "TestChildren/logging" || rows[1].Status != testquality.Violation {
		t.Fatalf("subtest scope: %+v", rows)
	}
}
func TestMalformedGoSourceRemainsFileScopedUnknown(t *testing.T) {
	rows := analyzeSource(t, "package p; func TestBroken(")
	if len(rows) != 1 || rows[0].Status != testquality.Unknown || rows[0].Reason != testquality.ParseFailure || rows[0].Target.Scope != "file" {
		t.Fatal(rows)
	}
}

func TestProcessHelperRequiresMatchingLauncherAndGuard(t *testing.T) {
	source := `package p;import("os";"testing");type command struct{Executable string; Args []string; Env map[string]string}
 func child()command{return command{Executable:os.Args[0],Args:[]string{"-test.run=^TestChild$"},Env:map[string]string{"GO_WANT_HELPER_PROCESS":"1"}}}
 func TestChild(t *testing.T){if os.Getenv("GO_WANT_HELPER_PROCESS")!="1"{return};os.Exit(0)}
 func TestLookalike(t *testing.T){if os.Getenv("GO_WANT_HELPER_PROCESS")!="1"{return};os.Exit(0)}`
	rows := analyzeSource(t, source)
	if len(rows) != 2 || rows[0].Status != testquality.NotApplicable || rows[1].Status != testquality.Unknown {
		t.Fatalf("helper registration: %+v", rows)
	}
}

func TestRepositoryExecutorProcessHelperConvention(t *testing.T) {
	rows, err := Analyze(Input{Root: filepath.Join("..", "..", "executor"), Workspace: "api", Files: []string{"executor_test.go"}, SelectedTests: []string{"TestExecutorHelperProcess"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Target.TestID != "TestExecutorHelperProcess" || rows[0].Status != testquality.NotApplicable {
		t.Fatalf("repository convention not recognized: %+v", rows)
	}
}

func TestTypeDeclarationsAreNotRuntimeTestCases(t *testing.T) {
	rows := analyzeSource(t, `package p
import "testing"
type TestStage = string
type TestRecord struct { Value string }
func TestBehavior(t *testing.T) { if len(TestStage("ready")) != 5 { t.Fatal("stage length") } }
`)
	if len(rows) != 1 || rows[0].Target.TestID != "TestBehavior" || rows[0].Status != testquality.CheckedClean {
		t.Fatalf("type declarations invented runtime cases or hid the behavior test: %+v", rows)
	}
}

func TestSkipDeclarationsDoNotInventRuntimeOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       testquality.Status
	}{
		{"placeholder", `t.Skip("TODO")`, testquality.Violation},
		{"formatted placeholder", `t.Skipf("TODO %s", "behavior")`, testquality.Violation},
		{"immediate placeholder", `t.SkipNow()`, testquality.Violation},
		{"conditional prerequisite", `if testing.Short() { t.Skip("requires tool") }; t.Fatal("behavior")`, testquality.Unknown},
		{"after other statements", `t.Log("setup"); t.Skip("not implemented")`, testquality.Unknown},
		{"unreachable declaration", `if false { t.Skip("unreachable") }; t.Fatal("behavior")`, testquality.Unknown},
		{"string lookalike", `t.Log("t.Skip TODO")`, ""},
		{"uncalled closure", `_ = func(){t.Skip("not called")}`, ""},
		{"shadowed receiver", `{ t := struct{Skip func(string)}{}; t.Skip("not testing") }`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := analyzeSource(t, `package p; import "testing"; func TestCase(t *testing.T) {`+tc.body+`}`)
			var declarations []testquality.Result
			for _, row := range rows {
				if row.RuleID == "skip-declaration" {
					declarations = append(declarations, row)
				}
			}
			if tc.want == "" {
				if len(declarations) != 0 {
					t.Fatalf("invented declaration: %+v", declarations)
				}
				return
			}
			if len(declarations) != 1 || declarations[0].Status != tc.want || declarations[0].EvidenceKind != testquality.Static || declarations[0].Enforcement != testquality.Advisory {
				t.Fatalf("declaration classification: %+v", declarations)
			}
			wantReason := testquality.ReasonNone
			if tc.want == testquality.Unknown {
				wantReason = testquality.NotExecuted
			}
			if declarations[0].Reason != wantReason {
				t.Fatalf("reason: %+v", declarations[0])
			}
		})
	}
}

func TestGoRequirementLinksUseDeclaredBoundariesWithoutExecutionClaims(t *testing.T) {
	root := t.TempDir()
	source := `package p
import "testing"
// [REQ:UH-CORE-001, UH-CORE-010]
func TestParent(t *testing.T) {
 t.Run("[REQ:UH-ANALYZE-003] first", func(t *testing.T) { t.Log("setup") })
 t.Run("second", func(t *testing.T) { t.Log("setup") })
}
func TestSeparate(t *testing.T) { // [REQ:UH-OTHER-001]
 // [REQ:UH-BODY-001] This body comment does not declare the test's responsibility.
 t.Log("setup")
}
func TestBare(t *testing.T) { t.Log("UH-CORE-001") }
`
	if err := os.WriteFile(filepath.Join(root, "case_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	_, links, err := AnalyzeEvidence(Input{Root: root, Workspace: "api", Files: []string{"case_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 4 {
		t.Fatalf("test boundaries: %+v", links)
	}
	for _, link := range links {
		if link.Execution != testquality.ExecutionNotRun || link.EvidenceKind != testquality.Static {
			t.Fatalf("source manufactured execution: %+v", link)
		}
	}
	if len(links[0].IDs) != 3 || len(links[1].IDs) != 2 || len(links[2].IDs) != 1 || links[2].IDs[0] != "UH-OTHER-001" || len(links[3].IDs) != 0 {
		t.Fatalf("tags leaked between tests: %+v", links)
	}
}
