package validation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"unit-health/internal/executor"
	"unit-health/internal/testquality"
)

func TestExecutionEvidencePreparationAndUnknownFailure(t *testing.T) {
	root := t.TempDir()
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "api", Executable: "go", Args: []string{"test", "./..."}, WorkingDirectory: root, TestKind: "unit"}}}
	collections := prepareExecutionEvidence(&plan, []Workspace{{ID: "api", Language: "go", RootPath: root}}, "request")
	if len(collections) != 1 || !plan.Commands[0].CaptureStdout || !buildExecCommands(plan.Commands)[0].CaptureStdout || buildExecCommands(plan.Commands)[0].StderrObserver != nil {
		t.Fatalf("capture not wired: %+v", plan)
	}
	declaration := testquality.TestLinks{Target: testquality.Target{Workspace: "api", File: "trace_test.go", TestID: "TestOne"}, IDs: []string{"UH-CORE-001"}, EvidenceKind: testquality.Static}
	links, reason := collectExecutionEvidence(collections, []CommandResult{{Status: "timeout"}}, []testquality.TestLinks{declaration})
	if reason == testquality.ReasonNone || links[0].Execution != testquality.ExecutionUnknown {
		t.Fatalf("timeout became execution proof: %+v %s", links, reason)
	}
	writeFile(t, filepath.Join(root, "go.mod"), "module example\n\ngo 1.25\n")
	result := CommandResult{Status: "passed", StdoutEvidenceComplete: true, StdoutEvidence: []byte(`{"Action":"start","Package":"example"} {"Action":"run","Package":"example","Test":"TestOne","Time":"now"} {"Action":"pass","Package":"example","Test":"TestOne","Time":"now"}`)}
	links, reason = collectExecutionEvidence(collections, []CommandResult{result}, []testquality.TestLinks{declaration})
	if reason != testquality.ReasonNone || links[0].Execution != testquality.ExecutionPassed || links[0].RunID != "request:command:0" {
		t.Fatalf("current event identity: %+v %s", links, reason)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var persisted CommandResult
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.StdoutEvidence != nil || persisted.StdoutEvidenceComplete {
		t.Fatal("ephemeral structured capture persisted as reusable current evidence")
	}
}

func TestUnregisteredNonGoCommandDoesNotGainObservers(t *testing.T) {
	root := t.TempDir()
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "misc", Executable: "pytest", Args: []string{"-q"}, WorkingDirectory: root, TestKind: "unit"}}}
	prepareExecutionEvidence(&plan, []Workspace{{ID: "misc", Language: "python", Framework: "pytest", RootPath: root}}, "request")
	command := buildExecCommands(plan.Commands)[0]
	if plan.Commands[0].ExecutionDiagnostics != nil || plan.Commands[0].StderrDiagnostics != nil || command.StdoutObserver != nil || command.StderrObserver != nil {
		t.Fatal("unregistered workspace gained an execution diagnostics observer")
	}
}

// longDiagnosticRunner models an overflowing exact capture while feeding the
// real command observer. Executor tests independently cover the stdout tee.
type longDiagnosticRunner struct{ t *testing.T }

func (r longDiagnosticRunner) Run(_ context.Context, command executor.Command) executor.Result {
	r.t.Helper()
	if command.StdoutObserver == nil || !command.CaptureStdout {
		r.t.Fatal("instrumented command missing bounded observer/capture")
	}
	first := []byte("{\"Action\":\"fail\",\"Package\":\"example/api\",\"Test\":\"TestEarly\"}\n{\"Action\":\"fail\",\"Package\":\"example/api\"}\n")
	for _, part := range [][]byte{first[:13], first[13:]} {
		command.StdoutObserver.Write(part)
	}
	passing := strings.Repeat("{\"Action\":\"pass\",\"Package\":\"example/later\",\"Test\":\"TestPassing\"}\n", 64)
	for written := 0; written <= 9<<20; written += len(passing) {
		command.StdoutObserver.Write([]byte(passing))
	}
	return executor.Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Command: "go test -json -count=1 ./...", Status: executor.StatusFailed, FailureClass: executor.ClassTestFailure, FailureReason: "exit status 1", ExitCode: 1, Stdout: "...[truncated]...\n" + passing, Stderr: "later stderr context", StdoutEvidenceComplete: false}
}

func TestGoFailureWitnessSurvivesLongOutputAndFindingSerialization(t *testing.T) {
	root := t.TempDir()
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "api", Executable: "go", Args: []string{"test", "./..."}, WorkingDirectory: root, TestKind: "unit"}}}
	prepareExecutionEvidence(&plan, []Workspace{{ID: "api", Language: "go", RootPath: root}}, "request")
	svc := Service{Executor: longDiagnosticRunner{t}, MaxConcurrency: 1}
	results, findings := svc.execute(context.Background(), "demo", plan, nil, "now", nil)
	if len(results) != 1 || len(findings) != 1 || results[0].StdoutEvidenceComplete {
		t.Fatal(results, findings)
	}
	// Finding evidence is retained, unlike the ephemeral complete-output proof.
	data, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	var retained []Finding
	if err := json.Unmarshal(data, &retained); err != nil {
		t.Fatal(err)
	}
	evidence := retained[0].Evidence
	if !strings.Contains(evidence, `package="example/api" test="TestEarly"`) || !strings.Contains(evidence, "package failure; no test identity supplied") || !strings.Contains(evidence, "--- stderr tail ---\nlater stderr context") || strings.Contains(evidence, "TestPassing") {
		t.Fatal(evidence)
	}
	if strings.Index(evidence, "observed Go failure") > strings.Index(evidence, "stderr tail") {
		t.Fatal("tail precedes diagnostic witness")
	}
}

func TestVitestFailureHeadersAndSummaryAreRetained(t *testing.T) {
	root := t.TempDir()
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "ui", Executable: "pnpm", Args: []string{"test"}, WorkingDirectory: root, TestKind: "unit"}}}
	prepareExecutionEvidence(&plan, []Workspace{{ID: "ui", Language: "typescript", Framework: "vitest", RootPath: root}}, "request")
	command := buildExecCommands(plan.Commands)[0]
	if plan.Commands[0].ExecutionDiagnostics == nil || plan.Commands[0].StderrDiagnostics == nil || command.StdoutObserver == nil || command.StderrObserver == nil {
		t.Fatal("Vitest bounded diagnostics were not wired to both streams")
	}
	svc := Service{Executor: vitestDiagnosticRunner{}, MaxConcurrency: 1}
	_, findings := svc.execute(context.Background(), "demo", plan, nil, "now", nil)
	if len(findings) != 1 || !strings.Contains(findings[0].Evidence, "FAIL src/example.test.ts > suite > assertion") || !strings.Contains(findings[0].Evidence, "Tests  1 failed | 10 passed") || !strings.Contains(findings[0].Evidence, "--- stderr tail ---\nordinary failure tail") {
		t.Fatal(findings)
	}
}

type vitestDiagnosticRunner struct{}

func (vitestDiagnosticRunner) Run(_ context.Context, command executor.Command) executor.Result {
	_, _ = command.StdoutObserver.Write([]byte("Test Files  1 failed | 2 passed (3)\nTests  1 failed | 10 passed (11)\n"))
	_, _ = command.StderrObserver.Write([]byte("FAIL src/example.test.ts > suite > assertion\n" + strings.Repeat("<div>large snapshot</div>\n", 2000)))
	return executor.Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Command: "pnpm run test:coverage", Status: executor.StatusFailed, FailureClass: executor.ClassTestFailure, FailureReason: "exit status 1", ExitCode: 1, Stderr: "ordinary failure tail"}
}
