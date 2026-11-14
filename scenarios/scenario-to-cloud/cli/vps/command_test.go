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
	// Apply is a thin alias: compile the plan, then submit its digest. A
	// digest supplied by the caller skips the compile.
	var paths []string
	var applyBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/vps/deploy/plan":
			_, _ = w.Write([]byte(`{"plan":{"commands":[]},"plan_digest":"sha256:abc","timestamp":"2026-01-01T00:00:00Z"}`))
		case "/api/v1/vps/deploy/apply":
			_ = json.NewDecoder(r.Body).Decode(&applyBody)
			_, _ = w.Write([]byte(`{"result":{"ok":true,"steps":[]},"timestamp":"2026-01-01T00:00:00Z"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := vpsTestClient(server.URL)
	if _, _, err := client.DeployApply(map[string]interface{}{"version": "1.0.0"}, ""); err != nil {
		t.Fatalf("DeployApply returned error: %v", err)
	}
	if len(paths) != 2 || paths[0] != "POST /api/v1/vps/deploy/plan" || paths[1] != "POST /api/v1/vps/deploy/apply" {
		t.Fatalf("expected plan then apply, got %v", paths)
	}
	if applyBody["plan_digest"] != "sha256:abc" {
		t.Fatalf("expected compiled plan_digest in apply body, got %#v", applyBody["plan_digest"])
	}

	paths, applyBody = nil, nil
	if _, _, err := client.DeployApply(map[string]interface{}{"version": "1.0.0"}, "sha256:reviewed"); err != nil {
		t.Fatalf("DeployApply with digest returned error: %v", err)
	}
	if len(paths) != 1 || applyBody["plan_digest"] != "sha256:reviewed" {
		t.Fatalf("expected a single apply with the reviewed digest, got paths=%v body=%#v", paths, applyBody)
	}
}

func TestExtractPlanDigest(t *testing.T) {
	digest, positional := extractPlanDigest([]string{"m.json", "--plan-digest", "sha256:x", "b.tar.gz"})
	if digest != "sha256:x" || len(positional) != 2 {
		t.Fatalf("got digest=%q positional=%v", digest, positional)
	}
	digest, positional = extractPlanDigest([]string{"--plan-digest=sha256:y", "m.json"})
	if digest != "sha256:y" || len(positional) != 1 {
		t.Fatalf("got digest=%q positional=%v", digest, positional)
	}
}
