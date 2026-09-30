package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestCredentialDoctorRelaysMetadataOnly(t *testing.T) {
	prior := credentialDoctorRelay
	credentialDoctorRelay = func(context.Context) ([]byte, error) {
		return []byte(`{"backend":"libsecret","condition":"available"}`), nil
	}
	t.Cleanup(func() { credentialDoctorRelay = prior })

	router := mux.NewRouter()
	NewCredentialHandlers(nil, nil, nil).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/doctor", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"backend":"libsecret"`) {
		t.Fatalf("status/body = %d/%s", rec.Code, rec.Body.String())
	}
}

func TestCredentialKeyringRepairRequiresConfirmation(t *testing.T) {
	router := mux.NewRouter()
	NewCredentialHandlers(nil, nil, nil).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/keyring/repair", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCredentialDoctorHidesRelayFailureDetails(t *testing.T) {
	prior := credentialDoctorRelay
	credentialDoctorRelay = func(context.Context) ([]byte, error) {
		return nil, errors.New("private host detail")
	}
	t.Cleanup(func() { credentialDoctorRelay = prior })

	router := mux.NewRouter()
	NewCredentialHandlers(nil, nil, nil).RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/doctor", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), "private host detail") {
		t.Fatalf("status/body = %d/%s", rec.Code, rec.Body.String())
	}
}
