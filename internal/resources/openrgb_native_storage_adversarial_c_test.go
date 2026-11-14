package resources

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func TestAdversarialCRetainedNamespaceRemainsLocatableAfterCatalogRetirement(t *testing.T) {
	_, m, ref, witness, _ := nativeTransportFixture(t)
	root := filepath.Dir(m.Runtime.StateDir)
	// Revocation must locate the exact retained namespace even if its catalog
	// no longer qualifies for a new launch. This is metadata, never liveness.
	p := m.NativePolicy
	p.Devices = append([]hostreqkit.OpenRGBNativeDevice(nil), p.Devices...)
	p.Devices[0].Detector, p.Devices[0].HardwareID = "Retired HID controller", "1234:5678"
	catalog := []byte(`{"retired-catalog":false}`)
	p.DetectorSHA256 = fmt.Sprintf("%x", sha256.Sum256(catalog))
	role := openRGBFrozenUpdateRole{ContractDigest: m.Contract.Digest, NativePolicy: &p, Catalog: catalog}
	binding, err := role.structuralBinding(m.Contract)
	if err != nil {
		t.Fatal(err)
	}
	if hostreqkit.ValidateOpenRGBNativeDetectorScope(p) == nil {
		t.Fatal("fixture did not retire launch qualification")
	}
	witness.Policy = binding
	state := ManagedServiceState{InstanceID: ref.Intent.AttemptID, OwnershipTokenHash: m.TokenHash, PID: ref.RootPID, Incarnation: ref.Native.RootIncarnation, ArtifactPath: m.Contract.Artifact.Path, ArtifactSHA256: m.Contract.Artifact.SHA256, ArtifactVersion: m.Contract.Artifact.Version, UpdateLaunch: &ManagedUpdateServingLaunch{OperationID: "operation", RequestID: "request", Role: managedUpdateRolePrimary, Contract: m.Contract, Runtime: m.Runtime, NativeRef: ref}}
	intent := lightingruntime.Intent{Phase: lightingruntime.Disabling, Selection: &lightingruntime.Selection{Version: m.Contract.Artifact.Version, SHA256: m.Contract.Artifact.SHA256, DevicePolicyID: binding.ID}}
	verify := func() error {
		return verifyNativeOpenRGBRetainedStorageEvidence(root, m.Runtime.Home, p.Account, p.MachineID, intent, state, m.Contract, m.Runtime, role, witness, binding.ID)
	}
	if err := verify(); err != nil {
		t.Fatal("exact retired namespace cannot be located for cleanup", err)
	}
	// A different birth or old IDv1 reference cannot borrow that locator.
	for _, change := range []string{"incarnation", "IDv1", "context", "generation", "claim"} {
		t.Run(change, func(t *testing.T) {
			copyState, copyW, copyRef := state, witness, ref
			copyNative := *ref.Native
			copyRef.Native = &copyNative
			launch := *state.UpdateLaunch
			copyState.UpdateLaunch = &launch
			switch change {
			case "incarnation":
				copyState.Incarnation = "different-process-birth"
			case "IDv1":
				copyNative.Version = 1
				launch.NativeRef = copyRef
			case "context":
				copyW.ContextSHA256 = ""
			case "generation":
				copyW.DeviceGenerationSHA256 = ""
			case "claim":
				copyW.ClaimSHA256 = ""
			}
			if err := verifyNativeOpenRGBRetainedStorageEvidence(root, m.Runtime.Home, p.Account, p.MachineID, intent, copyState, m.Contract, m.Runtime, role, copyW, binding.ID); err == nil {
				t.Fatal("incomplete retained native locator accepted")
			}
		})
	}
}

func TestAdversarialCFrozenOwnerCannotFollowAmbientRootOrSiblingPrefix(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "frozen")
	owner := openRGBNativeOwner{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: "uid:501", RuntimePrincipal: "uid:501", NativeAccount: "uid:501", resourceRoot: root}
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", filepath.Join(base, "ambient"))
	for _, path := range []string{filepath.Join(root, "launch", "attempt", "capsule.json"), filepath.Join(root, "runtime", "initial", "attempt", "config")} {
		if _, err := owner.policyFor(path); err != nil {
			t.Fatal("frozen private owner followed ambient root", err)
		}
	}
	for _, path := range []string{filepath.Join(root+"-sibling", "owner.json"), filepath.Join(base, "owner.json"), root + "/../frozen/owner.json"} {
		if _, err := owner.policyFor(path); err == nil {
			t.Fatal("alias or neighboring tree admitted", path)
		}
	}
	for _, goos := range []string{"darwin", "windows"} {
		for _, enrolled := range []string{base + "/../" + filepath.Base(base), base + "\n", "relative"} {
			if err := validateNativeOpenRGBStorageEnrollment(goos, root, enrolled, true, ""); err == nil {
				t.Fatal("invalid retained enrollment admitted", enrolled)
			}
		}
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("metadata guards created namespace", err)
	}
}
