package shell

import (
	"strings"
	"testing"
)

func TestShellCommandStampsToolchainFloorForToolchainPrograms(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOMAXPROCS", "")
	for _, name := range []string{"go", "/usr/local/go/bin/go", "pnpm", "npm", "cargo", "uv", "vite", "tsc"} {
		cmd := Command(Spec{Name: name, Args: []string{"build"}})
		goflags := envValue(cmd.Env, "GOFLAGS")
		if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") || envValue(cmd.Env, "GOMAXPROCS") == "" {
			t.Fatalf("%s: env lacks the floor: GOFLAGS=%q GOMAXPROCS=%q", name, goflags, envValue(cmd.Env, "GOMAXPROCS"))
		}
	}
	explicit := Command(Spec{Name: "go", Env: []string{"GOFLAGS=-p=1", "HOME=/h"}})
	if envValue(explicit.Env, "GOFLAGS") != "-p=1" || envValue(explicit.Env, "HOME") != "/h" {
		t.Fatalf("explicit env was replaced: %v", explicit.Env)
	}
	other := Command(Spec{Name: "ls"})
	if other.Env != nil {
		t.Fatalf("non-toolchain program gained an env: %v", other.Env)
	}
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		if strings.HasPrefix(entry, key+"=") {
			return strings.TrimPrefix(entry, key+"=")
		}
	}
	return ""
}
