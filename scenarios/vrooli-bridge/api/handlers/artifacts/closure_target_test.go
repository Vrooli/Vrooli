package artifacts

import (
	"context"
	"testing"

	internalartifacts "vrooli-bridge/internal/artifacts"
	internalmachines "vrooli-bridge/internal/machines"

	"github.com/stretchr/testify/require"
)

type closureTargetNodes struct{ node internalartifacts.TargetNode }

func (r closureTargetNodes) GetTarget(context.Context, string) (internalartifacts.TargetNode, error) {
	return r.node, nil
}

type closureTargetMachines struct{ machine internalmachines.Machine }

func (r closureTargetMachines) Get(context.Context, string) (internalmachines.Machine, error) {
	return r.machine, nil
}

func TestValidateClosureTargetRequiresCurrentExactEnrollment(t *testing.T) {
	shipment := internalartifacts.ClosureShipment{TargetID: "node-1", NodeID: "node-1", MachineID: "machine-1", EnrollmentGeneration: 7}
	nodes := closureTargetNodes{node: internalartifacts.TargetNode{ID: "node-1", EnrollmentGeneration: 7}}
	machines := closureTargetMachines{machine: internalmachines.Machine{
		ID: "machine-1", Lifecycle: internalmachines.LifecycleActive, EnrollmentGeneration: 7,
		Lineage: []internalmachines.NodeLineage{{NodeID: "node-1", Current: true, EnrollmentGeneration: 7}},
	}}
	require.NoError(t, validateClosureTarget(context.Background(), shipment, nodes, machines))

	staleNode := nodes
	staleNode.node.EnrollmentGeneration = 6
	require.ErrorContains(t, validateClosureTarget(context.Background(), shipment, staleNode, machines), "stale")

	staleLineage := machines
	staleLineage.machine.Lineage[0].EnrollmentGeneration = 6
	require.ErrorContains(t, validateClosureTarget(context.Background(), shipment, nodes, staleLineage), "lineage")

	revoked := nodes
	revoked.node.Revoked = true
	require.Error(t, validateClosureTarget(context.Background(), shipment, revoked, machines))
}
