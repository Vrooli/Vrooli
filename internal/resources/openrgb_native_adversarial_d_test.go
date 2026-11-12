package resources

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	platform "github.com/vrooli/platform-go"
)

// The public update owner must respect a new operator source even when the old
// caller disappeared after durable preparation and before its first native Run.
func TestNativeDPreparedUpdateCannotReplaceNewOperatorSource(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		t.Run(target, func(t *testing.T) {
			f := newOpenRGBPublicFixtureFor(t, target)
			control, _ := enablePublicLighting(t, f)
			u := f.admit(t, "d-stale-prepared-source")
			start := f.c.updateEffects.start
			ctx, cancel := context.WithCancel(t.Context())
			f.c.updateEffects.start = func(context.Context, platform.ServiceIntent, platform.OwnedServiceSpec) (platform.ServiceRef, error) {
				cancel()
				return platform.ServiceRef{}, context.Canceled
			}
			_, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
			cancel()
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			before, err := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
			if err != nil || before.Primary.Intent == nil || before.Phase != UpdatePrimaryStarting {
				t.Fatalf("prepared checkpoint: %s %v", before.Phase, err)
			}
			patch := fmt.Sprintf(`{"resources":{"openrgb":{"source":{"mode":"host","executable":"/fixture/new-operator-source","version":"next","sha256":"%s"}}}}`, updateTestDigest("d-new-source"))
			if _, err = f.c.operatorStateService().Apply(t.Context(), []byte(patch)); err != nil {
				t.Fatal(err)
			}
			f.c.updateEffects.start = start
			starts, claims := f.starts, f.execs
			if _, err = f.c.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false); !errors.Is(err, ErrManagedUpdateConflict) {
				t.Fatalf("stale prepared source resumed: %v", err)
			}
			after, err := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
			if err != nil || !reflect.DeepEqual(before, after) || f.starts != starts || f.execs != claims {
				t.Fatalf("stale recovery changed admission or dispatched: %v", err)
			}
			state, err := f.c.operatorStateService().Load(t.Context())
			if err != nil || state.Resources[openRGBResourceName].Source.SHA256 != updateTestDigest("d-new-source") {
				t.Fatalf("new source overwritten: %v", err)
			}
			// Opt-out remains available even though the old source is no longer selected.
			if err = control.Disable(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(f.active) != 0 || f.starts != starts || f.execs != claims {
				t.Fatal("disable restored stale source")
			}
		})
	}
}

// A successful durable claim is not permission to ignore a subsequent failed
// configuration publication or cancellation. The actual executor must abort the
// still-suspended Windows child and retain the consumed claim on replay.
func TestNativeDClaimedWindowsChildAbortsBeforeResumeOnPublicationFailure(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			f, child := nativeWindowsCapsuleExecFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failure := errors.New("private configuration publication failed")
			publications := 0
			f.owner.materialize = func(openRGBNativeMaterial, managedUpdateInvocation) error {
				publications++
				if canceled {
					cancel()
					return nil
				}
				return failure
			}
			err := executeOpenRGBNativeCapsule(ctx, f.capsule, f.owner)
			expected := failure
			if canceled {
				expected = context.Canceled
			}
			if !errors.Is(err, expected) || child.resumed || !f.session.claimed || publications != 1 || !reflect.DeepEqual(f.events, []string{"claim", "abort"}) {
				t.Fatalf("failure crossed resume boundary: %v events=%v", err, f.events)
			}
			if err = executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, ErrManagedUpdateExecClaimed) {
				t.Fatalf("consumed claim replay: %v", err)
			}
			if publications != 1 || child.resumed {
				t.Fatal("replay published config or resumed child")
			}
		})
	}
}
