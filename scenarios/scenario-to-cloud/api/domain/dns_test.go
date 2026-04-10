package domain

import "testing"

func TestDNSLookupErrorError(t *testing.T) {
	tests := []struct {
		name    string
		kind    DNSLookupErrorKind
		message string
	}{
		{
			name:    "timeout error",
			kind:    DNSLookupTimeout,
			message: "connection timed out after 5s",
		},
		{
			name:    "not found error",
			kind:    DNSLookupNotFound,
			message: "no such host",
		},
		{
			name:    "invalid host error",
			kind:    DNSLookupInvalidHost,
			message: "invalid hostname format",
		},
		{
			name:    "unknown error",
			kind:    DNSLookupUnknown,
			message: "unexpected DNS error",
		},
		{
			name:    "empty message",
			kind:    DNSLookupTimeout,
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DNSLookupError{Kind: tt.kind, Message: tt.message}
			if err.Error() != tt.message {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.message)
			}
			if err.Kind != tt.kind {
				t.Errorf("Kind = %q, want %q", err.Kind, tt.kind)
			}
		})
	}
}

func TestDNSLookupErrorKindConstants(t *testing.T) {
	// Verify that all error kinds have distinct, non-empty values
	kinds := []DNSLookupErrorKind{
		DNSLookupNotFound,
		DNSLookupTimeout,
		DNSLookupInvalidHost,
		DNSLookupUnknown,
	}

	seen := make(map[DNSLookupErrorKind]bool)
	for _, kind := range kinds {
		if kind == "" {
			t.Errorf("DNSLookupErrorKind should not be empty")
		}
		if seen[kind] {
			t.Errorf("duplicate DNSLookupErrorKind: %q", kind)
		}
		seen[kind] = true
	}
}

func TestDNSPolicyConstants(t *testing.T) {
	// Verify DNS policy constants are distinct and non-empty
	policies := []DNSPolicy{
		DNSPolicyRequired,
		DNSPolicyWarn,
		DNSPolicySkip,
	}

	seen := make(map[DNSPolicy]bool)
	for _, policy := range policies {
		if policy == "" {
			t.Errorf("DNSPolicy should not be empty")
		}
		if seen[policy] {
			t.Errorf("duplicate DNSPolicy: %q", policy)
		}
		seen[policy] = true
	}
}

func TestDNSLookupResultWithError(t *testing.T) {
	// Verify DNSLookupResult properly holds error information
	result := DNSLookupResult{
		Host: "example.com",
		IPs:  nil,
		Error: &DNSLookupError{
			Kind:    DNSLookupNotFound,
			Message: "no such host",
		},
	}

	if result.Host != "example.com" {
		t.Errorf("Host = %q, want %q", result.Host, "example.com")
	}
	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if result.Error.Kind != DNSLookupNotFound {
		t.Errorf("Error.Kind = %q, want %q", result.Error.Kind, DNSLookupNotFound)
	}
	if len(result.IPs) != 0 {
		t.Errorf("IPs should be empty when error is present, got %v", result.IPs)
	}
}

func TestDNSLookupResultWithSuccess(t *testing.T) {
	// Verify DNSLookupResult properly holds success information
	result := DNSLookupResult{
		Host:  "example.com",
		IPs:   []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"},
		Error: nil,
	}

	if result.Host != "example.com" {
		t.Errorf("Host = %q, want %q", result.Host, "example.com")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil on success, got %v", result.Error)
	}
	if len(result.IPs) != 2 {
		t.Errorf("IPs length = %d, want 2", len(result.IPs))
	}
}

func TestDNSComparisonResult(t *testing.T) {
	tests := []struct {
		name        string
		domainIPs   []string
		vpsIPs      []string
		pointsToVPS bool
	}{
		{
			name:        "domain points to VPS",
			domainIPs:   []string{"203.0.113.10"},
			vpsIPs:      []string{"203.0.113.10"},
			pointsToVPS: true,
		},
		{
			name:        "domain does not point to VPS",
			domainIPs:   []string{"104.16.0.1"},
			vpsIPs:      []string{"203.0.113.10"},
			pointsToVPS: false,
		},
		{
			name:        "multiple IPs with match",
			domainIPs:   []string{"104.16.0.1", "203.0.113.10"},
			vpsIPs:      []string{"203.0.113.10"},
			pointsToVPS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DNSComparisonResult{
				Domain: DNSLookupResult{
					Host: "example.com",
					IPs:  tt.domainIPs,
				},
				VPS: DNSLookupResult{
					Host: "203.0.113.10",
					IPs:  tt.vpsIPs,
				},
				PointsToVPS: tt.pointsToVPS,
			}

			if result.PointsToVPS != tt.pointsToVPS {
				t.Errorf("PointsToVPS = %v, want %v", result.PointsToVPS, tt.pointsToVPS)
			}
		})
	}
}

func TestDNSARecordHint(t *testing.T) {
	hint := DNSARecordHint{
		Domain:          "example.com",
		TargetIP:        "203.0.113.10",
		Providers:       []string{"Cloudflare", "Route53", "GoDaddy"},
		PropagationNote: "DNS changes may take up to 48 hours to propagate",
	}

	if hint.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", hint.Domain, "example.com")
	}
	if hint.TargetIP != "203.0.113.10" {
		t.Errorf("TargetIP = %q, want %q", hint.TargetIP, "203.0.113.10")
	}
	if len(hint.Providers) != 3 {
		t.Errorf("Providers length = %d, want 3", len(hint.Providers))
	}
	if hint.PropagationNote == "" {
		t.Error("PropagationNote should not be empty")
	}
}

func TestDNSRecordSet(t *testing.T) {
	recordSet := DNSRecordSet{
		Domain: "example.com",
		A: []DNSRecordValue{
			{Value: "93.184.216.34", TTL: 300},
		},
		AAAA: []DNSRecordValue{
			{Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
		},
		MX: []DNSMXRecord{
			{Host: "mail.example.com", Priority: 10, TTL: 3600},
		},
		TXT: []DNSRecordValue{
			{Value: "v=spf1 include:_spf.google.com ~all", TTL: 3600},
		},
	}

	if recordSet.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", recordSet.Domain, "example.com")
	}
	if len(recordSet.A) != 1 {
		t.Errorf("A records length = %d, want 1", len(recordSet.A))
	}
	if len(recordSet.AAAA) != 1 {
		t.Errorf("AAAA records length = %d, want 1", len(recordSet.AAAA))
	}
	if len(recordSet.MX) != 1 {
		t.Errorf("MX records length = %d, want 1", len(recordSet.MX))
	}
	if recordSet.MX[0].Priority != 10 {
		t.Errorf("MX Priority = %d, want 10", recordSet.MX[0].Priority)
	}
}

func TestReachabilityResult(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		reachable bool
		errorKind DNSLookupErrorKind
	}{
		{
			name:      "reachable host",
			target:    "203.0.113.10",
			reachable: true,
			errorKind: "",
		},
		{
			name:      "unreachable host with timeout",
			target:    "192.0.2.1",
			reachable: false,
			errorKind: DNSLookupTimeout,
		},
		{
			name:      "unreachable domain not found",
			target:    "nonexistent.invalid",
			reachable: false,
			errorKind: DNSLookupNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReachabilityResult{
				Target:    tt.target,
				Type:      "host",
				Reachable: tt.reachable,
				ErrorKind: tt.errorKind,
			}

			if result.Target != tt.target {
				t.Errorf("Target = %q, want %q", result.Target, tt.target)
			}
			if result.Reachable != tt.reachable {
				t.Errorf("Reachable = %v, want %v", result.Reachable, tt.reachable)
			}
			if result.ErrorKind != tt.errorKind {
				t.Errorf("ErrorKind = %q, want %q", result.ErrorKind, tt.errorKind)
			}
		})
	}
}
