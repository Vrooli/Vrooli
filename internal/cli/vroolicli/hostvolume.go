package vroolicli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/volumeremediation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// hostVolumeActions maps the CLI's action words onto the remediation registry.
// The CLI vocabulary is deliberately the same size as the registry: there is no
// format, partition, or clear action to reach.
var hostVolumeActions = map[string]volumeremediation.Action{
	"inspect":  volumeremediation.ActionInspect,
	"check":    volumeremediation.ActionCheck,
	"repair":   volumeremediation.ActionRepair,
	"unmount":  volumeremediation.ActionUnmount,
	"mount-rw": volumeremediation.ActionMountReadWrite,
}

func hostVolumeOptions() []commandtree.OptionArg {
	return []commandtree.OptionArg{
		commandtree.JSONOption(),
		{Name: "--device", ValueName: "path", Description: "Device to act on, e.g. /dev/sda1 (required)"},
		{Name: "--filesystem", ValueName: "type", Description: "Filesystem type; observed from the host when omitted"},
		{Name: "--uuid", ValueName: "uuid", Description: "Expected filesystem UUID; the action is refused if the host disagrees"},
		{Name: "--serial", ValueName: "serial", Description: "Expected device serial; the action is refused if the host disagrees"},
		{Name: "--mountpoint", ValueName: "path", Description: "Mount target for mount-rw (must be under /media, /run/media, /mnt or /Volumes)"},
		{Name: "--acknowledge-data-loss", Description: "Required for repair: acknowledges that repair can discard inconsistent filesystem metadata"},
		{Name: "--dry-run", Description: "Validate everything and report the command without running it"},
	}
}

func hostVolumeSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "volume",
		Summary: "Inspect, check, repair, or remount a storage volume",
		Help: commandtree.Help{
			Description: "Remediates a storage volume the kernel refuses to mount read/write — typically a filesystem carrying a dirty flag after an unclean disconnect. " +
				"Detection and remediation of host state are control-plane responsibilities; scenarios request this rather than carrying their own host-repair code. " +
				"On Linux the primary path is udisks2, which authorises a non-system device for an active session with no password and no root; hosts without it fall back to the setup-installed privilege broker. " +
				"Every mutating action re-observes the device identity immediately before acting, refuses system volumes, refuses to check or repair a mounted filesystem, and requires an explicit acknowledgement before writing.",
			Usage:   "vrooli host volume <inspect|check|repair|unmount|mount-rw> --device <path> [options]",
			Options: hostVolumeOptions(),
			Examples: []string{
				"vrooli host volume inspect --device /dev/sda1 --json",
				"vrooli host volume unmount --device /dev/sda1",
				"vrooli host volume check --device /dev/sda1",
				"vrooli host volume repair --device /dev/sda1 --acknowledge-data-loss --dry-run",
				"vrooli host volume mount-rw --device /dev/sda1",
			},
		},
		Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{
				{Name: "action", Required: true, Description: "inspect, check, repair, unmount, or mount-rw"},
			},
			Options: hostVolumeOptions(),
		},
		Handler: "volume",
	}
}

func (app *App) runHostVolumeCommand(ctx *CommandContext, args []string) error {
	spec := hostVolumeSpec()
	parsed, err := commandtree.ParseArgs("host volume", commandtree.SpecHelpText("", "vrooli host volume", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host volume", "%s", err.Error())
	}
	if len(parsed.Positionals) == 0 {
		return rootcli.UsageErrorf("host volume", "an action is required (inspect, check, repair, unmount, or mount-rw)")
	}
	actionName := strings.ToLower(strings.TrimSpace(parsed.Positionals[0]))
	action, ok := hostVolumeActions[actionName]
	if !ok {
		return rootcli.UsageErrorf("host volume", "unknown action %q (want inspect, check, repair, unmount, or mount-rw)", actionName)
	}
	device := strings.TrimSpace(parsed.FlagValue("--device"))
	if device == "" {
		return rootcli.UsageErrorf("host volume", "--device is required")
	}

	request := volumeremediation.Request{
		Action: action,
		Device: volumeremediation.Device{
			Path:       device,
			Filesystem: strings.TrimSpace(parsed.FlagValue("--filesystem")),
			UUID:       strings.TrimSpace(parsed.FlagValue("--uuid")),
			Serial:     strings.TrimSpace(parsed.FlagValue("--serial")),
			Mountpoint: strings.TrimSpace(parsed.FlagValue("--mountpoint")),
		},
		DesiredMountpoint:   strings.TrimSpace(parsed.FlagValue("--mountpoint")),
		AcknowledgeDataLoss: parsed.HasFlag("--acknowledge-data-loss"),
		DryRun:              parsed.HasFlag("--dry-run"),
	}

	service := volumeremediation.New(volumeremediation.Options{})
	// Identity and filesystem are optional on the command line because the host
	// is the authority on both. Filling them from an observation keeps the
	// operator from having to restate facts the host already publishes — while
	// any value they *did* supply is still checked against the observation.
	if err := fillHostVolumeDeviceFacts(context.Background(), service, &request); err != nil {
		return err
	}

	result, execErr := service.Execute(context.Background(), request)
	response := hostVolumeResponse(result, execErr)

	if ctx.Globals.JSON || parsed.HasFlag("--json") {
		if err := cliout.WriteProtoJSON(ctx.Stdout, response); err != nil {
			return err
		}
	} else {
		renderHostVolumeText(ctx.Stdout, response)
	}
	if hostVolumeOK(result.Status) {
		return nil
	}
	return rootcli.ExitCodeError{Code: 1, Silent_: true}
}

// fillHostVolumeDeviceFacts backfills the device facts the operator did not
// supply. It never overwrites a supplied value: a stated expectation is the
// caller's guard and must survive to be checked against the host.
func fillHostVolumeDeviceFacts(ctx context.Context, service *volumeremediation.Service, request *volumeremediation.Request) error {
	state, err := service.Inspect(ctx, request.Device)
	if err != nil {
		// Inspect failing is not fatal here: Execute re-observes and produces
		// the authoritative typed result, including the unsupported-platform
		// case that carries an operator command.
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

// hostVolumeOK reports whether a result should exit zero. A refusal and an
// unsupported platform are both non-zero: each means the requested change did
// not happen, and a script must be able to tell without parsing prose.
func hostVolumeOK(status string) bool {
	switch status {
	case volumeremediation.StatusVerified,
		volumeremediation.StatusChanged,
		volumeremediation.StatusAlreadySatisfied:
		return true
	default:
		return false
	}
}

func hostVolumeResponse(result volumeremediation.Result, execErr error) *cliv1.VolumeRemediationResponse {
	response := &cliv1.VolumeRemediationResponse{
		Action:  string(result.Action),
		Status:  result.Status,
		Changed: result.Changed,
		DryRun:  result.DryRun,
		Command: append([]string(nil), result.Command...),
		Backend: result.Backend,
		Detail:  result.Detail,
	}
	if result.Consistent != "" {
		response.Consistent = string(result.Consistent)
	}
	// A gate that fires before the host is observed leaves State zero-valued.
	// Rendering that would show "mounted: no, dirty: no" as if they were
	// findings, so an unobserved state is omitted rather than reported.
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
	return &cliv1.VolumeState{
		Device: &cliv1.VolumeDevice{
			Path:       state.Device.Path,
			Filesystem: state.Device.Filesystem,
			Uuid:       state.Device.UUID,
			Serial:     state.Device.Serial,
			Mountpoint: state.Device.Mountpoint,
			TotalBytes: state.Device.TotalBytes,
		},
		Mounted:      state.Mounted,
		ReadOnly:     state.ReadOnly,
		Dirty:        string(state.Dirty),
		Evidence:     state.Evidence,
		Observations: append([]string(nil), state.Observations...),
	}
}

func renderHostVolumeText(w io.Writer, response *cliv1.VolumeRemediationResponse) {
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
