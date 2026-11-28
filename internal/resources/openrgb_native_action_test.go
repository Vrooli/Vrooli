package resources

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func TestOpenRGBNativeResourceActionUsesSameQualifiedOwner(t *testing.T) {
	selected := lightingruntime.Selection{Version: "1.1", SHA256: openRGBAppImageSHA256, DevicePolicyID: updateTestDigest("policy")}
	for _, action := range []string{brokerTransportStart, brokerTransportRestart} {
		t.Run(action, func(t *testing.T) {
			calls := []string{}
			backend := lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: "instance-2"}
			selectCurrent := func(context.Context) (lightingruntime.Selection, error) {
				calls = append(calls, "persisted-selection")
				return selected, nil
			}
			handler := func(name string) func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
				return func(_ context.Context, s lightingruntime.Selection) (lightingruntime.Backend, error) {
					if s != selected {
						t.Fatal("native selection changed")
					}
					calls = append(calls, name)
					return backend, nil
				}
			}
			bind := func(_ context.Context, s lightingruntime.Selection, got lightingruntime.Backend) error {
				if s != selected || got != backend {
					t.Fatal("reconcile received a different selection or backend")
				}
				calls = append(calls, "reconcile")
				return nil
			}
			if err := runOpenRGBNativeActionAndReconcile(t.Context(), action, nil, selectCurrent, handler(brokerTransportStart), handler(brokerTransportRestart), bind); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []string{"persisted-selection", action, "reconcile"}) {
				t.Fatalf("owner route %v", calls)
			}
		})
	}
}

func TestOpenRGBStopPreservesDurableLightingIntentOnEveryPlatform(t *testing.T) {
	if !requiresOpenRGBNativeStop(openRGBResourceName, brokerTransportStop) || requiresOpenRGBNativeStop("redis", brokerTransportStop) || requiresOpenRGBNativeStop(openRGBResourceName, brokerTransportStart) {
		t.Fatal("OpenRGB stop routing does not cover exactly its native stop action")
	}
	home := t.TempDir()
	owners := nativeIntentFixtureOwners{}
	intent, err := lightingruntime.NewController(home, lightingruntime.Owners{Resource: owners, Scenario: owners})
	if err != nil {
		t.Fatal(err)
	}
	selected := lightingruntime.Selection{Version: "1.1", SHA256: openRGBAppImageSHA256, DevicePolicyID: updateTestDigest("policy")}
	if err := intent.Enable(t.Context(), selected); err != nil {
		t.Fatal(err)
	}
	steps := []string{}
	if err := runOpenRGBNativeStop(t.Context(), nil, intent, func(context.Context) error {
		steps = append(steps, "stop")
		return nil
	}, func(context.Context) error {
		steps = append(steps, "quiesce")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	current, err := intent.Current()
	if err != nil || current.Phase != lightingruntime.Enabled || current.Selection == nil || *current.Selection != selected {
		t.Fatalf("durable lighting intent after OpenRGB stop = %+v, err=%v", current, err)
	}
	if !reflect.DeepEqual(steps, []string{"quiesce", "stop"}) {
		t.Fatalf("stop ordering = %v", steps)
	}
	called := false
	if err := runOpenRGBNativeStop(t.Context(), []string{"--force"}, intent, func(context.Context) error { called = true; return nil }, func(context.Context) error { called = true; return nil }); err == nil || called {
		t.Fatal("native stop accepted a process override")
	}
}

func TestOpenRGBStopSerializesWithBackendPublication(t *testing.T) {
	home := t.TempDir()
	owners := nativeIntentFixtureOwners{}
	control, err := lightingruntime.NewController(home, lightingruntime.Owners{Resource: owners, Scenario: owners})
	if err != nil {
		t.Fatal(err)
	}
	selected := lightingruntime.Selection{Version: "1.1", SHA256: openRGBAppImageSHA256, DevicePolicyID: updateTestDigest("policy")}
	if err := control.Enable(t.Context(), selected); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	updateDone := make(chan error, 1)
	go func() {
		updateDone <- control.WithResourceUpdate(context.Background(), func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
			if intent.Phase != lightingruntime.Enabled || intent.Selection == nil || *intent.Selection != selected {
				return nil, errors.New("enabled intent changed before backend publication")
			}
			close(entered)
			<-release
			return intent.Selection, nil
		})
	}()
	<-entered
	steps := make(chan string, 2)
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- runOpenRGBNativeStop(context.Background(), nil, control,
			func(context.Context) error { steps <- "stop"; return nil },
			func(context.Context) error { steps <- "quiesce"; return nil })
	}()
	select {
	case err := <-stopDone:
		t.Fatalf("stop crossed an in-flight backend publication: %v", err)
	default:
	}
	close(release)
	if err := <-updateDone; err != nil {
		t.Fatal(err)
	}
	if err := <-stopDone; err != nil {
		t.Fatal(err)
	}
	if got := []string{<-steps, <-steps}; !reflect.DeepEqual(got, []string{"quiesce", "stop"}) {
		t.Fatalf("stop effects = %v", got)
	}
	current, err := control.Current()
	if err != nil || current.Phase != lightingruntime.Enabled || current.Selection == nil || *current.Selection != selected {
		t.Fatalf("durable intent after serialized stop = %+v, err=%v", current, err)
	}
}

func TestOpenRGBNativeResourceActionDisabledAndOverridesHaveNoEffects(t *testing.T) {
	for _, args := range [][]string{nil, {"--mode", "attach-only"}} {
		called := 0
		selection := func(context.Context) (lightingruntime.Selection, error) {
			return lightingruntime.Selection{}, errors.New("resource disabled")
		}
		effect := func(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
			called++
			return lightingruntime.Backend{}, nil
		}
		if err := runOpenRGBNativeAction(t.Context(), brokerTransportRestart, args, selection, effect, effect); err == nil || called != 0 {
			t.Fatalf("disabled native effect err=%v count=%d", err, called)
		}
	}
	c, _, _ := openRGBHeadlessFixture(t)
	if _, err := c.currentOpenRGBNativeSelection(t.Context()); err == nil {
		t.Fatal("default-disabled resource resolved a launch selection")
	}
}
