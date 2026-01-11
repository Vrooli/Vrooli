package tlsinfo

import (
	"errors"
	"testing"
)

func TestEvaluateALPNMissingDomain(t *testing.T) {
	check := EvaluateALPN("", nil, "", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn, got %q", check.Status)
	}
}

func TestEvaluateALPNReachabilityError(t *testing.T) {
	check := EvaluateALPN("example.com", errors.New("dial failed"), "", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn, got %q", check.Status)
	}
	if check.Error == "" {
		t.Fatalf("expected error details")
	}
}

func TestEvaluateALPNProbeError(t *testing.T) {
	check := EvaluateALPN("example.com", nil, "", errors.New("handshake failed"))
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn, got %q", check.Status)
	}
	if check.Error == "" {
		t.Fatalf("expected error details")
	}
}

func TestEvaluateALPNPass(t *testing.T) {
	check := EvaluateALPN("example.com", nil, "acme-tls/1", nil)
	if check.Status != ALPNPass {
		t.Fatalf("expected pass, got %q", check.Status)
	}
}

func TestEvaluateALPNWrongProtocol(t *testing.T) {
	check := EvaluateALPN("example.com", nil, "h2", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn, got %q", check.Status)
	}
}
