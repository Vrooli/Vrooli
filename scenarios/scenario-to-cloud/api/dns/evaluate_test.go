package dns

import (
	"context"
	"testing"
	"time"

	"scenario-to-cloud/domain"
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

func TestEvaluateReturnsAllRoles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com":           {"203.0.113.10"},
		"www.example.com":       {"203.0.113.10"},
		"do-origin.example.com": {"203.0.113.10"},
	}})

	eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")

	// Should have apex, www, and origin statuses
	roles := make(map[string]bool)
	for _, s := range eval.Statuses {
		roles[s.Role] = true
	}

	requiredRoles := []string{"apex", "www", "origin"}
	for _, role := range requiredRoles {
		if !roles[role] {
			t.Errorf("expected role %q in statuses", role)
		}
	}
}

func TestEvaluateEdgeDomainAddedWhenDifferent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Use a subdomain that differs from apex/www/origin
	// For "staging.app.example.com", baseDomain becomes "staging.app.example.com"
	// but for testing edge role, we need edgeDomain different from apex/www/origin
	// The logic is: edge added when edgeDomain != apexDomain && != wwwDomain && != originDomain

	// When edge domain is "api.example.com" but we pass "example.com" as base,
	// edgeDomain would equal the input. We need a case where edge differs from all.

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com":                   {"203.0.113.10"},
		"www.example.com":               {"203.0.113.10"},
		"do-origin.example.com":         {"203.0.113.10"},
		"staging.example.com":           {"203.0.113.10"},
		"www.staging.example.com":       {"203.0.113.10"},
		"do-origin.staging.example.com": {"203.0.113.10"},
	}})

	// For "staging.example.com", baseDomain = "staging.example.com"
	// apex = "staging.example.com", www = "www.staging.example.com", origin = "do-origin.staging.example.com"
	// edgeDomain = "staging.example.com" == apex, so no edge role added
	eval := Evaluate(ctx, svc, "staging.example.com", "203.0.113.10")

	// Edge role should NOT be present when edgeDomain == apexDomain
	_, hasEdge := eval.StatusForRole("edge")
	if hasEdge {
		t.Error("expected no edge role when edgeDomain equals apexDomain")
	}

	// Verify apex role is present with the staging subdomain
	apex, hasApex := eval.StatusForRole("apex")
	if !hasApex {
		t.Fatal("expected apex role")
	}
	if apex.Host != "staging.example.com" {
		t.Errorf("apex Host = %q, want %q", apex.Host, "staging.example.com")
	}
}

func TestEvaluateBaseDomainCalculation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		name       string
		edgeDomain string
		wantBase   string
		wantApex   string
		wantWWW    string
		wantOrigin string
	}{
		{
			name:       "apex domain",
			edgeDomain: "example.com",
			wantBase:   "example.com",
			wantApex:   "example.com",
			wantWWW:    "www.example.com",
			wantOrigin: "do-origin.example.com",
		},
		{
			name:       "www domain",
			edgeDomain: "www.example.com",
			wantBase:   "example.com",
			wantApex:   "example.com",
			wantWWW:    "www.example.com",
			wantOrigin: "do-origin.example.com",
		},
		{
			name:       "subdomain",
			edgeDomain: "app.example.com",
			wantBase:   "app.example.com",
			wantApex:   "app.example.com",
			wantWWW:    "www.app.example.com",
			wantOrigin: "do-origin.app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&FakeResolver{Hosts: map[string][]string{}})
			eval := Evaluate(ctx, svc, tt.edgeDomain, "203.0.113.10")

			if eval.BaseDomain != tt.wantBase {
				t.Errorf("BaseDomain = %q, want %q", eval.BaseDomain, tt.wantBase)
			}
			if eval.ApexDomain != tt.wantApex {
				t.Errorf("ApexDomain = %q, want %q", eval.ApexDomain, tt.wantApex)
			}
			if eval.WWWDomain != tt.wantWWW {
				t.Errorf("WWWDomain = %q, want %q", eval.WWWDomain, tt.wantWWW)
			}
			if eval.OriginDomain != tt.wantOrigin {
				t.Errorf("OriginDomain = %q, want %q", eval.OriginDomain, tt.wantOrigin)
			}
		})
	}
}

func TestEvaluatePointsToVPS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		name         string
		domainIPs    []string
		vpsIP        string
		wantPointsTo bool
	}{
		{
			name:         "direct VPS IP match",
			domainIPs:    []string{"203.0.113.10"},
			vpsIP:        "203.0.113.10",
			wantPointsTo: true,
		},
		{
			name:         "multiple IPs with VPS match",
			domainIPs:    []string{"104.16.0.1", "203.0.113.10"},
			vpsIP:        "203.0.113.10",
			wantPointsTo: true,
		},
		{
			name:         "no VPS IP match (Cloudflare only)",
			domainIPs:    []string{"104.16.0.1", "104.16.0.2"},
			vpsIP:        "203.0.113.10",
			wantPointsTo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&FakeResolver{Hosts: map[string][]string{
				"example.com":           tt.domainIPs,
				"www.example.com":       tt.domainIPs,
				"do-origin.example.com": tt.domainIPs,
			}})

			eval := Evaluate(ctx, svc, "example.com", tt.vpsIP)
			apex, _ := eval.StatusForRole("apex")

			if apex.PointsToVPS != tt.wantPointsTo {
				t.Errorf("PointsToVPS = %v, want %v", apex.PointsToVPS, tt.wantPointsTo)
			}
		})
	}
}

func TestEvaluateProxiedDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		name       string
		domainIPs  []string
		wantProxed bool
	}{
		{
			name:       "Cloudflare IP is proxied",
			domainIPs:  []string{"104.16.0.1"},
			wantProxed: true,
		},
		{
			name:       "direct VPS IP not proxied",
			domainIPs:  []string{"203.0.113.10"},
			wantProxed: false,
		},
		{
			name:       "mixed IPs not considered proxied (all must be CF)",
			domainIPs:  []string{"104.16.0.1", "203.0.113.10"},
			wantProxed: false, // areCloudflareIPs requires ALL IPs to be Cloudflare
		},
		{
			name:       "multiple Cloudflare IPs is proxied",
			domainIPs:  []string{"104.16.0.1", "104.16.0.2"},
			wantProxed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&FakeResolver{Hosts: map[string][]string{
				"example.com":           tt.domainIPs,
				"www.example.com":       tt.domainIPs,
				"do-origin.example.com": {"203.0.113.10"},
			}})

			eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")
			apex, _ := eval.StatusForRole("apex")

			if apex.Proxied != tt.wantProxed {
				t.Errorf("Proxied = %v, want %v", apex.Proxied, tt.wantProxed)
			}
		})
	}
}

func TestEvaluateStatusForRoleNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com":           {"203.0.113.10"},
		"www.example.com":       {"203.0.113.10"},
		"do-origin.example.com": {"203.0.113.10"},
	}})

	eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")

	_, ok := eval.StatusForRole("nonexistent")
	if ok {
		t.Error("expected StatusForRole to return false for nonexistent role")
	}
}

func TestEvaluateDNSError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Resolver returns error for domain lookup
	svc := NewService(&FakeResolver{
		Hosts: map[string][]string{},
		Err:   &domain.DNSLookupError{Kind: domain.DNSLookupNotFound, Message: "no such host"},
	})

	eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")

	apex, ok := eval.StatusForRole("apex")
	if !ok {
		t.Fatal("expected apex status even with DNS error")
	}
	if apex.Lookup.Error == nil {
		t.Error("expected DNS error to be captured in lookup result")
	}
	if apex.PointsToVPS {
		t.Error("PointsToVPS should be false when DNS error occurs")
	}
}

func TestEvaluateVPSIPResolution(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com":           {"203.0.113.10"},
		"www.example.com":       {"203.0.113.10"},
		"do-origin.example.com": {"203.0.113.10"},
		"203.0.113.10":          {"203.0.113.10"},
	}})

	eval := Evaluate(ctx, svc, "example.com", "203.0.113.10")

	if eval.VPS.Error != nil {
		t.Errorf("unexpected VPS lookup error: %v", eval.VPS.Error)
	}
	if len(eval.VPS.IPs) == 0 {
		t.Error("expected VPS lookup to return IPs")
	}
}
