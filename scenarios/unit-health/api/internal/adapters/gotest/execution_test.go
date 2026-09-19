package gotest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unit-health/internal/adapters"
	"unit-health/internal/executor"
	"unit-health/internal/testquality"
)

func TestNativeGoExecutionEvidenceFromBoundedCommand(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.test/trace\n\ngo 1.25\n",
		"trace_test.go": `package trace
import "testing"
// [REQ:UH-CORE-001]
func TestPassing(t *testing.T) { if 2+3 != 5 { t.Fatal("sum") } }
// [REQ:UH-CORE-010]
func TestSkipped(t *testing.T) { t.Skip("fixture deliberately skipped") }
`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, declarations, err := AnalyzeEvidence(Input{Root: root, Workspace: "api", Files: []string{"trace_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	args, ok := (Analyzer{}).PrepareExecutionEvidence("go", []string{"test", "./..."})
	if !ok {
		t.Fatal("canonical Go command unsupported")
	}
	result := (executor.Bounded{}).Run(context.Background(), executor.Command{Executable: "go", Args: args, Dir: root, CaptureStdout: true, TimeoutSeconds: 60})
	if result.Status != executor.StatusPassed {
		t.Fatalf("native fixture failed: %s %s", result.FailureReason, result.Stderr)
	}
	links, reason := (Analyzer{}).ObserveExecutionEvidence(adapters.ExecutionEvidenceInput{Root: root, RunID: "fixture-current", Stdout: result.StdoutEvidence, Complete: result.StdoutEvidenceComplete, Declarations: declarations})
	if reason != testquality.ReasonNone || len(links) != 2 || links[0].Execution != testquality.ExecutionPassed || links[1].Execution != testquality.ExecutionSkipped {
		t.Fatalf("native evidence: %+v reason=%s output=%s", links, reason, result.StdoutEvidence)
	}
}

func TestGoExecutionMatchesPackageAndIndividualState(t *testing.T) {
	data := `{"Action":"start","Package":"example/a"}
{"Action":"start","Package":"example/b"}
{"Action":"run","Package":"example/a","Test":"TestSame","Time":"2026-09-07T00:00:00Z"}
{"Action":"run","Package":"example/b","Test":"TestSame","Time":"2026-09-07T00:00:00Z"}
{"Action":"pass","Package":"example/a","Test":"TestSame","Time":"2026-09-07T00:00:01Z"}
{"Action":"skip","Package":"example/b","Test":"TestSame","Time":"2026-09-07T00:00:01Z"}`
	declarations := []testquality.TestLinks{{Target: testquality.Target{File: "a/test_test.go", TestID: "TestSame"}, IDs: []string{"UH-CORE-001"}}, {Target: testquality.Target{File: "b/test_test.go", TestID: "TestSame"}, IDs: []string{"UH-CORE-010"}}}
	rows, reason := ObserveExecution("example", "current", []byte(data), true, declarations)
	if reason != testquality.ReasonNone || len(rows) != 2 || rows[0].Execution != testquality.ExecutionPassed || rows[1].Execution != testquality.ExecutionSkipped || rows[1].IDs[0] != "UH-CORE-010" {
		t.Fatalf("package/state evidence: %+v %s", rows, reason)
	}
	rows, reason = ObserveExecution("example", "current", []byte(strings.ReplaceAll(data, `,"Time":"2026-09-07T00:00:00Z"`, "")), true, declarations)
	if reason != testquality.ReasonNone || rows[0].Execution != testquality.ExecutionUnknown {
		t.Fatalf("cached/non-running evidence became current: %+v %s", rows, reason)
	}
	if rows, reason := ObserveExecution("example", "current", []byte(data), false, declarations); rows != nil || reason != testquality.ResolutionLimit {
		t.Fatal("partial capture accepted")
	}
}

func TestGoExecutionRejectsMalformedOrTerminalOnlyOutput(t *testing.T) {
	declarations := []testquality.TestLinks{{Target: testquality.Target{File: "test_test.go", TestID: "TestSame"}}}
	for _, data := range []string{`...truncated...`, `{"Action":"pass","Package":"example","Test":"TestSame"}`} {
		if rows, reason := ObserveExecution("example", "current", []byte(data), true, declarations); rows != nil || reason != testquality.ParseFailure {
			t.Fatalf("invalid stream: %+v %s", rows, reason)
		}
	}
	rows, reason := ObserveExecution("example", "current", []byte(`{"Action":"start","Package":"example"} {"Action":"pass","Package":"example"}`), true, declarations)
	if reason != testquality.ReasonNone || rows[0].Execution != testquality.ExecutionUnknown {
		t.Fatalf("package success became test pass: %+v %s", rows, reason)
	}
}

func TestGoExecutionSanitizedNameCollisionsRemainUnknown(t *testing.T) {
	data := []byte(`{"Action":"start","Package":"example"} {"Action":"run","Package":"example","Test":"TestParent/a_b","Time":"now"} {"Action":"pass","Package":"example","Test":"TestParent/a_b","Time":"now"}`)
	declarations := []testquality.TestLinks{{Target: testquality.Target{File: "test_test.go", TestID: "TestParent/a b"}}, {Target: testquality.Target{File: "test_test.go", TestID: "TestParent/a_b"}}}
	rows, reason := ObserveExecution("example", "current", data, true, declarations)
	if reason != testquality.ReasonNone || len(rows) != 2 || rows[0].Execution != testquality.ExecutionUnknown || rows[1].Execution != testquality.ExecutionUnknown {
		t.Fatalf("colliding names borrowed a pass: %+v %s", rows, reason)
	}
}
