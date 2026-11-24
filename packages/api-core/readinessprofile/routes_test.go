package readinessprofile

import "testing"

func TestRouteTemplateMatchesWholeSegmentsAndStripsQueryAndFragment(t *testing.T) {
	cases := []struct {
		template string
		route    string
		want     bool
	}{
		{"/results", "/results?tab=active#top", true},
		{"/assets/:id", "/assets/react-component-library%3AButton?tab=preview", true},
		{"/assets/:id", "/assets/Button/history", false},
		{"/assets/:id", "/assets/", false},
		{"/", "/", true},
		{"/projects/:id", "/projects/:id", true},
	}
	for _, tc := range cases {
		t.Run(tc.template+" -> "+tc.route, func(t *testing.T) {
			if got := RouteTemplateMatches(tc.template, tc.route); got != tc.want {
				t.Fatalf("RouteTemplateMatches(%q, %q) = %t, want %t", tc.template, tc.route, got, tc.want)
			}
		})
	}
}

func TestContainsRoute(t *testing.T) {
	if !ContainsRoute([]string{"/", "/assets/:id"}, "/assets/Button?tab=preview") {
		t.Fatal("parameterized declared route was not matched")
	}
	if ContainsRoute([]string{"/results"}, "/results/other") {
		t.Fatal("undeclared subroute matched")
	}
}

func TestGeometryOptOutCodesRequireEveryMatchingPageAndIgnoreUnknownFloors(t *testing.T) {
	pages := []Page{
		{Routes: []string{"/assets/:id"}, GeometryOptOuts: []string{"tap-target-size", "future-unknown-floor"}},
		{Routes: []string{"/assets/:id"}, GeometryOptOuts: []string{"tap-target-size"}},
		{Routes: []string{"/other"}, GeometryOptOuts: []string{"no-document-horizontal-overflow"}},
	}
	codes := GeometryOptOutCodesForRoute(pages, "/assets/Button")
	if len(codes) != 1 {
		t.Fatalf("opt-out codes = %#v, want only shared tap-target waiver", codes)
	}
	if _, ok := codes["visual_touch_target_too_small"]; !ok {
		t.Fatalf("tap-target waiver missing: %#v", codes)
	}
	if _, ok := codes["visual_viewport_overflow"]; ok {
		t.Fatalf("unmatched page waived geometry: %#v", codes)
	}
	if got := GeometryOptOutCodesForRoute(pages, "/missing"); len(got) != 0 {
		t.Fatalf("unmatched route opt-outs = %#v, want empty", got)
	}
}

func TestGeometryOptOutCodesRespectQueryScopedRuntimeRoutes(t *testing.T) {
	page := Page{Routes: []string{"/results"}, RuntimeRoutes: []string{"/results?tab=preview"}, GeometryOptOuts: []string{"viewport-fill"}}
	if got := GeometryOptOutCodesForRoute([]Page{page}, "/results?tab=active"); len(got) != 0 {
		t.Fatalf("different query state inherited opt-out: %#v", got)
	}
	if got := GeometryOptOutCodesForRoute([]Page{page}, "/results?tab=preview#section"); len(got) != 1 {
		t.Fatalf("matching query state did not inherit opt-out: %#v", got)
	}
}
