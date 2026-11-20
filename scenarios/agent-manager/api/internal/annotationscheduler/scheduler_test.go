package annotationscheduler

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity/schedule"
)

func TestResolveModeDefaultsToPassive(t *testing.T) {
	t.Setenv(ModeEnv, "")
	mode, err := resolveMode("")
	if err != nil {
		t.Fatal(err)
	}
	if mode != schedule.ModePassive {
		t.Fatalf("mode = %q, want passive", mode)
	}
}

func TestResolveModeUsesEnvironmentAndExplicitOption(t *testing.T) {
	t.Setenv(ModeEnv, "idle")
	mode, err := resolveMode("")
	if err != nil || mode != schedule.ModeIdle {
		t.Fatalf("env mode = %q, err = %v", mode, err)
	}
	mode, err = resolveMode(schedule.ModeEager)
	if err != nil || mode != schedule.ModeEager {
		t.Fatalf("explicit mode = %q, err = %v", mode, err)
	}
}

func TestResolveModeRejectsUnknown(t *testing.T) {
	t.Setenv(ModeEnv, "")
	if _, err := resolveMode(schedule.Mode("sometimes")); err == nil {
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
