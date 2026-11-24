package readinessprofile

import "strings"

// RoutePath removes query and fragment data before comparing declared routes.
func RoutePath(route string) string {
	if before, _, found := strings.Cut(route, "#"); found {
		route = before
	}
	if before, _, found := strings.Cut(route, "?"); found {
		route = before
	}
	return route
}

// RouteTemplateMatches matches whole path segments, with colon-prefixed
// segments accepting one non-empty path segment.
func RouteTemplateMatches(template, route string) bool {
	template = RoutePath(template)
	route = RoutePath(route)
	if template == route {
		return true
	}
	pattern := strings.Split(strings.Trim(template, "/"), "/")
	actual := strings.Split(strings.Trim(route, "/"), "/")
	if len(pattern) != len(actual) {
		return false
	}
	for index, segment := range pattern {
		if strings.HasPrefix(segment, ":") {
			if len(segment) == 1 || actual[index] == "" {
				return false
			}
			continue
		}
		if segment != actual[index] {
			return false
		}
	}
	return true
}

// ContainsRoute reports whether any declared route template matches a route.
func ContainsRoute(routes []string, route string) bool {
	for _, candidate := range routes {
		if RouteTemplateMatches(candidate, route) {
			return true
		}
	}
	return false
}

var geometryFloorRules = map[string]string{
	"no-document-horizontal-overflow": "visual_viewport_overflow",
	"viewport-fill":                   "visual_surface_does_not_fill_viewport",
	"chrome-pinned":                   "visual_chrome_outside_viewport",
	"safe-area-tap-targets":           "visual_unsafe_edge_tap_zone",
	"single-line-chrome":              "visual_multiline_chrome_label",
	"tap-target-size":                 "visual_touch_target_too_small",
}

// GeometryRuleForFloor maps an authored page floor to its canonical geometry
// finding code. Unknown floor names stay outside the shared geometry pack.
func GeometryRuleForFloor(floor string) (string, bool) {
	rule, ok := geometryFloorRules[strings.TrimSpace(floor)]
	return rule, ok
}

// GeometryOptOutCodesForRoute returns only floor rules waived by every page
// composed at route. A single page's waiver cannot hide another page's defect.
func GeometryOptOutCodesForRoute(pages []Page, route string) map[string]struct{} {
	var waived map[string]struct{}
	matched := false
	for _, page := range pages {
		if !PageMatchesRoute(page, route) {
			continue
		}
		pageWaived := make(map[string]struct{}, len(page.GeometryOptOuts))
		for _, floor := range page.GeometryOptOuts {
			if code, ok := GeometryRuleForFloor(floor); ok {
				pageWaived[code] = struct{}{}
			}
		}
		if !matched {
			waived = pageWaived
			matched = true
			continue
		}
		for code := range waived {
			if _, ok := pageWaived[code]; !ok {
				delete(waived, code)
			}
		}
	}
	if !matched {
		return nil
	}
	return waived
}

// PageMatchesRoute applies query-scoped runtime routes before path-template
// matching, so a page can declare surfaces that exist only in a query state.
func PageMatchesRoute(page Page, route string) bool {
	if queryScoped(page.RuntimeRoutes) {
		requested := strings.SplitN(route, "#", 2)[0]
		for _, candidate := range page.RuntimeRoutes {
			if candidate == requested {
				return true
			}
		}
		return false
	}
	return ContainsRoute(append(append([]string(nil), page.Routes...), page.RuntimeRoutes...), route)
}

func queryScoped(runtimeRoutes []string) bool {
	if len(runtimeRoutes) == 0 {
		return false
	}
	for _, candidate := range runtimeRoutes {
		if !strings.Contains(candidate, "?") {
			return false
		}
	}
	return true
}
