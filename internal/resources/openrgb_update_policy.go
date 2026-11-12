package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var errOpenRGBReleasePayloadUnqualified = errors.New("OpenRGB release has no qualified envelope, extraction closure and serving executable profile")

type openRGBUpdateRolePolicy struct {
	binding      openRGBExecPolicy
	policy       hostreqkit.OpenRGBLinuxPolicy
	nativePolicy *hostreqkit.OpenRGBNativePolicy
}

func (r openRGBUpdateRolePolicy) frozen() openRGBFrozenUpdateRole {
	return openRGBFrozenUpdateRole{Policy: r.policy, NativePolicy: r.nativePolicy}
}

type openRGBUpdatePolicies struct {
	original, target, rollback openRGBUpdateRolePolicy
}

func qualifyOpenRGBUpdatePolicies(u ManagedServiceUpdate, load func(ManagedUpdateContract) (openRGBUpdateRolePolicy, error)) (openRGBUpdatePolicies, error) {
	var roles openRGBUpdatePolicies
	original, _, err := openRGBOriginalLaunch(u.Original)
	if err != nil || u.Resource != openRGBResourceName || !exclusiveStopStart(u) || u.OriginalScope != nil || original.Artifact.SHA256 != u.Rollback.Artifact.SHA256 || original.Artifact.Version != u.Rollback.Artifact.Version {
		return roles, errors.New("OpenRGB update requires a witnessed native original and its exact retained rollback artifact")
	}
	for _, role := range []struct {
		name     string
		contract ManagedUpdateContract
		out      *openRGBUpdateRolePolicy
	}{{"original", original, &roles.original}, {"target", u.Target, &roles.target}, {"rollback", u.Rollback, &roles.rollback}} {
		policy, err := load(role.contract)
		if err != nil {
			return openRGBUpdatePolicies{}, fmt.Errorf("qualify OpenRGB %s policy: %w", role.name, err)
		}
		id, artifact, catalog, err := policy.frozen().policyIdentity()
		if err != nil || !policy.binding.valid() || policy.binding.ID != id || policy.binding.ArtifactSHA256 != role.contract.Artifact.SHA256 || policy.binding.ArtifactSHA256 != artifact || policy.binding.DetectorCatalogSHA256 != catalog || policy.binding.ConfigSHA256 != role.contract.ConfigSHA256 {
			return openRGBUpdatePolicies{}, fmt.Errorf("OpenRGB %s policy differs from its exact artifact and frozen settings", role.name)
		}
		*role.out = policy
	}
	if roles.original.binding.ID != roles.rollback.binding.ID || !roles.original.frozen().samePhysicalGrant(roles.target.frozen()) {
		return openRGBUpdatePolicies{}, errors.New("OpenRGB update would change the installed physical device grant")
	}
	return roles, nil
}

func openRGBOriginalLaunch(state ManagedServiceState) (ManagedUpdateContract, ManagedUpdateRuntime, error) {
	if state.InitialLaunch != nil {
		a, _, err := readOpenRGBInitialServing(state)
		return a.Contract, a.Runtime, err
	}
	if state.UpdateLaunch != nil {
		return state.UpdateLaunch.Contract, state.UpdateLaunch.Runtime, nil
	}
	return ManagedUpdateContract{}, ManagedUpdateRuntime{}, errors.New("OpenRGB original has no witnessed native launch")
}

func sameOpenRGBPhysicalGrant(a, b hostreqkit.OpenRGBLinuxPolicy) bool {
	if a.MachineID != b.MachineID || a.GroupID != b.GroupID || a.OwnerUID != b.OwnerUID || a.RuleSHA256 != b.RuleSHA256 || len(a.Devices) != len(b.Devices) {
		return false
	}
	physical := func(devices []hostreqkit.OpenRGBLinuxDevice) []string {
		keys := make([]string, 0, len(devices))
		for _, d := range devices {
			keys = append(keys, strings.Join([]string{d.Kind, d.Topology, d.HardwareID, d.Adapter}, "\x00"))
		}
		slices.Sort(keys)
		return keys
	}
	return slices.Equal(physical(a.Devices), physical(b.Devices))
}

func readOpenRGBOriginalWitness(u ManagedServiceUpdate, live bool) (openRGBExecWitness, error) {
	if u.Original.InitialLaunch != nil {
		a, claim, err := readOpenRGBInitialServing(u.Original)
		if err != nil || claim.OpenRGBWitness == nil || live && claim.State != managedUpdateExecFenceClaimed || a.Contract.Artifact.SHA256 != u.Rollback.Artifact.SHA256 {
			return openRGBExecWitness{}, errors.Join(errors.New("OpenRGB initial original lost its consumed native witness"), err)
		}
		return *claim.OpenRGBWitness, nil
	}
	old, original, contract, err := managedUpdateRecordedOriginal(u)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	store, err := newManagedUpdateExecStore(u.Resource, old.ID, original.Intent.InstanceID)
	if err != nil {
		return openRGBExecWitness{}, err
	}
	claim, err := store.read()
	if err != nil {
		return openRGBExecWitness{}, err
	}
	if live {
		return verifyOpenRGBConsumedClaim(claim, *original.Intent, contract, *original.NativeRef, original.State.Incarnation)
	}
	return verifyOpenRGBRetainedClaim(claim, *original.Intent, contract, *original.NativeRef, original.State.Incarnation)
}

func (c *Controller) qualifyOpenRGBUpdateStop(ctx context.Context, u ManagedServiceUpdate) (openRGBUpdatePolicies, error) {
	roles, err := qualifyOpenRGBUpdatePolicies(u, func(contract ManagedUpdateContract) (openRGBUpdateRolePolicy, error) {
		return c.loadOpenRGBFrozenUpdateRolePolicy(ctx, u, contract)
	})
	if err != nil {
		return openRGBUpdatePolicies{}, err
	}
	old, err := readOpenRGBOriginalWitness(u, u.Phase != UpdateRestoring)
	if err != nil || old.Policy != roles.original.binding {
		return openRGBUpdatePolicies{}, errors.Join(errors.New("OpenRGB original process did not consume the current exact policy"), err)
	}
	if err := observeOpenRGBUpdateGrant(ctx, roles.original); err != nil {
		return openRGBUpdatePolicies{}, err
	}
	return roles, nil
}

func observeOpenRGBUpdateGrant(ctx context.Context, role openRGBUpdateRolePolicy) error {
	if role.nativePolicy != nil {
		// The control-plane updater does not impersonate the daemon account.
		// Recheck exact machine/attachment metadata here; the wrapper proves
		// actual account HID access before claiming and launching each role.
		e, err := hostreqkit.ValidateNativeOpenRGBDeviceSelection(ctx, *role.nativePolicy)
		if err != nil || e.PolicyID != role.binding.ID || len(e.SelectedPaths) != len(role.nativePolicy.Devices) {
			return errors.Join(errors.New("native OpenRGB selected attachment scope changed before update"), err)
		}
		return nil
	}
	b := role.binding
	evidence, err := hostreqkit.ObserveLinuxOpenRGBDevicePolicy(ctx, role.policy, hostreqkit.OpenRGBLinuxCheck{PolicyID: b.ID, ArtifactSHA256: b.ArtifactSHA256, DetectorSHA256: b.DetectorCatalogSHA256, EffectiveUID: os.Geteuid()}, nil)
	if err != nil || !evidence.StaticGrantConfigured || !evidence.CurrentScopeVerified || len(evidence.SelectedNodes) != len(role.policy.Devices) {
		return errors.Join(errors.New("OpenRGB persistent physical grant could not be confirmed before native update"), err)
	}
	return nil
}

func (c *Controller) openRGBManagedUpdateLaunchPolicy(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract) (*openRGBExecPolicy, error) {
	if !exclusiveStopStart(u) || u.Original.UpdateLaunch == nil && u.Original.InitialLaunch == nil || u.OriginalScope != nil {
		return nil, errors.New("OpenRGB native update cannot launch outside its exact exclusive owner")
	}
	if u.Phase == UpdatePrepared && contract.Digest == u.Target.Digest {
		roles, err := c.qualifyOpenRGBUpdateStop(ctx, u)
		if err != nil {
			return nil, err
		}
		return &roles.target.binding, nil
	}
	if u.Phase != UpdateRestoring || contract.Digest != u.Rollback.Digest {
		return nil, errors.New("OpenRGB launch policy is outside its admitted exclusive role")
	}
	rollback, err := c.loadOpenRGBFrozenUpdateRolePolicy(ctx, u, contract)
	if err != nil {
		return nil, err
	}
	old, err := readOpenRGBOriginalWitness(u, false)
	if err != nil || !sameOpenRGBRollbackPolicy(old.Policy, rollback.binding, contract) {
		return nil, errors.Join(errors.New("OpenRGB rollback policy differs from the original enrolled grant"), err)
	}
	if err := observeOpenRGBUpdateGrant(ctx, rollback); err != nil {
		return nil, err
	}
	return &rollback.binding, nil
}

func sameOpenRGBRollbackPolicy(original, rollback openRGBExecPolicy, contract ManagedUpdateContract) bool {
	return rollback.valid() && original.ID == rollback.ID && original.ArtifactSHA256 == rollback.ArtifactSHA256 && original.DetectorCatalogSHA256 == rollback.DetectorCatalogSHA256 && rollback.ArtifactSHA256 == contract.Artifact.SHA256 && rollback.ConfigSHA256 == contract.ConfigSHA256
}

func verifyOpenRGBManagedUpdateStop(ctx context.Context, u ManagedServiceUpdate) error {
	if u.Phase == UpdateRestoring {
		_, runtime, err := openRGBOriginalLaunch(u.Original)
		if err != nil {
			return err
		}
		c := &Controller{Root: runtime.Root, Home: runtime.Home}
		// Recovery completes an already admitted exact stop. A failed target
		// must not prevent qualification of the retained original/rollback.
		_, err = c.openRGBManagedUpdateLaunchPolicy(ctx, u, u.Rollback)
		return err
	}
	i := u.Primary.Intent
	if !validUpdateIntent(i, u.Target) || i.Runtime.validate() != nil {
		return errors.New("OpenRGB original stop lacks an exact target launch intent")
	}
	c := &Controller{Root: i.Runtime.Root, Home: i.Runtime.Home}
	roles, err := c.qualifyOpenRGBUpdateStop(ctx, u)
	if err != nil {
		return err
	}
	store, err := newManagedUpdateExecStore(u.Resource, u.ID, i.InstanceID)
	if err != nil {
		return err
	}
	f, err := store.read()
	if err != nil || f.Intent != *i || f.State != managedUpdateExecFencePrepared || !sameOpenRGBExecPolicy(f.OpenRGBPolicy, &roles.target.binding) {
		return errors.Join(errors.New("OpenRGB target fence no longer carries its admitted physical policy"), err)
	}
	return nil
}
