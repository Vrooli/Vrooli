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
				Kind   string   `json:"kind"`
				States []string `json:"states"`
			} `json:"lifecycle"`
		} `json:"regions"`
	} `json:"pages"`
}

type requiredSurface struct {
	id       string
	required bool
	kind     string
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
			if declared == route || matchesRouteTemplate(declared, route) {
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
			out = append(out, requiredSurface{id: region.ID, required: region.Required, kind: region.Lifecycle.Kind, states: states})
		}
	}
	return out
}

func (p *readinessProfile) routes() []string {
	if p == nil {
		return nil
	}
	seen := map[string]bool{}
	var candidates []string
	for _, page := range p.Pages {
		for _, route := range page.RuntimeRoutes {
			route = strings.TrimSpace(route)
			if route != "" && !isUnresolvedRouteTemplate(route) {
				candidates = appendUnique(candidates, seen, route)
			}
		}
		for _, route := range page.Routes {
			route = strings.TrimSpace(route)
			if route != "" && !isUnresolvedRouteTemplate(route) {
				candidates = appendUnique(candidates, seen, route)
			}
		}
	}
	seen = map[string]bool{}
	var collected []string
	for _, page := range p.Pages {
		pageRoutes := page.RuntimeRoutes
		if len(pageRoutes) == 0 {
			pageRoutes = page.Routes
		}
		for _, route := range pageRoutes {
			route = strings.TrimSpace(route)
			if route == "" {
				continue
			}
			if isUnresolvedRouteTemplate(route) {
				expanded, ok := instantiateRouteTemplate(route, candidates)
				if !ok {
					continue
				}
				route = expanded
			}
			collected = appendUnique(collected, seen, route)
		}
	}
	return collected
}

func appendUnique(routes []string, seen map[string]bool, route string) []string {
	if seen[route] {
		return routes
	}
	seen[route] = true
	return append(routes, route)
}

// unresolvedRouteTemplates returns route patterns that cannot be visited
// without a concrete example from a sibling route or state setup.
func (p *readinessProfile) unresolvedRouteTemplates() []string {
	if p == nil {
		return nil
	}
	candidates := make([]string, 0)
	for _, page := range p.Pages {
		for _, route := range append(append([]string(nil), page.RuntimeRoutes...), page.Routes...) {
			route = strings.TrimSpace(route)
			if route != "" && !isUnresolvedRouteTemplate(route) {
				candidates = append(candidates, route)
			}
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, page := range p.Pages {
		for _, route := range page.Routes {
			route = strings.TrimSpace(route)
			if route == "" || !isUnresolvedRouteTemplate(route) || seen[route] {
				continue
			}
			if _, ok := instantiateRouteTemplate(route, candidates); ok {
				continue
			}
			seen[route] = true
			out = append(out, route)
		}
	}
	return out
}

func isUnresolvedRouteTemplate(route string) bool {
	for _, segment := range strings.Split(routePath(route), "/") {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			return true
		}
	}
	return false
}

func instantiateRouteTemplate(template string, candidates []string) (string, bool) {
	templatePath := routePath(template)
	pattern := strings.Split(strings.Trim(templatePath, "/"), "/")
	lastParam := -1
	for i, segment := range pattern {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			lastParam = i
		}
	}
	if lastParam < 0 {
		return template, true
	}
	for _, candidate := range candidates {
		candidatePath := routePath(candidate)
		segments := strings.Split(strings.Trim(candidatePath, "/"), "/")
		if len(segments) <= lastParam {
			continue
		}
		instantiated := append([]string(nil), pattern...)
		valid := true
		for i := 0; i <= lastParam; i++ {
			if strings.HasPrefix(pattern[i], ":") {
				if segments[i] == "" || isUnresolvedRouteTemplate(segments[i]) {
					valid = false
					break
				}
				instantiated[i] = segments[i]
			} else if pattern[i] != segments[i] {
				valid = false
				break
			}
		}
		if valid {
			return "/" + strings.Join(instantiated, "/"), true
		}
	}
	return "", false
}

func routePath(route string) string {
	if i := strings.IndexByte(route, '?'); i >= 0 {
		return route[:i]
	}
	return route
}

func matchesRouteTemplate(template, route string) bool {
	pattern := strings.Split(strings.Trim(routePath(template), "/"), "/")
	actual := strings.Split(strings.Trim(routePath(route), "/"), "/")
	if len(pattern) != len(actual) || !isUnresolvedRouteTemplate(template) {
		return false
	}
	for i, segment := range pattern {
		if strings.HasPrefix(segment, ":") {
			if actual[i] == "" {
				return false
			}
			continue
		}
		if segment != actual[i] {
			return false
		}
	}
	return true
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
		if want.required && !isTerminalSurfaceState(want, got.state) {
			requiredFailure = true
			findings = append(findings, manifestvalidation.Finding{Severity: manifestvalidation.SeverityError, Code: "runtime_required_surface_unsettled", Location: location, Message: fmt.Sprintf("required experience surface %q remained in non-terminal state %q", want.id, got.state), Suggestion: "Wait for ready, empty, partial, or error before treating an async page as functionally settled."})
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

func isTerminalSurfaceState(surface requiredSurface, state string) bool {
	if surface.kind == "static" {
		return state == "static"
	}
	return state == "ready" || state == "empty" || state == "partial" || state == "error"
}
