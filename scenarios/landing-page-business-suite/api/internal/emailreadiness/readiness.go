// Package emailreadiness performs read-only deliverability checks. It never
// mutates DNS or provider configuration.
package emailreadiness

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Resolver interface {
	LookupTXT(context.Context, string) ([]string, error)
	LookupCNAME(context.Context, string) (string, error)
}

type Check struct{ Name, Status, Detail string }
type Report struct {
	Checks    []Check
	CheckedAt time.Time
}

type Service struct {
	resolver Resolver
	now      func() time.Time
	mu       sync.Mutex
	cache    Report
	cachedAt time.Time
	ttl      time.Duration
}

func New(resolver Resolver) *Service {
	return &Service{resolver: resolver, now: time.Now, ttl: time.Hour}
}
func (s *Service) UseClock(now func() time.Time) { s.now = now }

func (s *Service) Evaluate(ctx context.Context, fromEmail, publicBaseURL string, sendgridConfigured bool, smtpSelector string, webhookConfigured bool, webhookLastEvent time.Time) Report {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	now := s.now().UTC()
	s.mu.Lock()
	if !s.cachedAt.IsZero() && now.Sub(s.cachedAt) < s.ttl {
		report := s.cache
		s.mu.Unlock()
		return report
	}
	s.mu.Unlock()
	domain := emailDomain(fromEmail)
	org := registrableDomain(publicBaseURL)
	checks := []Check{
		{Name: "from_domain_configured", Status: "pass", Detail: "From address is configured"},
		{Name: "alignment", Status: "pass", Detail: "From domain aligns with the public origin"},
	}
	if domain == "" || domain == "example.com" {
		checks[0] = Check{"from_domain_configured", "fail", "EMAIL_FROM_ADDRESS must use a real sending domain"}
	}
	if org == "" || !sameOrSubdomain(domain, org) {
		checks[1] = Check{"alignment", "fail", "From domain must equal or be below the public origin domain"}
	}
	spf := s.txt(ctx, domain, "spf")
	if len(spf) != 1 {
		checks = append(checks, Check{"spf", "fail", "From domain must publish exactly one SPF record"})
	} else if sendgridConfigured && !strings.Contains(strings.ToLower(spf[0]), "include:sendgrid.net") {
		checks = append(checks, Check{"spf", "fail", "SPF must include sendgrid.net when SendGrid is configured"})
	} else {
		checks = append(checks, Check{"spf", "pass", "SPF record is present"})
	}
	if domain == "" {
		checks = append(checks, Check{"dkim", "fail", "No From domain is available for DKIM checks"})
	} else if smtpSelector != "" {
		checks = append(checks, s.cname(ctx, smtpSelector+"._domainkey."+domain, "dkim"))
	} else {
		checks = append(checks, s.cnamePair(ctx, domain))
	}
	dmarc := s.txt(ctx, "_dmarc."+org, "dmarc")
	dmarcCheck := Check{"dmarc", "fail", "DMARC v=DMARC1 record is missing"}
	for _, record := range dmarc {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(record)), "V=DMARC1") {
			dmarcCheck = Check{"dmarc", "pass", "DMARC is published"}
			if strings.Contains(strings.ToLower(record), "p=none") {
				dmarcCheck.Status = "warn"
				dmarcCheck.Detail = "DMARC is in monitoring mode (p=none)"
			}
			break
		}
	}
	checks = append(checks, dmarcCheck)
	webhook := Check{"event_webhook", "pass", "Signed Event Webhook is configured and recently received an event"}
	if !webhookConfigured {
		webhook = Check{"event_webhook", "warn", "SendGrid webhook verification key is not configured"}
	} else if webhookLastEvent.IsZero() || now.Sub(webhookLastEvent) > 7*24*time.Hour {
		webhook = Check{"event_webhook", "warn", "No signed delivery event received in the last 7 days"}
	}
	checks = append(checks, webhook)
	report := Report{Checks: checks, CheckedAt: now}
	s.mu.Lock()
	s.cache, s.cachedAt = report, now
	s.mu.Unlock()
	return report
}

func (s *Service) txt(ctx context.Context, name, _ string) []string {
	if s.resolver == nil {
		return nil
	}
	values, _ := s.resolver.LookupTXT(ctx, name)
	return values
}
func (s *Service) cname(ctx context.Context, name, label string) Check {
	if s.resolver == nil {
		return Check{label, "fail", "DNS resolver unavailable"}
	}
	value, err := s.resolver.LookupCNAME(ctx, name)
	if err != nil || strings.TrimSpace(value) == "" {
		return Check{label, "fail", "DKIM selector does not resolve"}
	}
	return Check{label, "pass", "DKIM selector resolves"}
}
func (s *Service) cnamePair(ctx context.Context, domain string) Check {
	for _, selector := range []string{"s1", "s2"} {
		if s.resolver == nil {
			return Check{"dkim", "fail", "DNS resolver unavailable"}
		}
		value, err := s.resolver.LookupCNAME(ctx, selector+"._domainkey."+domain)
		if err != nil || strings.TrimSpace(value) == "" {
			return Check{"dkim", "fail", selector + "._domainkey does not resolve"}
		}
	}
	return Check{"dkim", "pass", "SendGrid DKIM selectors resolve"}
}
func emailDomain(address string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(address)), "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSuffix(parts[1], ".")
}
func registrableDomain(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	return strings.Join(parts[len(parts)-2:], ".")
}
func sameOrSubdomain(domain, org string) bool {
	return domain == org || strings.HasSuffix(domain, "."+org)
}
