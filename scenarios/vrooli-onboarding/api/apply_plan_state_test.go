package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

// TestApplyStateFromReadinessNeverGuesses pins the mapping from a readiness
// verdict to the state shown to the operator before consent. "deferred" and
// "unsupported" both mean this process could not decide, and presenting either
// as "satisfied" would tell the operator an item is already in place when
// nothing verified that.
func TestApplyStateFromReadinessNeverGuesses(t *testing.T) {
	cases := map[string]string{
		"ready":       applyStateSatisfied,
		"missing":     applyStatePending,
		"deferred":    applyStateUnknown,
		"unsupported": applyStateUnknown,
		"":            applyStateUnknown,
		"something":   applyStateUnknown,
	}
	for status, want := range cases {
		if got := applyStateFromReadiness(status); got != want {
			t.Fatalf("applyStateFromReadiness(%q) = %q, want %q", status, got, want)
		}
	}
}

// TestObservedStateDefaultsToUnknown covers states absent from measured input.
// Runtime observation failures must remain unknown rather than guessed.
func TestObservedStateDefaultsToUnknown(t *testing.T) {
	observed := map[string]string{"tool:git": applyStateSatisfied, "tool:blank": ""}
	if got := observedState(observed, "tool:git"); got != applyStateSatisfied {
		t.Fatalf("measured item = %q, want %q", got, applyStateSatisfied)
	}
	for _, id := range []string{"resource:postgres", "scenario:web-console", "tool:blank"} {
		if got := observedState(observed, id); got != applyStateUnknown {
			t.Fatalf("unmeasured item %q = %q, want %q", id, got, applyStateUnknown)
		}
	}
}

type fakeApplyPlanStatusReader struct {
	resources    map[string]*cliv1.ResourceStatusResponse
	resourceList *cliv1.ResourceStatusesResponse
	scenarios    *cliv1.ScenarioStatusListResponse
	resourceErr  error
	scenarioErr  error
}

func (f fakeApplyPlanStatusReader) ResourceHealthStatus(_ context.Context, name string) (*cliv1.ResourceStatusResponse, error) {
	return f.resources[name], f.resourceErr
}

func (f fakeApplyPlanStatusReader) ResourceHealthStatuses(context.Context) (*cliv1.ResourceStatusesResponse, error) {
	return f.resourceList, f.resourceErr
}

func (f fakeApplyPlanStatusReader) ScenarioStatuses(context.Context) (*cliv1.ScenarioStatusListResponse, error) {
	return f.scenarios, f.scenarioErr
}

func TestApplyRuntimeReadbackMapsPositiveNegativeAndUnavailableEvidence(t *testing.T) {
	closure := closureResult{
		Resources: []closureMember{{Name: "postgres"}},
		Scenarios: []closureMember{{Name: "writer"}},
	}
	uncertain := map[string]struct{}{"resource:postgres": {}, "scenario:writer": {}}
	positive := fakeApplyPlanStatusReader{
		resources: map[string]*cliv1.ResourceStatusResponse{"postgres": {Success: true, Name: "postgres", Resource: &cliv1.ResourceStatus{Resource: &cliv1.Resource{Name: "postgres", Enabled: true}}}},
		scenarios: &cliv1.ScenarioStatusListResponse{Success: true, Scenarios: []*cliv1.ScenarioStatusItem{{Name: "writer", Status: "running"}}},
	}
	prior := []applyRun{{Target: "local", Items: []applyItemResult{{applyItem: applyItem{ID: "resource:postgres"}, Outcome: applyItemIndeterminate}, {applyItem: applyItem{ID: "scenario:writer"}, Outcome: applyItemIndeterminate}}}}
	eligibility := func(observed map[string]string) []string {
		plan := []applydomain.Item{{ID: "resource:postgres", ObservedState: observedState(observed, "resource:postgres")}, {ID: "scenario:writer", ObservedState: observedState(observed, "scenario:writer")}}
		return applyReplayBlockers(prior, "local", plan)
	}
	got := observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, closure, uncertain, positive)
	if got["resource:postgres"] != applyStateSatisfied || got["scenario:writer"] != applyStateSatisfied {
		t.Fatalf("positive readback = %#v, want both runtime items satisfied", got)
	}
	if blockers := eligibility(got); len(blockers) != 0 {
		t.Fatalf("positive readback did not release retry: %v", blockers)
	}

	batchClosure := closureResult{Resources: []closureMember{{Name: "postgres"}, {Name: "redis"}}}
	batchUncertain := map[string]struct{}{"resource:postgres": {}, "resource:redis": {}}
	batchReader := fakeApplyPlanStatusReader{resourceList: &cliv1.ResourceStatusesResponse{Success: true, Resources: []*cliv1.ResourceStatus{
		{Resource: &cliv1.Resource{Name: "postgres", Enabled: true}},
		{Resource: &cliv1.Resource{Name: "redis", Enabled: true}},
	}}}
	batchObserved := observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, batchClosure, batchUncertain, batchReader)
	if batchObserved["resource:postgres"] != applyStateSatisfied || batchObserved["resource:redis"] != applyStateSatisfied {
		t.Fatalf("batched resource status = %#v, want both enabled", batchObserved)
	}

	negative := fakeApplyPlanStatusReader{
		resources: map[string]*cliv1.ResourceStatusResponse{"postgres": {Success: true, Name: "postgres", Resource: &cliv1.ResourceStatus{Resource: &cliv1.Resource{Name: "postgres", Enabled: false}}}},
		scenarios: &cliv1.ScenarioStatusListResponse{Success: true, Scenarios: []*cliv1.ScenarioStatusItem{{Name: "writer", Status: "stopped"}}},
	}
	got = observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, closure, uncertain, negative)
	if got["resource:postgres"] != applyStatePending || got["scenario:writer"] != applyStatePending {
		t.Fatalf("negative readback = %#v, want pending and still ineligible", got)
	}
	if blockers := eligibility(got); len(blockers) != 2 {
		t.Fatalf("negative readback released uncertain retry: %v", blockers)
	}

	unavailable := fakeApplyPlanStatusReader{resourceErr: errors.New("resource status unavailable"), scenarioErr: errors.New("scenario status unavailable")}
	got = observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, closure, uncertain, unavailable)
	if observedState(got, "resource:postgres") != applyStateUnknown || observedState(got, "scenario:writer") != applyStateUnknown {
		t.Fatalf("unavailable readback = %#v, want unknown", got)
	}
	if blockers := eligibility(got); len(blockers) != 2 {
		t.Fatalf("unavailable readback released uncertain retry: %v", blockers)
	}

	partial := fakeApplyPlanStatusReader{
		resources: map[string]*cliv1.ResourceStatusResponse{"postgres": {Success: false, Name: "postgres", Resource: &cliv1.ResourceStatus{Resource: &cliv1.Resource{Name: "postgres", Enabled: true}}}},
		scenarios: &cliv1.ScenarioStatusListResponse{Success: true, Scenarios: []*cliv1.ScenarioStatusItem{{Name: "writer", Status: "running"}}, DiscoveryFailures: []*cliv1.DiscoveryFailure{{Name: "writer"}}},
	}
	got = observeApplyStates(context.Background(), "local", t.TempDir(), hostRequirementsResponse{}, closure, uncertain, partial)
	if observedState(got, "resource:postgres") != applyStateUnknown || observedState(got, "scenario:writer") != applyStateUnknown {
		t.Fatalf("partial discovery failures overrode missing evidence: %#v", got)
	}
	if blockers := eligibility(got); len(blockers) != 2 {
		t.Fatalf("partial discovery failure released uncertain retry: %v", blockers)
	}
}

// TestApplyPlanStateDoesNotChangeSelectionDigest is the safety property. State
// is observed from the host, so if it fed the digest, an unrelated host change
// between planning and applying would invalidate consent the operator had
// already given, or silently mark a stale selection as fresh.
func TestApplyPlanStateDoesNotChangeSelectionDigest(t *testing.T) {
	base := []applyItem{
		{ID: "tool:git", Kind: "tool", Name: "git", Required: true},
		{ID: "safeguard:clock", Kind: "safeguard", Name: "clock", Required: true, Privileged: true},
	}
	withState := []applyItem{
		{ID: "tool:git", Kind: "tool", Name: "git", Required: true, State: applyStateSatisfied},
		{ID: "safeguard:clock", Kind: "safeguard", Name: "clock", Required: true, Privileged: true, State: applyStatePending},
	}
	if selectionDigest(base) != selectionDigest(withState) {
		t.Fatal("observed state must not participate in the selection digest")
	}
}

// TestApplyPlanEmitsOnlyTheKindsBothSurfacesRender is the parity guard between
// the wizard CLI and the onboarding UI. Both renderers enumerate these four
// kinds to explain what apply does and to label each section. A fifth kind
// added to buildApplyPlan without updating them would reach an operator as an
// unlabelled item whose host effect is never disclosed, so this test fails
// until both surfaces are taught about it.
//
// CLI: cli/domains/wizard/register.go — pluralKind and the per-kind effect list.
// UI:  ui/src/lib/applyPlan.ts — KIND_LABELS, KIND_ORDER, APPLY_KIND_ACTIONS.
func TestApplyPlanEmitsOnlyTheKindsBothSurfacesRender(t *testing.T) {
	items := buildApplyPlan(applyPlanInput{
		Requirements: hostRequirementsResponse{
			Tools:      []hostItem{{hostRequirement: hostRequirement{Name: "git"}, Status: "required"}},
			Safeguards: []hostItem{{hostRequirement: hostRequirement{Name: "clock"}, Status: "required"}},
		},
		Closure: closureResult{
			Resources: []closureMember{{Name: "postgres"}},
			Scenarios: []closureMember{{Name: "web-console"}},
		},
	})

	rendered := map[string]bool{"tool": true, "safeguard": true, "resource": true, "scenario": true}
	seen := map[string]bool{}
	for _, item := range items {
		if !rendered[item.Kind] {
			t.Fatalf("buildApplyPlan emits kind %q, which neither the wizard CLI nor the onboarding UI can label or explain", item.Kind)
		}
		seen[item.Kind] = true
	}
	for kind := range rendered {
		if !seen[kind] {
			t.Fatalf("kind %q is rendered by both surfaces but this test no longer exercises it; the parity guard would miss a regression", kind)
		}
	}
}

func TestCatalogIdentityChangesWhenManifestEffectChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"service":{"name":"alpha","system_required":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	models := []ScenarioReadModel{{Name: "alpha"}}
	first, err := catalogIdentity(root, models)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"service":{"name":"alpha","system_required":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := catalogIdentity(root, models)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("catalog identity ignored a changed scenario manifest")
	}
}
