package elevation

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type elevationExecutor struct {
	name string
	args []string
}

func (e *elevationExecutor) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	e.name = name
	e.args = append([]string(nil), args...)
	return []byte("ok"), nil
}

func withElevationTestState(t *testing.T, osName string, present bool) {
	t.Helper()
	origOS, origPresent, origRoot, origFacts := osNameFn, grantPresent, hostreqkit.RunningAsRootFn, hostreqkit.ElevationFactsFn
	osNameFn = func() string { return osName }
	grantPresent = func() bool { return present }
	hostreqkit.RunningAsRootFn = func() bool { return false }
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: osName, Elevated: false, CanElevate: true, Mechanism: "sudo"}
	}
	t.Cleanup(func() {
		osNameFn, grantPresent, hostreqkit.RunningAsRootFn, hostreqkit.ElevationFactsFn = origOS, origPresent, origRoot, origFacts
	})
}

func TestRunMissingGrantReturnsNeedsSetupWithoutExecuting(t *testing.T) {
	withElevationTestState(t, "linux", false)
	executor := &elevationExecutor{}
	outcome, output, err := Run(context.Background(), executor, ServiceRestart, "docker")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if output != nil || outcome.State != NeedsSetup {
		t.Fatalf("outcome=%+v output=%q, want needs_setup without output", outcome, output)
	}
	if outcome.Command != "sudo vrooli setup" || executor.name != "" {
		t.Fatalf("outcome=%+v executor=%q, want setup command and no execution", outcome, executor.name)
	}
}

func TestRunGrantedUsesNonInteractiveTypedCommand(t *testing.T) {
	withElevationTestState(t, "linux", true)
	executor := &elevationExecutor{}
	outcome, output, err := Run(context.Background(), executor, ServiceRestart, "docker")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if outcome.State != Granted || string(output) != "ok" {
		t.Fatalf("outcome=%+v output=%q", outcome, output)
	}
	if executor.name != "sudo" || strings.Join(executor.args, " ") != "-n /usr/bin/systemctl restart docker" {
		t.Fatalf("executor command = %q %v", executor.name, executor.args)
	}
}

func TestRunRejectsUnknownUnitBeforeExecutor(t *testing.T) {
	withElevationTestState(t, "linux", true)
	executor := &elevationExecutor{}
	outcome, _, err := Run(context.Background(), executor, ServiceRestart, "attacker-controlled.service")
	if err == nil || outcome.State != Refused || executor.name != "" {
		t.Fatalf("outcome=%+v err=%v executor=%q, want refused without execution", outcome, err, executor.name)
	}
}

func TestRunUnsupportedPlatformIsTyped(t *testing.T) {
	withElevationTestState(t, "darwin", true)
	executor := &elevationExecutor{}
	outcome, output, err := Run(context.Background(), executor, ServiceRestart, "docker")
	if err != nil || output != nil || outcome.State != Unsupported {
		t.Fatalf("outcome=%+v output=%q err=%v", outcome, output, err)
	}
	withElevationTestState(t, "windows", true)
	outcome, output, err = Run(context.Background(), executor, ServiceRestart, "docker")
	if err != nil || output != nil || outcome.State != Unsupported {
		t.Fatalf("windows outcome=%+v output=%q err=%v", outcome, output, err)
	}
}

func TestRunRefusesNilExecutor(t *testing.T) {
	outcome, _, err := Run(context.Background(), nil, ServiceStart, "docker")
	if err == nil || outcome.State != Refused {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
}
