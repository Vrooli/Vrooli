package vps

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func vpsTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func TestRunSetupRejectsUnknownSubcommand(t *testing.T) {
	err := runSetup(nil, []string{"bogus"})
	if err == nil {
		t.Fatal("expected unknown setup subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown setup subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDeployRejectsUnknownSubcommand(t *testing.T) {
	err := runDeploy(nil, []string{"bogus"})
	if err == nil {
		t.Fatal("expected unknown deploy subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown deploy subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientSetupPlanSendsManifestAndBundlePath(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/vps/setup/plan" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plan":{"remote_tar_path":"/tmp/bundle.tar.gz","commands":[]},"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := vpsTestClient(server.URL)
	manifest := map[string]interface{}{"version": "1.0.0"}
	if _, _, err := client.SetupPlan(manifest, "/tmp/bundle.tar.gz"); err != nil {
		t.Fatalf("SetupPlan returned error: %v", err)
	}
	if got["bundle_path"] != "/tmp/bundle.tar.gz" {
		t.Fatalf("expected bundle_path in request, got %#v", got["bundle_path"])
	}
	if _, ok := got["manifest"]; !ok {
		t.Fatal("expected manifest field in request")
	}
}

func TestClientDeployApplyUsesExpectedEndpoint(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/vps/deploy/apply" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"ok":true,"steps":[]},"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := vpsTestClient(server.URL)
	if _, _, err := client.DeployApply(map[string]interface{}{"version": "1.0.0"}); err != nil {
		t.Fatalf("DeployApply returned error: %v", err)
	}
	if !called {
		t.Fatal("expected deploy apply endpoint to be called")
	}
}
