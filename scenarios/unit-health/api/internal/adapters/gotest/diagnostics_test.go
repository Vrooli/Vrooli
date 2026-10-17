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
