package cliapptest

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// failer is the minimum surface CaptureStdout needs from testing.TB. Carved
// out so this package's own tests can spy on Fatalf with a recording stub —
// testing.TB cannot be implemented externally because of its private() gate.
type failer interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}

// CaptureStdout redirects os.Stdout to a pipe for the duration of fn, then
// returns everything written. Useful for asserting on the human output
// produced by cli-core's RenderOperationalReport et al. when a test exercises
// a code path that writes to os.Stdout directly rather than through a
// RunContext writer.
//
// The original stdout is restored via tb.Cleanup; tests that fail mid-capture
// still get their stdout back for the test runner.
func CaptureStdout(tb testing.TB, fn func() error) string {
	tb.Helper()
	return captureStdout(tb, fn)
}

func captureStdout(tb failer, fn func() error) string {
	tb.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		tb.Fatalf("pipe: %v", err)
		return ""
	}
	os.Stdout = w
	tb.Cleanup(func() { os.Stdout = orig })

	runErr := fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		tb.Fatalf("copy stdout: %v", err)
		return ""
	}
	if runErr != nil {
		tb.Fatalf("command returned error: %v", runErr)
		return ""
	}
	return buf.String()
}
