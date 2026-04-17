package secrets

import (
	"net/http"
	"net/http/httptest"
	"scenario-to-cloud/cli/deployment"
	"strings"
	"testing"

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
	apiClient := cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions {
			return cliutil.APIBaseOptions{Override: baseURL}
		},
		nil,
	)
	return deployment.NewClient(apiClient)
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
	if err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment-id", "dep-1"}); err != nil {
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
	err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment-id", "dep-1"})
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
	err := runVerify(client, deploymentClient, []string{key, "--scenario", "landing-page-business-suite", "--targets", "scenario,deployment", "--deployment-id", "dep-1"})
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
