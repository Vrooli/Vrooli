package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	platform "github.com/vrooli/platform-go"
)

const openRGBResourceName = "openrgb"

type managedUpdateAdmissionSource struct {
	SourceRevision     string
	Process            ManagedServiceState
	Contract           ManagedUpdateContract
	Scope              *platform.ServiceScopeRef
	CredentialStateDir string
}

func managedUpdateRecordedOriginal(u ManagedServiceUpdate) (ManagedServiceUpdate, ManagedUpdateProcess, ManagedUpdateContract, error) {
	l := u.Original.UpdateLaunch
	if l == nil || validateManagedUpdateServing(u.Original) != nil || l.Contract.Artifact.SHA256 != u.Rollback.Artifact.SHA256 {
		return ManagedServiceUpdate{}, ManagedUpdateProcess{}, ManagedUpdateContract{}, fmt.Errorf("original recorded native launch is not qualified")
	}
	state := u.Original
	state.UpdateLaunch = nil
	i := ManagedUpdateLaunchIntent{InstanceID: state.InstanceID, OwnershipTokenHash: state.OwnershipTokenHash, ContractDigest: l.Contract.Digest, NativeIntent: platform.ServiceIntent(l.NativeRef.Intent), Runtime: l.Runtime}
	old := ManagedServiceUpdate{ID: l.OperationID, RequestID: l.RequestID, Resource: u.Resource, Credentials: l.Credentials}
	return old, ManagedUpdateProcess{Intent: &i, State: &state, NativeRef: &l.NativeRef}, l.Contract, nil
}

func (c *Controller) observeManagedUpdateAdmissionSource(ctx context.Context, m ResourceManifest) (managedUpdateAdmissionSource, error) {
	sup, paths, err := managedServiceSupervisorFor(m.Name)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	state, running, err := sup.Status()
	if err != nil || !running {
		return managedUpdateAdmissionSource{}, fmt.Errorf("serving resource owner is unavailable: %v", err)
	}
	if m.Name == openRGBResourceName && state.InitialLaunch != nil {
		return c.observeOpenRGBInitialAdmission(ctx, state)
	}
	if state.UpdateLaunch == nil {
		legacy, err := c.observeManagedServiceLegacyLaunchLocked(ctx, m)
		if err != nil {
			return managedUpdateAdmissionSource{}, err
		}
		contract, err := freezeManagedUpdateServingConfig(legacy.Process, paths.ConfigDir, legacy.Contract)
		if err != nil {
			return managedUpdateAdmissionSource{}, err
		}
		return managedUpdateAdmissionSource{SourceRevision: legacy.SourceRevision, Process: legacy.Process, Contract: contract, Scope: &legacy.NativeScope, CredentialStateDir: paths.StateDir}, nil
	}
	l := state.UpdateLaunch
	u := ManagedServiceUpdate{Resource: m.Name, Original: state, Rollback: l.Contract}
	old, p, contract, err := managedUpdateRecordedOriginal(u)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	if _, err := observeManagedUpdateProcess(ctx, old, p, contract); err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	contract, err = freezeManagedUpdateServingConfig(state, paths.ConfigDir, contract)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	if len(m.Ports) != 1 || m.Ports[0].Host != l.Runtime.Port {
		return managedUpdateAdmissionSource{}, fmt.Errorf("target changes the canonical serving port")
	}
	var image *os.File
	if m.Name == openRGBResourceName {
		image, err = c.openOpenRGBRollbackEnvelope(ctx, contract.Artifact)
	} else {
		image, err = platform.OpenProcessExecutable(state.PID)
	}
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	defer image.Close()
	retained, err := retainManagedUpdateFile(ctx, c, m.Name, state.ArtifactVersion, state.ArtifactSHA256, image)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	rollback, err := contract.withArtifact(retained)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	if _, err := observeManagedUpdateProcess(ctx, old, p, contract); err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	after, live, err := sup.Status()
	if err != nil || !live || !reflect.DeepEqual(state, after) {
		return managedUpdateAdmissionSource{}, fmt.Errorf("recorded serving owner changed during observation")
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	digest := sha256.Sum256(encoded)
	root, err := l.Runtime.roleRoot(l.OperationID, state.InstanceID)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	return managedUpdateAdmissionSource{SourceRevision: hex.EncodeToString(digest[:]), Process: state, Contract: rollback, CredentialStateDir: filepath.Join(root, "state")}, nil
}

func freezeManagedUpdateServingConfig(state ManagedServiceState, fallbackConfigDir string, contract ManagedUpdateContract) (ManagedUpdateContract, error) {
	declaration := contract.Manifest.ManagedService.Config
	if declaration == nil || declaration.ContentFile == "" {
		return contract, nil
	}
	configDir := fallbackConfigDir
	if state.InitialLaunch != nil {
		a, _, err := readOpenRGBInitialServing(state)
		if err != nil {
			return ManagedUpdateContract{}, err
		}
		configDir = filepath.Join(a.root(), "config")
	} else if state.UpdateLaunch != nil {
		root, err := state.UpdateLaunch.Runtime.roleRoot(state.UpdateLaunch.OperationID, state.InstanceID)
		if err != nil {
			return ManagedUpdateContract{}, err
		}
		configDir = filepath.Join(root, "config")
	}
	current, err := readManagedUpdateConfigFile(configDir, declaration.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contract, nil
		}
		return ManagedUpdateContract{}, err
	}
	if contract.Manifest.Name == openRGBResourceName {
		if err := verifyOpenRGBNormalizedServingSettings(contract.ConfigBytes, current); err != nil {
			return ManagedUpdateContract{}, fmt.Errorf("verify frozen OpenRGB serving config: %w", err)
		}
		return contract, nil
	}
	return contract.withConfigBytes(declaration.ContentFile, current)
}

func mergeManagedUpdateTargetConfig(target ManagedUpdateContract, rollback ManagedUpdateContract) (ManagedUpdateContract, error) {
	declaration := target.Manifest.ManagedService.Config
	if declaration == nil || declaration.ContentFile == "" || rollback.Manifest.Name != target.Manifest.Name || len(rollback.ConfigBytes) == 0 {
		return target, nil
	}
	if target.Manifest.Name == "openrgb" {
		merged, err := mergeOpenRGBConfigContent(rollback.ConfigBytes, string(target.ConfigBytes))
		if err != nil {
			return ManagedUpdateContract{}, fmt.Errorf("merge OpenRGB update config: %w", err)
		}
		return target.withConfigBytes(declaration.ContentFile, []byte(merged))
	}
	if target.ConfigFile == rollback.ConfigFile {
		return target.withConfigBytes(declaration.ContentFile, rollback.ConfigBytes)
	}
	return target, nil
}

func verifyManagedUpdateOriginalConfig(u ManagedServiceUpdate) error {
	declaration := u.Rollback.Manifest.ManagedService.Config
	if declaration == nil || declaration.ContentFile == "" {
		return nil
	}
	if u.Rollback.legacyFileConfig {
		// V1 checkpoints did not bind config bytes. Keep their prior recovery
		// behavior instead of comparing live config to an unavailable digest.
		return nil
	}
	configDir := ""
	if u.Original.InitialLaunch != nil {
		a, _, err := readOpenRGBInitialServing(u.Original)
		if err != nil {
			return err
		}
		configDir = filepath.Join(a.root(), "config")
	} else if u.Original.UpdateLaunch != nil {
		root, err := u.Original.UpdateLaunch.Runtime.roleRoot(u.Original.UpdateLaunch.OperationID, u.Original.InstanceID)
		if err != nil {
			return err
		}
		configDir = filepath.Join(root, "config")
	} else {
		paths, err := resourceStoragePaths(u.Resource)
		if err != nil {
			return err
		}
		configDir = paths.ConfigDir
	}
	data, err := readManagedUpdateConfigFile(configDir, declaration.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if u.Resource == openRGBResourceName {
		return verifyOpenRGBNormalizedServingSettings(u.Rollback.ConfigBytes, data)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != u.Rollback.ConfigSHA256 {
		return fmt.Errorf("serving config changed after update admission; refusing exclusive stop")
	}
	return nil
}

func managedUpdateOriginalQuiescent(ctx context.Context, u ManagedServiceUpdate) (bool, error) {
	if nativeRef := managedUpdateOriginalNativeRef(u.Original); nativeRef != nil {
		ref := *nativeRef
		native, err := platform.InspectOwnedService(ctx, ref)
		if err == nil {
			return !native.RootAlive && !native.Populated, nil
		}
		// Only this exact retained invocation's complete native absence can
		// reconcile a lost release acknowledgement.
		return managedUpdateNativeReleased(ctx, ref, &u.Original)
	}
	if u.OriginalScope == nil {
		return false, fmt.Errorf("original has no exact native owner")
	}
	native, err := platform.InspectServiceScope(ctx, *u.OriginalScope)
	return err == nil && !native.RootAlive && !native.TreePopulated, err
}

func managedUpdateOriginalNativeRef(state ManagedServiceState) *platform.ServiceRef {
	if state.InitialLaunch != nil {
		return &state.InitialLaunch.NativeRef
	}
	if state.UpdateLaunch != nil {
		return &state.UpdateLaunch.NativeRef
	}
	return nil
}

// Only a retained state for this exact root supplies cross-boot exclusion.
// Lost launch acknowledgements without a process state keep the conservative
// native-only release check; an intent is not a process-incarnation receipt.
func managedUpdateNativeReleased(ctx context.Context, ref platform.ServiceRef, state *ManagedServiceState) (bool, error) {
	if state == nil {
		return platform.OwnedServiceReleased(ctx, ref)
	}
	if state.PID != ref.RootPID {
		return false, fmt.Errorf("recorded state differs from released native root")
	}
	return platform.OwnedServiceReleasedForIncarnation(ctx, ref, state.Incarnation)
}
