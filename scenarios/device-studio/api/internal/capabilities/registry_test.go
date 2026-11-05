package capabilities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

// TestKnownDefinitionsAreComplete guards the declared capability surface: an
// entry missing an id, a dependency slug, or an operator action is not
// actionable by the console or by an agent, and the omission is silent.
func TestKnownDefinitionsAreComplete(t *testing.T) {
	if len(Known) == 0 {
		t.Fatal("Known is empty; the capability surface declares nothing")
	}
	seen := map[string]bool{}
	for _, def := range Known {
		if def.ID == "" {
			t.Error("capability definition has no ID")
			continue
		}
		if seen[def.ID] {
			t.Errorf("capability %q is declared more than once", def.ID)
		}
		seen[def.ID] = true
		if def.Name == "" || def.Description == "" {
			t.Errorf("capability %q must carry a name and a description", def.ID)
		}
		if def.DependencySlug == "" {
			t.Errorf("capability %q must name the dependency it needs", def.ID)
		}
		if def.OperatorCommand == "" || def.ActionLabel == "" {
			t.Errorf("capability %q must offer an operator action; an unavailable capability with no remedy is a dead end", def.ID)
		}
	}
}

func TestOptionalScenarioCheckUsesPinnedLoopbackHealthOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statusCode int
		resolved   string
		want       capabilityregistry.Status
	}{
		{name: "healthy local service", statusCode: http.StatusOK, resolved: "http://127.0.0.1:1234", want: capabilityregistry.StatusAvailable},
		{name: "unhealthy local service", statusCode: http.StatusServiceUnavailable, resolved: "http://127.0.0.1:1234", want: capabilityregistry.StatusUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" || r.Method != http.MethodGet {
					t.Errorf("request = %s %s, want GET /health", r.Method, r.URL.Path)
				}
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()
			checker := LoopbackScenarioChecker{
				Slug:    "ai-gateway",
				Resolve: func(context.Context, string) (string, error) { return server.URL, nil },
			}
			status, message := checker.Check(context.Background())
			if status != tc.want {
				t.Fatalf("status = %q, want %q (%s)", status, tc.want, message)
			}
			if strings.Contains(message, server.URL) || strings.Contains(message, "127.0.0.1") {
				t.Fatalf("status message leaked a discovered address: %q", message)
			}
		})
	}
}

func TestOptionalScenarioCheckRejectsNonLoopbackBeforeRequest(t *testing.T) {
	called := false
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}
	checker := LoopbackScenarioChecker{
		Slug: "workspace-sandbox", Client: client,
		Resolve: func(context.Context, string) (string, error) { return "http://203.0.113.8:1234", nil },
	}
	status, message := checker.Check(context.Background())
	if status != capabilityregistry.StatusUnavailable || called {
		t.Fatalf("status=%q called=%v", status, called)
	}
	if strings.Contains(message, "203.0.113.8") {
		t.Fatalf("message leaked rejected address: %q", message)
	}
}

func TestOptionalScenarioCheckDoesNotFollowRedirect(t *testing.T) {
	redirectTargetHit := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirectTargetHit = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/health", http.StatusFound)
	}))
	defer origin.Close()
	checker := LoopbackScenarioChecker{
		Slug:    "ai-gateway",
		Resolve: func(context.Context, string) (string, error) { return origin.URL, nil },
	}
	status, _ := checker.Check(context.Background())
	if status != capabilityregistry.StatusUnavailable || redirectTargetHit {
		t.Fatalf("status=%q redirect target hit=%v", status, redirectTargetHit)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestNewRegistryCoversEveryDeclaredCapability pins the pairing between a
// declared capability and its checker. A declaration without a checker reports
// nothing, which reads as healthy rather than as unknown.
func TestNewRegistryCoversEveryDeclaredCapability(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	for _, def := range Known {
		if def.DependencyKind != capabilityregistry.DependencyScenario {
			continue
		}
		var checker capabilityregistry.Checker = ScenarioChecker{}
		if def.ID == "ai-gateway" || def.ID == "workspace-sandbox" {
			checker = LoopbackScenarioChecker{Slug: def.DependencySlug}
		}
		if checker == nil {
			t.Fatalf("no checker is configured for %q", def.ID)
		}
	}
}
