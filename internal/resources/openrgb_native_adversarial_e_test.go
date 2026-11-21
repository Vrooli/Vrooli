package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// Failed native cleanup must leave a durable opt-out fence that survives the
// caller and forbids dispatch, while the ordinary Disable retry can converge.
// Only native manager/SDK leaves are modeled; public intent, role records and
// once-only execution fences use their production owners.
func TestNativeEInterruptedDisableRetainsRevocationAndConvergesOnRetry(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		t.Run(target, func(t *testing.T) {
			f := newOpenRGBPublicFixtureFor(t, target)
			control, model := enablePublicLighting(t, f)
			u := f.admit(t, "e-disable-retry")
			ctx, cancel := context.WithCancel(t.Context())
			wait := f.c.updateEffects.wait
			f.c.updateEffects.wait = func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess, ManagedUpdateContract) (ManagedUpdateProcess, error) {
				cancel()
				return ManagedUpdateProcess{}, context.Canceled
			}
			_, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
			cancel()
			if !errors.Is(err, context.Canceled) || f.starts != 1 || f.execs != 1 {
				t.Fatalf("claimed interruption: %v starts=%d claims=%d", err, f.starts, f.execs)
			}
			f.c.updateEffects.wait = wait
			pending, err := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
			if err != nil || pending.Primary.Intent == nil {
				t.Fatalf("missing prepared role: %v", err)
			}
			cleanup := f.c.updateEffects.cleanup
			denied := errors.New("native manager temporarily unavailable")
			f.c.updateEffects.cleanup = func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess) error { return denied }
			if err = control.Disable(t.Context()); !errors.Is(err, denied) {
				t.Fatalf("unconfirmed disable acknowledged: %v", err)
			}
			intent, err := control.Current()
			if err != nil || intent.Phase != lightingruntime.Disabling || model.bound != "" {
				t.Fatalf("failed disable lost durable fence or writer quiescence: %+v %v bound=%q", intent, err, model.bound)
			}
			fs, err := newManagedUpdateExecStore(openRGBResourceName, u.ID, pending.Primary.Intent.InstanceID)
			if err != nil {
				t.Fatal(err)
			}
			fence, err := fs.read()
			if err != nil || fence.State != managedUpdateExecFenceRevoked || fence.Claim == nil {
				t.Fatalf("failed stop lost revoked consumed claim: %v", err)
			}
			if _, err = fs.ClaimOpenRGB(t.Context(), fence, *fence.Claim, *fence.OpenRGBWitness); !errors.Is(err, ErrManagedUpdateExecRevoked) {
				t.Fatalf("failed cleanup reauthorized delayed claim: %v", err)
			}
			starts, claims := f.starts, f.execs
			if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err == nil {
				t.Fatal("failed disable permitted update")
			}
			f.c.updateEffects.cleanup = cleanup
			if err = control.Disable(t.Context()); err != nil {
				t.Fatal(err)
			}
			intent, err = control.Current()
			retired, lookupErr := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
			if err != nil || lookupErr != nil || intent.Phase != lightingruntime.Disabled || retired.Phase != UpdateResourceDisabled || len(f.active) != 0 || f.starts != starts || f.execs != claims {
				t.Fatalf("retry did not retire without dispatch: phase=%s update=%s active=%d err=%v/%v", intent.Phase, retired.Phase, len(f.active), err, lookupErr)
			}
		})
	}
}
