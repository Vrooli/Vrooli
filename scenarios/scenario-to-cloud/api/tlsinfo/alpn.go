package tlsinfo

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PortProbeFunc checks TCP reachability for a given host/port.
type PortProbeFunc func(ctx context.Context, host string, port int, timeout time.Duration) error

// ALPNProbeFunc checks TLS-ALPN negotiation for a given host/port.
type ALPNProbeFunc func(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error)

// ALPNCheckStatus represents the outcome of an ALPN readiness check.
type ALPNCheckStatus string

const (
	ALPNPass ALPNCheckStatus = "pass"
	ALPNWarn ALPNCheckStatus = "warn"
)

// ALPNCheck captures TLS-ALPN readiness details.
type ALPNCheck struct {
	Status   ALPNCheckStatus `json:"status"`
	Message  string          `json:"message"`
	Hint     string          `json:"hint,omitempty"`
	Protocol string          `json:"protocol,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// DefaultPortProbe checks TCP reachability for a given host/port.
func DefaultPortProbe(ctx context.Context, host string, port int, timeout time.Duration) error {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// DefaultALPNProbe checks TLS-ALPN negotiation for a given host/port.
func DefaultALPNProbe(ctx context.Context, host, serverName string, port int, timeout time.Duration) (string, error) {
	if serverName == "" {
		serverName = host
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: timeout}
	config := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
		NextProtos:         []string{"acme-tls/1"},
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, config)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := conn.HandshakeContext(ctx); err != nil {
		return "", err
	}
	return conn.ConnectionState().NegotiatedProtocol, nil
}

// RunALPNCheck runs reachability + ALPN probes and returns a readiness check.
func RunALPNCheck(
	ctx context.Context,
	domain string,
	portProbe PortProbeFunc,
	alpnProbe ALPNProbeFunc,
	portTimeout time.Duration,
	alpnTimeout time.Duration,
) ALPNCheck {
	if strings.TrimSpace(domain) == "" {
		return EvaluateALPN(domain, nil, "", nil)
	}
	if portProbe == nil {
		portProbe = DefaultPortProbe
	}
	if alpnProbe == nil {
		alpnProbe = DefaultALPNProbe
	}

	reachErr := portProbe(ctx, domain, 443, portTimeout)
	if reachErr != nil {
		return EvaluateALPN(domain, reachErr, "", nil)
	}
	proto, probeErr := alpnProbe(ctx, domain, domain, 443, alpnTimeout)
	return EvaluateALPN(domain, nil, proto, probeErr)
}

// EvaluateALPN converts probe outcomes into a readiness check.
func EvaluateALPN(domain string, reachErr error, proto string, probeErr error) ALPNCheck {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ALPNCheck{
			Status:  ALPNWarn,
			Message: "Edge domain not set for TLS-ALPN probe",
			Hint:    "Provide edge.domain to validate TLS issuance readiness.",
		}
	}

	if reachErr != nil {
		return ALPNCheck{
			Status:  ALPNWarn,
			Message: fmt.Sprintf("Unable to reach %s:443 for TLS-ALPN probe", domain),
			Hint:    "Ensure the edge domain resolves publicly and port 443 is reachable from the internet.",
			Error:   reachErr.Error(),
		}
	}

	if probeErr != nil {
		return ALPNCheck{
			Status:  ALPNWarn,
			Message: "TLS-ALPN probe failed",
			Hint:    "If using a proxy (e.g., Cloudflare), switch to DNS-only during issuance or configure DNS-01.",
			Error:   probeErr.Error(),
		}
	}

	if proto == "acme-tls/1" {
		return ALPNCheck{
			Status:   ALPNPass,
			Message:  "TLS-ALPN acme-tls/1 negotiated successfully",
			Protocol: proto,
		}
	}

	if proto == "" {
		return ALPNCheck{
			Status:  ALPNWarn,
			Message: "No ALPN protocol negotiated for acme-tls/1",
			Hint:    "If TLS-ALPN cannot be negotiated, use DNS-01 or DNS-only during issuance.",
		}
	}

	return ALPNCheck{
		Status:   ALPNWarn,
		Message:  fmt.Sprintf("Negotiated ALPN protocol %q instead of acme-tls/1", proto),
		Hint:     "If TLS-ALPN cannot be negotiated, use DNS-01 or DNS-only during issuance.",
		Protocol: proto,
	}
}
