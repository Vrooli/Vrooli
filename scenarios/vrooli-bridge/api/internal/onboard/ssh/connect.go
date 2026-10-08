package ssh

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// TestConnection tests key-based SSH to a host using the bridge-owned key and
// known_hosts. It offers only the supplied key (IdentitiesOnly) so a success
// means that exact key is authorized — the signal first touch keys off.
func (s *Service) TestConnection(ctx context.Context, req TestConnectionRequest) TestConnectionResponse {
	timestamp := nowTimestamp()

	knownHosts := s.knownHostsPath()
	if err := ensureKnownHostsFile(knownHosts); err != nil {
		return TestConnectionResponse{Outcome: Outcome{
			OK:        false,
			Status:    StatusError,
			Message:   "Failed to initialize known_hosts",
			Hint:      err.Error(),
			Timestamp: timestamp,
		}}
	}

	cfg := NewConfig(req.Host, req.Port, req.User, req.KeyPath, knownHosts)

	if _, err := os.Stat(cfg.KeyPath); os.IsNotExist(err) {
		return TestConnectionResponse{Outcome: Outcome{
			OK:        false,
			Status:    StatusNotFound,
			Message:   "SSH key file not found",
			Hint:      fmt.Sprintf("The file %s does not exist", cfg.KeyPath),
			Timestamp: timestamp,
		}}
	}

	testCmd := "echo ok && cat /etc/os-release 2>/dev/null | head -5"
	opts := TestConnectionOptions()

	start := time.Now()
	result, runErr := s.runner.Run(ctx, cfg, testCmd, opts)
	latency := time.Since(start).Milliseconds()

	if runErr != nil {
		errStr := runErr.Error() + " " + result.Stderr
		defaultHint := result.Stderr
		if defaultHint == "" {
			defaultHint = runErr.Error()
		}
		classified := ClassifyError(errStr, cfg.Host, defaultHint)

		slog.Info("ssh.connection_test", "host", cfg.Host, "status", StatusFromError(classified), "latency_ms", latency)
		if StatusFromError(classified) == StatusError {
			slog.Warn("ssh.classification_fallback", "host", cfg.Host, "raw_error", errStr)
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

	serverInfo := ""
	lines := strings.Split(result.Stdout, "\n")
	if len(lines) > 1 {
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				serverInfo = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	slog.Info("ssh.connection_test", "host", cfg.Host, "status", StatusSuccess, "latency_ms", latency)

	return TestConnectionResponse{
		Outcome: Outcome{
			OK:        true,
			Status:    StatusSuccess,
			Message:   "SSH connection successful",
			Timestamp: timestamp,
		},
		ServerInfo:  serverInfo,
		Fingerprint: hostFingerprint(knownHosts, cfg.Host, cfg.Port),
		LatencyMs:   latency,
	}
}
