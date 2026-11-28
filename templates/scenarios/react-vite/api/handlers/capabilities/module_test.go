package capabilities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type checkerFunc func(context.Context) (capabilityregistry.Status, string)

func (f checkerFunc) Check(ctx context.Context) (capabilityregistry.Status, string) {
	return f(ctx)
}

// TestModuleServiceUnavailableWithoutRegistry pins the degraded path: a nil
// registry must answer 503 rather than panicking the whole API. This is the
// branch that runs when an optional dependency was never wired.
func TestModuleServiceUnavailableWithoutRegistry(t *testing.T) {
	router := mux.NewRouter()
	Module(nil).Mount(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestModuleServiceUnavailableWhenRegistryCannotDescribe(t *testing.T) {
	registry := capabilityregistry.New([]capabilityregistry.Def{{
		ID:             "broken",
		Name:           "Broken",
		Description:    "A deliberately incomplete registry fixture.",
		DependencyKind: capabilityregistry.DependencyScenario,
		DependencySlug: "broken",
	}}, nil, time.Second)
	router := mux.NewRouter()
	Module(registry).Mount(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if rec.Body.String() != "definition \"broken\" has no reachability checker\n" {
		t.Fatalf("body = %q, want registry validation error", rec.Body.String())
	}
}

func TestModuleDescribesConfiguredRegistry(t *testing.T) {
	definitions := []capabilityregistry.Def{{
		ID:             "audio-tools",
		Name:           "Audio Tools",
		Description:    "Optional audio capability used by this test.",
		DependencyKind: capabilityregistry.DependencyScenario,
		DependencySlug: "audio-tools",
	}}
	registry := capabilityregistry.New(definitions, map[string]capabilityregistry.Checker{
		"audio-tools": checkerFunc(func(ctx context.Context) (capabilityregistry.Status, string) {
			_ = ctx
			return capabilityregistry.StatusUnavailable, "fixture is not running"
		}),
	}, time.Second)
	router := mux.NewRouter()
	Module(registry).Mount(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q, want application/json", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"id":"audio-tools"`) || !strings.Contains(body, `"status":"unavailable"`) {
		t.Fatalf("body = %q, want declared capability and unavailable state", body)
	}
}

// TestModuleDeclaresItsEndpoints keeps the module's declared endpoint surface
// non-empty, since cli-health and ui-health discover commands and widgets from
// it and an empty declaration is indistinguishable from an absent feature.
func TestModuleDeclaresItsEndpoints(t *testing.T) {
	m := Module(nil)
	if m.Name != "capabilities" {
		t.Errorf("module name = %q, want %q", m.Name, "capabilities")
	}
	if len(m.Endpoints) == 0 {
		t.Error("module declares no endpoints; discovery surfaces will not see it")
	}
}
