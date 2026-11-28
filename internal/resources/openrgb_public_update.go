package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

type openRGBUpdateIntentKey struct{}

type openRGBPublicUpdateEffects struct {
	quiesce          func(context.Context) error
	profile          func(context.Context, ManagedServiceArtifact) error
	grant            func(context.Context, openRGBUpdateRolePolicy) error
	source           func(context.Context, ResourceManifest) (managedUpdateAdmissionSource, error)
	target           func(context.Context, ResourceManifest, *managedUpdateStore, ManagedServiceUpdateAdmission) (ManagedUpdateContract, error)
	snapshot         func(context.Context, ManagedServiceUpdate) (*openRGBManagedUpdatePolicies, error)
	servingSelection func(context.Context, ManagedServiceUpdate) (*lightingruntime.Selection, error)
}

func (c *Controller) openRGBPublicOwners() openRGBPublicUpdateEffects {
	if c.openRGBUpdateEffects != nil {
		return *c.openRGBUpdateEffects
	}
	return openRGBPublicUpdateEffects{quiesce: func(ctx context.Context) error {
		owner, err := lightingruntime.NewNativeScenarioOwner(c.Home, func(context.Context) error { return errors.New("resource update does not start Light Desk") })
		if err != nil {
			return err
		}
		return owner.QuiesceLightingReconciler(ctx)
	}, profile: func(ctx context.Context, a ManagedServiceArtifact) error {
		_, err := c.verifyOpenRGBRoleArtifact(ctx, a)
		return err
	}, grant: observeOpenRGBUpdateGrant, source: c.observeManagedUpdateAdmissionSource, target: c.managedUpdateAdmissionTarget, snapshot: c.freezeOpenRGBUpdateSnapshot, servingSelection: c.openRGBPublishedSelection}
}

// Intent precedes resource ownership, matching the reconciler's lock order.
func (c *Controller) runOpenRGBPublicUpdate(ctx context.Context, run func(context.Context) (ManagedServiceUpdate, error)) (result ManagedServiceUpdate, err error) {
	if c == nil || ctx == nil {
		return result, errors.New("OpenRGB update requires a controller and context")
	}
	if err := requireOpenRGBNativeOperatorAuthority(); err != nil {
		return result, err
	}
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return result, err
	}
	if err := c.requireOpenRGBEnabled(ctx); err != nil {
		return result, err
	}
	var publicationRelease func()
	defer func() {
		if publicationRelease != nil {
			publicationRelease()
		}
	}()

	control, err := lightingruntime.NewController(c.Home, lightingruntime.Owners{})
	if err != nil {
		return result, err
	}
	err = control.WithResourceUpdate(ctx, func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		var runErr error
		result, runErr = run(context.WithValue(ctx, openRGBUpdateIntentKey{}, intent))
		if result.ID == "" || result.Phase == UpdateResourceDisabled {
			return nil, runErr
		}
		release, lockErr := acquireManagedServiceLifecycleLock(ctx, openRGBResourceName)
		if lockErr != nil {
			return nil, errors.Join(runErr, lockErr)
		}
		publicationRelease = release
		if err := c.requireOpenRGBEnabled(ctx); err != nil {
			return nil, errors.Join(runErr, err)
		}
		// Keep resource ownership through WithResourceUpdate's durable intent
		// write. A low-level opt-out cannot slip between proof and publication.
		selected, observeErr := c.openRGBPublicOwners().servingSelection(ctx, result)
		return selected, errors.Join(runErr, observeErr)
	})
	return result, err
}
func (c *Controller) qualifyOpenRGBResumption(ctx context.Context, u ManagedServiceUpdate) error {
	if err := c.requireOpenRGBEnabled(ctx); err != nil {
		return err
	}
	if err := validateOpenRGBUpdateSnapshot(u); err != nil {
		return err
	}
	if u.OpenRGBPolicies == nil {
		return errors.New("OpenRGB legacy update has no qualified frozen physical policies and catalogs")
	}
	pending, err := c.OpenRGBDeviceSetupPending(ctx)
	if err != nil || pending {
		return errors.Join(errors.New("finish pending OpenRGB device setup before updating"), err)
	}
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return err
	}
	source := state.Resources[openRGBResourceName].Source
	if source == nil || u.OpenRGBPolicies.Source == nil || *source != *u.OpenRGBPolicies.Source {
		return fmt.Errorf("%w: OpenRGB source selection changed after admission", ErrManagedUpdateConflict)
	}
	selected, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || selected == nil {
		return errors.Join(errors.New("OpenRGB selected device policy is unavailable"), err)
	}
	id, err := selected.ID()
	if err != nil {
		return err
	}
	authorized := false
	for _, role := range []openRGBFrozenUpdateRole{u.OpenRGBPolicies.Original, u.OpenRGBPolicies.Target, u.OpenRGBPolicies.Rollback} {
		candidate, _, _, e := role.policyIdentity()
		if e == nil && candidate == id {
			authorized = true
		}
	}
	if !authorized {
		return fmt.Errorf("%w: newer OpenRGB policy selection replaced this update", ErrManagedUpdateConflict)
	}
	if intent, ok := ctx.Value(openRGBUpdateIntentKey{}).(lightingruntime.Intent); ok && intent.Phase == lightingruntime.Enabled && intent.Selection != nil {
		matched := false
		for _, pair := range []struct {
			version, sha string
			role         openRGBFrozenUpdateRole
		}{{u.Original.ArtifactVersion, u.Original.ArtifactSHA256, u.OpenRGBPolicies.Original}, {u.Target.Artifact.Version, u.Target.Artifact.SHA256, u.OpenRGBPolicies.Target}, {u.Rollback.Artifact.Version, u.Rollback.Artifact.SHA256, u.OpenRGBPolicies.Rollback}} {
			id, _, _, e := pair.role.policyIdentity()
			if e == nil && *intent.Selection == (lightingruntime.Selection{Version: pair.version, SHA256: pair.sha, DevicePolicyID: id}) {
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("%w: newer lighting selection replaced this update", ErrManagedUpdateConflict)
		}
	}
	if !exclusiveStopStart(u) {
		return errors.New("OpenRGB profile requires exclusive native replacement")
	}
	_, runtime, err := openRGBOriginalLaunch(u.Original)
	if err != nil {
		return err
	}
	owners := c.openRGBPublicOwners()
	contracts := []ManagedUpdateContract{u.Target, u.Rollback}
	if u.PrimaryStopVerified || u.Phase == UpdateRestoring || u.Phase == UpdateRecoveryNeeded || u.Phase == UpdateRestoreStarting || u.Phase == UpdateRestoreVerifying || u.Phase == UpdateRestoreCommitting {
		contracts = []ManagedUpdateContract{u.Rollback}
	}
	for _, contract := range contracts {
		endpoint, _, err := openRGBHeadlessEndpoint(contract.Manifest)
		if err != nil {
			return err
		}
		if int(endpoint.Port()) != runtime.Port {
			return errors.New("OpenRGB update changes the retained canonical SDK port")
		}
		preflight := runtime
		preflight.Root = c.Root
		preflight.Home = c.Home
		base, err := renderManagedUpdateInvocation(contract, preflight, u.ID, "profile-preflight", "preflight-token")
		if err != nil {
			return err
		}
		if err := verifyOpenRGBExecInvocation(base, preflight.Port); err != nil {
			return err
		}
		if err := owners.profile(ctx, contract.Artifact); err != nil {
			return err
		}
		role, err := openRGBFrozenUpdateRoleFor(u, contract)
		if err != nil {
			return err
		}
		binding, err := role.binding(contract)
		if err != nil {
			return err
		}
		if err := owners.grant(ctx, openRGBUpdateRolePolicy{binding: binding, policy: role.Policy, nativePolicy: role.NativePolicy}); err != nil {
			return err
		}
	}
	return nil
}
func (c *Controller) publishManagedUpdateWithPolicy(ctx context.Context, u ManagedServiceUpdate, role string) error {
	if u.Resource != openRGBResourceName {
		return publishManagedUpdateServing(ctx, u, role)
	}
	if err := c.qualifyOpenRGBResumption(ctx, u); err != nil {
		return err
	}
	contract := u.Target
	if role == managedUpdateRoleRestored {
		contract = u.Rollback
	}
	frozen, err := openRGBFrozenUpdateRoleFor(u, contract)
	if err != nil {
		return err
	}
	if err := c.publishOpenRGBPolicy(ctx, u, frozen); err != nil {
		return err
	}
	return publishManagedUpdateServing(ctx, u, role)
}
func (c *Controller) publishOpenRGBPolicy(ctx context.Context, u ManagedServiceUpdate, desired openRGBFrozenUpdateRole) error {
	path, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return err
	}
	release, err := acquireOpenRGBDevicePolicyPublicationLock(ctx, path)
	if err != nil {
		return err
	}
	defer release()
	current, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || current == nil {
		return errors.Join(errors.New("OpenRGB selected policy disappeared before publication"), err)
	}
	id, err := current.ID()
	if err != nil {
		return err
	}
	allowed := false
	for _, r := range []openRGBFrozenUpdateRole{u.OpenRGBPolicies.Original, u.OpenRGBPolicies.Target, u.OpenRGBPolicies.Rollback} {
		p, _, _, e := r.policyIdentity()
		if e == nil && p == id {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("%w: refuse to replace a newer OpenRGB device policy", ErrManagedUpdateConflict)
	}
	data, err := json.Marshal(desired.record())
	if err != nil {
		return err
	}
	if err := writeResourceOwnedFileDurable(path, data, 0600); err != nil {
		return fmt.Errorf("%w: OpenRGB policy publication: %w", ErrManagedUpdateCommitUnknown, err)
	}
	return nil
}
func (c *Controller) openRGBPublishedSelection(ctx context.Context, u ManagedServiceUpdate) (*lightingruntime.Selection, error) {
	supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
	if err != nil {
		return nil, err
	}
	state, err := supervisor.readState()
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if state.UpdateLaunch == nil || state.UpdateLaunch.OperationID != u.ID || state.StoppedAt != nil {
		return nil, nil
	}
	frozen, err := openRGBFrozenUpdateRoleFor(u, state.UpdateLaunch.Contract)
	if err != nil {
		return nil, err
	}
	selected, e := c.readOpenRGBDevicePolicyRecord()
	if e != nil || selected == nil {
		return nil, errors.Join(errors.New("selected OpenRGB policy disappeared during serving publication"), e)
	}
	selectedID, e := selected.ID()
	if e != nil {
		return nil, e
	}
	id, _, _, err := frozen.policyIdentity()
	if err != nil {
		return nil, err
	}
	if selectedID != id {
		return nil, fmt.Errorf("%w: newer selected OpenRGB policy wins over serving publication", ErrManagedUpdateConflict)
	}
	binding, err := frozen.binding(state.UpdateLaunch.Contract)
	if err != nil {
		return nil, err
	}
	operator, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return nil, err
	}
	source := operator.Resources[openRGBResourceName].Source
	if source == nil || u.OpenRGBPolicies.Source == nil || *source != *u.OpenRGBPolicies.Source {
		return nil, ErrManagedUpdateConflict
	}
	uid := os.Geteuid()
	if binding.Account != "" {
		uid = 0
	}
	proofErr := c.OpenRGBDevicePolicyVerifier()(ctx, OpenRGBDevicePolicyCheck{Account: binding.Account, PID: state.PID, Incarnation: state.Incarnation, EffectiveUID: uid, Artifact: state.UpdateLaunch.Contract.Artifact, PolicyID: id, DetectorSHA256: binding.DetectorCatalogSHA256})
	if proofErr != nil && !errors.Is(proofErr, ErrOpenRGBDeviceGenerationChanged) {
		return nil, proofErr
	}
	// A kernel replacement does not change the verified canonical release.
	// Converge that static selection while reporting readiness failure, so the
	// next Ensure can refresh this exact profile rather than the obsolete one.
	return &lightingruntime.Selection{Version: state.ArtifactVersion, SHA256: state.ArtifactSHA256, DevicePolicyID: id}, proofErr
}
