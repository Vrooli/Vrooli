package hostapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/volumeremediation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ExecuteVolume performs one typed volume operation. Parsing and output
// rendering are owned by the CLI handler.
func (Service) ExecuteVolume(ctx context.Context, request volumeremediation.Request) (volumeremediation.Result, error) {
	if ctx == nil {
		return volumeremediation.Result{}, context.Canceled
	}
	service := volumeremediation.New(volumeremediation.Options{})
	if err := fillHostVolumeDeviceFacts(ctx, service, &request); err != nil {
		return volumeremediation.Result{}, err
	}
	return service.Execute(ctx, request)
}

// fillHostVolumeDeviceFacts backfills the device facts the operator did not
// supply. It never overwrites a supplied value: a stated expectation is the
// caller's guard and must survive to be checked against the host.
func fillHostVolumeDeviceFacts(ctx context.Context, service *volumeremediation.Service, request *volumeremediation.Request) error {
	state, err := service.Inspect(ctx, request.Device)
	if err != nil {
		// Inspect failing is not fatal here: Execute re-observes and produces
		// the authoritative typed result, including unsupported platforms.
		return nil
	}
	if request.Device.Filesystem == "" {
		request.Device.Filesystem = state.Device.Filesystem
	}
	if request.Device.UUID == "" {
		request.Device.UUID = state.Device.UUID
	}
	if request.Device.Serial == "" {
		request.Device.Serial = state.Device.Serial
	}
	if request.Device.TotalBytes == 0 {
		request.Device.TotalBytes = state.Device.TotalBytes
	}
	if request.Device.Mountpoint == "" {
		request.Device.Mountpoint = state.Device.Mountpoint
	}
	if request.Action == volumeremediation.ActionMountReadWrite && request.DesiredMountpoint == "" {
		request.DesiredMountpoint = state.Device.Mountpoint
	}
	return nil
}

// HostVolumeOK reports whether a volume operation reached a successful state.
func HostVolumeOK(status string) bool {
	switch status {
	case volumeremediation.StatusVerified,
		volumeremediation.StatusChanged,
		volumeremediation.StatusAlreadySatisfied:
		return true
	default:
		return false
	}
}

// HostVolumeResponse maps a remediation result to the CLI wire contract.
func HostVolumeResponse(result volumeremediation.Result, execErr error) *cliv1.VolumeRemediationResponse {
	response := &cliv1.VolumeRemediationResponse{Action: string(result.Action), Status: result.Status, Changed: result.Changed, DryRun: result.DryRun, Command: append([]string(nil), result.Command...), Backend: result.Backend, Detail: result.Detail}
	if result.Consistent != "" {
		response.Consistent = string(result.Consistent)
	}
	if strings.TrimSpace(result.State.Device.Path) != "" {
		response.State = hostVolumeState(result.State)
	}
	if response.Status == "" {
		response.Status = volumeremediation.StatusFailed
	}
	var refused volumeremediation.ErrRefused
	var unsupported volumeremediation.ErrUnsupported
	switch {
	case errors.As(execErr, &unsupported):
		response.RefusalReason = unsupported.Reason
		response.OperatorCommand = unsupported.OperatorCommand
	case errors.As(execErr, &refused):
		response.RefusalReason = refused.Reason
	case execErr != nil:
		response.RefusalReason = execErr.Error()
	}
	return response
}

func hostVolumeState(state volumeremediation.State) *cliv1.VolumeState {
	return &cliv1.VolumeState{Device: &cliv1.VolumeDevice{Path: state.Device.Path, Filesystem: state.Device.Filesystem, Uuid: state.Device.UUID, Serial: state.Device.Serial, Mountpoint: state.Device.Mountpoint, TotalBytes: state.Device.TotalBytes}, Mounted: state.Mounted, ReadOnly: state.ReadOnly, Dirty: string(state.Dirty), Evidence: state.Evidence, Observations: append([]string(nil), state.Observations...)}
}

// RenderHostVolumeText preserves the established human output contract.
func RenderHostVolumeText(w io.Writer, response *cliv1.VolumeRemediationResponse) {
	_, _ = fmt.Fprintf(w, "Action: %s\n", response.GetAction())
	_, _ = fmt.Fprintf(w, "Status: %s", response.GetStatus())
	if response.GetDryRun() {
		_, _ = fmt.Fprint(w, " (dry run)")
	}
	_, _ = fmt.Fprintln(w)
	if backend := response.GetBackend(); backend != "" {
		_, _ = fmt.Fprintf(w, "Backend: %s\n", backend)
	}
	if state := response.GetState(); state != nil && state.GetDevice() != nil {
		device := state.GetDevice()
		_, _ = fmt.Fprintf(w, "Device: %s", device.GetPath())
		if fs := device.GetFilesystem(); fs != "" {
			_, _ = fmt.Fprintf(w, " (%s)", fs)
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Mounted: %s", cliout.BoolLabel(state.GetMounted()))
		if mountpoint := device.GetMountpoint(); mountpoint != "" {
			_, _ = fmt.Fprintf(w, " at %s", mountpoint)
		}
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Read-only: %s\n", cliout.BoolLabel(state.GetReadOnly()))
		_, _ = fmt.Fprintf(w, "Filesystem dirty: %s\n", state.GetDirty())
		if evidence := state.GetEvidence(); evidence != "" {
			_, _ = fmt.Fprintf(w, "Evidence: %s\n", evidence)
		}
	}
	if command := response.GetCommand(); len(command) > 0 {
		_, _ = fmt.Fprintf(w, "Command: %s\n", strings.Join(command, " "))
	}
	if consistent := response.GetConsistent(); consistent != "" {
		_, _ = fmt.Fprintf(w, "Filesystem consistent: %s\n", consistent)
	}
	if detail := response.GetDetail(); detail != "" {
		_, _ = fmt.Fprintf(w, "Detail: %s\n", detail)
	}
	if reason := response.GetRefusalReason(); reason != "" {
		_, _ = fmt.Fprintf(w, "Reason: %s\n", reason)
	}
	if command := response.GetOperatorCommand(); command != "" {
		_, _ = fmt.Fprintf(w, "Run instead: %s\n", command)
	}
}
