package cliapp

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestStandardResourceEnv(t *testing.T) {
	env := StandardResourceEnv("example-resource", ResourceEnvOptions{
		ExtraSourceRootEnvVars:   []string{"CUSTOM_SOURCE_ROOT"},
		ExtraControlPlaneEnvVars: []string{"CUSTOM_VROOLI_BIN"},
	})

	wantSource := []string{"VROOLI_CLI_SOURCE_ROOT", "EXAMPLE_RESOURCE_CLI_SOURCE_ROOT", "CUSTOM_SOURCE_ROOT"}
	if !reflect.DeepEqual(env.SourceRootEnvVars, wantSource) {
		t.Fatalf("SourceRootEnvVars = %v, want %v", env.SourceRootEnvVars, wantSource)
	}

	wantControl := []string{"VROOLI_CLI_BIN", "EXAMPLE_RESOURCE_VROOLI_CLI_BIN", "CUSTOM_VROOLI_BIN"}
	if !reflect.DeepEqual(env.ControlPlaneEnvVars, wantControl) {
		t.Fatalf("ControlPlaneEnvVars = %v, want %v", env.ControlPlaneEnvVars, wantControl)
	}
}

func TestResourceAppDelegatesLifecycleCommand(t *testing.T) {
	var gotExec string
	var gotArgs []string

	env := StandardResourceEnv("postgres", ResourceEnvOptions{})
	app, err := NewResourceApp(ResourceOptions{
		Name:                "postgres",
		Version:             "0.1.0",
		Description:         "Postgres resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    "unknown",
		CommandRunner: func(cmd *exec.Cmd) error {
			gotExec = cmd.Path
			gotArgs = append([]string(nil), cmd.Args...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewResourceApp: %v", err)
	}
	app.SetCommands(app.StandardLifecycleCommands())

	t.Setenv("VROOLI_CLI_BIN", "/tmp/fake-vrooli")
	if err := app.CLI.Run([]string{"status", "--fast"}); err != nil {
		t.Fatalf("CLI.Run: %v", err)
	}

	if gotExec != "/tmp/fake-vrooli" {
		t.Fatalf("executed binary = %q, want %q", gotExec, "/tmp/fake-vrooli")
	}
	wantArgs := []string{"/tmp/fake-vrooli", "resource", "status", "postgres", "--fast"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("delegated args = %v, want %v", gotArgs, wantArgs)
	}
}

func TestResourceAppUsesStaleCheckerForDelegatingCommands(t *testing.T) {
	temp := t.TempDir()
	srcRoot := filepath.Join(temp, "resources", "postgres", "cli")
	if err := os.MkdirAll(filepath.Join(temp, "packages", "cli-core"), 0o755); err != nil {
		t.Fatalf("mkdir installer path: %v", err)
	}
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("mkdir src root: %v", err)
	}

	env := StandardResourceEnv("postgres", ResourceEnvOptions{})
	app, err := NewResourceApp(ResourceOptions{
		Name:                "postgres",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    "old",
		BuildTimestamp:      "ts",
		BuildSourceRoot:     srcRoot,
	})
	if err != nil {
		t.Fatalf("NewResourceApp: %v", err)
	}
	if got := strings.Join(app.StaleChecker.FreshnessInputs, ","); got != "cli/**,resource.json,../../packages/cli-core" {
		t.Fatalf("freshness inputs = %q", got)
	}
	restarted := false
	app.StaleChecker.FingerprintFunc = func(spec cliutil.FreshnessSpec) (string, error) {
		return "new", nil
	}
	app.StaleChecker.LookPathFunc = func(file string) (string, error) {
		return "/usr/bin/go", nil
	}
	app.StaleChecker.CommandRunner = func(cmd *exec.Cmd) error { return nil }
	app.StaleChecker.Reexec = func(executable string, args []string) error {
		restarted = true
		return nil
	}

	app.SetCommands(app.StandardLifecycleCommands())
	t.Setenv("VROOLI_CLI_BIN", "/tmp/fake-vrooli")
	originalArgs := os.Args
	os.Args = []string{"resource-postgres", "status"}
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	if err := app.CLI.Run([]string{"status"}); err != nil {
		t.Fatalf("CLI.Run: %v", err)
	}
	if !restarted {
		t.Fatal("expected stale checker to trigger reexec for resource command")
	}
	if got := app.StaleChecker.ReexecArgs; !reflect.DeepEqual(got, []string{"status"}) {
		t.Fatalf("ReexecArgs = %v, want %v", got, []string{"status"})
	}
}

func TestResourceAppStandardLifecycleCommandsExposeCompleteSurface(t *testing.T) {
	env := StandardResourceEnv("postgres", ResourceEnvOptions{})
	app, err := NewResourceApp(ResourceOptions{
		Name:                "postgres",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
	})
	if err != nil {
		t.Fatalf("NewResourceApp: %v", err)
	}

	groups := app.StandardLifecycleCommands()
	if len(groups) != 1 {
		t.Fatalf("group count = %d, want 1", len(groups))
	}

	var got []string
	for _, cmd := range groups[0].Commands {
		got = append(got, cmd.Name)
	}
	want := []string{"info", "status", "install", "uninstall", "start", "stop", "restart", "logs"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command names = %v, want %v", got, want)
	}
}
