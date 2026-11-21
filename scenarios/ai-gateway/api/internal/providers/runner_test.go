package providers

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactOutputKeepsCredentialIdentifiersAndValidJSON(t *testing.T) {
	input := `{"roles":[{"missing_credentials":["TYPESAFE_API_KEY"],"description":"api_key=sk-live-123","max_tokens":2048}],"api_token":"sk-live-456"}`
	output := redactOutput(input)
	var report struct {
		Roles []struct {
			MissingCredentials []string `json:"missing_credentials"`
			Description        string   `json:"description"`
			MaxTokens          int      `json:"max_tokens"`
		} `json:"roles"`
		APIToken string `json:"api_token"`
	}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("redacted JSON is malformed: %v; output=%q", err, output)
	}
	if len(report.Roles) != 1 || len(report.Roles[0].MissingCredentials) != 1 || report.Roles[0].MissingCredentials[0] != "TYPESAFE_API_KEY" {
		t.Fatalf("credential identifiers were changed: %+v", report.Roles)
	}
	if strings.Contains(output, "sk-live-123") || strings.Contains(output, "sk-live-456") {
		t.Fatalf("secret value was not redacted: %s", output)
	}
	if report.Roles[0].MaxTokens != 2048 {
		t.Fatalf("non-secret max_tokens was redacted: %s", output)
	}
}

// A decision's usage block is token counts, not credentials. Redacting it into
// strings made every Ollaya decision fail to decode (invalid_answers), so counts
// survive while a credential-named field stays redacted whatever its type.
func TestRedactOutputKeepsNumericTokenCountsAndRedactsCredentials(t *testing.T) {
	input := `{"model":"laya:en","usage":{"input_tokens":412,"output_tokens":0,"total_tokens":412},"answers":{},"token":"sk-live-1","tokens":["sk-live-2"],"api_key":12345,"input_tokens":"sk-live-3"}`
	output := redactOutput(input)
	var report struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
		Token       string `json:"token"`
		Tokens      string `json:"tokens"`
		APIKey      string `json:"api_key"`
		InputTokens string `json:"input_tokens"`
	}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("redacted JSON is malformed: %v; output=%q", err, output)
	}
	if report.Usage.InputTokens != 412 || report.Usage.OutputTokens != 0 || report.Usage.TotalTokens != 412 {
		t.Fatalf("numeric token counts were redacted: %s", output)
	}
	if report.Token != "[redacted]" || report.Tokens != "[redacted]" || report.APIKey != "[redacted]" || report.InputTokens != "[redacted]" || strings.Contains(output, "sk-live") {
		t.Fatalf("a credential-named field survived, whatever its shape: %s", output)
	}
}

// A typed provider failure an owner adapter writes to stderr must survive the
// resource-command boundary with its observed status and Retry-After intact.
// Without this, a rate limit or an account-credit exhaustion is flattened into
// a generic exit error and routing cannot park on the provider's own window.
func TestExecRunnerRecoversTypedProviderFailureMarker(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"echo 'VROOLI_PROVIDER_ERROR {\"code\":\"rate_limited\",\"http_status\":429,\"retry_after\":\"17\",\"message\":\"slow down\"}' 1>&2\n" +
		"echo 'Error: rate limited' 1>&2\n" +
		"exit 1\n"
	path := filepath.Join(dir, "resource-openrouter")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := (ExecRunner{}).Run(context.Background(), Command{Name: "resource-openrouter", Args: []string{"generate"}})
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("error = %T %v, want *CommandError", err, err)
	}
	if cmdErr.Code != CodeRateLimited {
		t.Fatalf("code = %q, want %q", cmdErr.Code, CodeRateLimited)
	}
	if cmdErr.HTTPStatus != 429 {
		t.Fatalf("http status = %d, want 429", cmdErr.HTTPStatus)
	}
	if cmdErr.RetryAfter != "17" {
		t.Fatalf("retry-after = %q, want observed 17", cmdErr.RetryAfter)
	}
}

// A decision resource that cannot fit a model writes the real marker line the
// resource emits; the gateway must keep the exact code, because routing and the
// walk treat a capacity refusal differently from a provider fault.
func TestExecRunnerRecoversCapacityRefusalMarkers(t *testing.T) {
	for _, code := range []string{CodeInsufficientVRAM, CodeInsufficientMemory, CodeLimitsExceeded} {
		t.Run(code, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\n" +
				"echo 'VROOLI_PROVIDER_ERROR {\"code\":\"" + code + "\",\"message\":\"kev:4b needs 7.5 GiB of VRAM\"}' 1>&2\n" +
				"echo 'Error: " + code + ": kev:4b needs 7.5 GiB of VRAM' 1>&2\n" +
				"exit 1\n"
			if err := os.WriteFile(filepath.Join(dir, "resource-ollaya"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			_, err := (ExecRunner{}).Run(context.Background(), Command{Name: "resource-ollaya", Args: []string{"policy", "resolve", "--role", "decide.local.kev"}})
			var cmdErr *CommandError
			if !errors.As(err, &cmdErr) || cmdErr.Code != code {
				t.Fatalf("error = %v, want CommandError code %q", err, code)
			}
		})
	}
}

// A resource that refuses because a credential is absent names the variable it
// needs. Redaction must not turn that refusal into a generic failure, and the
// stderr the gateway keeps must still be redacted.
func TestExecRunnerReadsMarkerNamingACredentialVariable(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"echo 'VROOLI_PROVIDER_ERROR {\"code\":\"unconfigured\",\"message\":\"role decide.jev.edenai needs EDENAI_API_KEY\"}' 1>&2\n" +
		"echo 'Error: api_key=sk-live-123' 1>&2\n" +
		"exit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "resource-ollaya"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := (ExecRunner{}).Run(context.Background(), Command{Name: "resource-ollaya", Args: []string{"policy", "resolve"}})
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) || cmdErr.Code != CodeUnconfigured {
		t.Fatalf("error = %v, want code %q", err, CodeUnconfigured)
	}
	if strings.Contains(result.Stderr, "sk-live-123") || strings.Contains(cmdErr.Stderr, "sk-live-123") {
		t.Fatalf("retained stderr must stay redacted: %q", result.Stderr)
	}
}

func TestApplyProviderErrorMarkerIgnoresMissingOrMalformedMarker(t *testing.T) {
	cases := []struct {
		name       string
		stderr     string
		wantCode   string
		wantStatus int
		wantRetry  string
	}{
		{name: "absent", stderr: "Error: boom", wantCode: "exit_error"},
		{name: "malformed json", stderr: ProviderErrorMarker + "{not json", wantCode: "exit_error"},
		{name: "unknown code", stderr: ProviderErrorMarker + `{"code":"made_up","http_status":500}`, wantCode: "exit_error"},
		{name: "valid", stderr: ProviderErrorMarker + `{"code":"insufficient_credits","http_status":402}`, wantCode: CodeInsufficientCredits, wantStatus: 402},
		{name: "valid with retry", stderr: ProviderErrorMarker + `{"code":"provider_overloaded","http_status":503,"retry_after":"30"}`, wantCode: CodeProviderOverloaded, wantStatus: 503, wantRetry: "30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmdErr := &CommandError{Code: "exit_error", Command: "resource-openrouter generate", ExitCode: 1}
			applyProviderErrorMarker(cmdErr, tc.stderr)
			if cmdErr.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", cmdErr.Code, tc.wantCode)
			}
			if cmdErr.HTTPStatus != tc.wantStatus {
				t.Fatalf("http status = %d, want %d", cmdErr.HTTPStatus, tc.wantStatus)
			}
			if cmdErr.RetryAfter != tc.wantRetry {
				t.Fatalf("retry-after = %q, want %q", cmdErr.RetryAfter, tc.wantRetry)
			}
		})
	}
}

func TestNormalizeProviderErrorCodeRejectsUnclaimedCodes(t *testing.T) {
	if got := normalizeProviderErrorCode("  rate_limited  "); got != CodeRateLimited {
		t.Fatalf("normalize rate_limited = %q", got)
	}
	for _, code := range []string{CodeUnconfigured, CodeContextOverflow, CodeInsufficientVRAM, CodeInsufficientMemory, CodeLimitsExceeded} {
		if got := normalizeProviderErrorCode(code); got != code {
			t.Fatalf("decision-resource code %q must survive the marker, got %q", code, got)
		}
	}
	if got := normalizeProviderErrorCode("openrouter_magic"); got != "" {
		t.Fatalf("unclaimed code must be rejected, got %q", got)
	}
	if !strings.Contains(ProviderErrorMarker, "VROOLI") {
		t.Fatalf("marker protocol must stay namespaced, got %q", ProviderErrorMarker)
	}
}
