package vroolicli

import (
	"context"
	"errors"
	"testing"
)

func TestOutputCombinedInjectsNoStaleCheck(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte("log line")}}}
	client := New(WithRunner(runner))

	out, err := client.OutputCombined(context.Background(), "scenario", "logs", "x")
	if err != nil {
		t.Fatalf("OutputCombined error: %v", err)
	}
	if string(out) != "log line" {
		t.Fatalf("output = %q, want %q", out, "log line")
	}
	if want := []string{"--no-stale-check", "scenario", "logs", "x"}; !equalArgs(runner.calls[0].args, want) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, want)
	}
}

// OutputCombined must return the captured output even when the command fails,
// matching exec.CombinedOutput — this is what lets the logs path surface a
// stderr message like "no lifecycle log found" as content instead of losing it.
func TestOutputCombinedReturnsOutputOnError(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{output: []byte("no lifecycle log found"), err: errors.New("exit status 1")},
	}}
	client := New(WithRunner(runner))

	out, err := client.OutputCombined(context.Background(), "scenario", "logs", "missing")
	if err == nil {
		t.Fatal("expected an error")
	}
	if string(out) != "no lifecycle log found" {
		t.Fatalf("output = %q, want the captured output preserved on error", out)
	}
}
