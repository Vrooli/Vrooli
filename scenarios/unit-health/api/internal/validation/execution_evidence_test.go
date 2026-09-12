package validation

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"unit-health/internal/testquality"
)

func TestExecutionEvidencePreparationAndUnknownFailure(t *testing.T) {
	root := t.TempDir()
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "api", Executable: "go", Args: []string{"test", "./..."}, WorkingDirectory: root, TestKind: "unit"}}}
	collections := prepareExecutionEvidence(&plan, []Workspace{{ID: "api", Language: "go", RootPath: root}}, "request")
	if len(collections) != 1 || !plan.Commands[0].CaptureStdout || !buildExecCommands(plan.Commands)[0].CaptureStdout {
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
