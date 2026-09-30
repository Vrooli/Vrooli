package spec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildReadinessProfileIsDeterministicAndUsesRegionBindings(t *testing.T) {
	spec := &ScenarioSpec{Index: IndexDocument{Scenario: "demo"}, Pages: map[string]PageDocument{
		"home": {
			Page: PageIdentity{ID: "home", Routes: []string{"/", "/home"}},
			Regions: []ExperienceRegion{
				{ID: "optional-help", Required: false, Component: ComponentReference{Local: "help"}, Lifecycle: RegionLifecycle{Kind: "static", States: []string{"static"}}},
				{ID: "results", Required: true, Component: ComponentReference{Local: "results"}, Lifecycle: RegionLifecycle{Kind: "async", States: []string{"ready", "loading", "error"}}},
			},
			Bindings: Bindings{Regions: map[string]Binding{"results": {TestID: "results-surface"}, "optional-help": {Selector: "#help"}}},
		},
	}}
	profile, err := BuildReadinessProfile(spec)
	require.NoError(t, err)
	require.Equal(t, "experience-readiness-profile/v1", profile.Version)
	require.Equal(t, []string{"/", "/home"}, profile.Pages[0].Routes)
	require.Equal(t, "optional-help", profile.Pages[0].Regions[0].ID)
	require.False(t, profile.Pages[0].Regions[0].Required)
	require.Equal(t, "results-surface", profile.Pages[0].Regions[1].Binding.TestID)
	require.Equal(t, []string{"error", "loading", "ready"}, profile.Pages[0].Regions[1].Lifecycle.States)
}

func TestBuildReadinessProfileProjectsStateSetupRoutesForRuntimeConsumers(t *testing.T) {
	spec := &ScenarioSpec{
		Index: IndexDocument{Scenario: "demo"},
		Pages: map[string]PageDocument{
			"asset": {
				Page:   PageIdentity{ID: "asset", Routes: []string{"/assets/:id"}},
				States: []State{{ID: "ready", Setup: Setup{Route: "/assets/Button", Query: map[string]string{"tab": "preview"}}}},
			},
		},
	}
	profile, err := BuildReadinessProfile(spec)
	require.NoError(t, err)
	require.Equal(t, []string{"/assets/Button?tab=preview"}, profile.Pages[0].RuntimeRoutes)
}
