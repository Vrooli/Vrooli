package vroolicli

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
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

func TestLocateTestGenieCLIUsesManifestDrivenInstalledPath(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	app := newRuntimeTestApp(t, root)
	testkitgo.WriteRepoContract(t, root, "scenarios")
	testscenario.WriteScenarioService(t, root, "test-genie", testscenario.ScenarioServiceManifest(
		"test-genie",
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: "test-genie",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(root, "scenarios", "test-genie", "cli", "go.mod"), "module test-genie/cli\n")

	app.EnsureScenarioCLIFn = func(root, home, name string) error { return nil }
	expected := testkitgo.WriteRelativeExecutable(t, home, filepath.Join(".vrooli", "bin", "test-genie"), shelltest.BashShebang()+"exit 0\n")

	path, err := app.LocateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("LocateTestGenieCLI() error = %v", err)
	}
	if path != expected {
		t.Fatalf("path = %q, want %q", path, expected)
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
		switch hostinventory.CurrentPlatform() {
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

	switch hostinventory.CurrentPlatform() {
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
	if hostinventory.CurrentPlatform() == "windows" {
		t.Skip("detached-process validation uses a bash fixture")
	}

	root := t.TempDir()
	app := newRuntimeTestApp(t, root)
	argsPath := filepath.Join(root, "args.txt")
	envPath := filepath.Join(root, "env.txt")
	executable := testkitgo.WriteRelativeExecutable(t, root, filepath.Join("bin", "fake-vrooli"), shelltest.BashShebang()+"printf '%s\\n' \"$@\" > "+argsPath+"\nenv | sort > "+envPath+"\n")
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

func TestEnsureScenarioCLIWarnsWhenPreviousPathWasNonCanonical(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testkitgo.WriteRepoContract(t, root, "scenarios")
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: "alpha",
			Adapter: scenario.CLIAdapterConfig{
				Kind:      "go_module",
				ModuleDir: "cli",
			},
		}),
	))
	testkitgo.WriteFile(t, filepath.Join(root, "scenarios", "alpha", "cli", "go.mod"), "module alpha/cli\n")

	app := newRuntimeTestApp(t, root)
	app.HomeDirFn = func() (string, error) { return home, nil }
	app.EnsureScenarioCLIFn = func(root, home, name string) error {
		path := filepath.Join(home, ".vrooli", "bin", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("installed"), 0o755)
	}
	lookups := 0
	nonCanonical := testkitgo.WriteRelativeExecutable(t, home, filepath.Join(".local", "bin", "alpha"), shelltest.BashShebang()+"exit 0\n")
	app.LookPathFn = func(file string) (string, error) {
		lookups++
		if lookups == 1 {
			return nonCanonical, nil
		}
		return filepath.Join(home, ".vrooli", "bin", file), nil
	}

	var stderr bytes.Buffer
	ctx := &CommandContext{
		Root:   root,
		Stdout: io.Discard,
		Stderr: &stderr,
		app:    app,
	}
	if err := app.ensureScenarioCLI(ctx, "alpha"); err != nil {
		t.Fatalf("ensureScenarioCLI() error = %v", err)
	}

	output := stderr.String()
	for _, want := range []string{
		"previously resolved to a non-canonical CLI path",
		nonCanonical,
		filepath.Join(home, ".vrooli", "bin", "alpha"),
		"hash -r",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in warning %q", want, output)
		}
	}
}

// TestEnsureScenarioCLIResolvesVariantToBareScenario guards the fix for the
// shadow-start bug the P1 live spike surfaced: a `--instance shadow` start
// arrives here as the instance slug "scenario@shadow", but the scenario CLI and
// its source tree are variant-independent (scenarios/<scenario>), so CLI
// resolution must use the bare scenario name — otherwise it looks for
// scenarios/scenario@shadow and fails.
func TestEnsureScenarioCLIResolvesVariantToBareScenario(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	app := newRuntimeTestApp(t, root)
	app.HomeDirFn = func() (string, error) { return home, nil }

	var gotName string
	app.EnsureScenarioCLIFn = func(_, _, name string) error {
		gotName = name
		return nil
	}
	app.LookPathFn = func(file string) (string, error) {
		return filepath.Join(home, ".vrooli", "bin", file), nil
	}

	ctx := &CommandContext{Root: root, Stdout: io.Discard, Stderr: io.Discard, app: app}
	if err := app.ensureScenarioCLI(ctx, "alpha@shadow"); err != nil {
		t.Fatalf("ensureScenarioCLI(alpha@shadow) error = %v", err)
	}
	if gotName != "alpha" {
		t.Fatalf("scenario CLI ensure used %q; want bare scenario %q (the source tree is variant-independent)", gotName, "alpha")
	}
}

func TestFormatScenarioCLIInstallWarningForPersistentMismatch(t *testing.T) {
	msg := formatScenarioCLIInstallWarning(
		cliinstall.InstallLocationStatus{},
		cliinstall.InstallLocationStatus{
			Command:           "alpha",
			CanonicalPath:     "/home/user/.vrooli/bin/alpha",
			ResolvedPath:      "/home/user/.local/bin/alpha",
			CanonicalExists:   true,
			Resolved:          true,
			ResolvedCanonical: false,
		},
	)
	for _, want := range []string{
		"alpha resolves to a non-canonical CLI path",
		"/home/user/.local/bin/alpha",
		"/home/user/.vrooli/bin/alpha",
		"hash -r",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in warning %q", want, msg)
		}
	}
}
