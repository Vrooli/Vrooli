package destinationreadiness

import (
	"context"
	"fmt"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ControlPlaneRemediator executes volume remediation through the Vrooli control
// plane, which owns host state.
//
// This scenario deliberately contains no filesystem repair, remount, fsck, or
// mount-namespace handling of its own. It observes, plans, confirms, and
// reports; the control plane decides what is safe to run on the host and runs
// it. That split is what keeps a single host-repair implementation in one
// reviewable place instead of one per scenario that happens to need it.
type ControlPlaneRemediator struct {
	client volumeClient
}

// volumeClient is the seam over the control-plane CLI wrapper so the gates and
// translation here are testable without a host.
type volumeClient interface {
	HostVolume(ctx context.Context, req vroolicli.VolumeRequest) (*cliv1.VolumeRemediationResponse, error)
}

// NewControlPlaneRemediator constructs the production remediation client.
func NewControlPlaneRemediator() *ControlPlaneRemediator {
	return &ControlPlaneRemediator{client: vroolicli.New()}
}

var _ Remediator = (*ControlPlaneRemediator)(nil)

// remediationCLIActions maps this scenario's vocabulary onto the control-plane
// CLI's. An action absent from the map is one the control plane does not offer,
// and it is reported as unsupported rather than approximated.
var remediationCLIActions = map[PreparationAction]vroolicli.VolumeAction{
	ActionUnmount:          vroolicli.VolumeUnmount,
	ActionCheckFilesystem:  vroolicli.VolumeCheck,
	ActionRepairFilesystem: vroolicli.VolumeRepair,
	ActionMountReadWrite:   vroolicli.VolumeMountReadWrite,
}

// Supported reports which remediation actions this client can execute.
func (r *ControlPlaneRemediator) Supported(action PreparationAction) (bool, string) {
	if _, ok := remediationCLIActions[action]; ok {
		return true, ""
	}
	return false, fmt.Sprintf("%s is not a host remediation action", action)
}

// Remediate asks the control plane to perform one action against the planned
// device. The plan's device identity is passed through as an expectation: the
// control plane re-observes the host and refuses if the disk disagrees, so a
// stale plan cannot act on a swapped drive.
func (r *ControlPlaneRemediator) Remediate(ctx context.Context, plan Plan, dryRun bool) (RemediationOutcome, error) {
	action, ok := remediationCLIActions[plan.Action]
	if !ok {
		return RemediationOutcome{Status: "unsupported"}, ErrPreparationRefused{Reason: fmt.Sprintf("%s is not a host remediation action", plan.Action)}
	}
	if r == nil || r.client == nil {
		return RemediationOutcome{Status: "unsupported"}, ErrPreparationRefused{Reason: "control-plane client is not configured"}
	}
	if strings.TrimSpace(plan.Identity.DevicePath) == "" {
		return RemediationOutcome{Status: "refused"}, ErrInvalidReadiness{Field: "device_path", Reason: "required for remediation"}
	}

	request := vroolicli.VolumeRequest{
		Action:     action,
		Device:     plan.Identity.DevicePath,
		Filesystem: plan.Identity.Filesystem,
		UUID:       plan.Identity.UUID,
		Serial:     plan.Identity.Serial,
		// A repair reaches the control plane's own acknowledgement gate only
		// because this scenario already collected a matching confirmation
		// phrase and an explicit data-loss acknowledgement for the same plan.
		AcknowledgeDataLoss: plan.Action == ActionRepairFilesystem,
		DryRun:              dryRun,
	}
	if plan.Action == ActionMountReadWrite {
		request.Mountpoint = plan.Identity.Mountpoint
	}

	response, err := r.client.HostVolume(ctx, request)
	if err != nil {
		return RemediationOutcome{Status: "failed"}, fmt.Errorf("control plane %s: %w", action, err)
	}
	outcome := RemediationOutcome{
		Status:          response.GetStatus(),
		Changed:         response.GetChanged(),
		Backend:         response.GetBackend(),
		Command:         append([]string(nil), response.GetCommand()...),
		Detail:          response.GetDetail(),
		OperatorCommand: response.GetOperatorCommand(),
		RefusalReason:   response.GetRefusalReason(),
		Consistent:      response.GetConsistent(),
	}
	if outcome.Satisfied() {
		return outcome, nil
	}
	// A refusal is a result, not a transport failure, but it must not read as
	// success: the caller gets the typed outcome *and* an error so a scripted
	// sequence stops rather than continuing to the next step.
	reason := outcome.RefusalReason
	if reason == "" {
		reason = "control plane reported status " + outcome.Status
	}
	if outcome.OperatorCommand != "" {
		reason += "; run: " + outcome.OperatorCommand
	}
	return outcome, ErrPreparationRefused{Reason: reason}
}
