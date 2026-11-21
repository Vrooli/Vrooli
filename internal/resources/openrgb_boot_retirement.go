package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

const openRGBNativeBootRetirementVersion = 1
const openRGBNativeBootRetirementLimit = 16

// A persisted native manager may start its frozen wrapper after a reboot. This
// receipt captures that separate incarnation for cleanup; the consumed payload
// claim and every published launch reference remain unchanged.
type openRGBNativeBootRetirement struct {
	SchemaVersion int                 `json:"version"`
	Original      platform.ServiceRef `json:"original"`
	Observed      platform.ServiceRef `json:"observed"`
	CapturedAt    time.Time           `json:"captured_at"`
	QuiescentAt   *time.Time          `json:"quiescent_at,omitempty"`
	ReleasedAt    *time.Time          `json:"released_at,omitempty"`
}

func sameOpenRGBNativeManager(a, b platform.ServiceRef) bool {
	if a.Native == nil || b.Native == nil || a.ValidateIdentity() != nil || b.ValidateIdentity() != nil {
		return false
	}
	ac, bc := a, b
	an, bn := *a.Native, *b.Native
	an.BootID = ""
	bn.BootID = ""
	an.RootIncarnation = ""
	bn.RootIncarnation = ""
	an.RootUniqueID = ""
	bn.RootUniqueID = ""
	ac.Native = &an
	bc.Native = &bn
	ac.RootPID = 0
	bc.RootPID = 0
	ac.RootStartTime = ""
	bc.RootStartTime = ""
	return ac.Equal(bc)
}

func (f managedUpdateExecFence) validateNativeBootRetirements() error {
	if len(f.NativeBootRetirements) == 0 {
		return nil
	}
	if f.Resource != openRGBResourceName || f.State != managedUpdateExecFenceRevoked || f.Claim == nil || f.Claim.Native == nil || f.Claim.Native.Platform != openRGBPlatformDarwin && f.Claim.Native.Platform != openRGBPlatformWindows || len(f.NativeBootRetirements) > openRGBNativeBootRetirementLimit {
		return errors.New("native boot cleanup requires an exact revoked native OpenRGB claim")
	}
	seen := map[string]bool{}
	for _, r := range f.NativeBootRetirements {
		if r.SchemaVersion != openRGBNativeBootRetirementVersion || !r.Original.Equal(*f.Claim) || !sameOpenRGBNativeManager(r.Original, r.Observed) || strings.EqualFold(r.Original.Native.BootID, r.Observed.Native.BootID) || r.CapturedAt.IsZero() || f.RevokedAt == nil || r.CapturedAt.Before(*f.RevokedAt) {
			return errors.New("native boot cleanup differs from the retained manager claim")
		}
		boot := strings.ToLower(r.Observed.Native.BootID)
		if seen[boot] {
			return errors.New("native boot cleanup repeats a boot identity")
		}
		seen[boot] = true
		if r.QuiescentAt != nil && (r.QuiescentAt.IsZero() || r.QuiescentAt.Before(r.CapturedAt)) {
			return errors.New("native boot cleanup quiescence precedes capture")
		}
		if r.ReleasedAt != nil && (r.QuiescentAt == nil || r.ReleasedAt.IsZero() || r.ReleasedAt.Before(*r.QuiescentAt)) {
			return errors.New("native boot cleanup release lacks prior quiescence")
		}
	}
	return nil
}

func (s *managedUpdateExecStore) changeNativeBootRetirement(ctx context.Context, expected managedUpdateExecFence, ref platform.ServiceRef, phase string) (managedUpdateExecFence, error) {
	release, err := s.lock(ctx)
	if err != nil {
		return managedUpdateExecFence{}, err
	}
	defer release()
	if err = ctx.Err(); err != nil {
		return managedUpdateExecFence{}, err
	}
	f, err := s.read()
	if err != nil {
		return f, err
	}
	if !sameManagedExecIdentity(f, expected) || f.State != managedUpdateExecFenceRevoked || f.Claim == nil || expected.Claim == nil || !f.Claim.Equal(*expected.Claim) {
		return f, ErrManagedUpdateConflict
	}
	index := -1
	for i, r := range f.NativeBootRetirements {
		if r.Observed.Equal(ref) {
			index = i
			break
		}
	}
	now := s.now().UTC()
	if phase == "capture" && index < 0 {
		if len(f.NativeBootRetirements) >= openRGBNativeBootRetirementLimit {
			return f, errors.New("native boot cleanup history bound reached")
		}
		f.NativeBootRetirements = append(f.NativeBootRetirements, openRGBNativeBootRetirement{SchemaVersion: openRGBNativeBootRetirementVersion, Original: *f.Claim, Observed: ref, CapturedAt: now})
		index = len(f.NativeBootRetirements) - 1
	} else if index < 0 {
		return f, ErrManagedUpdateConflict
	}
	r := &f.NativeBootRetirements[index]
	switch phase {
	case "capture":
	case "quiescent":
		if r.QuiescentAt == nil {
			r.QuiescentAt = &now
		}
	case "released":
		if r.QuiescentAt == nil {
			return f, errors.New("native boot cleanup release requires recorded quiescence")
		}
		if r.ReleasedAt == nil {
			r.ReleasedAt = &now
		}
	default:
		return f, errors.New("invalid native boot cleanup transition")
	}
	if err = s.save(f); err != nil {
		return f, err
	}
	return f, nil
}

// retireNativeBootWith handles only a new-boot incarnation or already retained
// boot cleanup. Same-boot replacements are never substituted for the claim.
// Each effect is preceded by durable capture and followed by its own receipt.
func retireOpenRGBNativeBootWith(ctx context.Context, s *managedUpdateExecStore, f managedUpdateExecFence, current *platform.ServiceRef, o openRGBInitialCleanupOwner) (bool, error) {
	if f.Resource != openRGBResourceName || f.Claim == nil || f.Claim.Native == nil {
		return false, nil
	}
	original := *f.Claim
	if current != nil {
		if !sameOpenRGBNativeManager(original, *current) {
			return false, errors.New("native cleanup manager differs from retained claim")
		}
		if strings.EqualFold(original.Native.BootID, current.Native.BootID) {
			if !original.Equal(*current) && !(current.RootPID == 0 && current.Native.RootIncarnation == "") {
				return false, errors.New("same-boot native cleanup root differs from retained claim")
			}
			if len(f.NativeBootRetirements) == 0 {
				return false, nil
			}
			return true, errors.New("native cleanup returned to a retired boot identity")
		}
		ref := *current
		for _, r := range f.NativeBootRetirements {
			if strings.EqualFold(r.Observed.Native.BootID, ref.Native.BootID) {
				if !r.Observed.Equal(ref) && !(ref.RootPID == 0 && ref.Native.RootIncarnation == "") {
					return true, errors.New("same-boot native cleanup root changed after capture")
				}
				ref = r.Observed
				break
			}
		}
		var captureErr error
		f, captureErr = s.changeNativeBootRetirement(ctx, f, ref, "capture")
		if captureErr != nil {
			return true, captureErr
		}
		var receipt *openRGBNativeBootRetirement
		for i := range f.NativeBootRetirements {
			if f.NativeBootRetirements[i].Observed.Equal(ref) {
				receipt = &f.NativeBootRetirements[i]
				break
			}
		}
		if receipt == nil {
			return true, ErrManagedUpdateConflict
		}
		if receipt.ReleasedAt != nil {
			return true, errors.New("released native manager appeared again in the same boot")
		}
		if receipt.QuiescentAt == nil {
			stopped, err := o.stop(ctx, ref)
			if err != nil {
				return true, err
			}
			if !stopped.NativeQuiescent || !platform.ServiceRef(stopped.Ref).Equal(ref) {
				return true, errors.New("native boot cleanup lacks exact quiescence acknowledgement")
			}
		}
		var persistErr error
		f, persistErr = s.changeNativeBootRetirement(ctx, f, ref, "quiescent")
		if persistErr != nil {
			return true, persistErr
		}
		released, err := o.released(ctx, ref)
		if err != nil {
			return true, err
		}
		if !released {
			if err = o.release(ctx, ref); err != nil {
				return true, err
			}
		}
		f, err = s.changeNativeBootRetirement(ctx, f, ref, "released")
		if err != nil {
			return true, err
		}
	}
	if len(f.NativeBootRetirements) == 0 {
		return false, nil
	}
	// A later reboot may have interrupted an earlier stop/release. Once the
	// current manager is removed, authoritative account-wide release reconciles
	// those historical references without inventing a process exit result.
	for _, r := range f.NativeBootRetirements {
		released, err := o.released(ctx, r.Observed)
		if err != nil {
			return true, err
		}
		if !released {
			return true, fmt.Errorf("native boot cleanup release remains unresolved")
		}
		if r.QuiescentAt == nil {
			f, err = s.changeNativeBootRetirement(ctx, f, r.Observed, "quiescent")
			if err != nil {
				return true, err
			}
		}
		f, err = s.changeNativeBootRetirement(ctx, f, r.Observed, "released")
		if err != nil {
			return true, err
		}
	}
	return true, nil
}

func openRGBNativeCleanupOwner() openRGBInitialCleanupOwner {
	return openRGBInitialCleanupOwner{lookup: platform.LookupOwnedService, absent: platform.OwnedServiceIntentAbsent, stop: platform.StopOwnedService, release: platform.ReleaseOwnedService, released: platform.OwnedServiceReleased}
}

func retireOpenRGBNativeBoot(ctx context.Context, s *managedUpdateExecStore, f managedUpdateExecFence, current *platform.ServiceRef) (bool, error) {
	return retireOpenRGBNativeBootWith(ctx, s, f, current, openRGBNativeCleanupOwner())
}

// reconcileOpenRGBServingBoot is cleanup of a persisted manager that cannot
// replay its old payload admission after boot. It is deliberately independent
// of current device presence or hardware access qualification.
func reconcileOpenRGBServingBoot(ctx context.Context, state ManagedServiceState) (bool, error) {
	ref := managedUpdateOriginalNativeRef(state)
	if ref == nil || ref.Native == nil {
		return false, nil
	}
	var store *managedUpdateExecStore
	var err error
	if state.InitialLaunch != nil {
		store, err = newManagedInitialExecStore(state.InitialLaunch.AttemptID)
	} else if state.UpdateLaunch != nil {
		store, err = newManagedUpdateExecStore(openRGBResourceName, state.UpdateLaunch.OperationID, state.InstanceID)
	} else {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	f, err := store.read()
	if err != nil {
		return false, err
	}
	if f.Claim == nil || !f.Claim.Equal(*ref) {
		return false, errors.New("native boot cleanup publication differs from consumed claim")
	}
	current, lookupErr := platform.LookupOwnedService(ctx, platform.ServiceIntent(ref.Intent))
	if lookupErr != nil && !errors.Is(lookupErr, platform.ErrOperationOutcomeLost) {
		return false, lookupErr
	}
	var observed *platform.ServiceRef
	if lookupErr == nil {
		observed = &current
		if current.Native == nil {
			return false, errors.New("native boot cleanup lookup lost platform identity")
		}
		if strings.EqualFold(current.Native.BootID, ref.Native.BootID) && len(f.NativeBootRetirements) == 0 {
			return false, nil
		}
	} else if len(f.NativeBootRetirements) == 0 {
		return false, nil
	}
	f, err = store.Revoke(ctx, f)
	if err != nil {
		return true, err
	}
	handled, err := retireOpenRGBNativeBoot(ctx, store, f, observed)
	if err != nil || !handled {
		return handled, err
	}
	released, err := platform.OwnedServiceReleased(ctx, *ref)
	if err != nil {
		return true, err
	}
	if !released {
		return true, errors.New("original native manager release remains unresolved after boot cleanup")
	}
	return true, store.recordNativeRetired(ctx, f)
}
