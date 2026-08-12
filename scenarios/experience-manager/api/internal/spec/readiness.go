package spec

import (
	"fmt"
	"net/url"
	"sort"
)

// ReadinessProfile is the deterministic consumer projection of an authored
// experience spec. It contains no selectors or lifecycle declarations not
// already authored in the spec; consumers must never maintain a second
// readiness registry.
type ReadinessProfile struct {
	Version  string                 `json:"version"`
	Scenario string                 `json:"scenario"`
	Pages    []ReadinessProfilePage `json:"pages"`
}

type ReadinessProfilePage struct {
	ID            string                   `json:"id"`
	Routes        []string                 `json:"routes"`
	RuntimeRoutes []string                 `json:"runtimeRoutes,omitempty"`
	Regions       []ReadinessProfileRegion `json:"regions"`
}

type ReadinessProfileRegion struct {
	ID        string             `json:"id"`
	Required  bool               `json:"required"`
	Binding   Binding            `json:"binding"`
	Component ComponentReference `json:"component"`
	Lifecycle RegionLifecycle    `json:"lifecycle"`
}

// BuildReadinessProfile resolves the authored page composition into a stable
// route-oriented profile. It is pure and sorted, so downstream BAS, UI Health,
// and Workflow Health observe the same result from the same source document.
func BuildReadinessProfile(spec *ScenarioSpec) (ReadinessProfile, error) {
	if spec == nil || spec.Index.Scenario == "" {
		return ReadinessProfile{}, fmt.Errorf("experience spec is required")
	}
	profile := ReadinessProfile{Version: "experience-readiness-profile/v1", Scenario: spec.Index.Scenario}
	pageIDs := make([]string, 0, len(spec.Pages))
	for id := range spec.Pages {
		pageIDs = append(pageIDs, id)
	}
	sort.Strings(pageIDs)
	for _, pageID := range pageIDs {
		page := spec.Pages[pageID]
		out := ReadinessProfilePage{ID: page.Page.ID, Routes: append([]string(nil), page.Page.Routes...)}
		for _, state := range page.States {
			if route := runtimeRoute(state.Setup); route != "" {
				out.RuntimeRoutes = append(out.RuntimeRoutes, route)
			}
		}
		sort.Strings(out.Routes)
		sort.Strings(out.RuntimeRoutes)
		for _, region := range page.Regions {
			out.Regions = append(out.Regions, ReadinessProfileRegion{
				ID:        region.ID,
				Required:  region.Required,
				Binding:   page.Bindings.Regions[region.ID],
				Component: region.Component,
				Lifecycle: RegionLifecycle{Kind: region.Lifecycle.Kind, States: append([]string(nil), region.Lifecycle.States...)},
			})
		}
		sort.Slice(out.Regions, func(i, j int) bool { return out.Regions[i].ID < out.Regions[j].ID })
		for i := range out.Regions {
			sort.Strings(out.Regions[i].Lifecycle.States)
		}
		profile.Pages = append(profile.Pages, out)
	}
	return profile, nil
}

func runtimeRoute(setup Setup) string {
	if setup.Route == "" {
		return ""
	}
	values := make(url.Values, len(setup.Query))
	for key, value := range setup.Query {
		values.Set(key, value)
	}
	if encoded := values.Encode(); encoded != "" {
		return setup.Route + "?" + encoded
	}
	return setup.Route
}

// ReadinessProfileForScenario parses then compiles the authoritative profile.
// Invalid authored contracts are rejected rather than silently degraded into a
// partial consumer-specific registry.
func ReadinessProfileForScenario(scenarioDir string) (ReadinessProfile, error) {
	report, err := ParseScenario(scenarioDir)
	if err != nil {
		return ReadinessProfile{}, err
	}
	for _, finding := range report.Findings {
		// Portable library contracts are validated by the scenario's contract
		// gate, but they do not affect route readiness for the authored pages.
		// Keeping this projection available lets capture consumers wait for the
		// rendered surface and report the contract debt independently, instead of
		// falling back to an unbounded navigation settle that can race React.
		if finding.Severity == SeverityError && finding.Code != CodeVacuousContract {
			return ReadinessProfile{}, fmt.Errorf("cannot compile readiness profile: %s", finding.Message)
		}
	}
	return BuildReadinessProfile(report.Spec)
}
