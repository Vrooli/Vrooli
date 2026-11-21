package vroolicli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestCLIRefreshCommandRejectsNoncanonicalExecutableBeforeInstall(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testkitgo.WriteRepoContract(t, root, "scenarios")
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithCLI(&scenario.CLIConfig{
			Enabled: true,
			Command: "alpha",
			Adapter: scenario.CLIAdapterConfig{Kind: "go_module", ModuleDir: "cli"},
		}),
	))
	testscenario.WriteScenarioCLIGoMod(t, root, "alpha", "alpha/cli")

	app := newRuntimeTestApp(t, root)
	app.HomeDirFn = func() (string, error) { return home, nil }
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"cli-refresh", "scenario", "alpha", filepath.Join(home, "bin", "unrelated")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("cli-refresh exit code = 0, expected noncanonical target rejection; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "running CLI path does not match declared scenario CLI") {
		t.Fatalf("cli-refresh stderr = %q, want manager canonical-target rejection", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "bin", "alpha")); !os.IsNotExist(err) {
		t.Fatalf("canonical CLI exists after rejecting the unrelated path: stat err=%v", err)
	}
}

func TestCLIRefreshDelegatesToHostWhenRunningInSandbox(t *testing.T) {
	app := newRuntimeTestApp(t, t.TempDir())
	app.HomeDirFn = func() (string, error) { return t.TempDir(), nil }
	var received lifecycle.CLIRefreshRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/host/vrooli/cli-refresh" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"exit_code":0,"stdout":"host installer output\n","stderr":"host installer warning\n"}`))
	}))
	defer server.Close()
	t.Setenv(lifecycle.HostLifecycleBaseEnv, server.URL)
	t.Setenv("VROOLI_SANDBOX_MERGED", "/repo")

	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"cli-refresh", "scenario", "scenario-qa", "/home/agent/.vrooli/bin/scenario-qa"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("sandboxed cli-refresh code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	want := lifecycle.CLIRefreshRequest{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/home/agent/.vrooli/bin/scenario-qa"}
	if received != want {
		t.Fatalf("host request = %+v, want %+v", received, want)
	}
	if stdout.String() != "host installer output\n" || stderr.String() != "host installer warning\n" {
		t.Fatalf("delegated output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCLIRefreshDoesNotFallBackLocallyAfterHostFailure(t *testing.T) {
	app := newRuntimeTestApp(t, t.TempDir())
	app.HomeDirFn = func() (string, error) { return t.TempDir(), nil }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"exit_code":1,"stderr":"capacity admission denied"}`))
	}))
	defer server.Close()
	t.Setenv(lifecycle.HostLifecycleBaseEnv, server.URL)
	t.Setenv("VROOLI_SANDBOX_MERGED", "/repo")

	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"cli-refresh", "scenario", "scenario-qa", "/home/agent/.vrooli/bin/scenario-qa"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "capacity admission denied") {
		t.Fatalf("sandboxed cli-refresh code=%d stderr=%q; want host denial", code, stderr.String())
	}
}
