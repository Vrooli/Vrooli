package delivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internal "landing-page-business-suite-api/internal/delivery"
)

type appCatalogTestStub struct {
	apps      []internal.App
	deleteErr error
}

func (s appCatalogTestStub) ListApps(string) ([]internal.App, error)           { return s.apps, nil }
func (s appCatalogTestStub) UpsertApp(app internal.App) (*internal.App, error) { return &app, nil }
func (s appCatalogTestStub) DeleteApp(string, string) error                    { return s.deleteErr }

func TestAppHandlersListCreateAndDeleteValidation(t *testing.T) {
	status := 0
	deps := AppDependencies{
		BundleKey: func() string { return "bundle" },
		PathParam: func(*http.Request, string) (string, bool) { return "missing", true },
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, target any) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteData: func(_ http.ResponseWriter, value any) {
			if len(value.(map[string]any)["apps"].([]internal.App)) != 1 {
				t.Fatal("apps")
			}
			status = http.StatusOK
		},
		WriteError: func(_ http.ResponseWriter, got int, _, _ string) { status = got },
	}
	ListApps(deps, appCatalogTestStub{apps: []internal.App{{AppKey: "desktop"}}}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if status != http.StatusOK {
		t.Fatalf("list status=%d", status)
	}
	CreateApp(deps, appCatalogTestStub{}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"app_key":"desktop"}`)))
	if status != http.StatusBadRequest {
		t.Fatalf("create status=%d", status)
	}
	DeleteApp(deps, appCatalogTestStub{deleteErr: internal.ErrAppNotFound}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/", nil))
	if status != http.StatusNotFound {
		t.Fatalf("delete status=%d", status)
	}
}
