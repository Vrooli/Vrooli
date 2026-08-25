package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestGlossarySupportsEmptyAndFilteredQueries(t *testing.T) {
	server := NewServer()
	for _, query := range []string{"", "postgres", "does-not-exist"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/glossary?q="+query, nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"entries"`) {
			t.Fatalf("query %q: %d %s", query, w.Code, w.Body.String())
		}
	}
}

func TestInspectHostReadinessBranches(t *testing.T) {
	if got := inspectToolReadiness(testHost("optional", "optional")); got.Status != "deferred" {
		t.Fatalf("unselected tool = %#v", got)
	}
	unsupported := testHost("unsupported", "opted_in")
	unsupported.Platforms = []string{"not-this-platform"}
	if got := inspectToolReadiness(unsupported); got.Status != "unsupported" {
		t.Fatalf("unsupported tool = %#v", got)
	}
	noCommand := testHost("no-command", "required")
	noCommand.Platforms = []string{runtime.GOOS}
	if got := inspectToolReadiness(noCommand); got.Status != "unsupported" {
		t.Fatalf("commandless tool = %#v", got)
	}
	missing := testHost("missing", "required")
	missing.Commands = []string{"vrooli-test-missing"}
	if got := inspectToolReadiness(missing); got.Status != "missing" {
		t.Fatalf("missing tool = %#v", got)
	}
	ready := testHost("ready", "required")
	ready.Commands = []string{"true"}
	if got := inspectToolReadiness(ready); got.Status != "ready" {
		t.Fatalf("ready tool = %#v", got)
	}

	root := t.TempDir()
	if got := inspectSafeguardReadiness(root, testHost("optional", "optional")); got.Status != "deferred" {
		t.Fatalf("unselected safeguard = %#v", got)
	}
	unsupportedSafeguard := testHost("unsupported", "required")
	unsupportedSafeguard.Platforms = []string{"not-this-platform"}
	if got := inspectSafeguardReadiness(root, unsupportedSafeguard); got.Status != "unsupported" {
		t.Fatalf("unsupported safeguard = %#v", got)
	}
	missingSafeguard := testHost("missing", "required")
	missingSafeguard.Platforms = []string{runtime.GOOS}
	if got := inspectSafeguardReadiness(root, missingSafeguard); got.Status != "unsupported" {
		t.Fatalf("missing manifest safeguard = %#v", got)
	}
	manifestDir := filepath.Join(root, "internal", "safeguards", "safe")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	verified := filepath.Join(root, "verified")
	manifest := `{"name":"safe","verificationCheck":{"files":["` + verified + `"]}}`
	if err := os.WriteFile(filepath.Join(manifestDir, "safeguard.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	safe := testHost("safe", "required")
	safe.Platforms = []string{runtime.GOOS}
	if got := inspectSafeguardReadiness(root, safe); got.Status != "missing" {
		t.Fatalf("unapplied safeguard = %#v", got)
	}
	if err := os.WriteFile(verified, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := inspectSafeguardReadiness(root, safe); got.Status != "ready" {
		t.Fatalf("verified safeguard = %#v", got)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "safeguard.json"), []byte(`{"name":"safe","verificationCheck":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := inspectSafeguardReadiness(root, safe); got.Status != "unsupported" {
		t.Fatalf("probe-less safeguard = %#v", got)
	}
}

func testHost(name, status string) hostItem {
	return hostItem{hostRequirement: hostRequirement{Name: name}, Status: status}
}

func TestControlPlaneExecutorUsesDeclaredCommands(t *testing.T) {
	var got []string
	previous := controlPlaneCommand
	controlPlaneCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		return exec.CommandContext(ctx, "true")
	}
	t.Cleanup(func() { controlPlaneCommand = previous })
	if err := (controlPlaneExecutor{}).InstallTool(context.Background(), "git"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "vrooli host install git --json --sudo-mode error" {
		t.Fatalf("command = %q", got)
	}
	for _, action := range []func(context.Context) error{
		func(ctx context.Context) error { return (controlPlaneExecutor{}).ApplySafeguard(ctx, "safe") },
		func(ctx context.Context) error { return (controlPlaneExecutor{}).EnableResource(ctx, "postgres") },
		func(ctx context.Context) error { return (controlPlaneExecutor{}).StartScenario(ctx, "demo") },
	} {
		if err := action(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	controlPlaneCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}
	if err := (controlPlaneExecutor{}).InstallTool(context.Background(), "git"); err == nil {
		t.Fatal("control-plane failure should be returned")
	}
}

func TestPrivilegedApplyUsesProvisionedGrantAndAuditsInvocation(t *testing.T) {
	var got []string
	previousCommand := controlPlaneCommand
	previousExecutable := controlPlaneExecutable
	previousStatePath := operatorStatePath
	root := t.TempDir()
	controlPlaneExecutable = func(string) (string, error) { return "/usr/local/bin/vrooli", nil }
	controlPlaneCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		return exec.CommandContext(ctx, "true")
	}
	operatorStatePath = func() (string, error) { return filepath.Join(root, "operator-state.json"), nil }
	t.Cleanup(func() {
		controlPlaneCommand = previousCommand
		controlPlaneExecutable = previousExecutable
		operatorStatePath = previousStatePath
	})
	if err := (controlPlaneExecutor{}).InstallToolPrivileged(context.Background(), "qemu"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "sudo -n /usr/local/bin/vrooli host install qemu --json --sudo-mode error" {
		t.Fatalf("granted command = %q", got)
	}
	audit, err := os.ReadFile(filepath.Join(root, "audit", "onboarding-privileged.jsonl"))
	if err != nil || !strings.Contains(string(audit), `"outcome":"applied"`) || !strings.Contains(string(audit), `"qemu"`) {
		t.Fatalf("audit = %s, err=%v", audit, err)
	}
}

func TestPrivilegedApplyReturnsTypedNeedsElevationWhenGrantIsMissing(t *testing.T) {
	previousCommand := controlPlaneCommand
	previousExecutable := controlPlaneExecutable
	controlPlaneExecutable = func(string) (string, error) { return "/usr/local/bin/vrooli", nil }
	controlPlaneCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo 'sudo: a password is required' >&2; exit 1")
	}
	t.Cleanup(func() { controlPlaneCommand = previousCommand; controlPlaneExecutable = previousExecutable })
	err := (controlPlaneExecutor{}).ApplySafeguardPrivileged(context.Background(), "clock")
	var typed *needsElevationError
	if !errors.As(err, &typed) || !strings.Contains(typed.Error(), "vrooli setup --sudo-mode=ask") {
		t.Fatalf("error = %v, want typed needs-elevation result", err)
	}
}

func TestV2ReadModelsDegradeWhenCatalogRootIsAbsent(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	server := NewServer()
	for _, path := range []string{"/api/v2/scenarios", "/api/v2/resources", "/api/v2/credentials", "/api/v2/host-requirements"} {
		w := doGet(t, server, path)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"degraded"`) {
			t.Fatalf("GET %s = %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestCatalogUnavailableErrorIsActionable(t *testing.T) {
	err := (&catalogUnavailableError{Missing: "catalog/resources", Remediation: "rebuild the bundle"}).Error()
	if !strings.Contains(err, "catalog/resources") || !strings.Contains(err, "rebuild the bundle") {
		t.Fatalf("error = %q", err)
	}
}

func TestCredentialClientReportsAuthorityConstructionFailures(t *testing.T) {
	previous := onboardingAuthority
	onboardingAuthority = func() (*credentialauthority.Authority, error) { return nil, credentialauthority.ErrProviderAbsent }
	t.Cleanup(func() { onboardingAuthority = previous })
	ctx := context.Background()
	if _, err := onboardingDoctorJSON(ctx); err == nil {
		t.Fatal("doctor should report authority construction failure")
	}
	if _, err := onboardingKeyringJSON(ctx, "inspect"); err == nil {
		t.Fatal("keyring inspect should report authority construction failure")
	}
	if err := onboardingProvision(ctx, "vrooli/demo", "value", "secret"); err == nil {
		t.Fatal("provision should report authority construction failure")
	}
	if _, err := onboardingStatusJSON(ctx, "vrooli/demo", "value"); err == nil {
		t.Fatal("status should report authority construction failure")
	}
}

func TestConfigureOperatorStateRootsUsesResolvedPath(t *testing.T) {
	root := t.TempDir()
	previous := operatorStatePath
	previousRoots := operatorStateRoots
	operatorStatePath = func() (string, error) { return filepath.Join(root, "state", "operator-state.json"), nil }
	t.Cleanup(func() { operatorStatePath = previous; operatorStateRoots = previousRoots })
	if err := configureOperatorStateRoots(); err != nil {
		t.Fatal(err)
	}
}
