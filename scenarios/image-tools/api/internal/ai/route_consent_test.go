package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/safety"
)

// localTierConsent is the shipped rule on a personal install: the route-locality
// rule only, so local runs stay unrestricted.
func localTierConsent(ctx context.Context, op, weight, tier string, faces func(context.Context) (int, error)) (bool, string) {
	need := safety.NewGate(safety.TierLocal, nil).ConsentForInput(ctx, safety.Weight(weight), op, tier, faces)
	return need.Required, need.Reason
}

func withConsentRule(d *Deps) { d.RouteConsent = localTierConsent }

func TestAutomaticCloudRouteNeverLeavesTheMachineWithoutConsent(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, withConsentRule)
	// No usable GPU: the cloud would otherwise be chosen automatically.
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	if plan.Tier == backends.TierBYOK.String() {
		t.Fatalf("an identity edit must not be sent off this machine without consent: %+v", plan)
	}
	if plan.Route == nil || plan.Route.Reason != routeStayConsent {
		t.Fatalf("the local choice must say why the cloud was not used: %+v", plan.Route)
	}
	if plan.Route.Fallback {
		t.Fatal("a wait-budget re-route must not be armed for a request that cannot consent")
	}

	affirmed := rig.plan(t, PlanRequest{Prompt: hatPrompt, ConsentAffirmed: true})
	if affirmed.Tier != backends.TierBYOK.String() {
		t.Fatalf("with consent the cloud route is available again: %+v", affirmed)
	}
}

func TestLocalRunsAreUnrestrictedOnAGPUHost(t *testing.T) {
	rig := newRouteRig(t, gpuHost, withConsentRule)
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt})
	if plan.Tier != backends.TierLocalGPU.String() {
		t.Fatalf("a local run needs no consent: %+v", plan)
	}
}

func TestDemandedCloudRouteWithoutConsentIsReportedAsCloudForTheSubmitEdgeToRefuse(t *testing.T) {
	rig := newRouteRig(t, gpuHost, withConsentRule)
	plan := rig.plan(t, PlanRequest{Prompt: hatPrompt, AllowBYOK: true, Role: RoleEditIdentity})
	if plan.Tier != backends.TierBYOK.String() {
		t.Fatalf("a caller who demands the cloud gets the cloud plan, so the edge can refuse it: %+v", plan)
	}
	need := safety.PolicyFor(safety.TierLocal).ConsentFor(safety.Weight(plan.Weight), "edit_instruct", plan.Tier)
	if !need.Required || !strings.Contains(need.Reason, "outside this machine") {
		t.Fatalf("the plan's tier must be enough for the gate to ask for consent: %+v", need)
	}
}

func TestExecutionRefusesACloudJobWhoseConsentWasNeverAffirmed(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, withConsentRule)
	blocked, why := rig.engine.consentBlocks(context.Background(), "edit_instruct", "high", false, nil)
	if !blocked || why == "" {
		t.Fatalf("the engine must hold an unaffirmed cloud job: blocked=%v why=%q", blocked, why)
	}
	if blocked, _ := rig.engine.consentBlocks(context.Background(), "edit_instruct", "high", true, nil); blocked {
		t.Fatal("an affirmed job must pass")
	}
	if blocked, _ := rig.engine.consentBlocks(context.Background(), "text_to_image", "none", false, nil); blocked {
		t.Fatal("a no-weight operation never needs consent")
	}
}

// A low-weight operation (a caption, boxes, tags, a colourisation) sent to a provider
// asks for consent only when its image shows people: the rule protects the people in
// the image, and a picture of a guinea pig has none.
func TestLowWeightCloudRouteAsksForConsentOnlyWhenTheImageShowsPeople(t *testing.T) {
	rig := newRouteRig(t, noGPUHost, withConsentRule)
	count := func(n int, err error) func(context.Context) (int, error) {
		return func(context.Context) (int, error) { return n, err }
	}
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		faces   func(context.Context) (int, error)
		blocked bool
	}{
		{"nobody shown", count(0, nil), false},
		{"someone shown", count(2, nil), true},
		{"the check cannot run", count(0, errors.New("detector down")), true},
		{"nobody can look", nil, true},
	} {
		blocked, why := rig.engine.consentBlocks(ctx, "caption", "low", false, tc.faces)
		if blocked != tc.blocked || (blocked && why == "") {
			t.Errorf("%s: blocked=%v (%q), want %v", tc.name, blocked, why, tc.blocked)
		}
	}
	if blocked, _ := rig.engine.consentBlocks(ctx, "caption", "low", true, nil); blocked {
		t.Error("a cleared request (affirmed, or already found to show nobody) must pass without another look")
	}
	// Identity-altering work keeps its rule whatever the check says.
	if blocked, _ := rig.engine.consentBlocks(ctx, "face_swap", "high", false, count(0, nil)); !blocked {
		t.Error("a high-weight operation asks for consent even when no face is found")
	}
}
