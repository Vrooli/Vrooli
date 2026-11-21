package resources

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func TestNativeRetainedServingEvidenceLocatesRootWithoutBootDefinition(t *testing.T) {
	_, m, ref, w, _ := nativeTransportFixture(t)
	root := filepath.Dir(m.Runtime.StateDir)
	role := openRGBFrozenUpdateRole{ContractDigest: m.Contract.Digest, NativePolicy: &m.NativePolicy, Catalog: m.Catalog}
	state := ManagedServiceState{InstanceID: ref.Intent.AttemptID, OwnershipTokenHash: m.TokenHash, PID: ref.RootPID, Incarnation: ref.Native.RootIncarnation, ArtifactPath: m.Contract.Artifact.Path, ArtifactSHA256: m.Contract.Artifact.SHA256, ArtifactVersion: m.Contract.Artifact.Version, UpdateLaunch: &ManagedUpdateServingLaunch{OperationID: "operation", RequestID: "request", Role: managedUpdateRolePrimary, Contract: m.Contract, Runtime: m.Runtime, NativeRef: ref}}
	intent := lightingruntime.Intent{Phase: lightingruntime.Disabling, Selection: &lightingruntime.Selection{Version: m.Contract.Artifact.Version, SHA256: m.Contract.Artifact.SHA256, DevicePolicyID: w.Policy.ID}}
	verify := func(root, home, account, machine string, state ManagedServiceState, run ManagedUpdateRuntime, role openRGBFrozenUpdateRole, w openRGBExecWitness, selected string) error {
		return verifyNativeOpenRGBRetainedStorageEvidence(root, home, account, machine, intent, state, m.Contract, run, role, w, selected)
	}
	if err := verify(root, m.Runtime.Home, m.NativePolicy.Account, m.NativePolicy.MachineID, state, m.Runtime, role, w, w.Policy.ID); err != nil {
		t.Fatal("retained complete namespace rejected without boot definition", err)
	}
	for _, name := range []string{"copiedToRootB", "otherHome", "otherAccount", "otherMachine", "otherSelection", "otherArtifact", "reusedPID", "missingWitness", "noNativePolicy", "ambiguousOwners"} {
		t.Run(name, func(t *testing.T) {
			current, home, account, machine := root, m.Runtime.Home, m.NativePolicy.Account, m.NativePolicy.MachineID
			copyState, copyRun, copyRole, copyW, selected := state, m.Runtime, role, w, w.Policy.ID
			switch name {
			case "copiedToRootB":
				current = filepath.Join(t.TempDir(), "root-b")
			case "otherHome":
				home = filepath.Join(t.TempDir(), "other-home")
			case "otherAccount":
				account = "uid:502"
			case "otherMachine":
				machine = strings.Repeat("f", 64)
			case "otherSelection":
				selected = strings.Repeat("f", 64)
			case "otherArtifact":
				copyState.ArtifactSHA256 = strings.Repeat("f", 64)
			case "reusedPID":
				copyState.PID++
			case "missingWitness":
				copyW.ClaimSHA256 = ""
			case "noNativePolicy":
				copyRole.NativePolicy = nil
			case "ambiguousOwners":
				copyState.InitialLaunch = &ManagedInitialServingLaunch{}
			}
			if err := verify(current, home, account, machine, copyState, copyRun, copyRole, copyW, selected); err == nil {
				t.Fatal("foreign or incomplete retained namespace accepted")
			}
		})
	}
}
