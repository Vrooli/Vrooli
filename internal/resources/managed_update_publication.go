package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	platform "github.com/vrooli/platform-go"
)

// ManagedUpdateServingLaunch retains native ownership and the frozen private
// launch contract after the update finishes. Ordinary stop must use that owner.
type ManagedUpdateServingLaunch struct {
	OperationID string                           `json:"operation_id"`
	RequestID   string                           `json:"request_id"`
	Role        string                           `json:"role"`
	Contract    ManagedUpdateContract            `json:"contract"`
	Runtime     ManagedUpdateRuntime             `json:"runtime"`
	NativeRef   platform.ServiceRef              `json:"native_ref"`
	Credentials *ManagedUpdateCredentialSnapshot `json:"credentials,omitempty"`
}

// The serving publication roles a launch record names; both images read them
// back by value.
const (
	managedUpdateRolePrimary  = "primary"
	managedUpdateRoleRestored = "restored"
)

func validateManagedUpdateServing(state ManagedServiceState) error {
	l := state.UpdateLaunch
	if l == nil || !managedUpdateKey.MatchString(l.OperationID) || !managedUpdateKey.MatchString(l.RequestID) || l.Role != managedUpdateRolePrimary && l.Role != managedUpdateRoleRestored || l.Contract.validate() != nil || l.Runtime.validate() != nil || l.NativeRef.ValidateIdentity() != nil {
		return fmt.Errorf("invalid native managed-service serving record")
	}
	if state.ArtifactPath != l.Contract.Artifact.Path || state.ArtifactSHA256 != l.Contract.Artifact.SHA256 || state.ArtifactVersion != l.Contract.Artifact.Version || state.PID != l.NativeRef.RootPID || len(l.Contract.Manifest.Ports) != 1 || l.Runtime.Port != l.Contract.Manifest.Ports[0].Host {
		return fmt.Errorf("native serving record differs from its resource launch")
	}
	i := ManagedUpdateLaunchIntent{InstanceID: state.InstanceID, OwnershipTokenHash: state.OwnershipTokenHash, ContractDigest: l.Contract.Digest, NativeIntent: platform.ServiceIntent(l.NativeRef.Intent), Runtime: l.Runtime}
	u := ManagedServiceUpdate{ID: l.OperationID, RequestID: l.RequestID, Resource: l.Contract.Manifest.Name}
	if !l.NativeRef.MatchesIncarnation(state.Incarnation) || !validUpdateIntent(&i, l.Contract) || i.NativeIntent.ValidateSpec(managedUpdateWrapperSpec(u, i)) != nil {
		return fmt.Errorf("native serving record lost exact process/specification binding")
	}
	return nil
}

func writeManagedUpdateServingState(supervisor *ManagedServiceSupervisor, state ManagedServiceState) error {
	if err := validateManagedUpdateServing(state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := writeResourceOwnedFileDurable(supervisor.statePath, data, 0o600); err != nil {
		return fmt.Errorf("%w: native serving publication: %w", ErrManagedUpdateCommitUnknown, err)
	}
	return nil
}

func managedUpdatePublishedState(u ManagedServiceUpdate, role string) (ManagedServiceState, error) {
	p, c := u.Primary, u.Target
	if role == managedUpdateRoleRestored {
		p, c = u.Restored, u.Rollback
	} else if role != managedUpdateRolePrimary {
		return ManagedServiceState{}, fmt.Errorf("invalid serving publication role")
	}
	if !validUpdateProcess(p, c) {
		return ManagedServiceState{}, fmt.Errorf("serving publication has no exact observed native process")
	}
	state := *p.State
	state.UpdateLaunch = &ManagedUpdateServingLaunch{OperationID: u.ID, RequestID: u.RequestID, Role: role, Contract: c, Runtime: p.Intent.Runtime, NativeRef: *p.NativeRef, Credentials: u.Credentials}
	return state, validateManagedUpdateServing(state)
}

// Caller holds the lifecycle lock. Publication is a durable CAS over the exact
// original or a prior publication by this operation. Unknown/newer owners refuse.
func publishManagedUpdateServing(ctx context.Context, u ManagedServiceUpdate, role string) error {
	if role == managedUpdateRolePrimary && u.Phase != UpdateCommitting || role == managedUpdateRoleRestored && u.Phase != UpdateRestoreCommitting {
		return fmt.Errorf("serving publication is outside its admitted phase")
	}
	p, c := u.Primary, u.Target
	if role == managedUpdateRoleRestored {
		p, c = u.Restored, u.Rollback
	}
	if _, err := observeManagedUpdateProcess(ctx, u, p, c); err != nil {
		return err
	}
	desired, err := managedUpdatePublishedState(u, role)
	if err != nil {
		return err
	}
	supervisor, _, err := managedServiceSupervisorFor(u.Resource)
	if err != nil {
		return err
	}
	current, err := supervisor.readState()
	if err != nil {
		return err
	}
	allowed := reflect.DeepEqual(current, desired)
	original := current
	if original.Incarnation == "" {
		original.Incarnation = u.Original.Incarnation
	}
	allowed = allowed || reflect.DeepEqual(original, u.Original)
	for _, name := range []string{managedUpdateRolePrimary, managedUpdateRoleRestored} {
		prior, e := managedUpdatePublishedState(u, name)
		if e == nil && reflect.DeepEqual(current, prior) {
			allowed = true
		}
	}
	for _, retired := range u.RetiredRestores {
		old := u
		old.Restored = retired.Process
		prior, e := managedUpdatePublishedState(old, managedUpdateRoleRestored)
		if e == nil && reflect.DeepEqual(current, prior) {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("%w: another canonical serving owner replaced the original", ErrManagedUpdateConflict)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeManagedUpdateServingState(supervisor, desired)
}

func (s *ManagedServiceSupervisor) stopManagedUpdateServing(ctx context.Context, state ManagedServiceState) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := validateManagedUpdateServing(state); err != nil {
		return err
	}
	if state.UpdateLaunch.Contract.Manifest.Name == openRGBResourceName {
		handled, err := reconcileOpenRGBServingBoot(ctx, state)
		if err != nil {
			return err
		}
		if handled {
			if state.StoppedAt == nil {
				now := time.Now().UTC()
				state.StoppedAt = &now
			}
			return writeManagedUpdateServingState(s, state)
		}
	}
	ref := state.UpdateLaunch.NativeRef
	if state.StoppedAt != nil {
		if released, err := managedUpdateNativeReleased(ctx, ref, &state); err != nil {
			return err
		} else if released {
			return writeManagedUpdateServingState(s, state)
		}
	}
	receipt, err := platform.StopOwnedService(ctx, ref)
	if err != nil {
		return err
	}
	if !receipt.NativeQuiescent {
		return fmt.Errorf("native serving stop remains unresolved")
	}
	if state.StoppedAt == nil {
		now := time.Now().UTC()
		state.StoppedAt = &now
	}
	// Retain acknowledged quiescence before releasing the native unit. A lost
	// release reply can then be reconciled without pretending to know an exit.
	if err := writeManagedUpdateServingState(s, state); err != nil {
		return err
	}
	return platform.ReleaseOwnedService(ctx, ref)
}
