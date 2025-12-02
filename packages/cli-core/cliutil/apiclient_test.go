package cliutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIClientAppliesBaseAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("expected bearer token")
		}
		fmt.Fprintf(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewAPIClient(NewHTTPClient(HTTPClientOptions{}), func() APIBaseOptions {
		return APIBaseOptions{DefaultBase: server.URL}
	}, func() string { return "secret" })

	body, err := client.Get("/ping", nil)
	if err != nil {
		t.Fatalf("api get: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}
