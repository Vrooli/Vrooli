package resources

import (
	"context"
	"errors"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// The ordinary resource command enters the same staged native owner as the
// lighting runtime, before the generic driver acquires its lifecycle locks.
func runOpenRGBNativeResourceAction(ctx context.Context, c *Controller, action string, args []string) error {
	if c == nil {
		return errors.New("OpenRGB resource controller is unavailable")
	}
	if err := requireOpenRGBNativeOperatorAuthority(); err != nil {
		return err
	}
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return err
	}
	owner := NewOpenRGBHeadless(c, c.OpenRGBDevicePolicyVerifier())
	restart := func(ctx context.Context, s lightingruntime.Selection) (lightingruntime.Backend, error) {

		return owner.nativeOpenRGBLifecycle(ctx, s, true)
	}
	if nativeOpenRGBUsesPrivateLocks(openRGBResourceName) {
		return withOpenRGBNativeClientIntent(ctx, c.Home, func(selected lightingruntime.Selection) error {
			return runOpenRGBNativeAction(ctx, action, args, func(context.Context) (lightingruntime.Selection, error) { return selected, nil }, owner.EnsureOpenRGB, restart)
		})
	}
	return runOpenRGBNativeAction(ctx, action, args, c.currentOpenRGBNativeSelection, owner.EnsureOpenRGB, restart)
}
func runOpenRGBNativeAction(ctx context.Context, action string, args []string, selection func(context.Context) (lightingruntime.Selection, error), ensure, restart func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if action != brokerTransportStart && action != brokerTransportRestart {
		return errors.New("unsupported OpenRGB native resource action")
	}
	if len(args) != 0 {
		return errors.New("OpenRGB native start/restart uses only the persisted source and device policy; launch overrides are unsupported")
	}
	selected, err := selection(ctx)
	if err != nil {
		return err
	}
	if action == brokerTransportRestart {
		_, err = restart(ctx, selected)
	} else {
		_, err = ensure(ctx, selected)
	}
	return err
}
func (c *Controller) currentOpenRGBNativeSelection(ctx context.Context) (lightingruntime.Selection, error) {
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return lightingruntime.Selection{}, err
	}
	choice := state.Resources[openRGBResourceName]
	if choice.Enabled == nil || !*choice.Enabled || choice.Source == nil {
		return lightingruntime.Selection{}, errors.New("OpenRGB requires explicit resource enablement and source selection")
	}
	policy, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil || policy == nil {
		return lightingruntime.Selection{}, errors.Join(errors.New("OpenRGB selected policy unavailable"), err)
	}
	id, err := policy.ID()
	if err != nil {
		return lightingruntime.Selection{}, err
	}
	artifact, err := c.InspectManagedServiceArtifact(ctx, openRGBResourceName)
	if err != nil {
		return lightingruntime.Selection{}, err
	}
	policyArtifact := policy.Policy.ArtifactSHA256
	if policy.NativePolicy != nil {
		policyArtifact = policy.NativePolicy.ArtifactSHA256
	}
	if artifact.SHA256 != policyArtifact {
		return lightingruntime.Selection{}, errors.New("OpenRGB selected source differs from the machine device policy")
	}
	return lightingruntime.Selection{Version: artifact.Version, SHA256: artifact.SHA256, DevicePolicyID: id}, nil
}

// Public resource commands have no reconciler-held operation lock. Retain
// the existing intent lock across the operation, before the resource lifecycle lock.
func withOpenRGBNativeClientIntent(ctx context.Context, home string, run func(lightingruntime.Selection) error) error {
	owner, err := lightingruntime.NewController(home, lightingruntime.Owners{})
	if err != nil {
		return err
	}
	return owner.WithResourceUpdate(ctx, func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		if intent.Phase != lightingruntime.Enabled || intent.Selection == nil {
			return nil, errors.New("native resource start requires durable enabled lighting selection")
		}
		return nil, run(*intent.Selection)
	})
}

// Resource stop enters the ordinary durable lighting transaction so quiescence
// always precedes revocation and native role retirement.
type openRGBNativeStopScenario struct{ quiesce func(context.Context) error }

func (s openRGBNativeStopScenario) QuiesceLightingReconciler(ctx context.Context) error {
	return s.quiesce(ctx)
}
func (openRGBNativeStopScenario) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	return errors.New("resource stop cannot start the lighting reconciler")
}
