package variant

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicGetPreservesPublicReadContract(t *testing.T) {
	logged := ""
	handler := PublicGet(testDependencies(func(string) (any, error) { return map[string]string{"slug": "spring"}, nil }, func(event string, _ map[string]any) { logged = event }))

	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/spring", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["slug"] != "spring" || logged != "" {
		t.Fatalf("body = %#v, log = %q", body, logged)
	}
}

func TestAdminGetRejectsSelectionPseudoSlug(t *testing.T) {
	handler := AdminGet(testDependencies(func(string) (any, error) { t.Fatal("Get must not be called for select"); return nil, nil }, nil))
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/variants/select", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPublicGetLogsAndReturnsNotFound(t *testing.T) {
	logged := ""
	handler := PublicGet(testDependencies(func(string) (any, error) { return nil, errors.New("missing") }, func(event string, _ map[string]any) { logged = event }))
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/missing", nil))
	if w.Code != http.StatusNotFound || logged != "public_variant_fetch_failed" {
		t.Fatalf("status = %d, log = %q", w.Code, logged)
	}
}

func TestListWrapsVariants(t *testing.T) {
	handler := List(Dependencies{List: func() any { return []string{"one"} }, WriteJSON: func(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) }, WriteError: func(tw http.ResponseWriter, status int, message, kind string) { tw.WriteHeader(status) }})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil))
	if w.Code != http.StatusOK || w.Body.String() != "{\"variants\":[\"one\"]}\n" {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

func testDependencies(get func(string) (any, error), log func(string, map[string]any)) Dependencies {
	return Dependencies{
		Get: get,
		Slug: func(r *http.Request) string {
			return r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		},
		WriteJSON:  func(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) },
		WriteError: func(w http.ResponseWriter, status int, message, kind string) { w.WriteHeader(status) },
		Log:        log,
	}
}
