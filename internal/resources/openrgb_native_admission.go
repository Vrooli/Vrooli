package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/vrooli/api-core/localprincipal"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/process"
)

// The logical owner retains this server only across one admitted dispatch and
// readiness observation. Its disappearance never permits the native wrapper to
// recreate the server, change the capsule or consume a second claim.
func (c *Controller) startOpenRGBNativeAdmission(ctx context.Context, id string, intent platform.ServiceIntent, expected managedUpdateExecFence, material openRGBNativeMaterial, fences *managedUpdateExecStore, validateCurrent func(context.Context) error) (*openRGBNativeClaimServer, error) {
	if material.Runtime.NativeAccount == "" || material.Runtime.NativeAccount != intent.NativeAccount || material.NativePolicy.Account != material.Policy.Account {
		return nil, errors.New("native OpenRGB admission has no exact enrolled service account")
	}
	principal, err := localprincipal.Current()
	if err != nil {
		return nil, err
	}
	incarnation, err := process.Incarnation(os.Getpid())
	if err != nil {
		return nil, err
	}
	endpoint, err := openRGBNativeCapsuleEndpoint(id)
	if err != nil {
		return nil, err
	}
	capsule := openRGBNativeCapsule{SchemaVersion: openRGBNativeCapsuleVersion, ID: id, Intent: intent, Account: intent.NativeAccount, LauncherSHA256: material.Runtime.LauncherSHA256, RuntimeRoot: material.Invocation.RuntimeRoot, MaterialSHA256: material.digest(), Endpoint: endpoint, OwnerPID: os.Getpid(), OwnerIncarnation: incarnation, OwnerPrincipal: principal.String()}
	current := func(ctx context.Context, ref platform.ServiceRef) (openRGBNativeMaterial, error) {
		if err := validateCurrent(ctx); err != nil {
			return openRGBNativeMaterial{}, err
		}
		f, err := fences.read()
		if err != nil {
			return openRGBNativeMaterial{}, err
		}
		if !sameManagedExecIdentity(f, expected) || f.State != managedUpdateExecFencePrepared || f.Claim != nil {
			return openRGBNativeMaterial{}, errors.New("native OpenRGB admission was changed, consumed or revoked")
		}
		if err := verifyOpenRGBNativeWrapper(ctx, material.Runtime, intent, ref); err != nil {
			return openRGBNativeMaterial{}, err
		}
		result := material
		result.ClaimSHA256 = f.openRGBClaimDigest(ref)
		return result, result.validate(capsule)
	}
	callbacks := openRGBNativeClaimCallbacks{Lookup: platform.LookupOwnedService, Inspect: platform.InspectOwnedService, Incarnation: process.Incarnation, ResolveAccount: platform.ResolveOwnedServiceAccount, Prepare: current}
	callbacks.Claim = func(ctx context.Context, ref platform.ServiceRef, witness openRGBExecWitness) error {
		m, err := current(ctx, ref)
		if err != nil {
			return err
		}
		if witness.Policy != m.Policy || witness.ClaimSHA256 != m.ClaimSHA256 || !ref.MatchesIncarnation(witness.Incarnation) {
			return errors.New("native OpenRGB witness differs from admitted exact owner")
		}
		contextSHA, err := openRGBServingContext(ref, witness.Incarnation)
		if err != nil || contextSHA != witness.ContextSHA256 {
			return errors.Join(errors.New("native OpenRGB witness security context changed"), err)
		}
		generation, err := hostreqkit.ValidateNativeOpenRGBDeviceSelection(ctx, m.NativePolicy)
		if err != nil {
			return err
		}
		selected := slices.Clone(generation.SelectedPaths)
		slices.Sort(selected)
		if generation.DeviceGenerationSHA256 != witness.DeviceGenerationSHA256 || !slices.Equal(selected, witness.SelectedNodes) {
			return errors.New("native OpenRGB selected attachment changed before claim")
		}
		if err := verifyOpenRGBNativeExecutionChild(ctx, m, ref, witness); err != nil {
			return err
		}
		_, err = fences.ClaimOpenRGB(ctx, expected, ref, witness)
		return err
	}
	server, err := startOpenRGBNativeClaimServer(ctx, capsule, callbacks)
	if err != nil {
		return nil, err
	}
	if _, err := publishOpenRGBNativeCapsule(capsule); err != nil {
		_ = server.Close()
		return nil, err
	}
	return server, nil
}

func verifyOpenRGBNativeWrapper(ctx context.Context, r ManagedUpdateRuntime, intent platform.ServiceIntent, ref platform.ServiceRef) error {
	if ref.ValidateIdentity() != nil || platform.ServiceIntent(ref.Intent) != intent || !ref.NativeOwnedIdentityComplete() {
		return errors.New("native OpenRGB wrapper differs from its admitted manager identity")
	}
	before, err := process.Incarnation(ref.RootPID)
	if err != nil || !ref.MatchesIncarnation(before) {
		return errors.Join(errors.New("native wrapper incarnation changed"), err)
	}
	image, err := platform.OpenOwnedServiceRootImage(ctx, ref)
	if err != nil {
		return err
	}
	digest, err := managedUpdateImageDigest(ctx, image)
	_ = image.Close()
	if err != nil || digest != r.LauncherSHA256 {
		return errors.Join(errors.New("native wrapper image differs from its frozen launcher"), err)
	}
	args, err := platform.ProcessArguments(ref.RootPID)
	if err != nil {
		return err
	}
	capsule, err := openRGBNativeCapsulePath(intent.AttemptID)
	if err != nil {
		return err
	}
	if !slices.Equal(args, []string{r.Launcher, "__vrooli-openrgb-native-capsule", capsule}) {
		return errors.New("native wrapper arguments differ from its fixed capsule")
	}
	env, err := platform.ReadProcessEnvironment(ref.RootPID)
	if err != nil {
		return err
	}
	if len(env) != 0 {
		return errors.New("native wrapper inherited an unadmitted environment")
	}
	dir := filepath.Dir(r.Launcher)
	matches, err := platform.ProcessWorkingDirectoryMatches(ref.RootPID, dir)
	if err != nil || !matches {
		return errors.Join(errors.New("native wrapper working directory differs from admission"), err)
	}
	spec := r.nativeWrapperSpec(intent.AttemptID)
	if err := intent.ValidateSpec(spec); err != nil {
		return err
	}
	after, err := process.Incarnation(ref.RootPID)
	if err != nil || after != before {
		return errors.Join(errors.New("native wrapper changed during image observation"), err)
	}
	return nil
}

func (c *Controller) startOpenRGBNativeInitialAdmission(ctx context.Context, a openRGBInitialAttempt, fences *managedUpdateExecStore) (*openRGBNativeClaimServer, error) {
	if a.NativeDevicePolicy == nil {
		return nil, errors.New("native initial launch has no native device scope")
	}
	token, err := readManagedUpdateToken(a.tokenPath())
	if err != nil {
		return nil, err
	}
	invocation, err := renderManagedServiceInvocation(a.Contract, a.Runtime, a.root(), token)
	if err != nil {
		return nil, err
	}
	stage, err := c.OpenRGBAppImageStagePath(a.Contract.Artifact)
	if err != nil {
		return nil, err
	}
	material := openRGBNativeMaterial{Contract: a.Contract, Runtime: a.Runtime, Invocation: invocation, StagePath: stage, NativePolicy: *a.NativeDevicePolicy, Catalog: slices.Clone(a.Catalog), Policy: a.Policy, TokenHash: a.TokenHash}
	if _, err := c.verifyOpenRGBRoleArtifact(ctx, material.Contract.Artifact); err != nil {
		return nil, err
	}
	return c.startOpenRGBNativeAdmission(ctx, a.ID, a.Intent, a.fence(), material, fences, func(ctx context.Context) error {
		if err := c.requireOpenRGBEnabled(ctx); err != nil {
			return err
		}
		if err := c.verifyOpenRGBNativeInitialLightingIntent(a); err != nil {
			return err
		}
		store, err := newOpenRGBInitialStore()
		if err != nil {
			return err
		}
		latest, err := store.read(a.ID)
		if err != nil {
			return err
		}
		current, err := store.current()
		if err != nil {
			return err
		}
		if current != a.ID || latest.State != openRGBInitialSubmitted || !reflect.DeepEqual(latest, a) {
			return errors.New("native initial admission is no longer current")
		}
		return nil
	})
}

func (c *Controller) startOpenRGBNativeUpdateAdmission(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract, intent ManagedUpdateLaunchIntent) (*openRGBNativeClaimServer, error) {
	role, err := openRGBFrozenUpdateRoleFor(u, contract)
	if err != nil {
		return nil, err
	}
	if role.NativePolicy == nil {
		return nil, errors.New("native update has no retained platform policy")
	}
	fences, err := newManagedUpdateExecStore(openRGBResourceName, u.ID, intent.InstanceID)
	if err != nil {
		return nil, err
	}
	fence, err := fences.read()
	if err != nil {
		return nil, err
	}
	if fence.Intent != intent || fence.OpenRGBPolicy == nil {
		return nil, errors.New("native update fence differs from dispatch intent")
	}
	tokenPath, err := intent.Runtime.tokenPath(u.ID, intent.InstanceID)
	if err != nil {
		return nil, err
	}
	token, err := readManagedUpdateToken(tokenPath)
	if err != nil {
		return nil, err
	}
	invocation, err := renderManagedUpdateInvocation(contract, intent.Runtime, u.ID, intent.InstanceID, token)
	if err != nil {
		return nil, err
	}
	stage, err := c.OpenRGBAppImageStagePath(contract.Artifact)
	if err != nil {
		return nil, err
	}
	material := openRGBNativeMaterial{Contract: contract, Runtime: intent.Runtime, Invocation: invocation, StagePath: stage, NativePolicy: *role.NativePolicy, Catalog: slices.Clone(role.Catalog), Policy: *fence.OpenRGBPolicy, TokenHash: intent.OwnershipTokenHash}
	if _, err := c.verifyOpenRGBRoleArtifact(ctx, material.Contract.Artifact); err != nil {
		return nil, err
	}
	return c.startOpenRGBNativeAdmission(ctx, intent.InstanceID, intent.NativeIntent, fence, material, fences, func(ctx context.Context) error {
		if err := c.requireOpenRGBEnabled(ctx); err != nil {
			return err
		}
		store, err := newManagedUpdateStore(openRGBResourceName)
		if err != nil {
			return err
		}
		latest, err := store.Lookup(u.RequestID)
		if err != nil {
			return err
		}
		admitted, selected, err := managedUpdateAdmittedRole(latest, intent.InstanceID)
		if err != nil {
			return err
		}
		if admitted.Intent == nil || *admitted.Intent != intent || selected.Digest != contract.Digest {
			return errors.New("native update role changed before execution")
		}
		return c.qualifyOpenRGBResumption(ctx, latest)
	})
}
