package uiruntime

import (
	"encoding/json"
	"fmt"
	"strings"

	"ui-health/internal/services/manifestvalidation"
)

type readinessProfile struct {
	Pages []struct {
		Routes        []string `json:"routes"`
		RuntimeRoutes []string `json:"runtimeRoutes"`
		Regions       []struct {
			ID        string `json:"id"`
			Required  bool   `json:"required"`
			Lifecycle struct {
				States []string `json:"states"`
			} `json:"lifecycle"`
		} `json:"regions"`
	} `json:"pages"`
}

type requiredSurface struct {
	id       string
	required bool
	states   map[string]bool
}

func (p *readinessProfile) requiredSurfacesForRoute(route string) []requiredSurface {
	if p == nil {
		return nil
	}
	var out []requiredSurface
	for _, page := range p.Pages {
		matched := false
		for _, declared := range append(page.Routes, page.RuntimeRoutes...) {
			if declared == route {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for _, region := range page.Regions {
			if strings.TrimSpace(region.ID) == "" {
				continue
			}
			states := make(map[string]bool, len(region.Lifecycle.States))
			for _, state := range region.Lifecycle.States {
				states[state] = true
			}
			out = append(out, requiredSurface{id: region.ID, required: region.Required, states: states})
		}
	}
	return out
}

func (p *readinessProfile) routes() []string {
	if p == nil {
		return nil
	}
	seen := map[string]bool{}
	var collected []string
	for _, page := range p.Pages {
		pageRoutes := page.RuntimeRoutes
		if len(pageRoutes) == 0 {
			pageRoutes = page.Routes
		}
		for _, route := range pageRoutes {
			route = strings.TrimSpace(route)
			if route == "" || seen[route] {
				continue
			}
			seen[route] = true
			collected = append(collected, route)
		}
	}
	return collected
}

func readinessSurfaceFindings(layoutJSON string, expected []requiredSurface, url, viewport string) []manifestvalidation.Finding {
	if len(expected) == 0 || strings.TrimSpace(layoutJSON) == "" {
		return nil
	}
	var layout struct {
		ExperienceSurfaces []struct {
			ID, State string
			Visible   bool
		} `json:"experienceSurfaces"`
	}
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil {
		return nil
	}
	actual := make(map[string]struct {
		state   string
		visible bool
	}, len(layout.ExperienceSurfaces))
	for _, surface := range layout.ExperienceSurfaces {
		actual[surface.ID] = struct {
			state   string
			visible bool
		}{surface.State, surface.Visible}
	}
	var findings []manifestvalidation.Finding
	requiredFailure := false
	optionalFailure := false
	for _, want := range expected {
		got, ok := actual[want.id]
		location := url + " [" + viewport + "]"
		if !ok {
			if want.required {
				requiredFailure = true
				findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityError, Code: "runtime_required_surface_missing", Location: location, Message: fmt.Sprintf("required experience surface %q was not rendered", want.id), Suggestion: "Render the declared surface with data-experience-surface and data-experience-state."})
			}
			continue
		}
		if !got.visible && want.required {
			requiredFailure = true
			findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityError, Code: "runtime_required_surface_hidden", Location: location, Message: fmt.Sprintf("required experience surface %q is hidden", want.id)})
		}
		if len(want.states) > 0 && !want.states[got.state] {
			if want.required {
				requiredFailure = true
				findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityError, Code: "runtime_required_surface_invalid_state", Location: location, Message: fmt.Sprintf("required experience surface %q reported undeclared state %q", want.id, got.state)})
			}
		}
		if got.state == "error" {
			if want.required {
				requiredFailure = true
			} else {
				optionalFailure = true
			}
		}
	}
	location := url + " [" + viewport + "]"
	if requiredFailure {
		findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityError, Code: "runtime_page_required_surface_error", Location: location, Message: "page aggregation is error because a required experience surface failed or is unavailable", Suggestion: "Restore the required surface or declare it optional only when the primary task remains usable."})
	} else if optionalFailure {
		findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityInfo, Code: "runtime_page_partial", Location: location, Message: "page aggregation is partial because an optional experience surface reported error", Suggestion: "Verify the primary task remains usable and expose partial-state guidance to the user."})
	}
	return findings
}
