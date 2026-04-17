package vps

import (
	"context"
	"scenario-to-cloud/ssh"
	"strings"
	"testing"
)

func TestCaddyTLSRenewCommand_QuotesDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "simple domain",
			domain: "example.com",
			want:   "'https://example.com'",
		},
		{
			name:   "domain with special chars",
			domain: "my-site.example.com",
			want:   "'https://my-site.example.com'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := CaddyTLSRenewCommand(tt.domain)
			if !strings.Contains(cmd, tt.want) {
				t.Errorf("CaddyTLSRenewCommand(%q) = %q, should contain %q", tt.domain, cmd, tt.want)
			}
		})
	}
}

func TestRunCaddyTLSRenew_Success(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			"caddy trust": {Stdout: "Certificate valid", ExitCode: 0},
		},
	}

	cfg := ssh.Config{Host: "test", User: "root", Port: 22}
	result := RunCaddyTLSRenew(context.Background(), runner, cfg, "example.com")

	if !result.OK {
		t.Errorf("expected OK, got: %s", result.Output)
	}
	if !strings.Contains(result.Message, "successfully") {
		t.Errorf("Message = %q, expected success message", result.Message)
	}
}

func TestRunCaddyTLSRenew_Failure(t *testing.T) {
	t.Parallel()

	runner := &testSSHRunner{
		responses: map[string]ssh.Result{
			"caddy trust": {Stderr: "renewal failed", ExitCode: 1},
		},
	}

	cfg := ssh.Config{Host: "test", User: "root", Port: 22}
	result := RunCaddyTLSRenew(context.Background(), runner, cfg, "example.com")

	if result.OK {
		t.Error("expected failure")
	}
	if result.Message == "" {
		t.Error("expected non-empty error message")
	}
}
