package dns

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/domain"
)

func TestDNSServiceResolveHostNormalizes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"example.com": {" 203.0.113.10", "203.0.113.10", "198.51.100.1"},
	}})

	result := svc.ResolveHost(ctx, "Example.com.")
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	expected := []string{"198.51.100.1", "203.0.113.10"}
	if !reflect.DeepEqual(result.IPs, expected) {
		t.Fatalf("expected %v, got %v", expected, result.IPs)
	}
}

func TestDNSServiceCheckDomainReachabilityNXDomain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Err: &net.DNSError{IsNotFound: true}})
	result := svc.CheckDomainReachability(ctx, "missing.example")

	if result.Reachable {
		t.Fatalf("expected unreachable, got reachable")
	}
	if !strings.Contains(result.Message, "NXDOMAIN") {
		t.Fatalf("expected NXDOMAIN message, got %q", result.Message)
	}
	if !strings.Contains(result.Hint, "DNS is not configured") {
		t.Fatalf("expected hint about DNS not configured, got %q", result.Hint)
	}
	if result.ErrorKind != domain.DNSLookupNotFound {
		t.Fatalf("expected not_found error kind, got %q", result.ErrorKind)
	}
}

func TestDNSServiceCheckDomainReachabilityTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Err: &net.DNSError{IsTimeout: true}})
	result := svc.CheckDomainReachability(ctx, "timeout.example")

	if result.Reachable {
		t.Fatalf("expected unreachable, got reachable")
	}
	if result.ErrorKind != domain.DNSLookupTimeout {
		t.Fatalf("expected timeout error kind, got %q", result.ErrorKind)
	}
	if !strings.Contains(result.Message, "timed out") {
		t.Fatalf("expected timeout message, got %q", result.Message)
	}
}

func TestDNSServiceCompareDomainToVPSMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{Hosts: map[string][]string{
		"vps.example.com": {"203.0.113.10"},
		"example.com":     {"198.51.100.5"},
	}})

	result := svc.CompareDomainToVPS(ctx, "example.com", "vps.example.com")

	if result.VPS.Error != nil || result.Domain.Error != nil {
		t.Fatalf("expected successful lookups, got: %+v", result)
	}
	if result.PointsToVPS {
		t.Fatalf("expected mismatch, got points_to_vps=true")
	}
}

func TestDNSServiceCompareDomainToVPSDomainResolveError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{
		Hosts: map[string][]string{
			"vps.example.com": {"203.0.113.10"},
		},
		Errs: map[string]error{
			"example.com": &net.DNSError{IsNotFound: true},
		},
	})

	result := svc.CompareDomainToVPS(ctx, "example.com", "vps.example.com")

	if result.Domain.Error == nil {
		t.Fatalf("expected domain error on lookup failure")
	}
	if result.Domain.Error.Kind != domain.DNSLookupNotFound {
		t.Fatalf("expected not_found error kind, got %q", result.Domain.Error.Kind)
	}
}

func TestDNSServiceResolveHostIPLiteral(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	svc := NewService(&FakeResolver{})
	result := svc.ResolveHost(ctx, "2001:db8::1")

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if len(result.IPs) != 1 || result.IPs[0] != "2001:db8::1" {
		t.Fatalf("expected IPv6 literal normalization, got %v", result.IPs)
	}
}
