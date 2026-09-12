package content

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicUsesPublicSectionReader(t *testing.T) {
	called := ""
	Public(testDependencies(&called)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/spring/sections", nil))
	if called != "public:spring" {
		t.Fatalf("called = %q", called)
	}
}

func TestAdminUsesAllSectionReader(t *testing.T) {
	called := ""
	Admin(testDependencies(&called)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/variants/spring/sections", nil))
	if called != "all:spring" {
		t.Fatalf("called = %q", called)
	}
}

func TestPublicRejectsMissingVariantSlug(t *testing.T) {
	status := 0
	deps := testDependencies(nil)
	deps.Path = func(*http.Request, string) (string, bool) { return "", false }
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	Public(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/public/variants//sections", nil))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d", status)
	}
}

func TestPublicLogsNotFound(t *testing.T) {
	event, status := "", 0
	deps := testDependencies(nil)
	deps.PublicSections = func(string) (any, error) { return nil, errors.New("missing") }
	deps.Log = func(got string, _ map[string]any) { event = got }
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	Public(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/spring/sections", nil))
	if status != http.StatusNotFound || event != "public_sections_get_failed" {
		t.Fatalf("status=%d event=%q", status, event)
	}
}

func testDependencies(called *string) Dependencies {
	set := func(kind string) func(string) (any, error) {
		return func(slug string) (any, error) {
			if called != nil {
				*called = kind + ":" + slug
			}
			return []string{"hero"}, nil
		}
	}
	return Dependencies{
		PublicSections: set("public"), AllSections: set("all"),
		Path:      func(*http.Request, string) (string, bool) { return "spring", true },
		WriteJSON: func(http.ResponseWriter, any) {}, WriteError: func(http.ResponseWriter, int, string, string) {}, Log: func(string, map[string]any) {},
	}
}
