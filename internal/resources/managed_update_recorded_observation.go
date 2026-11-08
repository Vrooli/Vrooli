package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	platform "github.com/vrooli/platform-go"
)

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
	if state.UpdateLaunch == nil {
		legacy, err := c.observeManagedServiceLegacyLaunchLocked(ctx, m)
		return managedUpdateAdmissionSource{SourceRevision: legacy.SourceRevision, Process: legacy.Process, Contract: legacy.Contract, Scope: &legacy.NativeScope, CredentialStateDir: paths.StateDir}, err
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
	if len(m.Ports) != 1 || m.Ports[0].Host != l.Runtime.Port {
		return managedUpdateAdmissionSource{}, fmt.Errorf("target changes the canonical serving port")
	}
	image, err := platform.OpenProcessExecutable(state.PID)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	defer image.Close()
	retained, err := retainManagedUpdateFile(ctx, c, m.Name, state.ArtifactVersion, state.ArtifactSHA256, image)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	rollback, err := newManagedUpdateContract(contract.Manifest, retained)
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

func managedUpdateOriginalQuiescent(ctx context.Context, u ManagedServiceUpdate) (bool, error) {
	if u.Original.UpdateLaunch != nil {
		ref := u.Original.UpdateLaunch.NativeRef
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
