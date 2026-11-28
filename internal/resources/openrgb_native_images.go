package resources

import (
	"context"
	"errors"
	"os"
	"strings"

	platform "github.com/vrooli/platform-go"
)

func (c *Controller) openOpenRGBLauncherImage(ctx context.Context, native bool, retained *ManagedServiceState) (*os.File, error) {
	if native {
		if retained != nil {
			rt, err := c.openRGBRetainedNativeLauncherRuntime(ctx, *retained)
			if err != nil {
				return nil, err
			}
			proof := platform.NativeLauncherProof{SchemaVersion: platform.NativeLauncherProofVersion, Profile: platform.NativeLauncherProfileOpenRGB, SHA256: rt.LauncherSHA256}
			return platform.OpenPublishedNativeLauncher(ctx, rt.Launcher, proof)
		}
		return platform.OpenInstalledNativeSupervisorImage(ctx)
	}
	return platform.OpenProcessExecutable(os.Getpid())
}

func openOpenRGBServingImage(ctx context.Context, ref platform.ServiceRef, w openRGBExecWitness) (*os.File, error) {
	if ref.Native == nil {
		return platform.OpenProcessExecutable(ref.RootPID)
	}
	if w.ExecutionChild == nil {
		if ref.Native.Platform == openRGBPlatformWindows {
			return nil, errors.New("Windows SDK image requires its exact claimed child")
		}
		return platform.OpenOwnedServiceRootImage(ctx, ref)
	}
	child := w.ExecutionChild
	if err := child.validateFor(ref); err != nil {
		return nil, err
	}
	return platform.OpenOwnedServiceChildImage(ctx, ref, platform.OwnedImageChildIdentity{PID: child.PID, Incarnation: child.Incarnation, ImageID: child.ImageID, ImageHandle: child.ImageHandle, SecurityProof: child.SecurityProof, NativeAccount: child.NativeAccount, ContainmentID: child.ContainmentID})
}

func openOpenRGBRoleImage(ctx context.Context, resource string, ref platform.ServiceRef, w *openRGBExecWitness, pid int) (*os.File, error) {
	if resource != openRGBResourceName || ref.Native == nil {
		return platform.OpenProcessExecutable(pid)
	}
	if w == nil {
		return nil, errors.New("native SDK image has no consumed device witness")
	}
	return openOpenRGBServingImage(ctx, ref, *w)
}

func readOpenRGBRecordedNativeClaim(state ManagedServiceState) (ManagedUpdateRuntime, platform.ServiceRef, openRGBExecWitness, error) {
	var runtime ManagedUpdateRuntime
	var zeroRef platform.ServiceRef
	var zeroWitness openRGBExecWitness
	ref := managedUpdateOriginalNativeRef(state)
	if ref == nil || ref.Native == nil {
		return runtime, zeroRef, zeroWitness, errors.New("retained native launcher has no native ownership record")
	}
	if state.InitialLaunch != nil {
		if err := validateManagedInitialServing(state); err != nil {
			return runtime, zeroRef, zeroWitness, err
		}
	} else if state.UpdateLaunch != nil {
		if err := validateManagedUpdateServing(state); err != nil {
			return runtime, zeroRef, zeroWitness, err
		}
	}
	var fence managedUpdateExecFence
	var intent ManagedUpdateLaunchIntent
	var contract ManagedUpdateContract
	if state.InitialLaunch != nil {
		a, f, err := readOpenRGBInitialServing(state)
		if err != nil {
			return runtime, zeroRef, zeroWitness, err
		}
		fence = f
		intent = a.fence().Intent
		contract = a.Contract
		runtime = a.Runtime
	} else if state.UpdateLaunch != nil {
		l := state.UpdateLaunch
		store, err := newManagedUpdateExecStore(openRGBResourceName, l.OperationID, state.InstanceID)
		if err != nil {
			return runtime, zeroRef, zeroWitness, err
		}
		fence, err = store.read()
		if err != nil {
			return runtime, zeroRef, zeroWitness, err
		}
		intent = ManagedUpdateLaunchIntent{InstanceID: state.InstanceID, OwnershipTokenHash: state.OwnershipTokenHash, ContractDigest: l.Contract.Digest, Runtime: l.Runtime, NativeIntent: platform.ServiceIntent(l.NativeRef.Intent)}
		contract = l.Contract
		runtime = l.Runtime
	} else {
		return runtime, zeroRef, zeroWitness, errors.New("native SDK image has no retained launch")
	}
	witness, err := verifyOpenRGBRetainedClaim(fence, intent, contract, *ref, state.Incarnation)
	if err != nil {
		return runtime, zeroRef, zeroWitness, err
	}
	return runtime, *ref, witness, nil
}

func openOpenRGBRecordedImage(ctx context.Context, state ManagedServiceState) (*os.File, error) {
	ref := managedUpdateOriginalNativeRef(state)
	if ref == nil || ref.Native == nil {
		return platform.OpenProcessExecutable(state.PID)
	}
	_, native, witness, err := readOpenRGBRecordedNativeClaim(state)
	if err != nil {
		return nil, err
	}
	return openOpenRGBServingImage(ctx, native, witness)
}

// A retained current role selects one exact launcher. This never enumerates
// published versions or guesses which historical code should become authority.
func (c *Controller) openRGBRetainedNativeLauncherRuntime(ctx context.Context, state ManagedServiceState) (ManagedUpdateRuntime, error) {
	var zero ManagedUpdateRuntime
	if c == nil || ctx == nil {
		return zero, errors.New("retained native launcher requires its resource owner")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	rt, ref, witness, err := readOpenRGBRecordedNativeClaim(state)
	if err != nil {
		return zero, err
	}
	if err = rt.validate(); err != nil {
		return zero, err
	}
	if rt.NativeAccount == "" || ref.Native == nil || rt.NativeAccount != ref.Native.Account || rt.OwnerPrincipal != rt.NativeAccount || witness.Policy.Platform != ref.Native.Platform || witness.Policy.Account != rt.NativeAccount {
		return zero, errors.New("retained launcher account differs from its consumed native role")
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return zero, err
	}
	samePath := func(a, b string) bool {
		if ref.Native.Platform == openRGBPlatformWindows {
			return strings.EqualFold(a, b)
		}
		return a == b
	}
	if !samePath(rt.StateDir, paths.StateDir) || !samePath(rt.Home, c.Home) {
		return zero, errors.New("retained launcher differs from the exact resource namespace")
	}
	contract, _, err := openRGBOriginalLaunch(state)
	if err != nil {
		return zero, err
	}
	role, err := c.openRGBFrozenOriginalRole(ctx, state)
	if err != nil {
		return zero, err
	}
	binding, err := role.binding(contract)
	if err != nil {
		return zero, err
	}
	if binding != witness.Policy {
		return zero, errors.New("retained launcher lost its frozen selected-device claim")
	}
	return rt, nil
}
