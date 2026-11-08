package elevation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type elevationExecutor struct {
	name   string
	args   []string
	output []byte
	err    error
}

func (e *elevationExecutor) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	e.name = name
	e.args = append([]string(nil), args...)
	return e.output, e.err
}
func withElevationOS(t *testing.T, name string) {
	t.Helper()
	old := osNameFn
	osNameFn = func() string { return name }
	t.Cleanup(func() { osNameFn = old })
}
func TestRunGrantedUsesControlPlaneWithoutSudo(t *testing.T) {
	withElevationOS(t, "linux")
	executor := &elevationExecutor{output: []byte(`{"version":"v1","action":"host.service.recover","status":"verified","evidence":{"active":true}}`)}
	outcome, _, err := Run(context.Background(), executor, ServiceRestart, "docker")
	if err != nil || outcome.State != Granted || executor.name != "vrooli" || strings.Join(executor.args, " ") != "host service restart docker --json" {
		t.Fatalf("outcome=%+v err=%v command=%s %v", outcome, err, executor.name, executor.args)
	}
}
func TestRunMissingBrokerReturnsSetupRecoveryWithoutSudoFallback(t *testing.T) {
	withElevationOS(t, "linux")
	executor := &elevationExecutor{err: errors.New("broker unavailable")}
	outcome, _, _ := Run(context.Background(), executor, ServiceRestart, "docker")
	if outcome.State != NeedsSetup || outcome.Command != "vrooli setup broker --sudo-mode=ask" || executor.name != "vrooli" {
		t.Fatalf("outcome=%+v command=%s", outcome, executor.name)
	}
}
func TestRunRejectsInvalidAndUnverifiedReceipts(t *testing.T) {
	withElevationOS(t, "linux")
	for _, output := range []string{"ok", `{"status":"verified"}`, `{"version":"v1","action":"host.service.recover","status":"verified","evidence":{"active":false}}`} {
		outcome, _, _ := Run(context.Background(), &elevationExecutor{output: []byte(output)}, ServiceRestart, "docker")
		if outcome.State == Granted {
			t.Fatalf("accepted invalid receipt %q", output)
		}
	}
}
func TestRunRejectsUnknownUnitAndStopBeforeExecution(t *testing.T) {
	withElevationOS(t, "linux")
	for _, r := range []struct {
		action Action
		unit   string
	}{{ServiceRestart, "attacker.service"}, {ServiceStop, "docker"}} {
		executor := &elevationExecutor{}
		outcome, _, err := Run(context.Background(), executor, r.action, r.unit)
		if err == nil || outcome.State != Refused || executor.name != "" {
			t.Fatalf("outcome=%+v err=%v executor=%s", outcome, err, executor.name)
		}
	}
}
func TestRunUnsupportedPlatformIsTyped(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			withElevationOS(t, platform)
			executor := &elevationExecutor{}
			outcome, _, err := Run(context.Background(), executor, ServiceRestart, "docker")
			if err != nil || outcome.State != Unsupported || executor.name != "" {
				t.Fatalf("outcome=%+v err=%v", outcome, err)
			}
		})
	}
}
func TestRunRefusesNilExecutor(t *testing.T) {
	outcome, _, err := Run(context.Background(), nil, ServiceStart, "docker")
	if err == nil || outcome.State != Refused {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
}
