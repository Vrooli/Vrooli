package main

import (
	"testing"
	"time"
)

// Commands read time and env through ambient, so a test substitutes them
// instead of touching the real clock or process environment.
func TestCommandsReadTimeAndEnvThroughTheSeam(t *testing.T) {
	saved := ambient
	t.Cleanup(func() { ambient = saved })
	frozen := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	ambient = ambientSeam{
		Now:    func() time.Time { return frozen },
		Getenv: func(key string) string { return map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "t"}[key] },
		LookupEnv: func(key string) (string, bool) {
			value, ok := map[string]string{"VROOLI_AGENT_IDENTITY_TOKEN": "t"}[key]
			return value, ok
		},
	}
	if err := rejectRunIdentityLifecycleCommand("approve"); err == nil {
		t.Fatal("an agent identity from the injected environment was not refused an operator-only command")
	}
	if got := waitClock.now(); !got.Equal(frozen) {
		t.Fatalf("waitClock.now() = %v, want the injected %v", got, frozen)
	}
	ambient.LookupEnv = func(string) (string, bool) { return "", false }
	if err := rejectRunIdentityLifecycleCommand("approve"); err != nil {
		t.Fatalf("with no identity token an operator command is refused: %v", err)
	}
}
