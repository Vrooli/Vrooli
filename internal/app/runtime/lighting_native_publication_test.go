package runtimeapp

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
)

// Public prepare owns only immutable target publication while enabled. This
// tests the durable intent boundary; OS/code publication is a separate leaf.
func TestNativeLightingServingPrepareOnlyPublishesTargetAndPreservesIntent(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "published", true: "publication-failed"}[fail], func(t *testing.T) {
			resource := &lightingResourceOwnerFake{}
			scenario := &lightingScenarioOwnerFake{}
			control, err := NewLightingControl(t.TempDir(), lightingruntime.Owners{Resource: resource, Scenario: scenario}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = control.Enable(t.Context(), lightingruntime.Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "old-adopted-policy"}); err != nil {
				t.Fatal(err)
			}
			before, err := control.controller.Current()
			if err != nil {
				t.Fatal(err)
			}
			initialEnsure, initialScenario := resource.ensured, scenario.ensured
			publications := 0
			denied := errors.New("qualified target publication failed")
			actions := LightingActions{Control: control, Resources: &resources.Controller{}, NativePrepare: func(context.Context) error { t.Fatal("serving prepare reprovisioned machine/device/boot"); return nil }, NativePublishArtifact: func(context.Context) error {
				publications++
				if fail {
					return denied
				}
				return nil
			}}
			err = actions.PrepareNative(t.Context())
			if fail && !errors.Is(err, denied) || !fail && err != nil {
				t.Fatal(err)
			}
			after, readErr := control.controller.Current()
			if readErr != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("publication changed serving intent: before=%+v after=%+v err=%v", before, after, readErr)
			}
			if publications != 1 || resource.ensured != initialEnsure || resource.disabled != 0 || scenario.ensured != initialScenario || scenario.quiesced != 0 {
				t.Fatal("content publication changed service or reconciler lifecycle")
			}
		})
	}
}

type nativePublicationPendingDisable struct{ lightingScenarioOwnerFake }

func (*nativePublicationPendingDisable) QuiesceLightingReconciler(context.Context) error {
	return errors.New("owner temporarily unavailable")
}

func TestNativeLightingPublicationCannotBypassPendingDisable(t *testing.T) {
	control, err := NewLightingControl(t.TempDir(), lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &nativePublicationPendingDisable{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = control.Enable(t.Context(), lightingruntime.Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "policy"}); err != nil {
		t.Fatal(err)
	}
	if err = control.Disable(t.Context()); err == nil {
		t.Fatal("fixture did not retain pending disable")
	}
	called := false
	effect := func(context.Context) error { called = true; return nil }
	actions := LightingActions{Control: control, Resources: &resources.Controller{}, NativePrepare: effect, NativePublishArtifact: effect}
	if err = actions.PrepareNative(t.Context()); err == nil || called {
		t.Fatal("pending disable admitted machine or content publication")
	}
}
