package gotest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func diagnosticEvent(action, pkg, test string) []byte {
	data, _ := json.Marshal(nativeEvent{Action: action, Package: pkg, Test: test})
	return append(data, '\n')
}

func TestGoFailureDiagnosticsSurvivePassingOutputBeyondCaptureBudget(t *testing.T) {
	d := (Analyzer{}).NewExecutionDiagnostics().(*failureDiagnostics)
	first := append(diagnosticEvent("fail", "example/api", "TestEarly/subtest"), diagnosticEvent("fail", "example/api", "")...)
	// Pipe writes may split JSON tokens and even the newline arbitrarily.
	for _, b := range first {
		if n, err := d.Write([]byte{b}); n != 1 || err != nil {
			t.Fatal(n, err)
		}
	}
	passing := []byte(strings.Repeat(string(diagnosticEvent("pass", "example/later", "TestPassing")), 64))
	written := 0
	for written <= 9<<20 {
		d.Write(passing)
		written += len(passing)
	}
	summary := d.Summary()
	if !strings.Contains(summary, `package="example/api" test="TestEarly/subtest"`) || !strings.Contains(summary, "package failure; no test identity supplied") || !strings.Contains(summary, "retained=2 omitted_failure_events=0") || strings.Contains(summary, "TestPassing") {
		t.Fatal(summary)
	}
	if cap(d.carry) > maxDiagnosticLineBytes || d.witnessBytes > maxDiagnosticWitnessBytes || len(summary) > maxDiagnosticSummaryBytes || len(d.witnesses) != 2 {
		t.Fatalf("unbounded retention: carry=%d witness=%d summary=%d", cap(d.carry), d.witnessBytes, len(summary))
	}
}

func TestGoFailureDiagnosticsRecoverAfterMalformedOversizedAndFragmentedLines(t *testing.T) {
	d := &failureDiagnostics{}
	d.Write([]byte("not json\n"))
	// No newline for many writes: storage stays bounded, then recovers at LF.
	for i := 0; i < 100; i++ {
		d.Write([]byte(strings.Repeat("x", 4096)))
		if cap(d.carry) > maxDiagnosticLineBytes {
			t.Fatal("line carry exceeded budget")
		}
	}
	d.Write([]byte("\n"))
	identity := "TestQuote/\"unicode_世界"
	event := diagnosticEvent("fail", "example/recovered", identity)
	for _, b := range event {
		d.Write([]byte{b})
	}
	d.Write([]byte(`{"Action":"fail"`))
	summary := d.Summary()
	if !strings.Contains(summary, fmt.Sprintf("test=%q", identity)) || !strings.Contains(summary, "malformed_lines=1 oversized_lines=1 unterminated_line=true") || !strings.Contains(summary, "diagnostics partial") || d.Summary() != summary {
		t.Fatal(summary)
	}
}

func TestGoFailureDiagnosticsBoundFailureCountAndRenderedBytes(t *testing.T) {
	for _, size := range []int{10, maxDiagnosticIdentityBytes} {
		d := &failureDiagnostics{}
		for i := 0; i < 100; i++ {
			d.Write(diagnosticEvent("fail", "example/api", fmt.Sprintf("%03d", i)+strings.Repeat("x", size-3)))
		}
		summary := d.Summary()
		if len(d.witnesses) > maxDiagnosticFailures || d.witnessBytes > maxDiagnosticWitnessBytes || len(summary) > maxDiagnosticSummaryBytes || d.omittedFailureEvents == 0 || !strings.Contains(summary, "diagnostics partial") || !strings.Contains(summary, `test="000`) {
			t.Fatalf("size=%d: %s", size, summary)
		}
	}
}

func TestGoFailureDiagnosticsNeverInferPassingFromMissingWitness(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte("invalid\n"),
		[]byte(`{"Action":"fail","Package":"example/api"}`), // unterminated
		[]byte("{\"Action\":\"fail\",\"Package\":\"bad\xff\"}\n"),
		diagnosticEvent("fail", "", "TestUnknownPackage"),
		diagnosticEvent("fail", "example/api", strings.Repeat("x", maxDiagnosticIdentityBytes+1)),
		diagnosticEvent("pass", "example/api", "TestPassing"),
	} {
		d := &failureDiagnostics{}
		d.Write(data)
		if len(d.witnesses) != 0 || !strings.Contains(d.Summary(), "No Go failure identity retained; this does not establish that tests passed.") {
			t.Fatal(d.Summary())
		}
	}
}

func TestGoFailureDiagnosticsKeepExactPackageIdentityAndCommandIsolation(t *testing.T) {
	one := (Analyzer{}).NewExecutionDiagnostics()
	two := (Analyzer{}).NewExecutionDiagnostics()
	one.Write(diagnosticEvent("fail", "example/one", "TestSame"))
	one.Write(diagnosticEvent("fail", "example/two", "TestSame"))
	if !strings.Contains(one.Summary(), `package="example/one" test="TestSame"`) || !strings.Contains(one.Summary(), `package="example/two" test="TestSame"`) || strings.Contains(two.Summary(), "example/") {
		t.Fatal(one.Summary(), two.Summary())
	}
}

func outputDiagnosticEvent(pkg, test, output string) []byte {
	data, _ := json.Marshal(nativeEvent{Action: "output", Package: pkg, Test: test, Output: output})
	return append(data, '\n')
}

func rawGoBuildEvent(action, importPath, output string) []byte {
	data, _ := json.Marshal(map[string]string{
		"Action": action, "ImportPath": importPath, "Output": output,
	})
	return append(data, '\n')
}

func TestGoFailureDiagnosticsRetainCompilerOutputAndFailedBuildIdentity(t *testing.T) {
	const packagePath = "personal-planner/cli/docs/historical-controls"
	d := &failureDiagnostics{}
	d.Write(rawGoBuildEvent("build-output", packagePath, "# "+packagePath+"\n"))
	d.Write(rawGoBuildEvent("build-output", packagePath, "historical/app.go:28:49: undefined: manifestBytes\n"))
	d.Write(rawGoBuildEvent("build-fail", packagePath, ""))
	d.Write(outputDiagnosticEvent(packagePath, "", "FAIL\t"+packagePath+" [build failed]\n"))
	failed, _ := json.Marshal(map[string]string{"Action": "fail", "Package": packagePath, "FailedBuild": packagePath})
	d.Write(append(failed, '\n'))

	summary := d.Summary()
	for _, want := range []string{
		`package="` + packagePath + `" failed_build="` + packagePath + `" (package failure; no test identity supplied)`,
		`undefined: manifestBytes`,
		"retained=1 omitted_failure_events=0 omitted_output_bytes=1 malformed_lines=0 oversized_lines=0 unterminated_line=false valid_events=5; diagnostics partial",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("compiler failure evidence %q missing from summary: %s", want, summary)
		}
	}
	if !strings.Contains(summary, "diagnostics partial") || len(summary) > maxDiagnosticSummaryBytes || d.pendingOutputBytes > maxPendingOutputTotalBytes || d.failureOutputBytes > maxDiagnosticFailureOutputTotalBytes {
		t.Fatalf("compiler failure evidence bypassed existing bounds: pending=%d failure=%d summary=%d: %s", d.pendingOutputBytes, d.failureOutputBytes, len(summary), summary)
	}
}

func TestGoFailureDiagnosticsJoinDependencyCompilerOutputToFailedBuild(t *testing.T) {
	const importingPackage = "example/application"
	const dependencyPackage = "example/internal/dependency"
	d := &failureDiagnostics{}
	d.Write(rawGoBuildEvent("build-output", dependencyPackage, "dependency/file.go:12:3: undefined: missingSymbol\n"))
	d.Write(rawGoBuildEvent("build-fail", dependencyPackage, ""))
	failed, _ := json.Marshal(map[string]string{
		"Action": "fail", "Package": importingPackage, "FailedBuild": dependencyPackage,
	})
	d.Write(append(failed, '\n'))

	summary := d.Summary()
	for _, want := range []string{
		`package="` + importingPackage + `" failed_build="` + dependencyPackage + `"`,
		`output="dependency/file.go:12:3: undefined: missingSymbol"`,
		"malformed_lines=0 oversized_lines=0",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("dependency build evidence %q missing from summary: %s", want, summary)
		}
	}
	if len(d.pendingOutputs) != 0 || d.pendingOutputBytes != 0 {
		t.Fatalf("dependency compiler output was not consumed by its failure: pending=%d bytes=%d", len(d.pendingOutputs), d.pendingOutputBytes)
	}
}

func TestGoFailureDiagnosticsRejectBuildOutputIdentityBeyondSharedKeyBound(t *testing.T) {
	d := &failureDiagnostics{}
	overlongImportPath := strings.Repeat("x", maxPendingOutputKeyBytes+1)
	d.Write(rawGoBuildEvent("build-output", overlongImportPath, "compiler detail that must not be retained\n"))
	summary := d.Summary()
	if !strings.Contains(summary, "malformed_lines=1") || strings.Contains(summary, "compiler detail") || len(d.pendingOutputs) != 0 || d.pendingOutputBytes != 0 {
		t.Fatalf("oversized build identity bypassed diagnostic bounds: pending=%d bytes=%d %s", len(d.pendingOutputs), d.pendingOutputBytes, summary)
	}
}

func TestGoFailureDiagnosticsRetainMatchedAssertionAfterLaterPassingOutput(t *testing.T) {
	d := &failureDiagnostics{}
	d.Write(diagnosticEvent("run", "example/api", "TestBroken"))
	d.Write(outputDiagnosticEvent("example/api", "TestBroken", "assertion failed: expected 4, got 9\n"))
	d.Write(diagnosticEvent("run", "example/api", "TestPassing"))
	d.Write(outputDiagnosticEvent("example/api", "TestPassing", "unrelated passing output\n"))
	d.Write(diagnosticEvent("pass", "example/api", "TestPassing"))
	d.Write(diagnosticEvent("run", "example/api", "TestSkipped"))
	d.Write(outputDiagnosticEvent("example/api", "TestSkipped", "skipped-only output\n"))
	d.Write(diagnosticEvent("skip", "example/api", "TestSkipped"))
	d.Write(diagnosticEvent("fail", "example/api", "TestBroken"))
	for i := 0; i < 40; i++ {
		d.Write(diagnosticEvent("pass", "example/later", fmt.Sprintf("TestPass%d", i)))
	}

	summary := d.Summary()
	if !strings.Contains(summary, `test="TestBroken" output="assertion failed: expected 4, got 9"`) {
		t.Fatalf("failed-test assertion was not retained: %s", summary)
	}
	if strings.Contains(summary, "unrelated passing output") || strings.Contains(summary, "skipped-only output") || !strings.Contains(summary, "retained=1 omitted_failure_events=0") || len(summary) > maxDiagnosticSummaryBytes {
		t.Fatalf("unrelated output leaked or bounds changed: %s", summary)
	}
}

func TestGoFailureDiagnosticsRetainPackageOnlyFailureOutput(t *testing.T) {
	d := &failureDiagnostics{}
	d.Write(outputDiagnosticEvent("example/api", "", "package setup assertion failed\n"))
	d.Write(diagnosticEvent("fail", "example/api", ""))
	summary := d.Summary()
	if !strings.Contains(summary, `package="example/api" (package failure; no test identity supplied) output="package setup assertion failed"`) {
		t.Fatalf("package-only failure output missing: %s", summary)
	}
}

func TestGoFailureDiagnosticsBoundPendingAndFailureOutput(t *testing.T) {
	d := &failureDiagnostics{}
	for i := 0; i < maxPendingDiagnosticTests+8; i++ {
		test := fmt.Sprintf("TestParallel%d", i)
		d.Write(diagnosticEvent("run", "example/api", test))
		d.Write(outputDiagnosticEvent("example/api", test, strings.Repeat("x", 1000)))
		if len(d.pendingOutputs) > maxPendingDiagnosticTests || d.pendingOutputBytes > maxPendingOutputTotalBytes {
			t.Fatalf("pending output exceeded bounds: tests=%d bytes=%d", len(d.pendingOutputs), d.pendingOutputBytes)
		}
	}
	if _, ok := d.pendingOutputs[outputKey{packageName: "example/api", testName: "TestParallel0"}]; ok {
		t.Fatal("oldest pending test was not evicted at the pending-test bound")
	}
	for i := 0; i < maxPendingDiagnosticTests+8; i++ {
		d.Write(diagnosticEvent("fail", "example/api", fmt.Sprintf("TestParallel%d", i)))
	}
	summary := d.Summary()
	if len(summary) > maxDiagnosticSummaryBytes || d.failureOutputBytes > maxDiagnosticFailureOutputTotalBytes || d.witnessBytes > maxDiagnosticWitnessBytes || !strings.Contains(summary, "omitted_output_bytes=") || !strings.Contains(summary, "diagnostics partial") {
		t.Fatalf("failure output exceeded summary bounds or lacked omission accounting: bytes=%d summary=%d %s", d.failureOutputBytes, len(summary), summary)
	}
}

func TestGoFailureDiagnosticsRejectOversizedOutputEventLine(t *testing.T) {
	d := &failureDiagnostics{}
	d.Write(outputDiagnosticEvent("example/api", "TestLarge", strings.Repeat("x", maxDiagnosticLineBytes)))
	d.Write(diagnosticEvent("fail", "example/api", "TestLarge"))
	summary := d.Summary()
	if !strings.Contains(summary, "oversized_lines=1") || strings.Contains(summary, "output=") || !strings.Contains(summary, `test="TestLarge"`) || len(summary) > maxDiagnosticSummaryBytes {
		t.Fatalf("oversized line changed identity or escaped bounds: %s", summary)
	}
}

func TestGoFailureDiagnosticsCountsUTF8BoundaryTrimming(t *testing.T) {
	d := &failureDiagnostics{}
	key := outputKey{packageName: "example/api", testName: "TestUnicode"}
	output := strings.Repeat("a", 100) + "界" + strings.Repeat("b", maxPendingOutputPerTestBytes-2)
	if got := len(output); got <= maxPendingOutputPerTestBytes {
		t.Fatalf("fixture must exceed pending output cap: %d", got)
	}
	d.rememberOutput(key, output)
	if got := d.omittedOutputBytes; got != uint64(len(output)-maxPendingOutputPerTestBytes+2) {
		t.Fatalf("omitted bytes should include UTF-8 boundary trim: got=%d want=%d", got, len(output)-maxPendingOutputPerTestBytes+2)
	}
	if got := d.pendingOutputBytes; got != maxPendingOutputPerTestBytes-2 {
		t.Fatalf("retained UTF-8 tail byte count mismatch: got=%d want=%d", got, maxPendingOutputPerTestBytes-2)
	}
}

func TestGoFailureDiagnosticsCountsOutputWhenIdentityWitnessBudgetIsFull(t *testing.T) {
	d := &failureDiagnostics{}
	key := outputKey{packageName: "example/api", testName: "TestBudget"}
	d.rememberOutput(key, "assertion output")
	witness := `package="example/api" test="TestBudget"`
	d.witnessBytes = maxDiagnosticWitnessBytes - len(witness) // one byte too little for witness newline
	d.recordFailure(nativeEvent{Package: key.packageName, Test: key.testName}, key, true)
	if d.omittedOutputBytes != uint64(len("assertion output")) || d.failureOutputBytes != 0 || d.omittedFailureEvents != 1 || len(d.witnesses) != 0 {
		t.Fatalf("discarded output accounting mismatch: omitted=%d excerpt=%d omitted_failures=%d witnesses=%d", d.omittedOutputBytes, d.failureOutputBytes, d.omittedFailureEvents, len(d.witnesses))
	}
}
