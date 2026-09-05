package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleStepEnvCarriesToolchainFloor(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOMAXPROCS", "")
	env := lifecycleStepEnv(phasesSetup, map[string]string{"FOO": "bar"})
	goflags := envValue(env, "GOFLAGS")
	if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") {
		t.Fatalf("GOFLAGS = %q, want the inherited flag composed with a -p width", goflags)
	}
	if envValue(env, "GOMAXPROCS") == "" {
		t.Fatalf("GOMAXPROCS is not set: %v", env)
	}
	if envValue(env, "FOO") != "bar" {
		t.Fatalf("override lost: %v", env)
	}
	tempRoot := t.TempDir()
	t.Setenv("VROOLI_HOME", tempRoot)
	t.Setenv("GOTMPDIR", "")
	env = lifecycleStepEnv(phasesSetup, nil)
	wantTemp := filepath.Join(tempRoot, "tmp", "go-work")
	if envValue(env, "GOTMPDIR") != wantTemp {
		t.Fatalf("GOTMPDIR = %q, want %q", envValue(env, "GOTMPDIR"), wantTemp)
	}
	if _, err := os.Stat(wantTemp); err != nil {
		t.Fatalf("GOTMPDIR was not created: %v", err)
	}
	t.Setenv("GOTMPDIR", filepath.Join(t.TempDir(), "operator-go-work"))
	if got := envValue(lifecycleStepEnv(phasesSetup, nil), "GOTMPDIR"); got != os.Getenv("GOTMPDIR") {
		t.Fatalf("explicit GOTMPDIR = %q, want %q", got, os.Getenv("GOTMPDIR"))
	}
}
