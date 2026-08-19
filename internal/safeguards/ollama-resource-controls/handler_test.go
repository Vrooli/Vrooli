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
		return unlimitedAddressSpace, unlimitedAddressSpace, nil
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
	notes := strings.Join(status.Notes, " ")
	if !strings.Contains(notes, "supervisor process 4321") {
		t.Fatalf("notes = %v, want supervised PID evidence", status.Notes)
	}
	// The unenforced memory percentages must be stated, not implied as applied.
	if !strings.Contains(notes, "NOT enforced") {
		t.Fatalf("notes = %v, want an explicit note that the memory percentages are unenforced", status.Notes)
	}
}

// TestInspectRejectsCappedAddressSpace guards the regression that took Ollama
// down: a memory declaration applied as an RLIMIT_AS cap. llama.cpp reserves a
// 32 GiB CUDA VMM pool on first inference, so any address-space cap kills every
// model load with "CUDA error: out of memory" on an idle GPU. The safeguard must
// report a capped address space as missing controls, never as healthy.
func TestInspectRejectsCappedAddressSpace(t *testing.T) {
	restoreHooks(t)
	statePath := filepath.Join(t.TempDir(), "managed-service.json")
	if err := os.WriteFile(statePath, []byte(`{"pid":4321}`), 0o600); err != nil {
		t.Fatal(err)
	}
	statePathFn = func() string { return statePath }
	processAlive = func(pid int) bool { return pid == 4321 }
	// The exact values observed on the host when every embedding request failed:
	// 60% and 70% of 60.9 GiB of physical memory.
	processLimitsFn = func(int) (uint64, uint64, error) {
		return 39259304755, 45802522214, nil
	}
	readFileFn = func(path string) ([]byte, error) {
		if path == "/proc/4321/oom_score_adj" {
			return []byte(fmt.Sprint(oomScoreAdjust)), nil
		}
		return os.ReadFile(path)
	}
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("status = %+v, want not applied while the address space is capped", status)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "address space is capped") {
		t.Fatalf("notes = %v, want the capped address space named as the defect", status.Notes)
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
