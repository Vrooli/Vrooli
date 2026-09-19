package ssh

import (
	"errors"
	"os/exec"
	"testing"
)

func TestExtractGitHubUsername(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "github success response",
			output: "Hi octocat! You've successfully authenticated, but GitHub does not provide shell access.",
			want:   "octocat",
		},
		{
			name:   "missing username",
			output: "successfully authenticated",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractGitHubUsername(tt.output); got != tt.want {
				t.Fatalf("extractGitHubUsername() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyGitHubSSHError(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantStatus string
	}{
		{
			name:       "permission denied",
			output:     "git@github.com: Permission denied (publickey).",
			wantStatus: "not_authorized",
		},
		{
			name:       "connection refused",
			output:     "ssh: connect to host github.com port 22: Connection refused",
			wantStatus: "connection_refused",
		},
		{
			name:       "timeout",
			output:     "operation timed out",
			wantStatus: "timeout",
		},
		{
			name:       "host key failed",
			output:     "Host key verification failed.",
			wantStatus: "host_key_failed",
		},
		{
			name:       "network unreachable",
			output:     "Network is unreachable",
			wantStatus: "network_error",
		},
		{
			name:       "dns error",
			output:     "Could not resolve hostname github.com",
			wantStatus: "dns_error",
		},
		{
			name:       "generic output",
			output:     "unexpected ssh failure",
			wantStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyGitHubSSHError(errors.New("ssh failed"), tt.output, 12, "2026-05-01T00:00:00Z")
			if got.Success {
				t.Fatal("Success = true, want false")
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.LatencyMs != 12 {
				t.Fatalf("LatencyMs = %d, want 12", got.LatencyMs)
			}
			if got.Timestamp != "2026-05-01T00:00:00Z" {
				t.Fatalf("Timestamp = %q, want fixed timestamp", got.Timestamp)
			}
		})
	}
}

func TestBuildGenericSSHErrorUsesExitErrorStderr(t *testing.T) {
	exitErr := &exec.ExitError{Stderr: []byte("stderr hint")}

	got := buildGenericSSHError(exitErr, "", 7, "2026-05-01T00:00:00Z")
	if got.Status != "error" {
		t.Fatalf("Status = %q, want error", got.Status)
	}
	if got.Hint != "stderr hint" {
		t.Fatalf("Hint = %q, want stderr hint", got.Hint)
	}
}
