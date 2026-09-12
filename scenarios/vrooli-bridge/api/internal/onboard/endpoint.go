package onboard

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ReachabilityMode describes how a node is expected to reach the control
// plane.  It is intentionally kept separate from transport: a tunnel or a
// managed-DNS endpoint is still proved from the candidate host before use.
type ReachabilityMode string

const (
	ReachabilityLAN    ReachabilityMode = "lan"
	ReachabilityTunnel ReachabilityMode = "tunnel"
	ReachabilityManual ReachabilityMode = "manual"
)

// ValidateControlPlaneURL accepts only an absolute HTTP(S) Bridge base URL.
// LAN/tunnel admission must never accidentally hand a node a loopback URL;
// loopback remains useful only for the explicitly manual same-host case.
func ValidateControlPlaneURL(raw string, mode ReachabilityMode) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("control-plane endpoint must be an absolute http(s) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("control-plane endpoint scheme must be http or https")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", fmt.Errorf("control-plane endpoint must be a Bridge base URL without credentials, path, query, or fragment")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("control-plane endpoint has no host")
	}
	if mode != ReachabilityManual && isLoopbackHost(host) {
		return "", fmt.Errorf("control-plane endpoint %q is loopback and cannot admit a remote %s node", host, mode)
	}
	u.Path = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeReachabilityMode(raw string) (ReachabilityMode, error) {
	switch ReachabilityMode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ReachabilityLAN:
		return ReachabilityLAN, nil
	case ReachabilityTunnel:
		return ReachabilityTunnel, nil
	case ReachabilityManual:
		return ReachabilityManual, nil
	default:
		return "", fmt.Errorf("reachability mode must be lan, tunnel, or manual")
	}
}

// NormalizeReachabilityMode is the public boundary for endpoint configuration
// surfaces. Keeping it beside URL validation prevents a saved configuration
// from accepting a mode that StartOnboarding would later reject.
func NormalizeReachabilityMode(raw string) (ReachabilityMode, error) {
	return normalizeReachabilityMode(raw)
}
