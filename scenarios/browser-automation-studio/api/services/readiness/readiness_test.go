package readiness

import (
	"context"
	"encoding/json"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestProfileReaderPreservesScenarioAndPathAtExperienceManagerBoundary(t *testing.T) {
	var gotRequest contractv1.GetReadinessProfileRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.experience_manager.v1.contract.ContractService/GetReadinessProfile" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		if err := proto.Unmarshal(body, &gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/proto")
		payload, err := proto.Marshal(&contractv1.GetReadinessProfileResponse{
			Scenario: "image-tools", ProfileJson: `{"pages":[]}`, ProfileVersion: "profile-v1",
		})
		if err != nil {
			t.Errorf("marshal response: %v", err)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	reader := &experienceResolver{
		client:  server.Client(),
		resolve: func(context.Context) (string, error) { return server.URL, nil },
	}
	profile, err := reader.GetReadinessProfile(context.Background(), "image-tools", "/editor?tab=assets")
	if err != nil {
		t.Fatalf("GetReadinessProfile: %v", err)
	}
	if profile.Scenario != "image-tools" || profile.ProfileVersion != "profile-v1" || profile.ProfileJSON != `{"pages":[]}` {
		t.Fatalf("profile = %+v", profile)
	}
	if gotRequest.GetScenario() != "image-tools" || gotRequest.GetPath() != "/editor?tab=assets" {
		t.Fatalf("request = %+v, want scenario and path", gotRequest)
	}
}

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

func TestResolveProfileCoalescesDuplicateGeometryWaits(t *testing.T) {
	var profile profileDocument
	if err := json.Unmarshal([]byte(`{
		"pages": [
			{"routes":["/"],"regions":[
				{"id":"shell-primary","required":true,"binding":{"selector":"[data-testid=\"shell\"]"},"lifecycle":{"kind":"static"}},
				{"id":"shell-alias","required":true,"binding":{"selector":"[data-testid=\"shell\"]"},"lifecycle":{"kind":"static"}},
				{"id":"shell-primary","required":true,"binding":{"selector":"[data-testid=\"shell\"]"},"lifecycle":{"kind":"static"}}
			]}
		]
	}`), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	resolution := resolveProfile(profile, "/", "v1")
	if got := len(resolution.Waits); got != 1 {
		t.Fatalf("wait count = %d, want one wait for the shared geometry selector", got)
	}
	if got, want := resolution.RequiredSurfaceIDs, []string{"shell-primary", "shell-alias"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("required surfaces = %#v, want %#v", got, want)
	}
}

func TestResolveProfileEscapesGeneratedTestIDSelector(t *testing.T) {
	var profile profileDocument
	if err := json.Unmarshal([]byte(`{
		"pages": [{"routes":["/"],"regions":[{
			"id":"quoted-id","required":true,"binding":{"testid":"panel\"[data-state=\"ready\"]\\x"},"lifecycle":{"kind":"static"}
		}]}]
	}`), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	resolution := resolveProfile(profile, "/", "v1")
	if got, want := resolution.Waits[0].GetSelector(), `[data-testid="panel\"[data-state=\"ready\"]\\x"]`; got != want {
		t.Fatalf("selector = %q, want escaped selector %q", got, want)
	}
}
