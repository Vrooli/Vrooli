package destinationreadiness

import (
	"context"
	"errors"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type stubVolumeClient struct {
	response *cliv1.VolumeRemediationResponse
	err      error
	requests []vroolicli.VolumeRequest
}

func (s *stubVolumeClient) HostVolume(_ context.Context, req vroolicli.VolumeRequest) (*cliv1.VolumeRemediationResponse, error) {
	s.requests = append(s.requests, req)
	return s.response, s.err
}

func remediationPlan(action PreparationAction) Plan {
	return Plan{
		Action: action,
		Identity: DeviceIdentity{
			DevicePath: "/dev/sda1",
			Filesystem: "ntfs3",
			UUID:       "E26A883E6A881189",
			Serial:     "WD-WX52A946D6VL",
			Mountpoint: "/media/user/Elements",
		},
	}
}

// The data-loss acknowledgement must be carried only by the action that can
// actually lose data. Sending it on every step would train the control plane's
// gate to be meaningless.
func TestRemediatorAcknowledgesDataLossOnlyForRepair(t *testing.T) {
	cases := map[PreparationAction]bool{
		ActionRepairFilesystem: true,
		ActionCheckFilesystem:  false,
		ActionUnmount:          false,
		ActionMountReadWrite:   false,
	}
	for action, wantAck := range cases {
		t.Run(string(action), func(t *testing.T) {
			client := &stubVolumeClient{response: &cliv1.VolumeRemediationResponse{Status: "changed", Changed: true}}
			remediator := &ControlPlaneRemediator{client: client}

			if _, err := remediator.Remediate(context.Background(), remediationPlan(action), false); err != nil {
				t.Fatalf("Remediate: %v", err)
			}
			if len(client.requests) != 1 {
				t.Fatalf("requests = %d", len(client.requests))
			}
			if got := client.requests[0].AcknowledgeDataLoss; got != wantAck {
				t.Fatalf("AcknowledgeDataLoss = %v, want %v", got, wantAck)
			}
		})
	}
}

// The device identity travels with every request so the control plane can
// refuse a disk that no longer matches.
func TestRemediatorPassesTheDeviceIdentityThrough(t *testing.T) {
	client := &stubVolumeClient{response: &cliv1.VolumeRemediationResponse{Status: "verified"}}
	remediator := &ControlPlaneRemediator{client: client}

	if _, err := remediator.Remediate(context.Background(), remediationPlan(ActionCheckFilesystem), true); err != nil {
		t.Fatalf("Remediate: %v", err)
	}
	req := client.requests[0]
	if req.Device != "/dev/sda1" || req.UUID != "E26A883E6A881189" || req.Serial != "WD-WX52A946D6VL" || req.Filesystem != "ntfs3" {
		t.Fatalf("request lost identity: %+v", req)
	}
	if !req.DryRun {
		t.Fatal("dry run was not propagated to the control plane")
	}
	// Only the mount step targets a mountpoint; sending one otherwise would
	// imply a target the action does not use.
	if req.Mountpoint != "" {
		t.Fatalf("mountpoint = %q, want empty for a check", req.Mountpoint)
	}
}

func TestRemediatorSendsTheMountpointOnlyForMount(t *testing.T) {
	client := &stubVolumeClient{response: &cliv1.VolumeRemediationResponse{Status: "changed", Changed: true}}
	remediator := &ControlPlaneRemediator{client: client}

	if _, err := remediator.Remediate(context.Background(), remediationPlan(ActionMountReadWrite), false); err != nil {
		t.Fatalf("Remediate: %v", err)
	}
	if client.requests[0].Mountpoint != "/media/user/Elements" {
		t.Fatalf("mountpoint = %q", client.requests[0].Mountpoint)
	}
}

// A control-plane refusal must surface as an error *and* keep its typed detail,
// so a scripted sequence stops and the operator still learns why.
func TestRemediatorSurfacesARefusalWithItsReasonAndNextCommand(t *testing.T) {
	client := &stubVolumeClient{response: &cliv1.VolumeRemediationResponse{
		Status:          "unsupported",
		RefusalReason:   "this host has no udisks2 client and the privilege broker is unavailable",
		OperatorCommand: "sudo vrooli setup",
	}}
	remediator := &ControlPlaneRemediator{client: client}

	outcome, err := remediator.Remediate(context.Background(), remediationPlan(ActionRepairFilesystem), false)

	var refused ErrPreparationRefused
	if !errors.As(err, &refused) {
		t.Fatalf("error = %v, want a refusal", err)
	}
	if !strings.Contains(refused.Reason, "udisks2") || !strings.Contains(refused.Reason, "sudo vrooli setup") {
		t.Fatalf("refusal must carry the reason and the next command, got %q", refused.Reason)
	}
	if outcome.OperatorCommand != "sudo vrooli setup" || outcome.Status != "unsupported" {
		t.Fatalf("outcome lost its detail: %+v", outcome)
	}
	if outcome.Satisfied() {
		t.Fatal("an unsupported result must not count as satisfied")
	}
}

// A step that finds its work already done is a success: recovery sequences get
// retried, and treating idempotence as failure would block the retry.
func TestRemediatorTreatsAlreadySatisfiedAsSuccess(t *testing.T) {
	client := &stubVolumeClient{response: &cliv1.VolumeRemediationResponse{Status: "already_satisfied"}}
	remediator := &ControlPlaneRemediator{client: client}

	outcome, err := remediator.Remediate(context.Background(), remediationPlan(ActionUnmount), false)
	if err != nil {
		t.Fatalf("Remediate: %v", err)
	}
	if !outcome.Satisfied() || outcome.Changed {
		t.Fatalf("outcome = %+v, want satisfied without a change", outcome)
	}
}

func TestRemediatorRejectsNonRemediationActions(t *testing.T) {
	remediator := &ControlPlaneRemediator{client: &stubVolumeClient{}}
	for _, action := range []PreparationAction{ActionCreateSubdir, ActionFormat, ActionClearDirectory, ActionRelabel} {
		if supported, reason := remediator.Supported(action); supported || reason == "" {
			t.Fatalf("%s must not be supported by the remediation client", action)
		}
		if _, err := remediator.Remediate(context.Background(), remediationPlan(action), false); err == nil {
			t.Fatalf("%s was accepted by the remediation client", action)
		}
	}
}

func TestRemediatorRequiresADevicePath(t *testing.T) {
	remediator := &ControlPlaneRemediator{client: &stubVolumeClient{}}
	plan := remediationPlan(ActionCheckFilesystem)
	plan.Identity.DevicePath = ""

	if _, err := remediator.Remediate(context.Background(), plan, false); err == nil {
		t.Fatal("expected an error without a device path")
	}
}
