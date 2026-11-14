package runtimeapp

import (
	"context"
	"errors"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
)

type lightingResourceOwnerFake struct {
	verified  []lightingruntime.Selection
	ensured   int
	disabled  int
	verifyErr error
}

func (f *lightingResourceOwnerFake) VerifyLightingSetup(_ context.Context, selection lightingruntime.Selection) error {
	f.verified = append(f.verified, selection)
	return f.verifyErr
}

func (f *lightingResourceOwnerFake) EnsureOpenRGB(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
	f.ensured++
	return lightingruntime.Backend{Endpoint: netip.MustParseAddrPort("127.0.0.1:6742"), Incarnation: "instance-1"}, nil
}
func (f *lightingResourceOwnerFake) DisableOpenRGB(context.Context) error { f.disabled++; return nil }

type lightingScenarioOwnerFake struct{ ensured, quiesced int }

func (f *lightingScenarioOwnerFake) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	f.ensured++
	return nil
}

func (f *lightingScenarioOwnerFake) QuiesceLightingReconciler(context.Context) error {
	f.quiesced++
	return nil
}

func TestLightingControlDisabledStatusDoesNotObserveResource(t *testing.T) {
	ctx := context.Background()
	home := filepath.Join(t.TempDir(), "operator")
	serving := false
	control, err := NewLightingControl(home, lightingruntime.Owners{}, func(context.Context) (resources.Status, error) {
		return resources.Status{Installed: true, Running: false, Serving: &serving, Health: "stopped", Message: "stopped"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := control.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Installed || got.Running || got.Ready || got.Degraded || got.State != LightingStateDisabled || got.ResourceObserved {
		t.Fatalf("status = %+v, want disabled state with resource facts unobserved", got)
	}
}

func TestLightingControlStatusUsesExplicitUnavailableNotInstalledReadyAndDegradedStates(t *testing.T) {
	ctx := context.Background()
	newControl := func(reader LightingResourceStatusReader) *LightingControl {
		control, err := NewLightingControl(filepath.Join(t.TempDir(), "operator"), lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &lightingScenarioOwnerFake{}}, reader)
		if err != nil {
			t.Fatal(err)
		}
		if err := control.Enable(ctx, lightingruntime.Selection{Version: "1.1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", DevicePolicyID: "policy"}); err != nil {
			t.Fatal(err)
		}
		return control
	}
	unknown, err := newControl(nil).Status(ctx)
	if err != nil || unknown.State != LightingStateDegraded || !unknown.Degraded || unknown.Installed || unknown.Running {
		t.Fatalf("missing status reader = %+v, err=%v; want explicit degraded/unavailable state", unknown, err)
	}

	for _, tc := range []struct {
		name     string
		resource resources.Status
		want     LightingState
	}{
		{name: "not installed", resource: resources.Status{Message: "no staged artifact"}, want: LightingStateNotInstalled},
		{name: "ready", resource: resources.Status{Installed: true, Running: true, Healthy: boolPointer(true), Serving: boolPointer(true)}, want: LightingStateReady},
		{name: "degraded", resource: resources.Status{Installed: true, Running: true, Healthy: boolPointer(false), Serving: boolPointer(true)}, want: LightingStateDegraded},
		{name: "running without readiness", resource: resources.Status{Installed: true, Running: true}, want: LightingStateDegraded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newControl(func(context.Context) (resources.Status, error) { return tc.resource, nil }).Status(ctx)
			if err != nil || got.State != tc.want {
				t.Fatalf("status = %+v, err=%v; want state %q", got, err, tc.want)
			}
		})
	}
}

func boolPointer(value bool) *bool { return &value }

func TestLightingControlSetupVerifiesOnlyAndEnableNeedsBothOwners(t *testing.T) {
	ctx := context.Background()
	selection := lightingruntime.Selection{Version: "1.1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", DevicePolicyID: "machine-policy"}
	resource := &lightingResourceOwnerFake{}
	control, err := NewLightingControl(filepath.Join(t.TempDir(), "operator"), lightingruntime.Owners{Resource: resource}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := control.Setup(ctx, selection); err != nil {
		t.Fatal(err)
	}
	if len(resource.verified) != 1 || resource.verified[0] != selection || resource.ensured != 0 {
		t.Fatalf("setup side effects: verified=%+v ensured=%d, want verification without launch", resource.verified, resource.ensured)
	}
	if err := control.Enable(ctx, selection); err == nil {
		t.Fatal("enable succeeded without scenario owner")
	}
}

func TestServiceComposesLightingControlOnlyWhenTypedOwnersExist(t *testing.T) {
	if _, err := (*Service)(nil).NewLightingControl(t.TempDir(), nil); err == nil {
		t.Fatal("nil service composed lighting operations")
	}
	app := &Service{}
	if _, err := app.NewLightingControl(t.TempDir(), nil); err == nil {
		t.Fatal("service without lighting owners composed lighting operations")
	}
	owners := lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &lightingScenarioOwnerFake{}}
	app.Lighting = &owners
	control, err := app.NewLightingControl(filepath.Join(t.TempDir(), "operator"), nil)
	if err != nil || control == nil {
		t.Fatalf("composed control = %v, err=%v", control, err)
	}
}

func TestLightingControlUsesFixedLifecycleAndDisableOrdering(t *testing.T) {
	ctx := context.Background()
	home := filepath.Join(t.TempDir(), "operator")
	resource := &lightingResourceOwnerFake{}
	scenario := &lightingScenarioOwnerFake{}
	control, err := NewLightingControl(home, lightingruntime.Owners{Resource: resource, Scenario: scenario}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection := lightingruntime.Selection{Version: "1.1", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", DevicePolicyID: "machine-policy"}
	if err := control.Enable(ctx, selection); err != nil {
		t.Fatal(err)
	}
	if err := control.Disable(ctx); err != nil {
		t.Fatal(err)
	}
	if resource.ensured != 0 || resource.disabled != 1 || scenario.quiesced != 1 {
		t.Fatalf("resource=%+v scenario=%+v, want quiesce then one backend disable and no immediate start", resource, scenario)
	}
	current, err := control.controller.Current()
	if err != nil || current.Phase != lightingruntime.Disabled {
		t.Fatalf("current intent = %+v, err=%v; want disabled", current, err)
	}
}

func TestLightingControlRejectsSetupFailureAndReportsStatusErrors(t *testing.T) {
	ctx := context.Background()
	setupErr := errors.New("device access witness unavailable")
	resource := &lightingResourceOwnerFake{verifyErr: setupErr}
	control, err := NewLightingControl(filepath.Join(t.TempDir(), "operator"), lightingruntime.Owners{Resource: resource}, func(context.Context) (resources.Status, error) {
		return resources.Status{}, errors.New("status owner failure")
	})
	if err != nil {
		t.Fatal(err)
	}
	selection := lightingruntime.Selection{Version: "1.1", SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", DevicePolicyID: "machine-policy"}
	if err := control.Setup(ctx, selection); !errors.Is(err, setupErr) {
		t.Fatalf("Setup error = %v, want verifier error", err)
	}
	if got, err := control.Status(ctx); err != nil || got.ResourceObserved {
		t.Fatalf("disabled status probed resource: %+v %v", got, err)
	}
}
