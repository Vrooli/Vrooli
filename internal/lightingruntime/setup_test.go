package lightingruntime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestPreparationRequiresDisabledIntent(t *testing.T) {
	ctx := context.Background()
	c, err := NewController(filepath.Join(t.TempDir(), "home"), Owners{})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := c.PrepareDisabled(ctx, func() error { calls++; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := c.write(Intent{SchemaVersion: lightingStateVersion, Phase: Enabled, Selection: &Selection{Version: "1.1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", DevicePolicyID: "policy"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.PrepareDisabled(ctx, func() error { calls++; return nil }); err == nil || calls != 1 {
		t.Fatalf("active preparation admitted: %v %d", err, calls)
	}
}

func TestEnablePreparationFailureDoesNotAdmitEnabledIntent(t *testing.T) {
	_, c, f := newLightingFixture(t)
	ctx := context.Background()
	failure := errors.New("resource enablement persistence failed")
	if err := c.EnablePrepared(ctx, selectedLighting(), func() error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	state, err := c.Current()
	if err != nil || state.Phase != Disabled {
		t.Fatalf("admitted failed preparation: %+v %v", state, err)
	}
	expectLightingCalls(t, f, "verify")
}

func TestPreparationReceiptSurvivesEnableDisableWithoutRuntimeClaims(t *testing.T) {
	_, c, _ := newLightingFixture(t)
	ctx := context.Background()
	proof := &Preparation{Source: PreparationManaged, Selection: selectedLighting(), Qualified: true, LaunchBlocker: "native launch must verify access"}
	if err := c.PrepareDisabledResult(ctx, func() (*Preparation, error) { return proof, nil }); err != nil {
		t.Fatal(err)
	}
	for _, action := range []func() error{func() error { return nil }, func() error { return c.Enable(ctx, selectedLighting()) }, func() error { return c.Disable(ctx) }} {
		if err := action(); err != nil {
			t.Fatal(err)
		}
		state, err := c.Current()
		if err != nil || state.Preparation == nil || *state.Preparation != *proof {
			t.Fatalf("lost setup receipt: %+v %v", state, err)
		}
	}
}

func TestFailedExplicitPreparationInvalidatesEarlierQualification(t *testing.T) {
	_, c, _ := newLightingFixture(t)
	ctx := context.Background()
	proof := &Preparation{Source: PreparationManaged, Selection: selectedLighting(), Qualified: true}
	if err := c.PrepareDisabledResult(ctx, func() (*Preparation, error) { return proof, nil }); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("selected artifact missing")
	if err := c.PrepareDisabledResult(ctx, func() (*Preparation, error) { return nil, failure }); !errors.Is(err, failure) {
		t.Fatalf("failure=%v", err)
	}
	state, err := c.Current()
	if err != nil || state.Preparation != nil || state.Phase != Disabled {
		t.Fatalf("stale qualification retained after known failed setup: %+v %v", state, err)
	}
}
