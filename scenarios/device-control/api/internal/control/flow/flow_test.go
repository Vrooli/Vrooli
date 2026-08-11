package flow

import (
	"testing"

	"device-control/internal/control/flow/generated"
	"device-control/internal/testutil/modeltest"
)

func TestDeviceLeaseFormalReplay(t *testing.T) {
	modeltest.AssertFormalArtifactFresh(t, modeltest.LoadFormalArtifact(t, "generated/artifact.json"), modeltest.FormalArtifactExpectation{
		ContractPath:    generated.DeviceLeaseContractPath,
		ContractSHA256:  generated.DeviceLeaseContractSHA256,
		ModelPath:       generated.DeviceLeaseModelPath,
		ModelSHA256:     generated.DeviceLeaseModelSHA256,
		GeneratorPath:   generated.DeviceLeaseGeneratorPath,
		GeneratorSHA256: generated.DeviceLeaseGeneratorSHA256,
		Invariants:      generated.DeviceLeaseFormalExpectedInvariants(),
		GeneratedChecks: generated.DeviceLeaseFormalExpectedGeneratedChecks(),
	})
	transition := func(status generated.Status, event generated.Event) (generated.Status, error) {
		return TransitionDeviceLease(status, event)
	}
	artifact := modeltest.LoadFormalArtifact(t, "generated/artifact.json")
	modeltest.AssertFormalTransitionsReplay(t, artifact, generated.AllDeviceLeaseStatuses(), generated.AllDeviceLeaseEvents(), transition)
	modeltest.AssertFormalTracesReplay(t, artifact, generated.AllDeviceLeaseStatuses(), generated.AllDeviceLeaseEvents(), transition)
}
