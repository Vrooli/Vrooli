package tlsinfo

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeTLSService struct {
	result ProbeResult
	err    error
}

func (f fakeTLSService) Probe(_ context.Context, _ string) (ProbeResult, error) {
	return f.result, f.err
}

func TestRunSnapshotUsesALPNRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := fakeTLSService{result: ProbeResult{Domain: "example.com", Valid: true}}
	seen := false
	_, err := RunSnapshot(ctx, "example.com", svc, func(_ context.Context, domain string) ALPNCheck {
		if domain != "example.com" {
			t.Fatalf("expected domain example.com, got %q", domain)
		}
		seen = true
		return ALPNCheck{Status: ALPNPass, Message: "ok"}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !seen {
		t.Fatalf("expected ALPN runner to be invoked")
	}
}

func TestRunSnapshotPropagatesProbeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	probeErr := errors.New("probe failed")
	svc := fakeTLSService{err: probeErr}
	snap, err := RunSnapshot(ctx, "example.com", svc, func(_ context.Context, _ string) ALPNCheck {
		return ALPNCheck{Status: ALPNWarn, Message: "warn"}
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if snap.ALPN.Status != ALPNWarn {
		t.Fatalf("expected ALPN check in snapshot on error")
	}
}
