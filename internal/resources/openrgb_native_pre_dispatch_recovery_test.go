package resources

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

// Stop at the durable intent/exec-fence boundary, before the first native Run.
// The platform and SDK leaves remain modeled; the public state machine, durable
// role loader, once-claim and disable fences are their production owners.
func TestOpenRGBPublicNativeReplacementAfterPrepareBeforeFirstDispatch(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		for _, disable := range []bool{false, true} {
			name := target + "/recover-restoration"
			if disable {
				name = target + "/disable"
			}
			t.Run(name, func(t *testing.T) {
				f := newOpenRGBPublicFixtureFor(t, target)
				control, _ := enablePublicLighting(t, f)
				u := f.admit(t, "prepared-before-native-run")
				start := f.c.updateEffects.start
				ctx, cancel := context.WithCancel(t.Context())
				f.c.updateEffects.start = func(context.Context, platform.ServiceIntent, platform.OwnedServiceSpec) (platform.ServiceRef, error) {
					cancel()
					return platform.ServiceRef{}, context.Canceled
				}
				_, err := f.c.AdvanceManagedServiceUpdate(ctx, openRGBResourceName, u.RequestID, false)
				cancel()
				if !errors.Is(err, context.Canceled) || f.starts != 0 || f.execs != 0 {
					t.Fatalf("pre-dispatch interruption err=%v starts=%d claims=%d", err, f.starts, f.execs)
				}
				stored, err := LookupManagedServiceUpdate(openRGBResourceName, u.RequestID)
				if err != nil || stored.Phase != UpdatePrimaryStarting || stored.Primary.Intent == nil {
					t.Fatalf("durable prepared role phase=%s err=%v", stored.Phase, err)
				}
				intent := stored.Primary.Intent.NativeIntent
				spec := managedUpdateWrapperSpec(stored, *stored.Primary.Intent)
				if _, _, _, err := loadOpenRGBNativeUpdateDispatch(t.Context(), intent, spec); err != nil {
					t.Fatalf("replacement could not reconstruct prepared admission: %v", err)
				}
				// Replace the public controller without a prepare/start map. Only
				// the modeled external-effect leaves survive this replacement.
				replacement := NewController(f.c.Root, f.c.Home)
				replacement.operatorState = operatorstate.New(operatorstate.Config{RepoRoot: f.c.Root, StateRoot: filepath.Join(f.c.Root, "operator-state")})
				effects := *f.c.updateEffects
				effects.start = start
				replacement.updateEffects = &effects
				replacement.openRGBUpdateEffects = f.c.openRGBUpdateEffects
				f.c = replacement
				if disable {
					if err := control.Disable(t.Context()); err != nil {
						t.Fatal(err)
					}
					if _, _, _, err := loadOpenRGBNativeUpdateDispatch(t.Context(), intent, spec); err == nil {
						t.Fatal("disabled prepared role retained dispatch authority")
					}
					if f.starts != 0 || f.execs != 0 {
						t.Fatal("disable dispatched the prepared role")
					}
					return
				}
				got, err := replacement.AdvanceManagedServiceUpdate(t.Context(), openRGBResourceName, u.RequestID, false)
				// Submission was unacknowledged. Recovery must preserve that old
				// invocation and restore once under a fresh admitted role, rather
				// than treating the absence of our modeled Run as replay authority.
				if err != nil || got.Phase != UpdateFailedRestored || f.starts != 1 || f.execs != 1 {
					t.Fatalf("replacement completion phase=%s err=%v starts=%d claims=%d", got.Phase, err, f.starts, f.execs)
				}
				if got.Primary.Intent.NativeIntent != intent {
					t.Fatal("replacement rewrote uncertain native invocation")
				}
				if got.Restored.Intent == nil || got.Restored.NativeRef == nil || got.Restored.Intent.NativeIntent == intent {
					t.Fatal("recovery did not retain a distinct admitted restoration")
				}
				if err := control.Disable(t.Context()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
