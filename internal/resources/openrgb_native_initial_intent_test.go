package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

type nativeIntentFixtureOwners struct{}

func (nativeIntentFixtureOwners) VerifyLightingSetup(context.Context, lightingruntime.Selection) error {
	return nil
}

func (nativeIntentFixtureOwners) EnsureOpenRGB(context.Context, lightingruntime.Selection) (lightingruntime.Backend, error) {
	return lightingruntime.Backend{}, nil
}
func (nativeIntentFixtureOwners) DisableOpenRGB(context.Context) error            { return nil }
func (nativeIntentFixtureOwners) QuiesceLightingReconciler(context.Context) error { return nil }
func (nativeIntentFixtureOwners) EnsureLightingReconciler(context.Context, lightingruntime.Backend) error {
	return nil
}

func TestNativeOpenRGBResourceOperationSerializesBeforeDisableFence(t *testing.T) {
	home := t.TempDir()
	owners := nativeIntentFixtureOwners{}
	control, err := lightingruntime.NewController(home, lightingruntime.Owners{Resource: owners, Scenario: owners})
	if err != nil {
		t.Fatal(err)
	}
	selected := lightingruntime.Selection{Version: "1.1", SHA256: updateTestDigest("artifact"), DevicePolicyID: updateTestDigest("policy")}
	if err = control.Enable(t.Context(), selected); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- control.WithResourceUpdate(context.Background(), func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
			if intent.Phase != lightingruntime.Enabled || intent.Selection == nil || *intent.Selection != selected {
				return nil, errors.New("selection changed")
			}
			close(entered)
			<-release
			return &selected, nil
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	err = control.Disable(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		close(release)
		<-done
		t.Fatalf("disable bypassed in-flight RPC intent lock: %v", err)
	}
	current, err := control.Current()
	if err != nil {
		close(release)
		<-done
		t.Fatal(err)
	}
	if current.Phase != lightingruntime.Enabled {
		close(release)
		<-done
		t.Fatal("durable disable fence appeared while public RPC was still admitted")
	}
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = control.Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	called := false
	if err = control.WithResourceUpdate(t.Context(), func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		called = true
		if intent.Phase != lightingruntime.Disabled {
			return nil, errors.New("disabled intent changed")
		}
		return nil, nil
	}); err != nil || !called {
		t.Fatalf("disabled resource action was not safely serialized: called=%v err=%v", called, err)
	}
}

func TestNativeOpenRGBInitialClaimRejectsNewDisableOrSelection(t *testing.T) {
	a := openRGBInitialAttempt{Contract: ManagedUpdateContract{Artifact: ManagedServiceArtifact{Version: "1.1", SHA256: updateTestDigest("artifact")}}, Policy: openRGBExecPolicy{ID: updateTestDigest("policy")}}
	selected := lightingruntime.Selection{Version: a.Contract.Artifact.Version, SHA256: a.Contract.Artifact.SHA256, DevicePolicyID: a.Policy.ID}
	current := lightingruntime.Intent{Phase: lightingruntime.Enabled, Selection: &selected}
	if err := verifyOpenRGBNativeInitialLightingSelection(current, a); err != nil {
		t.Fatal(err)
	}
	// Prepare may have succeeded before another owner publishes its fence. The
	// repeated Claim callback must reject it before granting payload execution.
	for _, phase := range []lightingruntime.Phase{lightingruntime.Disabling, lightingruntime.Disabled} {
		current.Phase = phase
		if err := verifyOpenRGBNativeInitialLightingSelection(current, a); err == nil {
			t.Fatal("late claim crossed durable disable fence")
		}
	}
	current.Phase = lightingruntime.Enabled
	selected.DevicePolicyID = updateTestDigest("successor")
	if err := verifyOpenRGBNativeInitialLightingSelection(current, a); err == nil {
		t.Fatal("late claim replaced newer lighting selection")
	}
}
