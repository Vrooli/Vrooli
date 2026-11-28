package resources

import (
	"context"
	"errors"
	"testing"

	platform "github.com/vrooli/platform-go"
)

// These journeys run the public durable update/lighting owners with native V2
// refs and native policy records. Native dispatch, artifact qualification and
// SDK leaves are modeled; the real capsule executor is proven separately.
func TestOpenRGBPublicNativeUpdateRecoveryJourneys(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		for _, journey := range []string{"update-rollback-disable", "target-failure", "interrupted-readiness", "lost-start-ack", "owner-replaced-before-dispatch"} {
			t.Run(target+"/"+journey, func(t *testing.T) {
				f := newOpenRGBPublicFixtureFor(t, target)
				control, model := enablePublicLighting(t, f)
				u := f.admit(t, "native-journey")
				if u.OpenRGBPolicies.Original.NativePolicy == nil || u.OpenRGBPolicies.Target.NativePolicy == nil || u.OpenRGBPolicies.Rollback.NativePolicy == nil {
					t.Fatal("native physical policies were not frozen")
				}
				if f.starts != 0 || f.stops != 0 {
					t.Fatal("admission changed the native process")
				}
				if journey == "target-failure" {
					f.failTarget = true
				}
				if journey == "owner-replaced-before-dispatch" {
					stop := f.c.updateEffects.stopOriginal
					ctx, cancel := context.WithCancel(t.Context())
					f.c.updateEffects.stopOriginal = func(ctx context.Context, u ManagedServiceUpdate) (managedUpdateOriginalStopReceipt, error) {
						r, err := stop(ctx, u)
						cancel()
						return r, err
					}
					_, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
					cancel()
					stopped, lookupErr := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
					if !errors.Is(err, context.Canceled) || lookupErr != nil || stopped.Phase != UpdatePrimaryStopping || f.starts != 0 {
						t.Fatalf("stop-edge interruption phase=%s err=%v lookup=%v starts=%d", stopped.Phase, err, lookupErr, f.starts)
					}
					// The replacement caller has no ephemeral prepare map. Its
					// start must reconstruct the existing durable admitted role.
					f.c.updateEffects.stopOriginal = stop
				}
				if journey == "interrupted-readiness" || journey == "lost-start-ack" {
					wait := f.c.updateEffects.wait
					ctx, cancel := context.WithCancel(t.Context())
					start := f.c.updateEffects.start
					if journey == "lost-start-ack" {
						f.c.updateEffects.start = func(ctx context.Context, intent platform.ServiceIntent, spec platform.OwnedServiceSpec) (platform.ServiceRef, error) {
							ref, err := start(ctx, intent, spec)
							if err != nil {
								return ref, err
							}
							cancel()
							return platform.ServiceRef{}, ctx.Err()
						}
					} else {
						f.c.updateEffects.wait = func(context.Context, ManagedServiceUpdate, ManagedUpdateProcess, ManagedUpdateContract) (ManagedUpdateProcess, error) {
							cancel()
							return ManagedUpdateProcess{}, ctx.Err()
						}
					}
					got, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
					cancel()
					if !errors.Is(err, context.Canceled) || got.Phase != UpdatePrimaryStarting || f.starts != 1 || f.execs != 1 {
						t.Fatalf("interruption phase=%s err=%v starts=%d claims=%d", got.Phase, err, f.starts, f.execs)
					}
					f.c.updateEffects.wait = wait
					f.c.updateEffects.start = start
				}
				got, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
				expected := UpdateSucceeded
				if journey == "target-failure" {
					expected = UpdateFailedRestored
				}
				if err != nil || got.Phase != expected {
					t.Fatalf("completion phase=%s err=%v", got.Phase, err)
				}
				if got.Primary.Intent != nil {
					// A completed/revoked role is never a fresh dispatch permit,
					// even when its exact once-admitted wrapper spec is replayed.
					if _, _, _, err := loadOpenRGBNativeUpdateDispatch(t.Context(), got.Primary.Intent.NativeIntent, managedUpdateWrapperSpec(got, *got.Primary.Intent)); err == nil {
						t.Fatal("terminal role admitted native dispatch")
					}
				}
				starts, claims := f.starts, f.execs
				if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err != nil || f.starts != starts || f.execs != claims {
					t.Fatalf("completion replay dispatched again: %v", err)
				}
				if err = control.Reconcile(t.Context()); err != nil {
					t.Fatal(err)
				}
				if journey == "update-rollback-disable" {
					rollback, replayed, err := f.c.AdmitManagedServiceUpdate(t.Context(), openRGBResourceName, ManagedServiceUpdateAdmission{RequestID: "native-rollback", ExpectedVersion: u.Rollback.Artifact.Version, ExpectedSHA256: u.Rollback.Artifact.SHA256, RollbackOf: u.RequestID})
					if err != nil || replayed {
						t.Fatalf("rollback admission %v replay=%v", err, replayed)
					}
					restored, err := f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, rollback.RequestID, false)
					if err != nil || restored.Phase != UpdateSucceeded {
						t.Fatalf("rollback phase=%s err=%v", restored.Phase, err)
					}
					if err = control.Reconcile(t.Context()); err != nil {
						t.Fatal(err)
					}
					current, err := control.Current()
					if err != nil || current.Selection == nil || current.Selection.SHA256 != u.Rollback.Artifact.SHA256 {
						t.Fatalf("rollback selection %+v err=%v", current, err)
					}
				}
				if err = control.Disable(t.Context()); err != nil {
					t.Fatal(err)
				}
				if len(f.active) != 0 || model.bound != "" {
					t.Fatal("disable retained native role or SDK writer")
				}
				starts = f.starts
				if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); err == nil || f.starts != starts {
					t.Fatal("disabled update redispatched")
				}
			})
		}
	}
}
