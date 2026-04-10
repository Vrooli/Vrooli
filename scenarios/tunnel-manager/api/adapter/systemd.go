package adapter

import (
	"context"
	"os/exec"
	"strings"
)

// CmdRunner executes a system command and returns its combined output.
type CmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultCmdRunner runs a command via exec.CommandContext.
func DefaultCmdRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// ExtractHostname strips the URL scheme and path, returning only the hostname.
// e.g., "https://api.example.com/path" → "api.example.com"
func ExtractHostname(publicURL string) string {
	host := strings.TrimPrefix(publicURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	return host
}
