// Package ollamaresourcecontrols verifies the resource controls applied by the
// managed-service supervisor to Ollama's actual supervised process.
//
// Older versions installed a systemd drop-in for ollama.service. That unit is
// not part of the managed-service lifecycle, so the drop-in could report
// success while the process serving inference remained unlimited. The control
// plane now applies the limits at launch and this safeguard reads them back
// from the supervisor's recorded PID.
package ollamaresourcecontrols

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	oomScoreAdjust = 500
	// unlimitedAddressSpace is RLIM_INFINITY as reported for RLIMIT_AS. The
	// supervised Ollama process must stay at this value; see verifyProcessControls.
	unlimitedAddressSpace = ^uint64(0)
)

type managedServiceState struct {
	PID int `json:"pid"`
}

var (
	readFileFn      = os.ReadFile
	statePathFn     = ollamaStatePath
	processAlive    = processAlivePID
	processLimitsFn = readProcessLimits
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "ollama resource controls require the Linux managed-service supervisor")
		return status
	}
	state, err := readState()
	if errors.Is(err, os.ErrNotExist) || state.PID <= 0 || !processAlive(state.PID) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "ollama managed-service is not running")
		return status
	}
	if err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "ollama managed-service state is unavailable: "+err.Error())
		return status
	}
	if err := verifyProcessControls(state.PID); err != nil {
		status.Notes = append(status.Notes, fmt.Sprintf("ollama supervisor process %d controls are missing or stale: %v", state.PID, err))
		return status
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, fmt.Sprintf("ollama supervisor process %d has the declared OOM control and an uncapped address space", state.PID))
	status.Notes = append(status.Notes, "memory_high_percent/memory_max_percent are NOT enforced: Linux has no rlimit that bounds resident memory, and capping address space instead breaks CUDA model loads. Enforcing them needs a dedicated cgroup v2 scope per managed service.")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: restart Ollama through `vrooli resource restart ollama` so the supervisor reapplies its process limits")
		return status, nil
	}
	status.Notes = append(status.Notes, "restart Ollama through `vrooli resource restart ollama`; limits are applied only by the managed-service supervisor")
	return status, nil
}

func readState() (managedServiceState, error) {
	body, err := readFileFn(statePathFn())
	if err != nil {
		return managedServiceState{}, err
	}
	var state managedServiceState
	if err := json.Unmarshal(body, &state); err != nil {
		return managedServiceState{}, fmt.Errorf("parse managed-service state: %w", err)
	}
	return state, nil
}

func ollamaStatePath() string {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return filepath.Join(".", "ollama", "managed-service.json")
	}
	return filepath.Join(home, ".local", "state", "vrooli", "resources", "ollama", "managed-service.json")
}

// verifyProcessControls checks the controls the supervisor can actually enforce.
//
// The address-space check is deliberately inverted from what it used to be. This
// safeguard once required RLIMIT_AS to equal 60%/70% of physical memory, reading
// a memory declaration as an address-space cap. That cap bounds virtual address
// space rather than resident memory, so it protected nothing measurable and
// broke CUDA: llama.cpp reserves a 32 GiB VMM pool on first inference, which the
// cap rejected as "CUDA error: out of memory" on an idle GPU, so every Ollama
// model load died and all embedding traffic failed. A capped address space on
// the supervised PID is now the failure condition, not the requirement.
func verifyProcessControls(pid int) error {
	cur, max, err := processLimitsFn(pid)
	if err != nil {
		return err
	}
	if cur != unlimitedAddressSpace || max != unlimitedAddressSpace {
		return fmt.Errorf("address space is capped at cur=%d max=%d; an address-space cap breaks CUDA model loads and must not be applied for a memory declaration", cur, max)
	}
	value, err := readFileFn(filepath.Join("/proc", strconv.Itoa(pid), "oom_score_adj"))
	if err != nil {
		return fmt.Errorf("read oom_score_adj: %w", err)
	}
	if strings.TrimSpace(string(value)) != strconv.Itoa(oomScoreAdjust) {
		return fmt.Errorf("oom_score_adj is %q, want %d", strings.TrimSpace(string(value)), oomScoreAdjust)
	}
	return nil
}
