package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTunnelsReadsEveryPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page != "1" && page != "2" {
			t.Errorf("unexpected page %q", page)
			return
		}
		fmt.Fprintf(w, `{"success":true,"result":[{"id":"tunnel-%s"}],"result_info":{"page":%s,"per_page":1,"count":1,"total_count":2}}`, page, page)
	}))
	defer server.Close()
	api := NewCloudflareBootstrapAPI(server.Client(), server.URL)
	tunnels, err := api.ListTunnels(context.Background(), "test-token", "account")
	if err != nil || len(tunnels) != 2 || tunnels[1].ID != "tunnel-2" {
		t.Fatalf("tunnels = %+v, error = %v", tunnels, err)
	}
}

func TestRotateConnectorSecretUsesFreshSecretAndScopedPatch(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/accounts/account/cfd_tunnel/tunnel" {
			t.Error("unexpected rotation target")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error("invalid rotation body")
			return
		}
		secret, err := base64.StdEncoding.DecodeString(body["tunnel_secret"])
		if err != nil || len(secret) != 32 || len(body) != 1 {
			t.Error("rotation body did not contain only a 32-byte secret")
		}
		seen = append(seen, body["tunnel_secret"])
		fmt.Fprint(w, `{"success":true,"result":{"token":"not-for-output"}}`)
	}))
	defer server.Close()
	api := NewCloudflareBootstrapAPI(server.Client(), server.URL)
	for i := 0; i < 2; i++ {
		secret := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%032d", i)))
		if err := api.RotateConnectorSecret(context.Background(), "test-token", "account", "tunnel", secret); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 2 || seen[0] == seen[1] {
		t.Fatal("rotation did not generate fresh secrets")
	}
}
