package annotationscheduler

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity/schedule"
)

// fakeEnv is the injected environment the mode tests read instead of the
// process's.
func fakeEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestResolveModeDefaultsToPassive(t *testing.T) {
	mode, err := resolveMode("", fakeEnv(map[string]string{}))
	if err != nil {
		t.Fatal(err)
	}
	if mode != schedule.ModePassive {
		t.Fatalf("mode = %q, want passive", mode)
	}
}

func TestResolveModeUsesEnvironmentAndExplicitOption(t *testing.T) {
	env := fakeEnv(map[string]string{ModeEnv: "idle"})
	mode, err := resolveMode("", env)
	if err != nil || mode != schedule.ModeIdle {
		t.Fatalf("env mode = %q, err = %v", mode, err)
	}
	mode, err = resolveMode(schedule.ModeEager, env)
	if err != nil || mode != schedule.ModeEager {
		t.Fatalf("explicit mode = %q, err = %v", mode, err)
	}
}

func TestResolveModeRejectsUnknown(t *testing.T) {
	if _, err := resolveMode(schedule.Mode("sometimes"), fakeEnv(nil)); err == nil {
		t.Fatal("unknown mode was accepted")
	}
}

func TestStartRequiresService(t *testing.T) {
	if _, err := Start(context.Background(), Options{}); err == nil {
		t.Fatal("nil service was accepted")
	}
}

func TestRunnerCloseIsIdempotentWithoutComposedDependencies(t *testing.T) {
	runner := &Runner{}
	if err := runner.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := runner.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
