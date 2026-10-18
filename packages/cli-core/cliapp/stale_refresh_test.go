package cliapp

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestAppDoesNotDispatchWhenStaleRefreshIsDenied(t *testing.T) {
	dispatched := false
	repoRoot := t.TempDir()
	sourceRoot := filepath.Join(repoRoot, "scenarios", "demo", "cli")
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir cli-core installer: %v", err)
	}
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir CLI source: %v", err)
	}
	checker := &cliutil.StaleChecker{
		BuildFingerprint: "installed",
		BuildSourceRoot:  sourceRoot,
		InstallKind:      "scenario",
		InstallName:      "demo",
		FingerprintFunc: func(cliutil.FreshnessSpec) (string, error) { return "current", nil },
		LookPathFunc:     func(string) (string, error) { return "/usr/bin/vrooli", nil },
		CommandRunner: func(*exec.Cmd) error {
			return errors.New("capacity denied")
		},
	}
	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{Title: "Demo", Commands: []Command{{
			Name: "read", Description: "Read data", NeedsAPI: true,
			Run: func([]string) error { dispatched = true; return nil },
		}}}},
		StaleChecker: checker,
	})

	err := app.Run([]string{"read"})
	if err == nil || !strings.Contains(err.Error(), "capacity denied") {
		t.Fatalf("App.Run() error = %v, want refresh denial", err)
	}
	if dispatched {
		t.Fatal("original CLI command dispatched after stale refresh denial")
	}
}
