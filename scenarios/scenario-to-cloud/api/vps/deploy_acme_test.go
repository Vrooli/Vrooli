package vps

import (
	"strings"
	"testing"
)

func TestCaddyACMEOriginUnreachableHintEmpty(t *testing.T) {
	if hint := caddyACMEOriginUnreachableHint("", false); hint != "" {
		t.Fatalf("expected empty hint, got %q", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintDNS01Missing(t *testing.T) {
	logs := "2025/01/01 00:00:00 acme: remaining=[dns-01] no solvers available"
	hint := caddyACMEOriginUnreachableHint(logs, false)
	if hint == "" {
		t.Fatalf("expected hint for dns-01 missing")
	}
	if !containsAll(hint, []string{"DNS-01", "CLOUDFLARE_API_TOKEN"}) {
		t.Fatalf("expected dns-01 remediation hint, got %q", hint)
	}
}

func TestCaddyACMEOriginUnreachableHintOriginBlocked(t *testing.T) {
	logs := "acme: error: challenge failed: 522 origin unreachable"
	hint := caddyACMEOriginUnreachableHint(logs, true)
	if hint == "" {
		t.Fatalf("expected hint for origin unreachable")
	}
	if !containsAll(hint, []string{"origin unreachable", "Open inbound 80/443"}) {
		t.Fatalf("expected origin unreachable remediation, got %q", hint)
	}
}

func containsAll(value string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
