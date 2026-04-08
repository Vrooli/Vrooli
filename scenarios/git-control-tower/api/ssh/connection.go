package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// TestGitHubConnection tests SSH connection to GitHub using the specified key.
// It extracts the GitHub username from the response on successful authentication.
func TestGitHubConnection(ctx context.Context, platform Platform, keyPath string) TestConnectionResponse {
	ts := timestamp()

	keyPath = ExpandPath(platform, keyPath)

	// Validate key path
	if err := ValidateSSHPath(platform, keyPath); err != nil {
		return TestConnectionResponse{
			Success:   false,
			Status:    "invalid_path",
			Message:   "Invalid SSH key path",
			Hint:      err.Error(),
			Timestamp: ts,
		}
	}

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return TestConnectionResponse{
			Success:   false,
			Status:    "key_not_found",
			Message:   "SSH key file not found",
			Hint:      fmt.Sprintf("The file %s does not exist", keyPath),
			Timestamp: ts,
		}
	}

	// Build SSH command to test connection to GitHub
	// GitHub always returns exit code 1 even on successful auth, but includes the username in stderr
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "IdentitiesOnly=yes", // Only use the specified key, not SSH agent
		"-i", keyPath,
		"-T", "git@github.com",
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, platform.SSHPath(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	latency := time.Since(start).Milliseconds()

	// Combine stdout and stderr for analysis
	output := stdout.String() + stderr.String()

	// GitHub's successful auth response looks like:
	// "Hi username! You've successfully authenticated, but GitHub does not provide shell access."
	if strings.Contains(output, "successfully authenticated") {
		username := extractGitHubUsername(output)
		return TestConnectionResponse{
			Success:    true,
			Status:     "authenticated",
			Message:    fmt.Sprintf("SSH key is authorized for GitHub user: %s", username),
			GitHubUser: username,
			LatencyMs:  latency,
			Timestamp:  ts,
		}
	}

	// Handle various error cases
	return classifyGitHubSSHError(runErr, output, latency, ts)
}

// extractGitHubUsername extracts the username from GitHub's SSH response.
func extractGitHubUsername(output string) string {
	// Pattern: "Hi username! You've successfully authenticated"
	re := regexp.MustCompile(`Hi ([^!]+)!`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return "unknown"
}

// sshErrorPattern defines a pattern for matching SSH error output to a classified response.
type sshErrorPattern struct {
	// substrings to match (case-insensitive via outputLower); any match triggers this pattern
	substrings []string
	status     string
	message    string
	hint       string
}

// sshErrorPatterns is the table of known SSH error patterns, checked in order.
var sshErrorPatterns = []sshErrorPattern{
	{
		substrings: []string{"permission denied"},
		status:     "not_authorized",
		message:    "SSH key is not authorized on GitHub",
		hint:       "Add this key to your GitHub account at https://github.com/settings/ssh/new",
	},
	{
		substrings: []string{"connection refused"},
		status:     "connection_refused",
		message:    "Connection to GitHub refused",
		hint:       "Check if GitHub.com is accessible from your network. You may be behind a firewall blocking SSH.",
	},
	{
		substrings: []string{"timed out", "connection timeout"},
		status:     "timeout",
		message:    "Connection to GitHub timed out",
		hint:       "Check your network connection. If you're behind a corporate firewall, SSH (port 22) may be blocked. Try using HTTPS instead.",
	},
	{
		substrings: []string{"host key verification failed"},
		status:     "host_key_failed",
		message:    "GitHub host key verification failed",
		hint:       "The GitHub server's identity could not be verified. This may indicate a network security issue.",
	},
	{
		substrings: []string{"network is unreachable", "no route to host"},
		status:     "network_error",
		message:    "Cannot reach GitHub",
		hint:       "Check your internet connection.",
	},
	{
		substrings: []string{"could not resolve", "name resolution"},
		status:     "dns_error",
		message:    "Cannot resolve github.com",
		hint:       "Check your DNS settings and internet connection.",
	},
}

// classifyGitHubSSHError analyzes SSH error output and returns a user-friendly response.
func classifyGitHubSSHError(err error, output string, latencyMs int64, ts string) TestConnectionResponse {
	outputLower := strings.ToLower(output)

	for _, p := range sshErrorPatterns {
		if matchesAnySubstring(outputLower, p.substrings) {
			return TestConnectionResponse{
				Success:   false,
				Status:    p.status,
				Message:   p.message,
				Hint:      p.hint,
				LatencyMs: latencyMs,
				Timestamp: ts,
			}
		}
	}

	return buildGenericSSHError(err, output, latencyMs, ts)
}

// matchesAnySubstring returns true if s contains any of the given substrings.
func matchesAnySubstring(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// buildGenericSSHError constructs a fallback error response when no known pattern matches.
func buildGenericSSHError(err error, output string, latencyMs int64, ts string) TestConnectionResponse {
	hint := output
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && hint == "" {
			hint = string(exitErr.Stderr)
		}
		if hint == "" {
			hint = err.Error()
		}
	}

	return TestConnectionResponse{
		Success:   false,
		Status:    "error",
		Message:   "SSH connection failed",
		Hint:      strings.TrimSpace(hint),
		LatencyMs: latencyMs,
		Timestamp: ts,
	}
}
