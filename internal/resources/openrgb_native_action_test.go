package resources

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

func TestOpenRGBNativeResourceActionUsesSameQualifiedOwner(t *testing.T) {
	selected := lightingruntime.Selection{Version: "1.1", SHA256: openRGBAppImageSHA256, DevicePolicyID: updateTestDigest("policy")}
	for _, action := range []string{brokerTransportStart, brokerTransportRestart} {
		t.Run(action, func(t *testing.T) {
			calls := []string{}
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
					return lightingruntime.Backend{}, nil
				}
			}
			if err := runOpenRGBNativeAction(t.Context(), action, nil, selectCurrent, handler("start"), handler("restart")); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []string{"persisted-selection", action}) {
				t.Fatalf("owner route %v", calls)
			}
		})
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
