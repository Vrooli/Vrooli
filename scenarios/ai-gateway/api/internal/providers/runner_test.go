package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if got := normalizeProviderErrorCode("openrouter_magic"); got != "" {
		t.Fatalf("unclaimed code must be rejected, got %q", got)
	}
	if !strings.Contains(ProviderErrorMarker, "VROOLI") {
		t.Fatalf("marker protocol must stay namespaced, got %q", ProviderErrorMarker)
	}
}
