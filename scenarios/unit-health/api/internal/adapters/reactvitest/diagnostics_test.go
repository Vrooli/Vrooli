package reactvitest

import (
	"strings"
	"testing"
)

func TestVitestDiagnosticsKeepFailureIdentityAndSummaryWithinBounds(t *testing.T) {
	stdout := newVitestStreamDiagnostics(vitestStdout)
	stderr := newVitestStreamDiagnostics(vitestStderr)
	writeChunks := func(dst interface{ Write([]byte) (int, error) }, value string) {
		t.Helper()
		for len(value) > 0 {
			n := 7
			if len(value) < n {
				n = len(value)
			}
			if _, err := dst.Write([]byte(value[:n])); err != nil {
				t.Fatal(err)
			}
			value = value[n:]
		}
	}

	writeChunks(stdout, "noise\nTest Files  11 failed | 98 passed (109)\nTests  21 failed | 556 passed (577)\n")
	writeChunks(stderr, "FAIL  src/early.test.ts > dialog > blocks submit\n")
	for i := 0; i < maxVitestFailureHeaders; i++ {
		writeChunks(stderr, "FAIL src/repeated.test.ts > suite > failure\n")
	}
	writeChunks(stderr, strings.Repeat("<section>large DOM snapshot</section>\n", 5000))
	stdoutSummary, stderrSummary := stdout.Summary(), stderr.Summary()
	for _, want := range []string{"Test Files  11 failed | 98 passed (109)", "Tests  21 failed | 556 passed (577)"} {
		if !strings.Contains(stdoutSummary, want) {
			t.Fatalf("stdout summary missing %q: %s", want, stdoutSummary)
		}
	}
	for _, want := range []string{"FAIL  src/early.test.ts > dialog > blocks submit", "omitted_entries=1", "diagnostics partial"} {
		if !strings.Contains(stderrSummary, want) {
			t.Fatalf("stderr diagnostics missing %q: %s", want, stderrSummary)
		}
	}
	if strings.Contains(stderrSummary, "large DOM snapshot") || len(stdoutSummary)+len(stderrSummary) > 2*maxVitestDiagnosticBytes+1024 {
		t.Fatalf("unbounded DOM or diagnostic summary: stdout=%d stderr=%d", len(stdoutSummary), len(stderrSummary))
	}
}

func TestVitestDiagnosticsStripANSIAndBoundOversizedLines(t *testing.T) {
	stderr := newVitestStreamDiagnostics(vitestStderr)
	_, _ = stderr.Write([]byte("\x1b[31mFAIL\x1b[0m src/file.test.ts > suite > name\n"))
	_, _ = stderr.Write([]byte("snapshot=" + strings.Repeat("x", maxVitestDiagnosticLineBytes+100) + "\n"))
	summary := stderr.Summary()
	if !strings.Contains(summary, "FAIL src/file.test.ts > suite > name") || !strings.Contains(summary, "oversized_lines=1") || len(summary) > maxVitestDiagnosticBytes+1024 {
		t.Fatalf("unexpected bounded ANSI diagnostics: %s", summary)
	}
}

func TestVitestDiagnosticsFlushesUnterminatedSummaryAndReportsInvalidBytes(t *testing.T) {
	stdout := newVitestStreamDiagnostics(vitestStdout)
	_, _ = stdout.Write([]byte("Tests  2 failed | 8 passed (10)"))
	first := stdout.Summary()
	second := stdout.Summary()
	if !strings.Contains(first, "Tests  2 failed | 8 passed (10)") || first != second {
		t.Fatalf("unterminated summary was lost or repeated: first=%q second=%q", first, second)
	}

	invalid := newVitestStreamDiagnostics(vitestStdout)
	_, _ = invalid.Write([]byte{0xff, 0xfe})
	summary := invalid.Summary()
	if !strings.Contains(summary, "omitted_bytes=2") || !strings.Contains(summary, "diagnostics partial") {
		t.Fatalf("invalid UTF-8 omission was hidden: %q", summary)
	}
}
