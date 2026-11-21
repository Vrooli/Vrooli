package programsvalidation

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProgram(t *testing.T, root, scenario, name, contract, source string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli", "program-runtime")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".py"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasFinding(findings []string, want string) bool {
	for _, finding := range findings {
		if finding == want {
			return true
		}
	}
	return false
}

// A local copy of the transport table is the drift the kernel helper retired,
// and `program.report()` is an envelope print. Both are graded by the phase so
// the canon's "a copy is a finding" is enforced, not just written.
func TestDuplicatedHelperIsAFindingAndProgramReportIsAnEnvelope(t *testing.T) {
	root := t.TempDir()
	contract := `{"name": "x", "bindings": []}`
	writeProgram(t, root, "demo", "copied", contract,
		"import json\nenvelope = {'status': 'failed'}\n"+
			"def classify_transport(exc):\n    return ('failed', 'binding_error')\n"+
			"print(json.dumps(envelope))\n")
	writeProgram(t, root, "demo", "helper", contract,
		"envelope = program.envelope('demo.helper', '1')\n"+
			"def step_validate():\n    envelope['status'] = 'ok'\n    return 'report'\n"+
			"def step_report():\n    envelope['phase'] = 'report'\n    program.report()\n"+
			"program.run({'validate': step_validate, 'report': step_report})\n")

	findings := validateScenario(root, "demo", nil, false)
	if !hasFinding(findings, "programs.duplicated_helper") {
		t.Fatalf("copied classify_transport not reported: %v", findings)
	}
	if hasFinding(findings, "programs.envelope_missing") {
		t.Fatalf("program.report() not recognised as the envelope print: %v", findings)
	}
	if hasFinding(findings, "programs.preflight_diagnostic") {
		t.Fatalf("program helper members refused by preflight: %v", findings)
	}
}

func TestHelperFindingHelpers(t *testing.T) {
	if !hasDuplicatedHelper("def classify_transport(exc):\n    pass") {
		t.Fatal("copy not detected")
	}
	if hasDuplicatedHelper("status, klass = program.classify(exc)") {
		t.Fatal("helper call misreported as a copy")
	}
	if !hasEnvelopePrint("program.report()") || !hasEnvelopePrint("print(json.dumps({'status': 'ok'}))") {
		t.Fatal("envelope print not recognised")
	}
	if hasEnvelopePrint("x = 1") {
		t.Fatal("non-envelope source accepted")
	}
}
