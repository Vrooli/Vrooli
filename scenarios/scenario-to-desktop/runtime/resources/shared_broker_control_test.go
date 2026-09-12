package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBrokerControlGrantClientUsesScopedLoopbackProtocol(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer app-control-token" {
			t.Errorf("Authorization = %q", got)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, sentScope := body["scope"]; sentScope {
			t.Error("desktop request must not choose a broker scope")
		}
		requests = append(requests, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/acquire":
			_, _ = w.Write([]byte(`{"ID":"lease-1"}`))
		case "/v1/authorize-use":
			_, _ = w.Write([]byte(`{"Resource":"vault","Provider":"managed-shared","Endpoint":"http://127.0.0.1:8200"}`))
		case "/v1/credentials":
			_, _ = w.Write([]byte(`{"LeaseID":"lease-1","Resource":"vault","ExpiresAt":"2030-01-02T03:04:05Z","Credential":"app-vault-token"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewBrokerControlGrantClient(BrokerControlCredential{Endpoint: server.URL, Scope: "app:desktop", Token: "app-control-token"})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := client.GrantSharedService(context.Background(), "vault", time.Minute)
	if err != nil {
		t.Fatalf("GrantSharedService: %v", err)
	}
	if grant.Endpoint != "http://127.0.0.1:8200" || grant.Credential != "app-vault-token" {
		t.Fatalf("grant = %#v", grant)
	}
	if got := strings.Join(requests, ","); got != "/v1/acquire,/v1/authorize-use,/v1/credentials" {
		t.Fatalf("requests = %q", got)
	}
}

func TestBrokerControlGrantClientRejectsNonLoopbackControlOrServiceEndpoint(t *testing.T) {
	if _, err := NewBrokerControlGrantClient(BrokerControlCredential{Endpoint: "http://broker.example:8080", Scope: "app:desktop", Token: "token"}); err == nil {
		t.Fatal("non-loopback broker endpoint was accepted")
	}
	if isLoopbackManagedServiceEndpoint("https://vault.example/v1") {
		t.Fatal("non-loopback managed endpoint was accepted")
	}
}
