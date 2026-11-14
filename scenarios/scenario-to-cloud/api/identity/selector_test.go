package identity

import (
	"context"
	"testing"

	"scenario-to-cloud/apierrors"
)

type fakeResolver struct {
	refs []DeploymentRef
	seen Selector
}

func (f *fakeResolver) ResolveDeployments(_ context.Context, selector Selector) ([]DeploymentRef, error) {
	f.seen = selector
	var out []DeploymentRef
	for _, ref := range f.refs {
		if selector.ID != "" && ref.ID != selector.ID {
			continue
		}
		if selector.ScenarioID != "" && ref.ScenarioID != selector.ScenarioID {
			continue
		}
		if selector.Environment != "" && ref.Environment != selector.Environment {
			continue
		}
		if selector.Host != "" && ref.Target.Locator.Host != selector.Host {
			continue
		}
		out = append(out, ref)
	}
	return out, nil
}

func sampleRefs() []DeploymentRef {
	return []DeploymentRef{
		{ID: "dep-prod", ScenarioID: "demo-app", Environment: "production", Target: TargetRef{Transport: TransportSSH, Locator: TargetLocator{Host: "203.0.113.10"}}},
		{ID: "dep-staging", ScenarioID: "demo-app", Environment: "staging", Target: TargetRef{Transport: TransportSSH, Locator: TargetLocator{Host: "203.0.113.20"}}},
	}
}

// TestSelectorKindAcceptsExactlyFourForms [REQ:STC-P0-015] pins the selector
// grammar: id alone, or scenario_id with exactly one of environment, domain,
// host.
func TestSelectorKindAcceptsExactlyFourForms(t *testing.T) {
	cases := []struct {
		name     string
		selector Selector
		want     SelectorKind
		wantErr  bool
	}{
		{name: "by id", selector: Selector{ID: "x"}, want: SelectByID},
		{name: "scenario+environment", selector: Selector{ScenarioID: "a", Environment: "production"}, want: SelectByScenarioEnvironment},
		{name: "scenario+domain", selector: Selector{ScenarioID: "a", Domain: "example.com"}, want: SelectByScenarioDomain},
		{name: "scenario+host", selector: Selector{ScenarioID: "a", Host: "203.0.113.1"}, want: SelectByScenarioHost},
		{name: "scenario alone is ambiguous by construction", selector: Selector{ScenarioID: "a"}, wantErr: true},
		{name: "id with facets", selector: Selector{ID: "x", ScenarioID: "a"}, wantErr: true},
		{name: "two facets", selector: Selector{ScenarioID: "a", Environment: "p", Host: "h"}, wantErr: true},
		{name: "empty", selector: Selector{}, wantErr: true},
		{name: "whitespace is empty", selector: Selector{ScenarioID: "  ", Domain: "d"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, err := tc.selector.Kind()
			if tc.wantErr {
				if !apierrors.Is(err, apierrors.CodeDeploymentSelectorInvalid) {
					t.Fatalf("expected deployment_selector_invalid, got kind=%q err=%v", kind, err)
				}
				return
			}
			if err != nil || kind != tc.want {
				t.Fatalf("kind=%q err=%v want %q", kind, err, tc.want)
			}
		})
	}
}

// TestResolveTwoInstallationsResolveDistinctRecords [REQ:STC-P0-015] proves
// P03-A02: two installations of one scenario in distinct environments resolve
// to distinct deployment records.
func TestResolveTwoInstallationsResolveDistinctRecords(t *testing.T) {
	repo := &fakeResolver{refs: sampleRefs()}
	prod, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app", Environment: "production"})
	if err != nil {
		t.Fatalf("resolve production: %v", err)
	}
	staging, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app", Environment: "staging"})
	if err != nil {
		t.Fatalf("resolve staging: %v", err)
	}
	if prod.ID != "dep-prod" || staging.ID != "dep-staging" || prod.ID == staging.ID {
		t.Fatalf("prod=%s staging=%s", prod.ID, staging.ID)
	}
	byHost, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app", Host: "203.0.113.20"})
	if err != nil || byHost.ID != "dep-staging" {
		t.Fatalf("resolve by host = %+v, %v", byHost, err)
	}
}

// TestResolveAmbiguousSelectorIsTypedConflict [REQ:STC-P0-015] proves P03-A03:
// a selector matching more than one record returns the typed
// deployment_selector_ambiguous conflict listing every candidate.
func TestResolveAmbiguousSelectorIsTypedConflict(t *testing.T) {
	refs := sampleRefs()
	// Two environments on the same host: (scenario, host) is now ambiguous.
	refs[1].Target.Locator.Host = refs[0].Target.Locator.Host
	repo := &fakeResolver{refs: refs}

	_, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app", Host: "203.0.113.10"})
	typed := apierrors.As(err)
	if typed.Code != apierrors.CodeDeploymentSelectorAmbiguous {
		t.Fatalf("code = %q (%v)", typed.Code, err)
	}
	if typed.Status() != 409 {
		t.Fatalf("status = %d", typed.Status())
	}
	candidates, ok := typed.Details["candidates"].([]map[string]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("candidates = %#v", typed.Details["candidates"])
	}
	if candidates[0]["id"] != "dep-prod" || candidates[1]["id"] != "dep-staging" {
		t.Fatalf("candidates not sorted by id: %v", candidates)
	}
	if typed.NextAction == nil || typed.NextAction.Reference != "id" {
		t.Fatalf("next action must point at selecting by id: %+v", typed.NextAction)
	}
}

// TestResolveNoMatchIsTypedNotFound proves the empty case is a typed 404, not
// a nil ref.
func TestResolveNoMatchIsTypedNotFound(t *testing.T) {
	repo := &fakeResolver{refs: sampleRefs()}
	_, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app", Environment: "qa"})
	if !apierrors.Is(err, apierrors.CodeDeploymentNotFound) {
		t.Fatalf("expected deployment_not_found, got %v", err)
	}
	if apierrors.As(err).Status() != 404 {
		t.Fatalf("status = %d", apierrors.As(err).Status())
	}
}

// TestResolveRejectsInvalidSelectorBeforeStorage proves a malformed selector
// never reaches the repository.
func TestResolveRejectsInvalidSelectorBeforeStorage(t *testing.T) {
	repo := &fakeResolver{refs: sampleRefs()}
	_, err := Resolve(context.Background(), repo, Selector{ScenarioID: "demo-app"})
	if !apierrors.Is(err, apierrors.CodeDeploymentSelectorInvalid) {
		t.Fatalf("expected deployment_selector_invalid, got %v", err)
	}
	if repo.seen != (Selector{}) {
		t.Fatalf("repository was queried with %+v", repo.seen)
	}
}

// TestTargetKeyPrefersMachineIdentity pins the uniqueness key: a Bridge
// machine id wins over the host, and the namespaces never collide.
func TestTargetKeyPrefersMachineIdentity(t *testing.T) {
	ssh := TargetRef{Transport: TransportSSH, Locator: TargetLocator{Host: "203.0.113.10"}}
	bridge := TargetRef{MachineID: "203.0.113.10", Transport: TransportBridge, Locator: TargetLocator{Host: "203.0.113.10"}}
	if ssh.Key() != "host:203.0.113.10" || bridge.Key() != "machine:203.0.113.10" {
		t.Fatalf("keys = %q, %q", ssh.Key(), bridge.Key())
	}
	if (TargetRef{}).Key() != "" || !(TargetRef{}).IsZero() {
		t.Fatalf("empty target must have no key")
	}
	if NormalizeEnvironment("  ") != DefaultEnvironment || NormalizeEnvironment(" staging ") != "staging" {
		t.Fatalf("environment normalisation broken")
	}
}
