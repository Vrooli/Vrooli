package studio_test

import (
	"context"
	"testing"

	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	"flow-verifier/internal/flows/kinds/navigation/studio"
	"flow-verifier/internal/flows/schemas"
)

func TestBuild_ProjectsFullExample(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	d := studio.Build(spec.(*navigation.Spec).Graph())

	if d.Renderer != "navigation-graph" {
		t.Errorf("renderer = %q, want navigation-graph", d.Renderer)
	}
	if len(d.Routes) == 0 {
		t.Error("routes empty; want at least one")
	}
	if len(d.Contexts) == 0 {
		t.Error("contexts empty; want at least one for the toggle UI")
	}
	// Affordances should have at least one presentation each (schema-enforced).
	for _, a := range d.Affordances {
		if len(a.Presentations) == 0 {
			t.Errorf("affordance %q has no presentations", a.ID)
		}
	}
	// Without findings, invariants must be empty (Build, not BuildWithFindings).
	if len(d.Invariants) != 0 {
		t.Errorf("Build should not populate invariants; got %d", len(d.Invariants))
	}
}

func TestVerifyAndBuild_StampsInvariantPassFail(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	d, err := studio.VerifyAndBuild(context.Background(), spec.(*navigation.Spec).Graph())
	if err != nil {
		t.Fatalf("VerifyAndBuild: %v", err)
	}
	if len(d.Invariants) == 0 {
		t.Fatal("VerifyAndBuild produced no invariants; expected one per reachability_invariant in the example")
	}
}

func TestKindStudioDescriptor_PassesThrough(t *testing.T) {
	k, _ := kind.Get(navigation.Name)
	spec, err := k.Load(schemas.NavigationFullExample, "schemas/examples/navigation-full.json")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sd := k.StudioDescriptor(spec)
	if sd.Renderer != "navigation-graph" {
		t.Errorf("renderer = %q", sd.Renderer)
	}
	if sd.Nodes == nil || sd.Edges == nil || sd.Toggles == nil {
		t.Errorf("kind.StudioDescriptor should populate nodes, edges, toggles; got %+v", sd)
	}
}
