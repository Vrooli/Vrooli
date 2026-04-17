package orchestrator

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/logx"
)

// TestLayeringDoesNotStackSubsystem guards against a regression where
// each layer (bootstrap → orchestrator → lifecycle) chained WithSubsystem
// onto an already-scoped logger, producing multiple subsystem= tokens
// per record. Each layer must wrap the unscoped base it was given, so
// downstream wrapping stays flat.
func TestLayeringDoesNotStackSubsystem(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewTextHandler(&buf, nil))

	// Simulate what bootstrap correctly does today: it keeps the unscoped
	// base and hands that to orchestrator.New (not its own scoped Logger).
	svc := New("/root", "/home", nil, nil, base)

	svc.logger().Info("orchestrator emission")
	got := buf.String()
	if strings.Count(got, "subsystem=") != 1 {
		t.Fatalf("expected exactly one subsystem= on orchestrator log, got %q", got)
	}
	if !strings.Contains(got, "subsystem=orchestrator") {
		t.Fatalf("missing subsystem=orchestrator: %q", got)
	}

	// And the logger orchestrator hands to a child runner must be the
	// unscoped base so lifecycle's WithSubsystem doesn't stack atop
	// orchestrator.
	buf.Reset()
	runnerLog := svc.runnerBaseLogger()
	lifecycleScoped := logx.WithSubsystem(runnerLog, "lifecycle")
	lifecycleScoped.Info("lifecycle emission")
	got = buf.String()
	if strings.Count(got, "subsystem=") != 1 {
		t.Fatalf("expected exactly one subsystem= on lifecycle log, got %q", got)
	}
	if !strings.Contains(got, "subsystem=lifecycle") {
		t.Fatalf("missing subsystem=lifecycle: %q", got)
	}
}
