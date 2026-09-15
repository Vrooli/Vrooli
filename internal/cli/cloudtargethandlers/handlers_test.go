package cloudtargethandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cloudtarget"
	"github.com/vrooli/vrooli/internal/privilegebroker"
)

func runVerb(t *testing.T, store *cloudtarget.Store, args ...string) (map[string]any, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx := &rootcli.CommandContext{Root: t.TempDir(), Stdout: &stdout, Stderr: &stderr, Context: context.Background()}
	err := Run(ctx, Deps{Store: func() (*cloudtarget.Store, error) { return store, nil }}, args)
	exit := 0
	if err != nil {
		var exitErr rootcli.ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("Run(%q) returned an untyped error: %v (stderr %s)", args, err, stderr.String())
		}
		exit = exitErr.Code
	}
	var value map[string]any
	if strings.TrimSpace(stdout.String()) != "" {
		if err := json.Unmarshal(stdout.Bytes(), &value); err != nil {
			t.Fatalf("stdout is not JSON: %s", stdout.String())
		}
	}
	return value, exit
}

func TestRegisteredCommandPathsMatchManifestGroups(t *testing.T) {
	paths := RegisteredCommandPaths()
	want := []string{"cloud-target receipt get", "cloud-target release verify", "cloud-target release stage", "cloud-target release activate", "cloud-target release rollback", "cloud-target release list", "cloud-target release prune", "cloud-target data inventory", "cloud-target data backup", "cloud-target data restore", "cloud-target data verify", "cloud-target host observe", "cloud-target host repair", "cloud-target credential ingest", "cloud-target credential acknowledge", "cloud-target credential revoke", "cloud-target edge route-apply", "cloud-target edge route-rollback", "cloud-target edge route-status"}
	if strings.Join(paths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("paths = %q", paths)
	}
}

// [REQ:STC-P0-007] Refusals print a typed JSON error and exit 2 without
// touching the store; a missing receipt is a refusal, not a crash.
func TestVerbsPrintTypedErrorsWithExitCodes(t *testing.T) {
	store := cloudtarget.NewStore(filepath.Join(t.TempDir(), "deployments"))
	value, exit := runVerb(t, store, "receipt", "get", "--deployment", "dep", "--operation", "op", "--step", "stage")
	if exit != cloudtarget.ExitRefused {
		t.Fatalf("exit = %d value=%v", exit, value)
	}
	errValue, _ := value["error"].(map[string]any)
	if errValue["code"] != cloudtarget.CodeReceiptNotFound {
		t.Fatalf("error = %v", value)
	}
	value, exit = runVerb(t, store, "release", "activate", "--deployment", "dep", "--operation", "op", "--step", "activate", "--fence", "not-a-number", "--release", strings.Repeat("a", 64))
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeInvalidArgument {
		t.Fatalf("bad fence exit=%d value=%v", exit, value)
	}
	value, exit = runVerb(t, store, "release", "activate", "--deployment", "dep", "--operation", "op", "--step", "activate", "--fence", "1", "--release", strings.Repeat("a", 64))
	if exit != cloudtarget.ExitRefused {
		t.Fatalf("unstaged activate exit=%d value=%v", exit, value)
	}
	receipt, _ := value["receipt"].(map[string]any)
	if receipt["outcome"] != string(cloudtarget.OutcomeFailed) || value["error"].(map[string]any)["code"] != cloudtarget.CodeReleaseNotStaged {
		t.Fatalf("unstaged activate value=%v", value)
	}
	value, exit = runVerb(t, store, "release", "list", "--deployment", "dep")
	if exit != 0 || value["deployment_id"] != "dep" {
		t.Fatalf("list exit=%d value=%v", exit, value)
	}
}

func TestHostObserveUsesTargetOwnerContract(t *testing.T) {
	runner := &recordingRunner{}
	value, exit := runVerbWith(t, Deps{Runner: runner}, "host", "observe", "--kind", "file", "--arg", "--", "--arg", "/etc/os-release")
	if exit != 0 {
		t.Fatalf("observe exit=%d value=%v", exit, value)
	}
	result, ok := value["result"].(map[string]any)
	if !ok || result["kind"] != "file" {
		t.Fatalf("observe result=%v", value)
	}
	if len(runner.calls) != 1 || strings.Join(runner.calls[0], " ") != "cat -- /etc/os-release" {
		t.Fatalf("owner argv=%v", runner.calls)
	}
}

// [REQ:STC-P0-036] The edge verbs are reachable in both spellings, refuse a
// spec that routes a private listener before touching the proxy, and print
// the transactional report beside the receipt.
func TestEdgeRouteVerbsThroughManifest(t *testing.T) {
	root := t.TempDir()
	store := cloudtarget.NewStore(filepath.Join(root, "deployments"))
	confDir := filepath.Join(root, "etc", "caddy", "conf.d")
	main := filepath.Join(root, "etc", "caddy", "Caddyfile")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(main, []byte("other.example.test {\n  respond \"ok\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &recordingRunner{}
	deps := Deps{Store: func() (*cloudtarget.Store, error) { return store, nil }, Broker: func() cloudtarget.BrokerClient { return unavailableBroker{} }, Runner: runner}
	pathFlags := []string{"--caddy-main", main, "--caddy-conf-dir", confDir, "--caddy-data-dir", filepath.Join(root, "data")}
	spec := `{"schema_version":1,"deployment_id":"dep","domain":"app.example.test","routes":[{"host":"app.example.test","upstream_port":3000,"listener_id":"app/ui"}],"snippet":"app.example.test {\n  reverse_proxy 127.0.0.1:3000\n}\n","acme_environment":"staging"}`
	value, exit := runVerbWith(t, deps, append([]string{"edge", "route", "apply", "--deployment", "dep", "--operation", "op-1", "--step", "edge", "--fence", "1", "--spec", spec}, pathFlags...)...)
	if exit != 0 {
		t.Fatalf("apply exit=%d value=%v", exit, value)
	}
	receipt, _ := value["receipt"].(map[string]any)
	report, _ := value["report"].(map[string]any)
	if receipt["outcome"] != "succeeded" || report["import_line_added"] != true || report["rolled_back"] != false {
		t.Fatalf("apply value=%v", value)
	}
	want := [][]string{{"caddy", "validate", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"}, {"systemctl", "reload", "caddy"}}
	if len(runner.calls) != 2 || strings.Join(runner.calls[0], " ") != strings.Join(want[0], " ") || strings.Join(runner.calls[1], " ") != strings.Join(want[1], " ") {
		t.Fatalf("argv calls = %q", runner.calls)
	}
	private := strings.Replace(strings.Replace(spec, "3000", "18767", -1), "op-1", "op-2", -1)
	value, exit = runVerbWith(t, deps, append([]string{"edge", "route-apply", "--deployment", "dep", "--operation", "op-2", "--step", "edge", "--fence", "2", "--spec", private}, pathFlags...)...)
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeEdgePrivateListener || len(runner.calls) != 2 {
		t.Fatalf("private listener exit=%d value=%v calls=%d", exit, value, len(runner.calls))
	}
	value, exit = runVerbWith(t, deps, append([]string{"edge", "route", "status", "--deployment", "dep"}, pathFlags...)...)
	if exit != 0 || value["snippet_present"] != true || value["import_line_present"] != true {
		t.Fatalf("status exit=%d value=%v", exit, value)
	}
	value, exit = runVerbWith(t, deps, append([]string{"edge", "route", "rollback", "--deployment", "dep", "--operation", "op-3", "--step", "rb", "--fence", "3"}, pathFlags...)...)
	if exit != cloudtarget.ExitRefused || value["error"].(map[string]any)["code"] != cloudtarget.CodeEdgeRollbackNotEligible {
		t.Fatalf("rollback exit=%d value=%v", exit, value)
	}
}

type unavailableBroker struct{}

func (unavailableBroker) Available() bool { return false }

func (unavailableBroker) Do(context.Context, privilegebroker.Request) (privilegebroker.Result, error) {
	return privilegebroker.Result{}, errors.New("unavailable")
}

type recordingRunner struct{ calls [][]string }

func (r *recordingRunner) LookPath(name string) (string, error) { return name, nil }

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	return []byte("Valid configuration"), nil
}

func runVerbWith(t *testing.T, deps Deps, args ...string) (map[string]any, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx := &rootcli.CommandContext{Root: t.TempDir(), Stdout: &stdout, Stderr: &stderr, Context: context.Background()}
	err := Run(ctx, deps, args)
	exit := 0
	if err != nil {
		var exitErr rootcli.ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("Run(%q) returned an untyped error: %v (stderr %s)", args, err, stderr.String())
		}
		exit = exitErr.Code
	}
	var value map[string]any
	if strings.TrimSpace(stdout.String()) != "" {
		if err := json.Unmarshal(stdout.Bytes(), &value); err != nil {
			t.Fatalf("stdout is not JSON: %s", stdout.String())
		}
	}
	return value, exit
}
