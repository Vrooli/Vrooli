package companion

import (
	"context"
	"testing"

	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
)

type fakeCommandPusher struct {
	commands []*companionv1.CompanionCommand
}

type fakeArtifactDistributor struct {
	decision ArtifactDecision
	request  ArtifactRequest
}

func (d *fakeArtifactDistributor) Distribute(_ context.Context, request ArtifactRequest) (ArtifactDecision, error) {
	d.request = request
	return d.decision, nil
}

func (p *fakeCommandPusher) PushCompanionCommand(_ context.Context, _ string, command *companionv1.CompanionCommand) error {
	p.commands = append(p.commands, command)
	return nil
}

func TestManagerWaitsForArtifactReceiptBeforeLifecycleCommand(t *testing.T) {
	pusher := &fakeCommandPusher{}
	distributor := &fakeArtifactDistributor{decision: ArtifactDecision{DistributionID: "dist-1", Pending: true}}
	m := NewManager(func(node string) bool { return node == "node-1" }, pusher)
	m.SetArtifactDistributor(distributor)
	op, err := m.InstallWithArtifact(context.Background(), "node-1", "1.0.0", "display-1", "file:///companion", "companion", "/Users/alice/.vrooli/bin/device-control-companion")
	if err != nil || op.GetReasonCode() != "artifact_pending" || len(pusher.commands) != 0 {
		t.Fatalf("install=%+v err=%v commands=%v", op, err, len(pusher.commands))
	}
	if distributor.request.DestinationPath == "" || distributor.request.SourceRef == "" {
		t.Fatalf("artifact request=%+v", distributor.request)
	}
	if !m.DeliverArtifactReceipt(context.Background(), "dist-1", "node-1", true, "") || len(pusher.commands) != 1 {
		t.Fatalf("receipt did not release lifecycle command: %+v", pusher.commands)
	}
	if got := pusher.commands[0].GetOperationId(); got != op.GetOperationId() {
		t.Fatalf("command operation=%q want %q", got, op.GetOperationId())
	}
}

func TestManagerFailsPendingCompanionOnRejectedArtifact(t *testing.T) {
	pusher := &fakeCommandPusher{}
	m := NewManager(func(node string) bool { return node == "node-1" }, pusher)
	m.SetArtifactDistributor(&fakeArtifactDistributor{decision: ArtifactDecision{DistributionID: "dist-2", Pending: true}})
	op, err := m.InstallWithArtifact(context.Background(), "node-1", "1.0.0", "", "file:///companion", "companion", "/Users/alice/.vrooli/bin/device-control-companion")
	if err != nil || !m.DeliverArtifactReceipt(context.Background(), "dist-2", "node-1", false, "checksum mismatch") {
		t.Fatalf("setup failed op=%+v err=%v", op, err)
	}
	current, _ := m.Inspect("node-1")
	if current.GetState() != companionv1.CompanionState_COMPANION_STATE_FAILED || current.GetReasonCode() != "artifact_placement_failed" {
		t.Fatalf("current=%+v", current)
	}
}

func TestManagerLifecycleIsTargetGatedAndReversible(t *testing.T) {
	pusher := &fakeCommandPusher{}
	m := NewManager(func(node string) bool { return node == "node-1" }, pusher)
	if _, err := m.Install("revoked", "1.0.0", "display-1"); err != ErrTargetUnavailable {
		t.Fatalf("revoked target error = %v", err)
	}
	installed, err := m.Install("node-1", "1.0.0", "display-1")
	if err != nil || installed.GetState() != companionv1.CompanionState_COMPANION_STATE_INSTALLING || len(pusher.commands) != 1 {
		t.Fatalf("install = %+v, err=%v", installed, err)
	}
	if !m.Deliver(&companionv1.CompanionResponse{OperationId: installed.GetOperationId(), NodeId: "node-1", Version: "1.0.0", State: companionv1.CompanionState_COMPANION_STATE_READY, ReasonCode: "ready", UserLaunchAgent: true}) {
		t.Fatal("expected lifecycle response to update operation")
	}
	upgraded, err := m.Upgrade("node-1", "1.1.0")
	if err != nil || upgraded.GetVersion() != "1.1.0" || upgraded.GetState() != companionv1.CompanionState_COMPANION_STATE_INSTALLING || len(pusher.commands) != 2 {
		t.Fatalf("upgrade = %+v, err=%v", upgraded, err)
	}
	revoked, err := m.Revoke("node-1", "operator_request")
	if err != nil || revoked.GetState() != companionv1.CompanionState_COMPANION_STATE_INSTALLING || len(pusher.commands) != 3 {
		t.Fatalf("revoke = %+v, err=%v", revoked, err)
	}
	removed, err := m.Remove("node-1")
	if err != nil || removed.GetState() != companionv1.CompanionState_COMPANION_STATE_INSTALLING || len(pusher.commands) != 4 {
		t.Fatalf("remove = %+v, err=%v", removed, err)
	}
}
