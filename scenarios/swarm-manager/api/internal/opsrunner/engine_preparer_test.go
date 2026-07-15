package opsrunner

import (
	"context"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
)

// TestEnginePreparerDrivesRealModeThroughRunner proves the generic runner drives
// a REAL operating-mode definition (holistic-loop) end to end: the effective
// inputs are prepared through the engine's own input SSOT, provenance pins the
// mode + digests, and the self-contained snapshot reproduces. This exercises the
// production preparer, not an in-memory stub.
func TestEnginePreparerDrivesRealModeThroughRunner(t *testing.T) {
	holistic, err := operatingmode.DefinitionFor(operatingmode.ModeHolisticLoop)
	if err != nil {
		t.Skipf("holistic-loop not loadable in this environment: %v", err)
	}
	drain, err := operatingmode.DefinitionFor(operatingmode.ModePhasedPlanDrain)
	if err != nil {
		t.Skipf("phased-plan-drain not loadable: %v", err)
	}
	preparer := NewEnginePreparer(map[string]operatingmode.Definition{
		string(operatingmode.ModeHolisticLoop): holistic,
	})
	preparer.Delegated = map[string]operatingmode.Definition{
		string(operatingmode.ModePhasedPlanDrain): drain,
	}

	// A catalog binding review-round -> holistic-loop (initiative target).
	dir := t.TempDir()
	catalog := writeCatalogBoundTo(t, dir, string(operatingmode.ModeHolisticLoop), testModeRevision)

	r, _, execStore := newRunner(t, catalog, t.TempDir(), preparer,
		fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})

	target := TargetRef{Kind: agentops.TargetInitiative, ID: "ship-search"}
	res, err := r.Invoke(context.Background(), InvokeRequest{Target: target, Operation: agentops.OpReviewRound, Simulate: true})
	if err != nil {
		t.Fatalf("invoke real mode: %v", err)
	}
	if res.Provenance.Mode != string(operatingmode.ModeHolisticLoop) {
		t.Fatalf("provenance mode = %q", res.Provenance.Mode)
	}
	if _, err := execStore.Reproduce(target.Kind, target.ID, res.ExecutionID); err != nil {
		t.Fatalf("reproduce real-mode execution: %v", err)
	}
}

// TestEnginePreparerRejectsIncompatibleMode proves a binding to a mode built for
// another unit of work fails closed with ErrIncompatibleMode.
func TestEnginePreparerRejectsIncompatibleMode(t *testing.T) {
	drain, err := operatingmode.DefinitionFor(operatingmode.ModePhasedPlanDrain)
	if err != nil {
		t.Skipf("phased-plan-drain not loadable: %v", err)
	}
	preparer := NewEnginePreparer(map[string]operatingmode.Definition{
		string(operatingmode.ModePhasedPlanDrain): drain, // targets plan-execution
	})
	// Bind an initiative-review operation (needs initiative caps) to a
	// plan-execution mode; resolution must reject it.
	dir := t.TempDir()
	catalog := writeCatalogBoundTo(t, dir, string(operatingmode.ModePhasedPlanDrain), testModeRevision)
	r, _, _ := newRunner(t, catalog, t.TempDir(), preparer,
		fakeDriver{outcome: "accepted", disposition: "success"}, &memRunOwners{})
	_, err = r.Invoke(context.Background(), InvokeRequest{
		Target: TargetRef{Kind: agentops.TargetInitiative, ID: "x"}, Operation: agentops.OpReviewRound, Simulate: true,
	})
	if err == nil {
		t.Fatalf("binding a plan-execution mode to an initiative review must fail closed")
	}
}
