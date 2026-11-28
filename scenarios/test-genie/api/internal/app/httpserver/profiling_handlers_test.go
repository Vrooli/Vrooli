package httpserver

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdmissionProfileIsDisabledUnlessExplicitlyEnabled(t *testing.T) {
	t.Setenv("TEST_GENIE_PROFILING_ENABLED", "")
	server := &Server{logger: log.New(io.Discard, "", 0)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admission/profile?kind=heap", nil)
	res := httptest.NewRecorder()
	server.handleAdmissionProfile(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
}

func TestAdmissionProfileRequiresToken(t *testing.T) {
	t.Setenv("TEST_GENIE_PROFILING_ENABLED", "1")
	t.Setenv("TEST_GENIE_PROFILING_TOKEN", "secret")
	server := &Server{logger: log.New(io.Discard, "", 0)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admission/profile?kind=heap", nil)
	res := httptest.NewRecorder()
	server.handleAdmissionProfile(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
}
