package capabilityregistry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type httpChecker struct{}

func (httpChecker) Check(context.Context) (Status, string) {
	return StatusAvailable, "ready"
}

func TestDescribeHandlerPreservesSharedResponseContract(t *testing.T) {
	registry := New([]Def{{
		ID: "demo", Name: "Demo", Description: "Example capability",
		DependencyKind: DependencyScenario, DependencySlug: "demo",
	}}, map[string]Checker{"demo": httpChecker{}}, time.Minute)
	recorder := httptest.NewRecorder()
	DescribeHandler(registry).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}
	if !strings.Contains(recorder.Body.String(), `"definitions":[{"id":"demo"`) || !strings.Contains(recorder.Body.String(), `"status":"available"`) {
		t.Fatalf("unexpected capability response: %s", recorder.Body.String())
	}
}

func TestDescribeHandlerRejectsNilRegistry(t *testing.T) {
	recorder := httptest.NewRecorder()
	DescribeHandler(nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), "capabilities registry is not configured") {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestDescribeHandlerReturnsRegistryErrorsAsUnavailable(t *testing.T) {
	registry := New([]Def{{ID: "demo"}}, nil, time.Minute)
	recorder := httptest.NewRecorder()
	DescribeHandler(registry).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities/describe", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), `definition "demo" has no description`) {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}
