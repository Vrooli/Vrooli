package tlsinfo

import (
	"errors"
	"testing"
)

func TestEvaluateALPNEmptyDomain(t *testing.T) {
	check := EvaluateALPN("", nil, "", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn for empty domain, got %q", check.Status)
	}
	if check.Message == "" || check.Hint == "" {
		t.Fatalf("expected message and hint for empty domain")
	}
}

func TestEvaluateALPNReachabilityError(t *testing.T) {
	check := EvaluateALPN("example.com", errors.New("no route"), "", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn on reachability error, got %q", check.Status)
	}
	if check.Error == "" {
		t.Fatalf("expected error to be set")
	}
}

func TestEvaluateALPNProbeError(t *testing.T) {
	check := EvaluateALPN("example.com", nil, "", errors.New("handshake failed"))
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn on probe error, got %q", check.Status)
	}
	if check.Error == "" {
		t.Fatalf("expected error to be set")
	}
}

func TestEvaluateALPNWrongProtocol(t *testing.T) {
	check := EvaluateALPN("example.com", nil, "h2", nil)
	if check.Status != ALPNWarn {
		t.Fatalf("expected warn on wrong protocol, got %q", check.Status)
	}
	if check.Protocol != "h2" {
		t.Fatalf("expected protocol to be set")
	}
}
