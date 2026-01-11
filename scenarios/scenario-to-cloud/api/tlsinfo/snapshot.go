package tlsinfo

import (
	"context"
	"fmt"
	"time"
)

// ALPNRunner runs a TLS-ALPN readiness check for a domain.
type ALPNRunner func(ctx context.Context, domain string) ALPNCheck

// DefaultALPNRunner uses the standard ALPN probe with conservative timeouts.
func DefaultALPNRunner(ctx context.Context, domain string) ALPNCheck {
	return RunALPNCheck(ctx, domain, nil, nil, 3*time.Second, 4*time.Second)
}

// Snapshot contains certificate details and ALPN readiness.
type Snapshot struct {
	Probe ProbeResult
	ALPN  ALPNCheck
}

// RunSnapshot gathers TLS certificate info and ALPN readiness for a domain.
func RunSnapshot(ctx context.Context, domain string, svc Service, alpnRunner ALPNRunner) (Snapshot, error) {
	if svc == nil {
		return Snapshot{}, fmt.Errorf("tls probe service is nil")
	}
	if alpnRunner == nil {
		alpnRunner = DefaultALPNRunner
	}
	alpn := alpnRunner(ctx, domain)
	probe, err := svc.Probe(ctx, domain)
	if err != nil {
		return Snapshot{ALPN: alpn}, err
	}
	return Snapshot{Probe: probe, ALPN: alpn}, nil
}
