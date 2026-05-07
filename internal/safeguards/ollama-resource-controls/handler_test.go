package ollamaresourcecontrols

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
	}
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "ollama_resource_controls",
		Handler: "ollama_resource_controls",
	})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "ollama_resource_controls",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:              "linux",
		PackageManager:  "apt-get",
		SupportsSystemd: true,
	}
}

func ollamaPresentStub(t *testing.T) {
	t.Helper()
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" && len(args) > 0 && args[0] == "list-unit-files" {
			return []byte("ollama.service enabled enabled\n"), nil
		}
		return nil, errors.New("unexpected CombinedOutputFn call")
	}
}

func ollamaAbsentStub(t *testing.T) {
	t.Helper()
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "systemctl" {
			return []byte(""), nil
		}
		return nil, errors.New("unexpected CombinedOutputFn call")
	}
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "ollama_resource_controls" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	req := linuxReq()
	req.Manual = true
	status := h.Inspect(linuxHost(), req)
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNoSystemdNotApplicable(t *testing.T) {
	h := newTestHandler()
	host := linuxHost()
	host.SupportsSystemd = false
	status := h.Inspect(host, linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectOllamaAbsentNotApplicable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	ollamaAbsentStub(t)

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want NotApplicable; notes: %v", status.SupportClass, status.Notes)
	}
}

func TestInspectDropinAlreadyPresent(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	ollamaPresentStub(t)

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == dropinPath {
			return []byte(buildDropinContent()), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied=true; notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectDropinMissingNeedsApply(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	ollamaPresentStub(t)

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("expected Applied=false when drop-in missing")
	}
	joined := strings.Join(status.Notes, " ")
	if !strings.Contains(joined, dropinPath) {
		t.Errorf("expected notes to mention drop-in path %q; got %v", dropinPath, status.Notes)
	}
}

func TestApplyDryRunDoesNotMutate(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	ollamaPresentStub(t)

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want WouldApply", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("dry-run ran %d commands; want 0", calls)
	}
}

func TestApplyAlreadyAppliedShortCircuits(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("already-applied path ran %d commands; want 0", calls)
	}
}

func TestApplyNotApplicableShortCircuits(t *testing.T) {
	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportNotApplicable,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
}

func TestBuildDropinContentIncludesAllDirectives(t *testing.T) {
	content := buildDropinContent()
	for _, d := range managedDirectives {
		want := d.Key + "=" + d.Value
		if !strings.Contains(content, want) {
			t.Errorf("drop-in missing %q", want)
		}
	}
	if !strings.Contains(content, "[Service]") {
		t.Error("drop-in missing [Service] section header")
	}
	if !strings.Contains(content, "Managed by Vrooli") {
		t.Error("drop-in missing managed-by marker")
	}
}
