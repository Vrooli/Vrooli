package secrets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenario-to-cloud/cli/deployment"
	"scenario-to-cloud/cli/internal/transport"

	"github.com/vrooli/cli-core/cliutil"
)

func secretsTestClient(baseURL string) *Client {
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return NewClient(apiClient)
}

func secretsTestDeploymentClient(baseURL string) *deployment.Client {
	return deployment.NewClient(transport.ForBaseURL(baseURL))
}

func TestParseTargets(t *testing.T) {
	targets, err := parseTargets("scenario,deployment")
	if err != nil {
		t.Fatalf("parseTargets error: %v", err)
	}
	if !targets["scenario"] || !targets["deployment"] || targets["workspace"] {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestParseTargetsRejectsUnknown(t *testing.T) {
	if _, err := parseTargets("scenario,unknown"); err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestGenerateSecretFromSpecHex(t *testing.T) {
	value, err := generateSecretFromSpec("hex:64")
	if err != nil {
		t.Fatalf("generateSecretFromSpec error: %v", err)
	}
	if len(value) != 64 {
		t.Fatalf("expected length 64, got %d", len(value))
	}
}

func TestGenerateSecretFromSpecFormats(t *testing.T) {
	for _, spec := range []string{"hex:17", "base64:17", "alnum:17"} {
		t.Run(spec, func(t *testing.T) {
			value, err := generateSecretFromSpec(spec)
			if err != nil {
				t.Fatalf("generateSecretFromSpec(%q): %v", spec, err)
			}
			if len(value) != 17 {
				t.Fatalf("generated length = %d, want 17", len(value))
			}
		})
	}
	value, err := generateSecretFromSpec("uuid")
	if err != nil {
		t.Fatalf("generateSecretFromSpec(uuid): %v", err)
	}
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		t.Fatalf("invalid UUID format %q", value)
	}
}

func TestGenerateSecretFromSpecRejectsUnsupportedValues(t *testing.T) {
	for _, spec := range []string{"hex", "unknown:2", "hex:0", "hex:4097"} {
		if _, err := generateSecretFromSpec(spec); err == nil {
			t.Errorf("generateSecretFromSpec(%q) accepted unsupported input", spec)
		}
	}
}

func TestClientLegacyGetParsesCurrentShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/secrets/landing-page-business-suite" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secrets":{"bundle_secrets":[],"summary":{"total":0}}}`))
	}))
	defer server.Close()

	client := secretsTestClient(server.URL)
	_, resp, err := client.LegacyGet("landing-page-business-suite", true)
	if err != nil {
		t.Fatalf("LegacyGet returned error: %v", err)
	}
	if resp.Secrets == nil {
		t.Fatal("expected non-nil secrets payload")
	}
}

func TestLooksLikeHTTPStatus(t *testing.T) {
	err := &fakeErr{msg: "api error (409): conflict"}
	if !looksLikeHTTPStatus(err, 409) {
		t.Fatal("expected status match")
	}
}

func TestRunSetUsageRequiresKey(t *testing.T) {
	if err := runSet(nil, nil, []string{"--value", "x"}); err == nil {
		t.Fatal("expected usage error when key is missing")
	}
}

func TestParseSetRequestValidatesValueModes(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "missing value", args: []string{"API_KEY"}},
		{name: "value and generate", args: []string{"API_KEY", "--value", "x", "--generate", "uuid"}},
		{name: "stdin and value", args: []string{"API_KEY", "--value-stdin", "--value", "x"}},
		{name: "stdin and generate", args: []string{"API_KEY", "--value-stdin", "--generate", "uuid"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseSetRequest(test.args); err == nil {
				t.Fatal("expected invalid value mode to be rejected")
			}
		})
	}
	request, err := parseSetRequest([]string{"API_KEY", "--value", "x", "--targets", "workspace"})
	if err != nil || request.key != "API_KEY" || request.value != "x" || request.targetsRaw != "workspace" {
		t.Fatalf("valid request parsed incorrectly: request=%+v err=%v", request, err)
	}
}

func TestRunSetWritesWorkspaceSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/local-secrets/workspace/API_KEY" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body LocalSecretSetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Value != "value-123" {
			t.Fatalf("request value = %q, want value-123", body.Value)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"key":"API_KEY","scope":"workspace"}`))
	}))
	defer server.Close()
	if err := runSet(secretsTestClient(server.URL), nil, []string{"API_KEY", "--value", "value-123", "--targets", "workspace"}); err != nil {
		t.Fatalf("runSet returned error: %v", err)
	}
}

func TestRunGetReadsWorkspaceSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/local-secrets/workspace/API_KEY" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"API_KEY","scope":"workspace","value":"value-123"}`))
	}))
	defer server.Close()
	if err := runGet(secretsTestClient(server.URL), nil, []string{"API_KEY", "--targets", "workspace", "--reveal"}); err != nil {
		t.Fatalf("runGet returned error: %v", err)
	}
}

func TestRunDeleteRemovesWorkspaceSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/local-secrets/workspace/API_KEY" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"key":"API_KEY","action":"deleted"}`))
	}))
	defer server.Close()
	if err := runDelete(secretsTestClient(server.URL), nil, []string{"API_KEY", "--targets", "workspace"}); err != nil {
		t.Fatalf("runDelete returned error: %v", err)
	}
}

func TestRunDeleteUsageRequiresKey(t *testing.T) {
	if err := runDelete(nil, nil, []string{}); err == nil {
		t.Fatal("expected usage error when key is missing")
	}
}

func TestRunGetUsageRequiresKey(t *testing.T) {
	if err := runGet(nil, nil, []string{}); err == nil {
		t.Fatal("expected usage error when key is missing")
	}
}

func TestRunVerifyUsageRequiresKey(t *testing.T) {
	if err := runVerify(nil, nil, []string{}); err == nil {
		t.Fatal("expected usage error when key is missing")
	}
}

func TestResolveDeploymentTargetsValidatesScopeBeforeListing(t *testing.T) {
	if ids, err := resolveDeploymentTargets(nil, false, nil); err != nil || len(ids) != 0 {
		t.Fatalf("disabled deployment target should not need a client: ids=%v err=%v", ids, err)
	}
	if _, err := resolveDeploymentTargets(nil, true, &selectorFlags{}); err == nil {
		t.Fatal("deployment targets should require a client")
	}
	client := &deployment.Client{}
	if _, err := resolveDeploymentTargets(client, true, &selectorFlags{all: true}); err == nil || !strings.Contains(err.Error(), "requires --scenario") {
		t.Fatalf("missing all-deployments scenario should be refused, got %v", err)
	}
	if _, err := resolveDeploymentTargets(client, true, &selectorFlags{all: true, scenarioID: "lpbs", domain: "example.test"}); err == nil || !strings.Contains(err.Error(), "single selector") {
		t.Fatalf("combined all and domain selector should be refused, got %v", err)
	}
}

func TestRunVerifyPassesWhenFingerprintsMatch(t *testing.T) {
	const key = "LPBS_SERVICE_SECRET"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/local-secrets/scenario/"+key:
			_, _ = w.Write([]byte(`{"key":"LPBS_SERVICE_SECRET","value":"secret-123","masked":false,"scope":"scenario","scenario_id":"landing-page-business-suite","path":"x","timestamp":"x"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-1/secrets/"+key:
			_, _ = w.Write([]byte(`{"secret":{"key":"LPBS_SERVICE_SECRET","value":"secret-123","masked":false,"source":"deployment"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := secretsTestClient(server.URL)
	deploymentClient := secretsTestDeploymentClient(server.URL)
	if err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment", "dep-1"}); err != nil {
		t.Fatalf("runVerify returned error: %v", err)
	}
}

func TestRunVerifyFailsWhenFingerprintsMismatch(t *testing.T) {
	const key = "LPBS_SERVICE_SECRET"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/local-secrets/scenario/"+key:
			_, _ = w.Write([]byte(`{"key":"LPBS_SERVICE_SECRET","value":"secret-123","masked":false,"scope":"scenario","scenario_id":"landing-page-business-suite","path":"x","timestamp":"x"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-1/secrets/"+key:
			_, _ = w.Write([]byte(`{"secret":{"key":"LPBS_SERVICE_SECRET","value":"different","masked":false,"source":"deployment"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := secretsTestClient(server.URL)
	deploymentClient := secretsTestDeploymentClient(server.URL)
	err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment", "dep-1"})
	if err == nil {
		t.Fatal("expected verification failure on mismatched secret values")
	}
	if !strings.Contains(err.Error(), "secret verification failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunVerifyHandlesMissingSecretAsVerificationFailure(t *testing.T) {
	const key = "LPBS_SERVICE_SECRET"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/local-secrets/scenario/"+key:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"secret_not_found"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/deployments/dep-1/secrets/"+key:
			_, _ = w.Write([]byte(`{"secret":{"key":"LPBS_SERVICE_SECRET","value":"secret-123","masked":false,"source":"deployment"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := secretsTestClient(server.URL)
	deploymentClient := secretsTestDeploymentClient(server.URL)
	err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment", "dep-1"})
	if err == nil {
		t.Fatal("expected verification failure when one target is missing")
	}
	if !strings.Contains(err.Error(), "secret verification failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeErr struct {
	msg string
}

func (f *fakeErr) Error() string { return strings.TrimSpace(f.msg) }
