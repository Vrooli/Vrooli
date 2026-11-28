package modules_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"google.golang.org/protobuf/proto"
)

func TestClosureShipmentChannelContractRoundTripsTypedFrames(t *testing.T) {
	request := &channelv1.ClosureShipmentRequest{
		ShipmentId: "shipment-1", IdempotencyKey: "key-1", TargetId: "node-1",
		SourceRef: "artifact://closure", Destination: "/srv/vrooli",
		Closure: &channelv1.DependencyClosure{SchemaVersion: 1, Digest: "graph-sha256", Members: []*channelv1.ClosureMember{{
			Component: &channelv1.ClosureComponentID{Kind: "scenario", Name: "web-console"}, Selected: true,
			Root: &channelv1.ClosureComponentID{Kind: "scenario", Name: "web-console"},
		}}},
		InventoryDigest: "inventory-sha256", PriorRevision: "rev-old", NewRevision: "rev-new",
		SelectionReceiptRef: "vrooli-onboarding://handoffs/handoff-1", SelectionDigest: "selection-sha256", DeploymentId: "deployment-1", MachineId: "machine-1", NodeId: "node-1", EnrollmentGeneration: 7, DesiredRevision: 12,
		Files: []*channelv1.ClosureFile{{Path: "scenarios/web-console/README.md", Sha256: "file-sha256", Mode: 0o640, SizeBytes: 42}},
	}
	command := &channelv1.ServerFrame{FrameId: "frame-1", Payload: &channelv1.ServerFrame_ClosureShipment{ClosureShipment: &channelv1.ClosureShipmentCommand{
		Action: channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, Request: request,
	}}}
	wire, err := proto.Marshal(command)
	require.NoError(t, err)
	var decodedCommand channelv1.ServerFrame
	require.NoError(t, proto.Unmarshal(wire, &decodedCommand))
	require.Equal(t, "node-1", decodedCommand.GetClosureShipment().GetRequest().GetTargetId())
	require.Equal(t, "graph-sha256", decodedCommand.GetClosureShipment().GetRequest().GetClosure().GetDigest())
	require.Equal(t, "inventory-sha256", decodedCommand.GetClosureShipment().GetRequest().GetInventoryDigest())
	require.Equal(t, "rev-old", decodedCommand.GetClosureShipment().GetRequest().GetPriorRevision())
	require.Equal(t, "rev-new", decodedCommand.GetClosureShipment().GetRequest().GetNewRevision())

	receipt := &channelv1.NodeFrame{Payload: &channelv1.NodeFrame_ClosureShipmentReceipt{ClosureShipmentReceipt: &channelv1.ClosureShipmentReceipt{
		ShipmentId: "shipment-1", IdempotencyKey: "key-1", TargetId: "node-1", ClosureDigest: "graph-sha256",
		InventoryDigest: "inventory-sha256", ActualInventoryDigest: "inventory-sha256", PriorRevision: "rev-old",
		NewRevision: "rev-new", ActiveRevision: "rev-new", State: channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED,
		SelectionReceiptRef: "vrooli-onboarding://handoffs/handoff-1", SelectionDigest: "selection-sha256", DeploymentId: "deployment-1", MachineId: "machine-1", NodeId: "node-1", EnrollmentGeneration: 7, DesiredRevision: 12,
	}}}
	wire, err = proto.Marshal(receipt)
	require.NoError(t, err)
	var decodedReceipt channelv1.NodeFrame
	require.NoError(t, proto.Unmarshal(wire, &decodedReceipt))
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED, decodedReceipt.GetClosureShipmentReceipt().GetState())
	require.Equal(t, "rev-new", decodedReceipt.GetClosureShipmentReceipt().GetActiveRevision())
}
