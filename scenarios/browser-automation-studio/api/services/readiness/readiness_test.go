package readiness

import (
	"encoding/json"
	"testing"
)

// TestPageMatchesRoute_TabScopedRegions covers the case that made the injected
// wait cost more than it saved: a page whose async content is reachable only
// with a query string.
func TestPageMatchesRoute_TabScopedRegions(t *testing.T) {
	// BAS's dashboard: route "/", but every declared state pins ?tab=projects.
	dashRoutes := []string{"/"}
	dashRuntime := []string{"/?tab=projects"}

	if PageMatchesRoute(dashRoutes, dashRuntime, "/") {
		t.Fatal("bare / must not resolve a region that only mounts under ?tab=projects")
	}
	if !PageMatchesRoute(dashRoutes, dashRuntime, "/?tab=projects") {
		t.Fatal("the declared runtime route must still match")
	}

	// A page whose runtime routes carry no query keeps path-only matching, so
	// scenarios that are not tab-scoped are unaffected.
	if !PageMatchesRoute([]string{"/projects/:projectId"}, []string{"/projects/:projectId"}, "/projects/abc") {
		t.Fatal("a query-free page must keep template path matching")
	}
	if !PageMatchesRoute([]string{"/settings"}, nil, "/settings?section=general") {
		t.Fatal("a page with no runtime routes must keep path-only matching")
	}
}

func TestResolveProfileCombinesMatchingPages(t *testing.T) {
	var profile profileDocument
	if err := json.Unmarshal([]byte(`{
		"pages": [
			{"routes":["/"],"regions":[]},
			{"routes":["/"],"regions":[{
				"id":"workspace-shell",
				"required":true,
				"binding":{"selector":"[data-testid=\"workspace-top-edge\"]"},
				"lifecycle":{"kind":"static","states":["static"]}
			}]}
		]
	}`), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	resolution := resolveProfile(profile, "/", "experience-readiness-profile/v1")
	if !resolution.RouteMatched {
		t.Fatal("matching pages must mark the route as matched")
	}
	if len(resolution.Waits) != 1 {
		t.Fatalf("wait count = %d, want one wait from the later matching page", len(resolution.Waits))
	}
	if len(resolution.RequiredSurfaceIDs) != 1 || resolution.RequiredSurfaceIDs[0] != "workspace-shell" {
		t.Fatalf("required surfaces = %#v, want workspace-shell", resolution.RequiredSurfaceIDs)
	}
}
