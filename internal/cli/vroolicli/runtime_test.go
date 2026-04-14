package vroolicli

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func newRuntimeTestApp(t *testing.T, root string) *App {
	t.Helper()

	return New(Config{
		VersionInfo:         VersionInfo{CLIVersion: "1.0.0", PlatformVersion: "2.0.0"},
		ResolveSourceRootFn: func() (string, error) { return root, nil },
		HomeDirFn:           func() (string, error) { return t.TempDir(), nil },
		CheckStalenessFn:    func() (buildinfo.StaleCheck, error) { return buildinfo.StaleCheck{}, nil },
		RebuildAndReexecFn:  func([]string) error { return nil },
		NewLoggerFn: func(rootcli.GlobalOptions, io.Writer) (*slog.Logger, func()) {
			return slog.New(slog.NewTextHandler(io.Discard, nil)), func() {}
		},
		DebugLogFn: func(*slog.Logger, string, ...any) {},
	})
}

func TestRunInfoTopLevelCommandHandlesHelp(t *testing.T) {
	app := New(Config{
		VersionInfo:         VersionInfo{CLIVersion: "1.0.0", PlatformVersion: "2.0.0"},
		ResolveSourceRootFn: func() (string, error) { return "/repo", nil },
		HomeDirFn:           func() (string, error) { return "/home/test", nil },
	})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx := app.NewCommandContext("/repo", rootcli.GlobalOptions{}, stdout, stderr)

	if err := app.runInfoTopLevelCommand(ctx, []string{"--help"}); err != nil {
		t.Fatalf("runInfoTopLevelCommand() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "vrooli info") {
		t.Fatalf("runInfoTopLevelCommand() help missing info usage: %q", got)
	}
}

func TestWriteVersionHumanOutput(t *testing.T) {
	var stdout bytes.Buffer
	if err := WriteVersion(&stdout, "/repo", rootcli.GlobalOptions{}, VersionInfo{
		CLIVersion:      "1.2.3",
		PlatformVersion: "4.5.6",
	}); err != nil {
		t.Fatalf("WriteVersion() error = %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"Vrooli CLI v1.2.3", "Vrooli Platform v4.5.6", "Root: /repo"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output %q", want, output)
		}
	}
}

func TestCommandEnvPreservesExistingSourceRootAndNoColor(t *testing.T) {
	t.Setenv("VROOLI_SOURCE_ROOT", "/custom/source")

	env := newRuntimeTestApp(t, "/repo").CommandEnv("/repo", rootcli.GlobalOptions{NoColor: true})
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "VROOLI_ROOT=/repo") {
		t.Fatalf("env missing VROOLI_ROOT: %s", got)
	}
	if !strings.Contains(got, buildinfo.SourceRootEnvVar+"=/custom/source") {
		t.Fatalf("env missing preserved source root: %s", got)
	}
	if !strings.Contains(got, "NO_COLOR=1") {
		t.Fatalf("env missing NO_COLOR: %s", got)
	}
}

func TestLocateTestGenieCLIResolutionOrder(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	app := newRuntimeTestApp(t, root)

	overrideCLI := testkitgo.WriteRelativeExecutable(t, root, filepath.Join("override", "test-genie"), "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_TEST_GENIE_CLI", overrideCLI)
	path, err := app.LocateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("LocateTestGenieCLI() override error = %v", err)
	}
	if path != overrideCLI {
		t.Fatalf("override path = %q", path)
	}

	t.Setenv("VROOLI_TEST_GENIE_CLI", "")
	homeCLI := testkitgo.WriteRelativeExecutable(t, home, filepath.Join(".vrooli", "bin", "test-genie"), "#!/usr/bin/env bash\nexit 0\n")
	app.LookPathFn = func(string) (string, error) { return "", exec.ErrNotFound }
	path, err = app.LocateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("LocateTestGenieCLI() home error = %v", err)
	}
	if path != homeCLI {
		t.Fatalf("home path = %q", path)
	}

	if err := os.Remove(homeCLI); err != nil {
		t.Fatalf("remove home CLI: %v", err)
	}
	pathCLI := testkitgo.WriteRelativeExecutable(t, root, filepath.Join("bin", "test-genie"), "#!/usr/bin/env bash\nexit 0\n")
	app.LookPathFn = func(string) (string, error) { return pathCLI, nil }
	path, err = app.LocateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("LocateTestGenieCLI() PATH error = %v", err)
	}
	if path != pathCLI {
		t.Fatalf("PATH path = %q", path)
	}
}

func TestOpenScenarioURLUsesPlatformLauncher(t *testing.T) {
	app := newRuntimeTestApp(t, t.TempDir())
	var opened scenarioexec.SubprocessSpec
	app.RunScenarioSubprocess = func(spec scenarioexec.SubprocessSpec) error {
		opened = spec
		return nil
	}
	app.LookPathFn = func(file string) (string, error) {
		switch runtime.GOOS {
		case "linux":
			if file == "xdg-open" {
				return "/usr/bin/xdg-open", nil
			}
		default:
			return "", exec.ErrNotFound
		}
		return "", exec.ErrNotFound
	}

	url := "http://localhost:1234"
	if err := app.OpenScenarioURL(url); err != nil {
		t.Fatalf("OpenScenarioURL() error = %v", err)
	}

	switch runtime.GOOS {
	case "linux":
		if opened.Name != "/usr/bin/xdg-open" || strings.Join(opened.Args, "|") != url {
			t.Fatalf("opened = %+v", opened)
		}
	case "darwin":
		if opened.Name != "open" || strings.Join(opened.Args, "|") != url {
			t.Fatalf("opened = %+v", opened)
		}
	case "windows":
		if opened.Name != "cmd" || strings.Join(opened.Args, "|") != "/c|start||"+url {
			t.Fatalf("opened = %+v", opened)
		}
	}
}

func TestLaunchDetachedScenarioPropagatesExpectedArgsAndEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached-process validation uses a bash fixture")
	}

	root := t.TempDir()
	app := newRuntimeTestApp(t, root)
	argsPath := filepath.Join(root, "args.txt")
	envPath := filepath.Join(root, "env.txt")
	executable := testkitgo.WriteRelativeExecutable(t, root, filepath.Join("bin", "fake-vrooli"), "#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > "+argsPath+"\nenv | sort > "+envPath+"\n")
	app.ScenarioExecutableFn = func() (string, error) { return executable, nil }

	t.Setenv("VROOLI_SANDBOX_ID", "sandbox-123")
	t.Setenv("VROOLI_SANDBOX_MERGED", "/merged")
	t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/alpha")
	t.Setenv("SANDBOX_MERGED_DIR", "/merged")
	t.Setenv(buildinfo.SourceRootEnvVar, "/source-root")

	if err := app.LaunchDetachedScenario(root, rootcli.GlobalOptions{JSON: true, Verbose: true, NoColor: true}, "start", "alpha"); err != nil {
		t.Fatalf("LaunchDetachedScenario() error = %v", err)
	}

	testkitgo.WaitForFile(t, argsPath)
	testkitgo.WaitForFile(t, envPath)

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	if got, want := strings.Join(strings.Fields(string(argsData)), "|"), "scenario|start|alpha|--json|--verbose|--no-color"; got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	env := string(envData)
	for _, forbidden := range []string{
		"VROOLI_SANDBOX_ID=",
		"VROOLI_SANDBOX_MERGED=",
		"VROOLI_SANDBOX_SCOPE=",
		"SANDBOX_MERGED_DIR=",
	} {
		if strings.Contains(env, forbidden) {
			t.Fatalf("env should not contain %q: %s", forbidden, env)
		}
	}
	for _, want := range []string{"VROOLI_ROOT=" + root, buildinfo.SourceRootEnvVar + "=/source-root", "NO_COLOR=1"} {
		if !strings.Contains(env, want) {
			t.Fatalf("missing %q in env %s", want, env)
		}
	}
}

func TestRunVersionDoesNotRequireRootResolution(t *testing.T) {
	app := newRuntimeTestApp(t, "/repo")
	app.ResolveSourceRootFn = func() (string, error) { return "", errors.New("boom") }
	app.CheckStalenessFn = nil

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vrooli CLI v1.0.0") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
