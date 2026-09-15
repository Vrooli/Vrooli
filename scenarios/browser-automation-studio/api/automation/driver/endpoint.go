package driver

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// ResolveEndpoint returns a validated Playwright driver endpoint. By default
// only loopback hosts are allowed; production deployments may explicitly add
// trusted driver hosts through PLAYWRIGHT_DRIVER_ALLOWED_HOSTS.
func ResolveEndpoint(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = DefaultDriverURL
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
		return "", fmt.Errorf("invalid Playwright driver endpoint")
	}
	if !isTrustedDriverHost(parsed.Hostname()) {
		return "", fmt.Errorf("untrusted Playwright driver host %q", parsed.Hostname())
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isTrustedDriverHost(host string) bool {
	if strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback() {
		return true
	}
	for _, candidate := range strings.Split(os.Getenv("PLAYWRIGHT_DRIVER_ALLOWED_HOSTS"), ",") {
		if strings.EqualFold(strings.TrimSpace(candidate), host) {
			return true
		}
	}
	return false
}
