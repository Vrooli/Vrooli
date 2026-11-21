package monetization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJourneyModuleReportsCapabilitiesAndObservations(t *testing.T) {
	module := JourneyModule{Deps: JourneyDeps{AppKey: "web-console", BundleKey: "business_suite", Operations: map[JourneyOperation]func(context.Context) (string, error){
		JourneySignInSharedSession: func(context.Context) (string, error) { return "session=shared", nil },
	}}}
	server := httptest.NewServer(module.Handler())
	defer server.Close()
	response, err := http.Get(server.URL + "?operation=capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	var body struct {
		AppKey     string   `json:"app_key"`
		Operations []string `json:"operations"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.AppKey != "web-console" || len(body.Operations) != 1 || body.Operations[0] != string(JourneySignInSharedSession) {
		t.Fatalf("body=%+v", body)
	}
}

func TestJourneyModuleUnsupportedUnknownAndRemote(t *testing.T) {
	module := JourneyModule{Deps: JourneyDeps{AppKey: "web-console", BundleKey: "business_suite"}}
	for _, test := range []struct {
		name, query string
		wantStatus  int
		wantBody    string
	}{
		{"unsupported", "signin_shared_session", http.StatusOK, "unsupported:dependency unavailable"},
		{"unknown", "not-a-real-operation", http.StatusBadRequest, "unknown operation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/?operation="+test.query, nil)
			request.RemoteAddr = "127.0.0.1:1234"
			recorder := httptest.NewRecorder()
			module.Handler().ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus || !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
	request := httptest.NewRequest(http.MethodGet, "/?operation=signin_shared_session", nil)
	request.RemoteAddr = "192.0.2.1:1234"
	recorder := httptest.NewRecorder()
	module.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("remote status=%d", recorder.Code)
	}
}
