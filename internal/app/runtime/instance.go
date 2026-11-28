package runtimeapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

// RuntimeInstanceInspection is the bounded owner record needed to interpret
// scenario-instance references in process ownership evidence.
type RuntimeInstanceInspection struct {
	InstanceID                    string                        `json:"instance_id"`
	Scenario                      string                        `json:"scenario"`
	Variant                       string                        `json:"variant"`
	Generation                    int64                         `json:"generation"`
	Status                        string                        `json:"status"`
	StartedAt                     string                        `json:"started_at"`
	StoppedAt                     string                        `json:"stopped_at,omitempty"`
	ProcessRefs                   []RuntimeProcessRefInspection `json:"process_refs"`
	ProcessRefObservationComplete bool                          `json:"process_ref_observation_complete"`
	ProcessRefsTruncated          bool                          `json:"process_refs_truncated,omitempty"`
}

type RuntimeProcessRefInspection struct {
	RefID        string `json:"ref_id"`
	ProcessID    string `json:"process_id,omitempty"`
	Step         string `json:"step,omitempty"`
	PID          *int   `json:"pid"`
	PGID         *int   `json:"pgid,omitempty"`
	Status       string `json:"recorded_status"`
	ProcessState string `json:"process_state"`
	StartedAt    string `json:"started_at,omitempty"`
	EndedAt      string `json:"ended_at,omitempty"`
}

// InspectRuntimeInstance reads one exact instance through the canonical
// runtime registry in read-only mode. It never reconciles or updates state.
func (app *Service) InspectRuntimeInstance(ctx context.Context, home, instanceID string, out io.Writer, jsonOutput bool) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return errors.New("runtime instance ID is required")
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, ReadOnly: true})
	if err != nil {
		return fmt.Errorf("open runtime registry read-only: %w", err)
	}
	defer store.Close()
	instance, err := store.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("read runtime instance %s: %w", instanceID, err)
	}
	inspection := RuntimeInstanceInspection{
		InstanceID: instance.InstanceID, Scenario: instance.Scenario,
		Variant: instance.Variant, Generation: instance.Generation,
		Status: instance.Status, StartedAt: instance.StartedAt.UTC().Format(time.RFC3339Nano),
	}
	if instance.StoppedAt != nil {
		inspection.StoppedAt = instance.StoppedAt.UTC().Format(time.RFC3339Nano)
	}
	refs, err := store.ListProcessRefs(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("read runtime process refs for %s: %w", instanceID, err)
	}
	const processRefLimit = 128
	inspection.ProcessRefs = make([]RuntimeProcessRefInspection, 0, min(len(refs), processRefLimit))
	inspection.ProcessRefsTruncated = len(refs) > processRefLimit
	processes, processErr := (supervision.NativeProcessTableSource{}).ProcessesContext(ctx)
	bootID := hostsession.CurrentBootID()
	inspection.ProcessRefObservationComplete = processErr == nil && bootID != "" && !inspection.ProcessRefsTruncated
	for _, ref := range refs[:min(len(refs), processRefLimit)] {
		row := RuntimeProcessRefInspection{
			RefID: ref.RefID, ProcessID: ref.ProcessID, Step: ref.Step,
			PID: ref.PID, PGID: ref.PGID, Status: ref.Status,
			ProcessState: runtimeProcessRefState(ref, bootID, processes, processErr),
		}
		if !ref.StartedAt.IsZero() {
			row.StartedAt = ref.StartedAt.UTC().Format(time.RFC3339Nano)
		}
		if ref.EndedAt != nil {
			row.EndedAt = ref.EndedAt.UTC().Format(time.RFC3339Nano)
		}
		if row.ProcessState == "unknown" {
			inspection.ProcessRefObservationComplete = false
		}
		inspection.ProcessRefs = append(inspection.ProcessRefs, row)
	}
	if jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(inspection)
	}
	if _, err := fmt.Fprintf(out, "Instance: %s\nScenario: %s/%s generation %d\nStatus: %s\nStarted: %s\n", inspection.InstanceID, inspection.Scenario, inspection.Variant, inspection.Generation, inspection.Status, inspection.StartedAt); err != nil {
		return err
	}
	if inspection.StoppedAt != "" {
		if _, err = fmt.Fprintf(out, "Stopped: %s\n", inspection.StoppedAt); err != nil {
			return err
		}
	}
	for _, ref := range inspection.ProcessRefs {
		if _, err := fmt.Fprintf(out, "Process: %s/%s PID %v recorded=%s current=%s\n", ref.ProcessID, ref.Step, ref.PID, ref.Status, ref.ProcessState); err != nil {
			return err
		}
	}
	return nil
}

func runtimeProcessRefState(ref scenarioruntime.ProcessRef, currentBootID string, processes map[int]supervision.ProcessInfo, processErr error) string {
	if processErr != nil || currentBootID == "" || ref.PID == nil || *ref.PID <= 1 || ref.HostBootID == "" || ref.StartedAt.IsZero() {
		return "unknown"
	}
	if ref.HostBootID != currentBootID {
		return "absent"
	}
	process, exists := processes[*ref.PID]
	if !exists {
		return "absent"
	}
	const pidReuseAllowance = 2 * time.Second
	delta := process.StartedAt.Sub(ref.StartedAt)
	if delta < -pidReuseAllowance || delta > pidReuseAllowance {
		return "absent"
	}
	return "present"
}
