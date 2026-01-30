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

// classifyGitHubSSHError analyzes SSH error output and returns a user-friendly response.
func classifyGitHubSSHError(err error, output string, latencyMs int64, ts string) TestConnectionResponse {
	outputLower := strings.ToLower(output)

	// Check for permission denied (key not authorized)
	if strings.Contains(output, "Permission denied") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "not_authorized",
			Message:   "SSH key is not authorized on GitHub",
			Hint:      "Add this key to your GitHub account at https://github.com/settings/ssh/new",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Check for connection refused
	if strings.Contains(outputLower, "connection refused") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "connection_refused",
			Message:   "Connection to GitHub refused",
			Hint:      "Check if GitHub.com is accessible from your network. You may be behind a firewall blocking SSH.",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Check for timeout
	if strings.Contains(outputLower, "timed out") || strings.Contains(outputLower, "connection timeout") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "timeout",
			Message:   "Connection to GitHub timed out",
			Hint:      "Check your network connection. If you're behind a corporate firewall, SSH (port 22) may be blocked. Try using HTTPS instead.",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Check for host key verification failure
	if strings.Contains(output, "Host key verification failed") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "host_key_failed",
			Message:   "GitHub host key verification failed",
			Hint:      "The GitHub server's identity could not be verified. This may indicate a network security issue.",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Check for network unreachable
	if strings.Contains(outputLower, "network is unreachable") || strings.Contains(outputLower, "no route to host") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "network_error",
			Message:   "Cannot reach GitHub",
			Hint:      "Check your internet connection.",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Check for DNS resolution failure
	if strings.Contains(outputLower, "could not resolve") || strings.Contains(outputLower, "name resolution") {
		return TestConnectionResponse{
			Success:   false,
			Status:    "dns_error",
			Message:   "Cannot resolve github.com",
			Hint:      "Check your DNS settings and internet connection.",
			LatencyMs: latencyMs,
			Timestamp: ts,
		}
	}

	// Generic error
	errMsg := "SSH connection failed"
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
		Message:   errMsg,
		Hint:      strings.TrimSpace(hint),
		LatencyMs: latencyMs,
		Timestamp: ts,
	}
}
