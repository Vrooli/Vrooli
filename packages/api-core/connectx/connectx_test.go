package connectx

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
)

func TestWrapErrorUsesDomainCodeAndPreservesCause(t *testing.T) {
	cause := errors.New("invalid request")
	wrapped := WrapError(cause, func(err error) connect.Code {
		if err != cause {
			t.Fatalf("mapper received %v, want original cause", err)
		}
		return connect.CodeInvalidArgument
	})
	if wrapped == nil || connect.CodeOf(wrapped) != connect.CodeInvalidArgument || !errors.Is(wrapped, cause) {
		t.Fatalf("wrapped error = %v, code = %v", wrapped, connect.CodeOf(wrapped))
	}
}

func TestWrapErrorHandlesNilAndMissingMapper(t *testing.T) {
	if WrapError(nil, nil) != nil {
		t.Fatal("nil error must remain nil")
	}
	cause := errors.New("missing mapper")
	wrapped := WrapError(cause, nil)
	if wrapped == nil || connect.CodeOf(wrapped) != connect.CodeInternal || !errors.Is(wrapped, cause) {
		t.Fatalf("wrapped error = %v, code = %v", wrapped, connect.CodeOf(wrapped))
	}
}

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

func TestRegisterServicesNormalizesBarePathAndNilRouter(t *testing.T) {
	RegisterServices(nil, ServiceMount{Path: "ignored", Handler: http.NotFoundHandler()})

	router := mux.NewRouter()
	RegisterServices(router, ServiceMount{
		Path: "demo.v1.Bare",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "bare")
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/demo.v1.Bare/Call", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "bare" {
		t.Fatalf("body = %q", resp.Body.String())
	}
}
