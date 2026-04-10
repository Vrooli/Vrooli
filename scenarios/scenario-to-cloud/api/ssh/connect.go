// DOC: docs/concepts/architecture.md#ssh-subsystem — SSH subsystem overview
package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

// IsIPv6 checks if the given host is an IPv6 address.
func IsIPv6(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.To4() == nil
}

// IPv6ConnectivityHint is a helpful hint for IPv6 connectivity issues.
const IPv6ConnectivityHint = "You entered an IPv6 address, but your network may not have IPv6 connectivity. Most ISPs still only provide IPv4. Try using the IPv4 address of your server instead."

// TestConnection tests SSH connection to a host using key authentication.
// If runner is non-nil it is used for the SSH invocation (for testing);
// otherwise the default os/exec-based runSSH is used.
func TestConnection(ctx context.Context, runner Runner, req TestConnectionRequest) TestConnectionResponse {
	timestamp := nowTimestamp()

	cfg := NewConfig(req.Host, req.Port, req.User, ExpandPath(req.KeyPath))

	// Validate key path
	if err := ValidateSSHPath(cfg.KeyPath); err != nil {
		return TestConnectionResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusNotFound,
				Message:   "Invalid SSH key path",
				Hint:      err.Error(),
				Timestamp: timestamp,
			},
		}
	}

	// Check if key file exists
	if _, err := os.Stat(cfg.KeyPath); os.IsNotExist(err) {
		return TestConnectionResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusNotFound,
				Message:   "SSH key file not found",
				Hint:      fmt.Sprintf("The file %s does not exist", cfg.KeyPath),
				Timestamp: timestamp,
			},
		}
	}

	// Build SSH command with IdentitiesOnly=yes to test ONLY the specified key
	// (not SSH agent keys). This ensures the test accurately reflects whether
	// the key file itself is authorized on the server.
	testCmd := "echo ok && cat /etc/os-release 2>/dev/null | head -5"

	opts := TestConnectionOptions()
	start := time.Now()
	var result Result
	var runErr error
	if runner != nil {
		result, runErr = runner.Run(ctx, cfg, testCmd, opts)
	} else {
		result, runErr = runSSH(ctx, cfg, testCmd, opts)
	}
	latency := time.Since(start).Milliseconds()

	if runErr != nil {
		// Combine error sources for classification
		errStr := runErr.Error() + " " + result.Stderr
		defaultHint := result.Stderr
		if defaultHint == "" {
			defaultHint = runErr.Error()
		}

		classified := ClassifyError(errStr, cfg.Host, defaultHint)

		slog.Info("ssh.connection_test",
			"host", cfg.Host,
			"status", StatusFromError(classified),
			"latency_ms", latency,
		)

		if StatusFromError(classified) == StatusError {
			slog.Warn("ssh.classification_fallback",
				"host", cfg.Host,
				"raw_error", errStr,
			)
		}

		return TestConnectionResponse{
			Outcome: Outcome{
				OK:        false,
				Status:    StatusFromError(classified),
				Message:   classified.Message,
				Hint:      classified.Hint,
				Timestamp: timestamp,
			},
			LatencyMs: latency,
		}
	}

	// Parse OS info from output
	serverInfo := ""
	lines := strings.Split(result.Stdout, "\n")
	if len(lines) > 1 {
		// Skip "ok" line, parse os-release
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				serverInfo = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	// Populate Fingerprint by reading host key from known_hosts after
	// successful connection (the key was accepted via accept-new).
	fingerprint := readHostFingerprint(cfg.Host)

	slog.Info("ssh.connection_test",
		"host", cfg.Host,
		"status", StatusSuccess,
		"latency_ms", latency,
	)

	return TestConnectionResponse{
		Outcome: Outcome{
			OK:        true,
			Status:    StatusSuccess,
			Message:   "SSH connection successful",
			Timestamp: timestamp,
		},
		ServerInfo:  serverInfo,
		Fingerprint: fingerprint,
		LatencyMs:   latency,
	}
}

// readHostFingerprint extracts the host key fingerprint from known_hosts.
// Returns empty string on any failure (best-effort).
func readHostFingerprint(host string) string {
	cmd := ExecCommandRunner{}
	stdout, _, err := cmd.Run(context.Background(), "ssh-keygen", "-lF", host)
	if err != nil {
		return ""
	}
	// Output format: "# Host <host> found: line N\n<bits> <fingerprint> ..."
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[1] // SHA256:xxxx
		}
	}
	return ""
}
