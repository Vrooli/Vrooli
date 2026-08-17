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
	memoryHighPercent = 60
	memoryMaxPercent  = 70
	oomScoreAdjust    = 500
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
	status.Notes = append(status.Notes, fmt.Sprintf("ollama supervisor process %d has the declared memory and OOM controls", state.PID))
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

func verifyProcessControls(pid int) error {
	total, err := physicalMemory()
	if err != nil {
		return err
	}
	cur, max, err := processLimitsFn(pid)
	if err != nil {
		return err
	}
	wantCur := total * memoryHighPercent / 100
	wantMax := total * memoryMaxPercent / 100
	if cur != wantCur || max != wantMax {
		return fmt.Errorf("address-space limit is cur=%d max=%d, want cur=%d max=%d", cur, max, wantCur, wantMax)
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

func physicalMemory() (uint64, error) {
	data, err := readFileFn("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("read physical memory: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kib, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil || kib == 0 {
				break
			}
			return kib * 1024, nil
		}
	}
	return 0, fmt.Errorf("physical memory is missing from /proc/meminfo")
}
