package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

func (c *Controller) requireOpenRGBEnabled(ctx context.Context) error {
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return err
	}
	choice := state.Resources[openRGBResourceName]
	if choice.Enabled == nil || !*choice.Enabled {
		return errors.New("OpenRGB resource is disabled; no native update execution is admitted")
	}
	return nil
}

// The caller holds the resource lifecycle lock after persisting disabled
// intent. Retirement never restores, dispatches or republishes a backend.
func (c *Controller) retireOpenRGBUpdatesForDisable(ctx context.Context) error {
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		return err
	}
	ledger, err := store.read()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return retireOpenRGBUpdateRecords(ctx, ledger, revokeOpenRGBUpdateRole, cleanupManagedUpdateRole, stopManagedUpdateOriginalNative, managedUpdatePortUnbound, func(ctx context.Context, u ManagedServiceUpdate) error {
		_, err := store.Commit(ctx, u, UpdateFinishDisabled)
		return err
	})
}

func retireOpenRGBUpdateRecords(ctx context.Context, ledger managedUpdateLedger, revoke func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error, cleanup func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error, stopOriginal func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error), portAbsent func(int) error, finish func(context.Context, ManagedServiceUpdate) error) error {
	var failures []error
	blocked := map[string]bool{}
	// Revoke all delayed wrappers before potentially slow or failing stops.
	// One failed native cleanup must not leave another admitted role executable.
	for _, entry := range ledger.Records {
		u := entry.State
		if u.Resource != openRGBResourceName {
			failures = append(failures, errors.New("OpenRGB disable ledger contains a different resource"))
			blocked[u.ID] = true
			continue
		}
		for _, role := range openRGBUpdateRoles(u) {
			if err := revoke(ctx, u, role); err != nil {
				failures = append(failures, err)
				blocked[u.ID] = true
			}
		}
	}
	for _, entry := range ledger.Records {
		u := entry.State
		if u.Resource != openRGBResourceName {
			continue
		}
		retired := !blocked[u.ID]
		for _, role := range openRGBUpdateRoles(u) {
			if err := cleanup(ctx, u, role); err != nil {
				failures = append(failures, err)
				retired = false
			}
		}
		if u.Phase.Terminal() {
			continue
		}
		if managedUpdateOriginalNativeRef(u.Original) == nil && u.OriginalScope == nil {
			failures = append(failures, errors.New("OpenRGB disable cannot retire an original with unknown native ownership"))
			continue
		}
		receipt, err := stopOriginal(ctx, u)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !receipt.RootGone || !receipt.TreeQuiescent {
			failures = append(failures, errors.New("OpenRGB original native quiescence is unconfirmed during disable"))
			continue
		}
		if !retired {
			continue
		}
		u.PrimaryStopVerified, u.PrimaryQuiescent, u.CandidateQuiescent = true, true, true
		u.RollbackProbeQuiescent = u.RollbackProbeQuiescent || u.RollbackProbe.Intent != nil
		if u.Restored.Intent != nil {
			if err := portAbsent(u.Restored.Intent.Runtime.Port); err != nil {
				failures = append(failures, err)
				continue
			}
			q := &ManagedUpdateQuiescence{InstanceID: u.Restored.Intent.InstanceID, OwnershipTokenHash: u.Restored.Intent.OwnershipTokenHash, ObservedAt: time.Now().UTC(), NativeAbsent: true, ProcessesAbsent: true, ListenerAbsent: true}
			if u.Restored.State != nil {
				q.PID, q.Incarnation = u.Restored.State.PID, u.Restored.State.Incarnation
			}
			u.Restored.Quiescence = q
		}
		if err := finish(ctx, u); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func revokeOpenRGBUpdateRole(ctx context.Context, u ManagedServiceUpdate, role ManagedUpdateProcess) error {
	if role.Intent == nil {
		return nil
	}
	store, err := newManagedUpdateExecStore(u.Resource, u.ID, role.Intent.InstanceID)
	if err != nil {
		return err
	}
	_, err = store.Revoke(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: *role.Intent, State: managedUpdateExecFencePrepared})
	return err
}

func openRGBUpdateRoles(u ManagedServiceUpdate) []ManagedUpdateProcess {
	roles := []ManagedUpdateProcess{u.Candidate, u.Primary, u.RollbackProbe, u.Restored}
	for _, retired := range u.RetiredRestores {
		roles = append(roles, retired.Process)
	}
	return roles
}

// Metadata-only admission for a privileged device-access change. A revoked
// fence prevents delayed wrappers from running, and any already claimed root
// and cgroup must be empty before device permission mutation.
func ensureOpenRGBUpdateFencesRetired() error {
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		return err
	}
	ledger, err := store.read()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range ledger.Records {
		for _, role := range openRGBUpdateRoles(entry.State) {
			if role.Intent == nil {
				continue
			}
			fences, err := newManagedUpdateExecStore(openRGBResourceName, entry.State.ID, role.Intent.InstanceID)
			if err != nil {
				return err
			}
			f, err := fences.read()
			if err != nil || f.State != managedUpdateExecFenceRevoked || f.Intent != *role.Intent {
				return errors.Join(errors.New("OpenRGB admitted update role has not been retired"), err)
			}
			if f.Claim != nil {
				if err := verifyOpenRGBRetiredNative(*f.Claim); err != nil {
					return err
				}
			} else if f.NativeRetiredAt == nil {
				return errors.New("OpenRGB revoked update role still requires exact native cleanup acknowledgement")
			}
			if role.NativeRef != nil {
				if err := verifyOpenRGBRetiredNative(*role.NativeRef); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func verifyOpenRGBRetiredNative(ref platform.ServiceRef) error {
 if ref.Native!=nil{
  ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel()
  return verifyOpenRGBNativeRelease(ctx,ref,func(ctx context.Context,ref platform.ServiceRef,incarnation string)(bool,error){
   if incarnation!=""{return platform.OwnedServiceReleasedForIncarnation(ctx,ref,incarnation)}
   return platform.OwnedServiceReleased(ctx,ref)
  })
 }

	incarnation, err := process.Incarnation(ref.RootPID)
	if err == nil {
		_, ticks, ok := strings.Cut(incarnation, ":")
		if !ok || ticks == ref.RootStartTime {
			return errors.New("OpenRGB retired native root is still alive or unknown")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("observe retired OpenRGB native root: %w", err)
	}
	pids, err := platform.ScopeProcesses(ref.Scope)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(pids) != 0 {
		return errors.New("OpenRGB retired native cgroup remains populated")
	}
	return nil
}

// Opt-out has no replacement-serving phase. Permit only monotone observed
// cleanup receipts for its explicit terminal event; source, launch identity,
// phase, revision, abort intent and serving proofs remain immutable inputs.
func validOpenRGBUpdateRetirementObservation(old, next ManagedServiceUpdate) error {
	if old.Resource != openRGBResourceName || old.Phase.Terminal() {
		return ErrManagedUpdateConflict
	}
	immutable := next
	immutable.CandidateQuiescent = old.CandidateQuiescent
	immutable.PrimaryStopVerified = old.PrimaryStopVerified
	immutable.PrimaryQuiescent = old.PrimaryQuiescent
	immutable.RollbackProbeQuiescent = old.RollbackProbeQuiescent
	immutable.Restored.Quiescence = old.Restored.Quiescence
	if !reflect.DeepEqual(immutable, old) {
		return fmt.Errorf("%w: retirement changed immutable update inputs", ErrManagedUpdateConflict)
	}
	if old.CandidateQuiescent && !next.CandidateQuiescent || old.PrimaryStopVerified && !next.PrimaryStopVerified || old.PrimaryQuiescent && !next.PrimaryQuiescent || old.RollbackProbeQuiescent && !next.RollbackProbeQuiescent {
		return errors.New("retirement cannot revoke recorded quiescence")
	}
	if old.Restored.Quiescence != nil && next.Restored.Quiescence == nil || next.Restored.Quiescence != nil && !matchingUpdateQuiescence(next.Restored) {
		return errors.New("retirement changed exact restored ownership evidence")
	}
	return nil
}

// Native release observes the exact manager slot and containment, including an
// uncertain submit with no root PID. A PID disappearance alone is insufficient.
func verifyOpenRGBNativeRelease(ctx context.Context,ref platform.ServiceRef,released func(context.Context,platform.ServiceRef,string)(bool,error))error{
 if ref.Native==nil || ref.ValidateIdentity()!=nil{return platform.ErrOperationIdentity}
 incarnation:="";if ref.RootPID>0{incarnation=ref.Native.RootIncarnation;if !ref.MatchesIncarnation(incarnation){return platform.ErrOperationIdentity}}
 absent,err:=released(ctx,ref,incarnation);if err!=nil{return err};if !absent{return errors.New("OpenRGB retired native manager or containment remains present")};return nil
}
