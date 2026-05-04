package connectx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterServicesMountsPathPrefix(t *testing.T) {
	router := mux.NewRouter()
	RegisterServices(router, ServiceMount{
		Path: "/demo.v1.Notes/",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/demo.v1.Notes/List" {
				t.Fatalf("path = %q", r.URL.Path)
			}
			_, _ = io.WriteString(w, "ok")
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/demo.v1.Notes/List", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "ok" {
		t.Fatalf("body = %q", resp.Body.String())
	}
}

func TestRegisterServicesSkipsInvalidMounts(t *testing.T) {
	router := mux.NewRouter()
	RegisterServices(router,
		ServiceMount{},
		ServiceMount{Path: "/missing-handler"},
	)
	req := httptest.NewRequest(http.MethodPost, "/missing-handler/Call", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d", resp.Code)
	}
}
