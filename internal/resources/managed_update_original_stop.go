package resources

import (
	"context"
	"fmt"
	"os"
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
	if u.Phase != UpdatePrimaryStopping || (u.OriginalScope == nil && managedUpdateOriginalNativeRef(u.Original) == nil) {
		return managedUpdateOriginalStopReceipt{}, fmt.Errorf("original primary stop lacks its admitted exact native owner")
	}
	if u.Resource == openRGBResourceName {
		if handled, err := reconcileOpenRGBServingBoot(ctx, u.Original); handled || err != nil {
			if err != nil {
				return managedUpdateOriginalStopReceipt{}, err
			}
			return managedUpdateOriginalStopReceipt{Native: managedUpdateOriginalNativeRef(u.Original), RootGone: true, TreeQuiescent: true}, nil
		}
	}
	if exclusiveStopStart(u) {
		if u.Candidate.Intent != nil || u.RollbackProbe.Intent != nil || u.RollbackQualifiedAt == nil ||
			!freshStateUpdateProof(u.OriginalProof, u.Original, *u.RollbackQualifiedAt) {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("exclusive stop lacks in-place rollback qualification or contains a shadow role")
		}
		quiescent, err := managedUpdateOriginalQuiescent(ctx, u)
		if err != nil {
			return managedUpdateOriginalStopReceipt{}, err
		}
		if quiescent {
			// A prior stop may have completed before its release acknowledgement.
			// Reconcile that exact owner instead of leaving an empty native unit.
			return stopManagedUpdateOriginalNative(ctx, u)
		}
		if _, err := observeManagedUpdateOriginal(ctx, u); err != nil {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse original stop without current exact-owner readiness: %w", err)
		}
		if err := verifyManagedUpdateOriginalConfig(u); err != nil {
			return managedUpdateOriginalStopReceipt{}, err
		}
		if u.Resource == openRGBResourceName {
			if err := verifyOpenRGBManagedUpdateStop(ctx, u); err != nil {
				return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse OpenRGB original stop without current role policy proof: %w", err)
			}
		}
	} else {
		if _, err := observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target); err != nil {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse original stop without a current verified candidate: %w", err)
		}
		if u.RollbackQualificationRequired {
			if _, err := observeManagedUpdateProcess(ctx, u, u.RollbackProbe, u.Rollback); err != nil {
				return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse original stop without its live qualified rollback probe: %w", err)
			}
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
	if u.Resource == openRGBResourceName {
		if handled, err := reconcileOpenRGBServingBoot(ctx, u.Original); handled || err != nil {
			if err != nil {
				return managedUpdateOriginalStopReceipt{}, err
			}
			return managedUpdateOriginalStopReceipt{Native: managedUpdateOriginalNativeRef(u.Original), RootGone: true, TreeQuiescent: true}, nil
		}
	}
	if !exclusiveStopStart(u) {
		_, rollbackErr := observeManagedUpdateProcess(ctx, u, u.RollbackProbe, u.Rollback)
		if rollbackErr != nil {
			if _, candidateErr := observeManagedUpdateProcess(ctx, u, u.Candidate, u.Target); candidateErr != nil {
				return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse unresolved original stop without a current serving shadow")
			}
		}
	}
	if u.Resource == openRGBResourceName && exclusiveStopStart(u) {
		if err := verifyOpenRGBManagedUpdateStop(ctx, u); err != nil {
			return managedUpdateOriginalStopReceipt{}, fmt.Errorf("refuse OpenRGB restoration stop without current retained policy proof: %w", err)
		}
	}
	return stopManagedUpdateOriginalNative(ctx, u)
}

func stopManagedUpdateOriginalNative(ctx context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
	return stopManagedUpdateOriginalNativeWithBoot(ctx, u, reconcileOpenRGBServingBoot)
}

func stopManagedUpdateOriginalNativeWithBoot(ctx context.Context, u ManagedServiceUpdate, boot func(context.Context, ManagedServiceState) (bool, error)) (managedUpdateOriginalStopReceipt, error) {
	if u.Resource == openRGBResourceName {
		if handled, err := boot(ctx, u.Original); handled || err != nil {
			if err != nil {
				return managedUpdateOriginalStopReceipt{}, err
			}
			return managedUpdateOriginalStopReceipt{Native: managedUpdateOriginalNativeRef(u.Original), RootGone: true, TreeQuiescent: true}, nil
		}
	}
	if nativeRef := managedUpdateOriginalNativeRef(u.Original); nativeRef != nil {
		ref := *nativeRef
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
	var image *os.File
	var err error
	if u.Resource == openRGBResourceName {
		image, err = openOpenRGBRecordedImage(ctx, u.Original)
	} else {
		image, err = platform.OpenProcessExecutable(u.Original.PID)
	}
	if err != nil {
		return err
	}
	defer image.Close()
	if u.Resource == openRGBResourceName && managedUpdateOriginalNativeRef(u.Original) != nil {
		_, runtime, err := openRGBOriginalLaunch(u.Original)
		if err != nil {
			return err
		}
		owner := &Controller{Root: runtime.Root, Home: runtime.Home}
		stage, err := owner.verifyOpenRGBRoleArtifact(ctx, u.Rollback.Artifact)
		if err != nil {
			return err
		}
		if stage != nil {
			info, err := image.Stat()
			if err != nil || !os.SameFile(stage.payload, info) {
				return fmt.Errorf("OpenRGB original is not the witnessed staged payload: %v", err)
			}
			digest, err := managedUpdateImageDigest(ctx, image)
			if err != nil || digest != stage.payloadSHA256 {
				return fmt.Errorf("OpenRGB original payload changed before native stop: %v", err)
			}
			return nil
		}
	}
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return err
	}
	if digest != u.Rollback.Artifact.SHA256 {
		return fmt.Errorf("original image changed before native stop")
	}
	return nil
}
