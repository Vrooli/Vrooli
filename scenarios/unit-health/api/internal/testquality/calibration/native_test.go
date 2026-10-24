package calibration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeMatcherRequiresNamedIntendedFailure(t *testing.T) {
	spec := NativeSpecification{Version: "2.1.9", Cases: []NativeExpectation{{CaseID: "C023", Name: "empty", Status: "failed", FailureContains: "no assertions"}}}
	var observed NativeObservation
	if err := json.Unmarshal([]byte(`{"version":"2.1.9","runnerExitCode":1,"cases":[{"name":"empty","status":"failed","failureMessages":["no assertions"]}]}`), &observed); err != nil {
		t.Fatal(err)
	}
	good := CompareNative(spec, observed)
	if good.Matched != 1 || len(good.Differences) != 0 {
		t.Fatalf("control %+v", good)
	}
	observed.Cases[0].FailureMessages = []string{"unrelated network outage"}
	bad := CompareNative(spec, observed)
	if bad.Matched != 0 || len(bad.Differences) == 0 {
		t.Fatal("unrelated failure satisfied expectation")
	}
	observed.Cases = nil
	if missing := CompareNative(spec, observed); missing.Matched != 0 || len(missing.Differences) == 0 {
		t.Fatal("missing case passed")
	}
}

func TestLoadNativeSpecificationRejectsMalformedAndUnsafeFixtures(t *testing.T) {
	root := t.TempDir()
	write := func(raw string) {
		if err := os.WriteFile(filepath.Join(root, "native-expected.json"), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{}`, `{"version":"1","provenance":"p","root":".","files":[]}`, `{"version":"1","provenance":"p","root":"../escape","files":["x"]}`} {
		write(raw)
		if _, err := LoadNativeSpecification(root); err == nil {
			t.Fatalf("malformed native fixture accepted: %s", raw)
		}
	}
	write(`{"version":"1","provenance":"p","root":"fixtures","files":["case.js"],"cases":[{"caseId":"C1","name":"broken","status":"failed"}]}`)
	os.MkdirAll(filepath.Join(root, "fixtures"), 0o755)
	os.WriteFile(filepath.Join(root, "fixtures", "case.js"), []byte(""), 0o600)
	if _, err := LoadNativeSpecification(root); err == nil {
		t.Fatal("failed case without rationale accepted")
	}
	write(`{"version":"1","provenance":"p","root":"fixtures","files":["case.js"],"cases":[{"caseId":"C1","name":"same","status":"passed"},{"caseId":"C2","name":"same","status":"passed"}]}`)
	if _, err := LoadNativeSpecification(root); err == nil {
		t.Fatal("duplicate native name accepted")
	}
}

func TestCompareNativeReportsRetriesUnexpectedCasesAndLimitations(t *testing.T) {
	spec := NativeSpecification{Version: "1", Cases: []NativeExpectation{{CaseID: "C1", Name: "pass", Status: "passed", RetryCount: 1, RetainedErrorContains: "old", Limitation: "runner note"}, {CaseID: "C2", Name: "skip", Status: "skipped"}}}
	var observed NativeObservation
	if err := json.Unmarshal([]byte(`{"version":"1","cases":[{"name":"pass","status":"passed"},{"name":"extra","status":"passed"}],"reporter":{"events":[{"name":"pass","retryCount":1,"errors":[{"message":"old error"}]}]}}`), &observed); err != nil {
		t.Fatal(err)
	}
	report := CompareNative(spec, observed)
	if report.Matched != 1 || len(report.Limitations) != 1 || len(report.Differences) == 0 {
		t.Fatalf("native comparison = %+v", report)
	}
}

func TestNativeUnknownVersionAndUnhandledErrorCannotCertify(t *testing.T) {
	spec := NativeSpecification{Version: "2.1.9", Cases: []NativeExpectation{{CaseID: "C021", Name: "direct", Status: "passed"}}}
	if report := CompareNative(spec, NativeObservation{Version: "99.0.0"}); !report.UnknownVersion || report.Matched != 0 {
		t.Fatal("unsupported version certified")
	}
	var observed NativeObservation
	if err := json.Unmarshal([]byte(`{"version":"2.1.9","cases":[{"name":"direct","status":"passed"}],"reporter":{"errors":[{"name":"Error","message":"unhandled"}]}}`), &observed); err != nil {
		t.Fatal(err)
	}
	if report := CompareNative(spec, observed); len(report.Differences) == 0 {
		t.Fatal("unhandled runner error ignored")
	}
}

func TestNativeExpectationsPreserveKnownRunnerLimitations(t *testing.T) {
	spec, err := LoadNativeSpecification("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Cases) != 18 {
		t.Fatalf("lost native cases: %d", len(spec.Cases))
	}
	for _, id := range []string{"C025", "C026", "C027", "C028", "C033"} {
		found := false
		for _, c := range spec.Cases {
			if c.CaseID == id && c.Limitation != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing explicit native limitation for %s", id)
		}
	}
}
