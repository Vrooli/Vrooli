package resources

import (
	"context"
	"errors"
	platform "github.com/vrooli/platform-go"
)

// Instance-private effects keep the public state-machine path hermetically
// testable. Production always composes the canonical platform/resource owners.
type managedUpdateEffects struct {
	close             func() error
	prepare           func(context.Context, ManagedServiceUpdate, ManagedUpdateContract, string, int) (ManagedUpdateLaunchIntent, error)
	original          func(context.Context, ManagedServiceUpdate) (*ManagedUpdateProof, error)
	stopOriginal      func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error)
	stopRestoration   func(context.Context, ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error)
	originalQuiescent func(context.Context, ManagedServiceUpdate) (bool, error)
	portUnbound       func(int) error
	start             func(context.Context, platform.ServiceIntent, platform.OwnedServiceSpec) (platform.ServiceRef, error)
	wait              func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess, ManagedUpdateContract) (ManagedUpdateProcess, error)
	observe           func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess, ManagedUpdateContract) (ManagedUpdateProcess, error)
	publish           func(context.Context, ManagedServiceUpdate, string) error
	cleanup           func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error
	reserve, release  func(context.Context, ManagedServiceUpdate) error
	candidate         func(context.Context, ManagedServiceUpdate, *managedUpdateStore) (ManagedServiceUpdate, error)
}

func defaultManagedUpdateEffects(c *Controller) managedUpdateEffects {
	effects := managedUpdateEffects{prepare: c.prepareManagedUpdateLaunch, original: observeManagedUpdateOriginal, stopOriginal: stopManagedUpdateOriginal, stopRestoration: stopManagedUpdateOriginalForRestoration, originalQuiescent: managedUpdateOriginalQuiescent, portUnbound: managedUpdatePortUnbound, start: platform.StartOwnedService, wait: waitManagedUpdateProcess, observe: observeManagedUpdateProcess, publish: c.publishManagedUpdateWithPolicy, cleanup: cleanupManagedUpdateRole, reserve: reserveManagedUpdateCapacity, release: releaseManagedUpdateCapacity, candidate: c.advanceManagedUpdateCandidateLocked}

	var servers []*openRGBNativeClaimServer
	effects.start = func(ctx context.Context, intent platform.ServiceIntent, spec platform.OwnedServiceSpec) (platform.ServiceRef, error) {
		if intent.NativeAccount != "" {
			u, contract, launch, err := loadOpenRGBNativeUpdateDispatch(ctx, intent, spec)
			if err != nil {
				return platform.ServiceRef{}, err
			}
			server, err := c.startOpenRGBNativeUpdateAdmission(ctx, u, contract, launch)
			if err != nil {
				return platform.ServiceRef{}, err
			}
			servers = append(servers, server)
		}
		return platform.StartOwnedService(ctx, intent, spec)
	}
	effects.close = func() error {
		var err error
		for _, s := range servers {
			err = errors.Join(err, s.Close())
		}
		return err
	}
	return effects
}
func (c *Controller) managedUpdateEffectOwner() managedUpdateEffects {
	if c.updateEffects != nil {
		return *c.updateEffects
	}
	return defaultManagedUpdateEffects(c)
}

// A caller may be replaced after preparing a role and stopping the original.
// Reconstruct admission from the existing durable owner rather than retaining
// an ephemeral prepare/start map. The runner holds the resource lifecycle lock;
// Confirm establishes durability before the admission listener or dispatch.
func loadOpenRGBNativeUpdateDispatch(ctx context.Context, intent platform.ServiceIntent, spec platform.OwnedServiceSpec) (ManagedServiceUpdate, ManagedUpdateContract, ManagedUpdateLaunchIntent, error) {
	var zero ManagedServiceUpdate
	var contract ManagedUpdateContract
	var launch ManagedUpdateLaunchIntent
	if ctx == nil || intent.NativeAccount == "" || intent.ValidateSpec(spec) != nil {
		return zero, contract, launch, errors.New("native update dispatch differs from its durable intent")
	}
	if err := ctx.Err(); err != nil {
		return zero, contract, launch, err
	}
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		return zero, contract, launch, err
	}
	ledger, err := store.read()
	if err != nil {
		return zero, contract, launch, err
	}
	for _, entry := range ledger.Records {
		u := entry.State
		p, selected, err := managedUpdateAdmittedRole(u, intent.AttemptID)
		if err != nil || p.Intent == nil || p.Intent.NativeIntent != intent {
			continue
		}
		if err := intent.ValidateSpec(managedUpdateWrapperSpec(u, *p.Intent)); err != nil {
			return zero, contract, launch, err
		}
		confirmed, err := store.Confirm(ctx, u.RequestID, u.Revision)
		if err != nil {
			return zero, contract, launch, err
		}
		return confirmed, selected, *p.Intent, nil
	}
	return zero, contract, launch, errors.New("native resource dispatch has no current durable admitted role")
}
