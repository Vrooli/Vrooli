package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestMonetizationJourneyReportsUnwiredOperationsAsUnsupported(t *testing.T) {
	server := &Server{router: mux.NewRouter()}
	registerMonetizationJourneyRoute(server)

	for _, operation := range []string{
		"signin_shared_session",
		"second_app_resolves",
		"tampered_class_a",
		"class_b_local",
		"offline_class_b",
		"offline_gate_degrades",
		"outbox_drains_once",
		"expired_lease_falls_back",
	} {
		t.Run(operation, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/internal/monetization/journey?operation="+operation, nil)
			request.RemoteAddr = "127.0.0.1:1234"
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Observed string `json:"observed"`
				AppKey   string `json:"app_key"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode journey response: %v", err)
			}
			if !strings.HasPrefix(response.Observed, "unsupported:") {
				t.Fatalf("observation=%q; unimplemented journey must stay unsupported", response.Observed)
			}
			if response.AppKey != "landing-page-business-suite" {
				t.Fatalf("app_key=%q", response.AppKey)
			}
		})
	}
}
