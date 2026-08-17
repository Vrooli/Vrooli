package ollamaresourcecontrols

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "ollama_resource_controls", Handler: "ollama_resource_controls"})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "ollama_resource_controls", Kind: hostreqspec.KindSafeguard, Required: true}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux"}
}

func restoreHooks(t *testing.T) {
	t.Helper()
	originalRead := readFileFn
	originalState := statePathFn
	originalAlive := processAlive
	originalLimits := processLimitsFn
	t.Cleanup(func() {
		readFileFn = originalRead
		statePathFn = originalState
		processAlive = originalAlive
		processLimitsFn = originalLimits
	})
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "ollama_resource_controls" || h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("handler identity = %q/%q", h.Name(), h.Kind())
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	status := newTestHandler().Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q, want unsupported", status.SupportClass)
	}
}

func TestInspectWithoutSupervisedOllamaIsNotApplicable(t *testing.T) {
	restoreHooks(t)
	statePathFn = func() string { return filepath.Join(t.TempDir(), "managed-service.json") }
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q, want not applicable; notes=%v", status.SupportClass, status.Notes)
	}
}

func TestInspectReadsBackSupervisedProcessLimits(t *testing.T) {
	restoreHooks(t)
	statePath := filepath.Join(t.TempDir(), "managed-service.json")
	if err := os.WriteFile(statePath, []byte(`{"pid":4321}`), 0o600); err != nil {
		t.Fatal(err)
	}
	statePathFn = func() string { return statePath }
	processAlive = func(pid int) bool { return pid == 4321 }
	processLimitsFn = func(pid int) (uint64, uint64, error) {
		if pid != 4321 {
			return 0, 0, errors.New("unexpected pid")
		}
		total, err := physicalMemory()
		if err != nil {
			return 0, 0, err
		}
		return total * memoryHighPercent / 100, total * memoryMaxPercent / 100, nil
	}
	readFileFn = func(path string) ([]byte, error) {
		if path == "/proc/4321/oom_score_adj" {
			return []byte(fmt.Sprint(oomScoreAdjust)), nil
		}
		return os.ReadFile(path)
	}
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("status = %+v, want applied", status)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "supervisor process 4321") {
		t.Fatalf("notes = %v, want supervised PID evidence", status.Notes)
	}
}

func TestApplyReportsRestartRequirementWhenControlsAreMissing(t *testing.T) {
	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed || !strings.Contains(strings.Join(out.Notes, " "), "restart Ollama") {
		t.Fatalf("status = %+v, want failed restart guidance", out)
	}
}

func TestApplyDryRunDoesNotClaimHostMutation(t *testing.T) {
	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want would-apply", out.ExecutionState)
	}
}
