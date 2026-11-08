package resources

import (
	"context"
	"fmt"
	"time"

	platform "github.com/vrooli/platform-go"
)

type managedUpdateOriginalStopReceipt struct {
	Legacy                  *platform.ServiceScopeRef
	Native                  *platform.ServiceRef
	RootGone, TreeQuiescent bool
}

// stopManagedUpdateOriginal is the original-primary effect boundary. The caller
// holds the lifecycle lock and has durably admitted primary_stopping. A fresh
// candidate proof is required again immediately before any original stop.
// Returned native quiescence is an observation; the runner must still verify
// primary-port exclusion and commit its checkpoint before a replacement starts.
func stopManagedUpdateOriginal(ctx context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := u.Validate(); err != nil {
		return managedUpdateOriginalStopReceipt{}, err
	}
	if u.Phase != UpdatePrimaryStopping || (u.OriginalScope == nil && u.Original.UpdateLaunch == nil) {
		return managedUpdateOriginalStopReceipt{}, fmt.Errorf("original primary stop lacks its admitted exact native owner")
	}
	if _, err := observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target); err != nil {
		return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse original stop without a current verified candidate: %w", err)
	}
	if u.RollbackQualificationRequired {
		if _, err := observeManagedUpdateProcess(ctx, u, u.RollbackProbe, u.Rollback); err != nil {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse original stop without its live qualified rollback probe: %w", err)
		}
	}
	return stopManagedUpdateOriginalNative(ctx, u)
}

// An ambiguous original stop can leave the old root alive and shutting down.
// Do not call that preserved success or leave restoration permanently unable
// to proceed. A current serving shadow must be proved before finishing the
// already-admitted exact stop and restoring the retained launch.
func stopManagedUpdateOriginalForRestoration(ctx context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := u.Validate(); err != nil {
		return managedUpdateOriginalStopReceipt{}, err
	}
	if u.Phase != UpdateRestoring || u.PrimaryStopVerified || u.Primary.Intent == nil {
		return managedUpdateOriginalStopReceipt{}, fmt.Errorf("original stop recovery is outside its admitted restoration phase")
	}
	_, rollbackErr := observeManagedUpdateProcess(ctx, u, u.RollbackProbe, u.Rollback)
	if rollbackErr != nil {
		if _, candidateErr := observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target); candidateErr != nil {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse unresolved original stop without a current serving shadow")
		}
	}
	return stopManagedUpdateOriginalNative(ctx, u)
}

func stopManagedUpdateOriginalNative(ctx context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
	if u.Original.UpdateLaunch != nil {
		ref := u.Original.UpdateLaunch.NativeRef
		quiescent, err := managedUpdateOriginalQuiescent(ctx, u)
		if err != nil {
			return managedUpdateOriginalStopReceipt{}, err
		}
		if !quiescent {
			native, err := platform.InspectOwnedService(ctx, ref)
			if err != nil {
				return managedUpdateOriginalStopReceipt{}, err
			}
			if native.RootAlive {
				if err := verifyManagedUpdateOriginalImage(ctx, u); err != nil {
					return managedUpdateOriginalStopReceipt{}, err
				}
			}
			receipt, err := platform.StopOwnedService(ctx, ref)
			if err != nil {
				return managedUpdateOriginalStopReceipt{}, err
			}
			if !receipt.NativeQuiescent {
				return managedUpdateOriginalStopReceipt{}, fmt.Errorf("original native service remains populated")
			}
		}
		if err := platform.ReleaseOwnedService(ctx, ref); err != nil {
			if released, e := managedUpdateNativeReleased(ctx, ref, &u.Original); e != nil || !released {
				return managedUpdateOriginalStopReceipt{}, err
			}
		}
		return managedUpdateOriginalStopReceipt{Native: &ref, RootGone: true, TreeQuiescent: true}, nil
	}
	native, err := platform.InspectServiceScope(ctx, *u.OriginalScope)
	if err != nil {
		return managedUpdateOriginalStopReceipt{}, err
	}
	if native.RootAlive {
		if err := verifyManagedUpdateOriginalImage(ctx, u); err != nil {
			return managedUpdateOriginalStopReceipt{}, err
		}
	}
	receipt, err := platform.StopServiceScope(ctx, *u.OriginalScope)
	return managedUpdateOriginalStopReceipt{Legacy: u.OriginalScope, RootGone: receipt.RootGone, TreeQuiescent: receipt.TreeQuiescent}, err
}

func verifyManagedUpdateOriginalImage(ctx context.Context, u ManagedServiceUpdate) error {
	if err := verifyManagedServiceOwnership(u.Original); err != nil {
		return err
	}
	image, err := platform.OpenProcessExecutable(u.Original.PID)
	if err != nil {
		return err
	}
	defer image.Close()
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return err
	}
	if digest != u.Rollback.Artifact.SHA256 {
		return fmt.Errorf("original image changed before native stop")
	}
	return nil
}
