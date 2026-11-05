package flow

import (
	_ "embed"
	"encoding/json"
	"testing"

	"device-control/internal/control/flow/generated"
	"github.com/vrooli/vrooli/packages/proto/modeltest"
)

// Embedding keeps the formal replay independent of the test runner's
// -trimpath working directory. The artifact's source hashes are still checked
// by modeltest against the module files on disk.
//
//go:embed generated/artifact.json
var formalArtifactJSON []byte

func loadFormalArtifact(t *testing.T) modeltest.FormalArtifact {
	t.Helper()
	var artifact modeltest.FormalArtifact
	if err := json.Unmarshal(formalArtifactJSON, &artifact); err != nil {
		t.Fatalf("parse embedded formal artifact: %v", err)
	}
	return artifact
}

func TestDeviceLeaseFormalReplay(t *testing.T) { // [REQ:DVC-P0-005]
	artifact := loadFormalArtifact(t)
	modeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{
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
	modeltest.AssertFormalTransitionsReplay(t, artifact, generated.AllDeviceLeaseStatuses(), generated.AllDeviceLeaseEvents(), transition)
	modeltest.AssertFormalTracesReplay(t, artifact, generated.AllDeviceLeaseStatuses(), generated.AllDeviceLeaseEvents(), transition)
}
