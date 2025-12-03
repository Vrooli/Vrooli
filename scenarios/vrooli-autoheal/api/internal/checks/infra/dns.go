// Package infra provides infrastructure health checks
// [REQ:INFRA-DNS-001]
package infra

import (
	"context"
	"net"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DNSCheck verifies DNS resolution.
// Domain is required - operational defaults should be set by the bootstrap layer.
type DNSCheck struct {
	domain string
}

// NewDNSCheck creates a DNS resolution check.
// The domain parameter is required (e.g., "google.com").
func NewDNSCheck(domain string) *DNSCheck {
	return &DNSCheck{domain: domain}
}

func (c *DNSCheck) ID() string                  { return "infra-dns" }
func (c *DNSCheck) Description() string         { return "DNS resolution check" }
func (c *DNSCheck) IntervalSeconds() int        { return 30 }
func (c *DNSCheck) Platforms() []platform.Type  { return nil }

func (c *DNSCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{"domain": c.domain},
	}

	ips, err := net.LookupIP(c.domain)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "DNS resolution failed"
		result.Details["error"] = err.Error()
		return result
	}

	result.Status = checks.StatusOK
	result.Message = "DNS resolution OK"
	result.Details["resolved"] = len(ips)
	return result
}
