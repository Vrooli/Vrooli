package resources

import (
	"context"
	"errors"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// The real desired-intent, initial-attempt and exec-fence owners compose here.
// Native observation/stop effects are modeled; the matching platform regression
// exercises the actual Darwin backend/file store with those native edges.
func TestOpenRGBNativeDisableRetainsUnclaimedRootlessUncertainty(t *testing.T) {
	for _, recovery := range []string{"exact-root", "origin-boot-ended", "lost-release-ack"} {
		t.Run(recovery, func(t *testing.T) {
			flow := newNativeLifecycleInitialFlow(t, "unclaimed-submitted-wrapper", "12345678-1234-1234-1234-123456789abc")
			rootless := flow.ref
			native := *rootless.Native
			rootless.Native = &native
			rootless.RootPID, rootless.RootStartTime = 0, ""
			native.RootIncarnation, native.RootUniqueID = "", ""
			if err := rootless.ValidateIdentity(); err != nil {
				t.Fatal(err)
			}
			unknown, removed := true, false
			releases, starts := 0, 0
			lookupRef := rootless
			cleanup := openRGBInitialCleanupOwner{
				revoke: flow.fence.Revoke,
				lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
					if removed {
						return platform.ServiceRef{}, platform.ErrOperationNotFound
					}
					return lookupRef, nil
				},
				stop: func(_ context.Context, ref platform.ServiceRef) (platform.OperationStopReceipt, error) {
					if unknown {
						return platform.OperationStopReceipt{Ref: platform.OperationRef(ref)}, platform.ErrOperationOutcomeLost
					}
					return platform.OperationStopReceipt{Ref: platform.OperationRef(ref), NativeQuiescent: true}, nil
				},
				release: func(_ context.Context, ref platform.ServiceRef) error {
					a, err := flow.store.read(flow.attempt.ID)
					if err != nil || a.QuiescentAt == nil || a.NativeRef == nil || !a.NativeRef.Equal(ref) {
						t.Fatal("release preceded durable exact quiescence", err)
					}
					releases++
					removed = true
					if recovery == "lost-release-ack" {
						return platform.ErrOperationOutcomeLost
					}
					return nil
				},
				released: func(context.Context, platform.ServiceRef) (bool, error) { return removed, nil },
			}
			var control *lightingruntime.Controller
			resource := &nativeLifecycleLightingResource{ensure: func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
				starts++
				return lightingruntime.Backend{}, errors.New("disable must not submit another wrapper")
			}}
			resource.disable = func(ctx context.Context) error {
				intent, err := control.Current()
				if err != nil || intent.Phase != lightingruntime.Disabling {
					return errors.New("native retirement preceded durable Disabling")
				}
				a, err := flow.store.read(flow.attempt.ID)
				if err != nil {
					return err
				}
				return retireOpenRGBInitialWith(ctx, flow.store, a, cleanup)
			}
			var err error
			control, err = lightingruntime.NewController(t.TempDir(), lightingruntime.Owners{Resource: resource, Scenario: openRGBNativeStopScenario{quiesce: func(context.Context) error { return nil }}})
			if err != nil {
				t.Fatal(err)
			}
			selection := lightingruntime.Selection{Version: flow.attempt.Contract.Artifact.Version, SHA256: flow.attempt.Contract.Artifact.SHA256, DevicePolicyID: flow.attempt.Policy.ID}
			if err := control.Enable(t.Context(), selection); err != nil {
				t.Fatal(err)
			}
			for retry := 0; retry < 2; retry++ {
				if err := control.Disable(t.Context()); !errors.Is(err, platform.ErrOperationOutcomeLost) {
					t.Fatalf("unknown submitted root reported disabled: %v", err)
				}
				intent, err := control.Current()
				if err != nil || intent.Phase != lightingruntime.Disabling || releases != 0 || starts != 0 {
					t.Fatalf("uncertainty lost %+v %v release=%d start=%d", intent, err, releases, starts)
				}
				a, err := flow.store.read(flow.attempt.ID)
				if err != nil || a.State != openRGBInitialSubmitted || a.NativeRef != nil || a.QuiescentAt != nil || a.StoppedAt != nil {
					t.Fatalf("unknown root became durable extinction %+v %v", a, err)
				}
				fence, err := flow.fence.read()
				if err != nil || fence.State != managedUpdateExecFenceRevoked || fence.Claim != nil {
					t.Fatalf("unclaimed execution not fenced %+v %v", fence, err)
				}
			}
			unknown = false
			lookupRef = flow.ref
			if recovery == "origin-boot-ended" {
				lookupRef = rootless
				nextBoot := *rootless.Native
				lookupRef.Native = &nextBoot
				nextBoot.BootID = "22345678-1234-1234-1234-123456789abc"
			}
			err = control.Reconcile(t.Context())
			if recovery == "lost-release-ack" {
				if !errors.Is(err, platform.ErrOperationOutcomeLost) {
					t.Fatalf("lost release acknowledgement reported completion: %v", err)
				}
				err = control.Reconcile(t.Context())
			}
			if err != nil {
				t.Fatal(err)
			}
			intent, err := control.Current()
			a, readErr := flow.store.read(flow.attempt.ID)
			if err != nil || readErr != nil || intent.Phase != lightingruntime.Disabled || a.State != openRGBInitialStopped || releases != 1 || starts != 0 {
				t.Fatalf("recovery incomplete intent=%+v attempt=%+v err=%v/%v release=%d start=%d", intent, a, err, readErr, releases, starts)
			}
		})
	}
}
