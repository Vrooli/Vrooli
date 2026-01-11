package dns

import "testing"

func TestBaseDomainPreservesSubdomain(t *testing.T) {
	if got := BaseDomain("app.example.com"); got != "app.example.com" {
		t.Fatalf("expected subdomain to remain intact, got %q", got)
	}
	if got := BaseDomain("www.example.com"); got != "example.com" {
		t.Fatalf("expected www to collapse to apex, got %q", got)
	}
	if got := BaseDomain("example.com"); got != "example.com" {
		t.Fatalf("expected apex to remain unchanged, got %q", got)
	}
}
