package artifacts

import (
	"context"
	"errors"
	"fmt"

	internalartifacts "vrooli-bridge/internal/artifacts"
	internalmachines "vrooli-bridge/internal/machines"
)

type closureMachineReader interface {
	Get(context.Context, string) (internalmachines.Machine, error)
}

// validateClosureTarget re-reads the Bridge-owned Node and Machine lineage at
// each dispatch/reconciliation boundary. A selection receipt cannot authorize
// a stale or revoked enrollment after the handoff was created.
func validateClosureTarget(ctx context.Context, shipment internalartifacts.ClosureShipment, nodes internalartifacts.NodeReader, machines closureMachineReader) error {
	if nodes == nil || machines == nil {
		return errors.New("closure target identity readers are unavailable")
	}
	if shipment.TargetID == "" || shipment.TargetID != shipment.NodeID || shipment.MachineID == "" || shipment.EnrollmentGeneration == 0 {
		return errors.New("closure shipment target binding is incomplete")
	}
	node, err := nodes.GetTarget(ctx, shipment.TargetID)
	if err != nil {
		return err
	}
	if node.Revoked {
		return internalartifacts.ErrNodeRevoked{ID: shipment.TargetID}
	}
	if node.ID != shipment.NodeID || node.EnrollmentGeneration != int64(shipment.EnrollmentGeneration) {
		return fmt.Errorf("closure shipment enrollment generation is stale")
	}
	machine, err := machines.Get(ctx, shipment.MachineID)
	if err != nil {
		return err
	}
	if machine.ID != shipment.MachineID || machine.Lifecycle != internalmachines.LifecycleActive || machine.EnrollmentGeneration != int64(shipment.EnrollmentGeneration) {
		return errors.New("closure shipment machine binding is stale or inactive")
	}
	current := 0
	for _, lineage := range machine.Lineage {
		if !lineage.Current {
			continue
		}
		current++
		if lineage.NodeID != shipment.NodeID || lineage.EnrollmentGeneration != int64(shipment.EnrollmentGeneration) {
			return errors.New("closure shipment current node lineage does not match its enrollment generation")
		}
	}
	if current != 1 {
		return errors.New("closure shipment requires exactly one current machine node lineage")
	}
	return nil
}
