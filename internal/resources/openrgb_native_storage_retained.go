package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// Retained records locate the existing namespace; they do not certify that a
// process is alive or transfer device access. The ordinary stopper still proves
// exact current native ownership before any effect.
func (c *Controller) verifyOpenRGBRetainedStorageRoot(ctx context.Context, intent lightingruntime.Intent) error {
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return err
	}
	if _, _, err := nativeOpenRGBStoragePaths(); err != nil {
		return err
	}
	data, err := readManagedUpdatePrivateFile(filepath.Join(root, "state", managedServiceStateFile), managedUpdateMaxBytes)
	if err != nil {
		return err
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return err
	}
	var state ManagedServiceState
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return err
	}
	if (state.InitialLaunch == nil) == (state.UpdateLaunch == nil) {
		return errors.New("native namespace has no exclusive retained serving owner")
	}
	if state.InitialLaunch != nil {
		if err := validateManagedInitialServing(state); err != nil {
			return err
		}
	} else {
		if err := validateManagedUpdateServing(state); err != nil {
			return err
		}
	}
	contract, run, err := openRGBOriginalLaunch(state)
	if err != nil {
		return err
	}
	role, err := c.openRGBFrozenOriginalRole(ctx, state)
	if err != nil {
		return err
	}
	witness, err := readOpenRGBOriginalWitness(ManagedServiceUpdate{Resource: openRGBResourceName, Original: state, Rollback: contract}, false)
	if err != nil {
		return err
	}
	selected, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || selected == nil || selected.NativePolicy == nil {
		return errors.Join(errors.New("native namespace selected policy is unavailable"), err)
	}
	selectedID, err := selected.ID()
	if err != nil {
		return err
	}
	account, err := c.openRGBNativeRuntimeAccount(ctx)
	if err != nil {
		return err
	}
	machine, err := hostreqkit.NativeOpenRGBMachineIdentity()
	if err != nil {
		return err
	}
	return verifyNativeOpenRGBRetainedStorageEvidence(root, c.Home, account, machine, intent, state, contract, run, role, witness, selectedID)
}

func verifyNativeOpenRGBRetainedStorageEvidence(root, home, account, machine string, intent lightingruntime.Intent, state ManagedServiceState, contract ManagedUpdateContract, run ManagedUpdateRuntime, role openRGBFrozenUpdateRole, witness openRGBExecWitness, selectedID string) error {
	if intent.Selection == nil || role.NativePolicy == nil || run.validate() != nil || run.Home != home || run.StateDir != filepath.Join(root, "state") || run.NativeAccount != account || run.OwnerPrincipal != account || role.NativePolicy.Account != account || role.NativePolicy.MachineID != machine {
		return errors.New("retained native namespace differs from exact operator, machine or runtime root")
	}
	if (state.InitialLaunch == nil) == (state.UpdateLaunch == nil) {
		return errors.New("retained native namespace has ambiguous serving ownership")
	}
	ref := managedUpdateOriginalNativeRef(state)
	if ref == nil || ref.Native == nil || !ref.NativeOwnedIdentityComplete() || ref.RootPID != state.PID || !ref.MatchesIncarnation(state.Incarnation) || ref.Native.Account != account || ref.Native.Platform != role.NativePolicy.Platform {
		return errors.New("retained namespace has no complete exact native reference")
	}
	binding, err := role.structuralBinding(contract)
	expected := lightingruntime.Selection{Version: contract.Artifact.Version, SHA256: contract.Artifact.SHA256, DevicePolicyID: binding.ID}
	if err != nil || *intent.Selection != expected || selectedID != binding.ID || witness.Policy != binding || witness.Incarnation != state.Incarnation || !updateDigest(witness.DeviceGenerationSHA256) || !updateDigest(witness.ClaimSHA256) || !updateDigest(witness.ContextSHA256) || state.ArtifactPath != contract.Artifact.Path || state.ArtifactSHA256 != contract.Artifact.SHA256 || state.ArtifactVersion != contract.Artifact.Version {
		return errors.Join(errors.New("retained namespace differs from selected frozen policy, artifact or consumed witness"), err)
	}
	return nil
}
