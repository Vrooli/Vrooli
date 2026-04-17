package lifecycle

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressfEmitsAtQuiet(t *testing.T) {
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityQuiet}
	runner.progressf("starting %s...", "alpha")
	if got := out.String(); got != "starting alpha...\n" {
		t.Fatalf("quiet progressf = %q", got)
	}
}

func TestProgressfEmitsAtNormal(t *testing.T) {
	// Normal mode is the new default on TTYs (where slog info is
	// suppressed by the top-level vrooli binary), so pings must reach
	// the user here or they see a silent 10+ second gap.
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityNormal}
	runner.progressf("running setup phase for %s...", "alpha")
	if !strings.Contains(out.String(), "running setup phase for alpha...") {
		t.Fatalf("normal progressf dropped: %q", out.String())
	}
}

func TestProgressfSuppressedAtVerbose(t *testing.T) {
	// Verbose replays the full slog/debug stream plus child tool stdout,
	// so duplicating pings here adds noise.
	var out bytes.Buffer
	runner := &Runner{Out: &out, Verbosity: VerbosityVerbose}
	runner.progressf("starting %s...", "alpha")
	if got := out.String(); got != "" {
		t.Fatalf("verbose progressf should be silent, got %q", got)
	}
}

func TestProgressfNoOpOnNilWriter(t *testing.T) {
	runner := &Runner{Out: nil, Verbosity: VerbosityNormal}
	// Must not panic.
	runner.progressf("hello %s", "world")
}
