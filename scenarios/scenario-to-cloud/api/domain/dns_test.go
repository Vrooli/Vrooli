package domain

import "testing"

func TestDNSLookupErrorError(t *testing.T) {
	err := DNSLookupError{Kind: DNSLookupTimeout, Message: "timeout"}
	if err.Error() != "timeout" {
		t.Fatalf("expected error message to match, got %q", err.Error())
	}
}
