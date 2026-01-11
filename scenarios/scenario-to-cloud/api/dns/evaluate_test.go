package dns

import (
	"context"
	"testing"
	"time"
)

func TestEvaluateIncludesEdgeDomains(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com":           {"104.16.0.1"},
		"www.example.com":       {"104.16.0.2"},
		"do-origin.example.com": {"203.0.113.10"},
		"app.example.com":       {"203.0.113.10"},
	}})

	eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")

	apex, ok := eval.StatusForRole("apex")
	if !ok {
		t.Fatalf("expected apex status")
	}
	if !apex.Proxied {
		t.Fatalf("expected apex to be proxied via Cloudflare: ips=%v err=%v", apex.Lookup.IPs, apex.Lookup.Error)
	}

	origin, ok := eval.StatusForRole("origin")
	if !ok {
		t.Fatalf("expected origin status")
	}
	if origin.AllowProxy {
		t.Fatalf("expected origin to disallow proxying")
	}
	if !origin.PointsToVPS {
		t.Fatalf("expected origin to point to VPS")
	}
}
