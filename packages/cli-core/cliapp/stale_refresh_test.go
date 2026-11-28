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

// staleDemoChecker is a stale CLI whose synchronous rebuild fails the test and
// whose background refresh launches are recorded instead of executed.
func staleDemoChecker(t *testing.T) (*cliutil.StaleChecker, *[]*exec.Cmd) {
	t.Helper()
	repoRoot := t.TempDir()
	sourceRoot := filepath.Join(repoRoot, "scenarios", "demo", "cli")
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir cli-core installer: %v", err)
	}
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir CLI source: %v", err)
	}
	controlPlane := filepath.Join(t.TempDir(), "vrooli")
	if err := os.WriteFile(controlPlane, []byte("test executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cliutil.LaunchRuntimeBinaryEnv, controlPlane)
	var launched []*exec.Cmd
	return &cliutil.StaleChecker{
		BuildFingerprint: "installed",
		BuildSourceRoot:  sourceRoot,
		InstallKind:      "scenario",
		InstallName:      "demo",
		RefreshStateDir:  t.TempDir(),
		FingerprintFunc:  func(cliutil.FreshnessSpec) (string, error) { return "current", nil },
		LookPathFunc:     func(string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(*exec.Cmd) error {
			t.Error("a latency-bound command waited on a synchronous rebuild")
			return errors.New("synchronous rebuild")
		},
		BackgroundStarter: func(cmd *exec.Cmd) error { launched = append(launched, cmd); return nil },
	}, &launched
}

func TestAppRunsLatencyBoundCommandWithoutWaitingForRebuild(t *testing.T) {
	for name, build := range map[string]func(Command, *cliutil.StaleChecker) *App{
		"top-level command": func(cmd Command, checker *cliutil.StaleChecker) *App {
			cmd.NeedsAPI = true
			return NewApp(AppOptions{Name: "demo", StaleChecker: checker,
				Commands: []CommandGroup{{Title: "Demo", Commands: []Command{cmd}}}})
		},
		"subcommand": func(cmd Command, checker *cliutil.StaleChecker) *App {
			return NewApp(AppOptions{Name: "demo", StaleChecker: checker,
				SubcommandGroups: []SubcommandGroup{{Name: "brief", Subcommands: []Command{cmd}}}})
		},
	} {
		t.Run(name, func(t *testing.T) {
			checker, launched := staleDemoChecker(t)
			dispatched := false
			app := build(Command{
				Name: "hook", Description: "Hook entry point", LatencyBound: true,
				Run: func([]string) error { dispatched = true; return nil },
			}, checker)
			args := []string{"hook"}
			if name == "subcommand" {
				args = []string{"brief", "hook"}
			}

			if err := app.Run(args); err != nil {
				t.Fatalf("App.Run() error = %v", err)
			}
			if !dispatched {
				t.Fatal("latency-bound command did not run the current binary")
			}
			if len(*launched) != 1 {
				t.Fatalf("background refreshes started = %d, want 1", len(*launched))
			}
		})
	}
}

func TestAppStillRebuildsOrdinaryCommandSynchronously(t *testing.T) {
	checker, launched := staleDemoChecker(t)
	rebuilt := false
	checker.CommandRunner = func(*exec.Cmd) error { rebuilt = true; return nil }
	checker.Reexec = func(string, []string) error { return nil }
	dispatched := false
	app := NewApp(AppOptions{Name: "demo", StaleChecker: checker,
		SubcommandGroups: []SubcommandGroup{{Name: "brief", Subcommands: []Command{{
			Name: "show", Description: "Show", Run: func([]string) error { dispatched = true; return nil },
		}}}}})

	if err := app.Run([]string{"brief", "show"}); err != nil {
		t.Fatalf("App.Run() error = %v", err)
	}
	if !rebuilt {
		t.Fatal("ordinary stale command did not rebuild synchronously")
	}
	if dispatched {
		t.Fatal("ordinary command ran the stale binary instead of the rebuilt one")
	}
	if len(*launched) != 0 {
		t.Fatalf("ordinary command started %d background refreshes", len(*launched))
	}
}

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
		FingerprintFunc:  func(cliutil.FreshnessSpec) (string, error) { return "current", nil },
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
