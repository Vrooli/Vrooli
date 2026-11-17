package resources

import (
	"context"
	"errors"
	"time"

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
	scenario, err := lightingruntime.NewNativeScenarioOwner(c.Home, lightingruntime.EnsureNativeScenarioStarted)
	if err != nil {
		return err
	}
	control, err := lightingruntime.NewController(c.Home, lightingruntime.Owners{Resource: owner, Scenario: scenario})
	if err != nil {
		return err
	}
	return control.WithResourceUpdate(ctx, func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		var selected lightingruntime.Selection
		if intent.Phase == lightingruntime.Enabled && intent.Selection != nil {
			selected = *intent.Selection
		} else if nativeOpenRGBUsesPrivateLocks(openRGBResourceName) {
			if intent.Phase != lightingruntime.Enabled || intent.Selection == nil {
				return nil, errors.New("native OpenRGB start/restart requires durable enabled lighting selection")
			}
			selected = *intent.Selection
		} else {
			var err error
			selected, err = c.currentOpenRGBNativeSelection(ctx)
			if err != nil {
				return nil, err
			}
		}
		err := runOpenRGBNativeActionAndReconcile(ctx, action, args,
			func(context.Context) (lightingruntime.Selection, error) { return selected, nil }, owner.EnsureOpenRGB, restart,
			func(ctx context.Context, selected lightingruntime.Selection, backend lightingruntime.Backend) error {
				if intent.Phase != lightingruntime.Enabled {
					return nil
				}
				// The backend restart quiesces Light Desk before retiring the old
				// process. Publish the replacement binding before this command
				// reports success instead of waiting for the supervisor's next tick.
				if bindErr := scenario.EnsureLightingReconciler(ctx, backend); bindErr != nil {
					cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
					defer cancel()
					return errors.Join(bindErr, scenario.QuiesceLightingReconciler(cleanup))
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
		if intent.Phase != lightingruntime.Enabled {
			return nil, nil
		}
		return &selected, nil
	})
}

func runOpenRGBNativeAction(ctx context.Context, action string, args []string, selection func(context.Context) (lightingruntime.Selection, error), ensure, restart func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error)) error {
	_, err := runOpenRGBNativeActionWithBackend(ctx, action, args, selection, ensure, restart)
	return err
}

func requiresOpenRGBNativeStop(resource, action string) bool {
	return resource == openRGBResourceName && action == brokerTransportStop
}

func runOpenRGBNativeStop(ctx context.Context, args []string, owner *lightingruntime.Controller, stop, quiesce func(context.Context) error) error {
	if len(args) != 0 {
		return errors.New("OpenRGB native stop accepts no process or launch overrides")
	}
	if owner == nil || stop == nil || quiesce == nil {
		return errors.New("OpenRGB native stop owner is unavailable")
	}
	return owner.WithResourceUpdate(ctx, func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		if err := quiesce(ctx); err != nil {
			return nil, err
		}
		if err := stop(ctx); err != nil {
			if intent.Phase == lightingruntime.Enabled {
				return intent.Selection, err
			}
			return nil, err
		}
		if intent.Phase == lightingruntime.Enabled {
			return intent.Selection, nil
		}
		return nil, nil
	})
}

func runOpenRGBNativeActionAndReconcile(ctx context.Context, action string, args []string, selection func(context.Context) (lightingruntime.Selection, error), ensure, restart func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error), reconcile func(context.Context, lightingruntime.Selection, lightingruntime.Backend) error) error {
	if reconcile == nil {
		return errors.New("OpenRGB native action requires a reconciliation callback")
	}
	selected, err := selection(ctx)
	if err != nil {
		return err
	}
	backend, err := runOpenRGBNativeActionWithBackend(ctx, action, args,
		func(context.Context) (lightingruntime.Selection, error) { return selected, nil }, ensure, restart)
	if err != nil {
		return err
	}
	return reconcile(ctx, selected, backend)
}

func runOpenRGBNativeActionWithBackend(ctx context.Context, action string, args []string, selection func(context.Context) (lightingruntime.Selection, error), ensure, restart func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error)) (lightingruntime.Backend, error) {
	if err := ctx.Err(); err != nil {
		return lightingruntime.Backend{}, err
	}
	if action != brokerTransportStart && action != brokerTransportRestart {
		return lightingruntime.Backend{}, errors.New("unsupported OpenRGB native resource action")
	}
	if len(args) != 0 {
		return lightingruntime.Backend{}, errors.New("OpenRGB native start/restart uses only the persisted source and device policy; launch overrides are unsupported")
	}
	selected, err := selection(ctx)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	var backend lightingruntime.Backend
	if action == brokerTransportRestart {
		backend, err = restart(ctx, selected)
	} else {
		backend, err = ensure(ctx, selected)
	}
	return backend, err
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

// Resource stop enters the ordinary durable lighting transaction so quiescence
// always precedes revocation and native role retirement.
type openRGBNativeStopScenario struct{ quiesce func(context.Context) error }

func (s openRGBNativeStopScenario) QuiesceLightingReconciler(ctx context.Context) error {
	return s.quiesce(ctx)
}

func (openRGBNativeStopScenario) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	return errors.New("resource stop cannot start the lighting reconciler")
}
