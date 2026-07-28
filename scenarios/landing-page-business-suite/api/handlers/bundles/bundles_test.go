package bundles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogWritesTheCatalogEnvelope(t *testing.T) {
	handler := Catalog(testDependencies())
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/bundles", nil))
	if w.Code != http.StatusOK || w.Body.String() != "{\"bundles\":[\"starter\"]}\n" {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

func TestUpdatePriceRejectsAKeyOutsideTheActiveBundle(t *testing.T) {
	called := false
	deps := testDependencies()
	deps.Update = func(_ context.Context, _, _ string, _ UpdatePriceRequest) (any, error) {
		called = true
		return nil, nil
	}
	handler := UpdatePrice(deps)
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodPatch, "/api/v1/admin/bundles/other/prices/price", strings.NewReader(`{}`)))
	if w.Code != http.StatusBadRequest || called {
		t.Fatalf("status = %d, update called = %t", w.Code, called)
	}
}

func TestUpdatePriceClassifiesProviderFailures(t *testing.T) {
	deps := testDependencies()
	deps.Update = func(_ context.Context, _, _ string, _ UpdatePriceRequest) (any, error) {
		return nil, errors.New("provider unavailable")
	}
	deps.ClassifyError = func(error) (int, string, string, bool) {
		return http.StatusBadGateway, "server_error", "Stripe unavailable", true
	}
	handler := UpdatePrice(deps)
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodPatch, "/api/v1/admin/bundles/starter/prices/price", strings.NewReader(`{}`)))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "Stripe unavailable") {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

func testDependencies() Dependencies {
	return Dependencies{
		Catalog:   func(context.Context) (any, error) { return map[string]any{"bundles": []string{"starter"}}, nil },
		ActiveKey: func() string { return "starter" },
		Update:    func(_ context.Context, _, _ string, request UpdatePriceRequest) (any, error) { return request, nil },
		Path: func(r *http.Request, key string) (string, bool) {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			for index, part := range parts {
				if key == "bundle_key" && part == "bundles" && index+1 < len(parts) {
					return parts[index+1], true
				}
				if key == "price_id" && part == "prices" && index+1 < len(parts) {
					return parts[index+1], true
				}
			}
			return "", false
		},
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, target any) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(message))
		},
		WriteSuccess:  func(w http.ResponseWriter, response any) { _ = json.NewEncoder(w).Encode(response) },
		ClassifyError: func(error) (int, string, string, bool) { return 0, "", "", false },
	}
}
