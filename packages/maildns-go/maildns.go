// Package maildns verifies provider-specific mail authorization without
// writing DNS. It is deliberately independent of any scenario or provider SDK
// so deployment preflight and runtime health use the same verdicts.
package maildns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Resolver interface {
	LookupTXT(context.Context, string) ([]string, error)
	LookupCNAME(context.Context, string) (string, error)
}

type DKIMRequirement struct {
	Selector string
	Type     string // TXT or CNAME
	Expected string // provider-owned key or target suffix
}

type ProviderRequirement struct {
	Name         string
	SPFMechanism string
	SPFHost      string
	DKIM         []DKIMRequirement
}

type Status string

const (
	Pass Status = "pass"
	Warn Status = "warn"
	Fail Status = "fail"
)

type Verdict struct {
	Provider string
	Status   Status
	Detail   string
	Record   string
	Cost     int
}

type Report struct {
	Domain    string
	Providers []Verdict
	DMARC     Verdict
	CheckedAt time.Time
}

type Service struct {
	resolver Resolver
	now      func() time.Time
	ttl      time.Duration
	mu       sync.Mutex
	cache    map[string]cachedReport
}

type cachedReport struct {
	report Report
	at     time.Time
}

func New(resolver Resolver) *Service {
	return &Service{resolver: resolver, now: time.Now, ttl: 3 * time.Minute, cache: make(map[string]cachedReport)}
}

func (s *Service) UseClock(now func() time.Time) { s.now = now }

func (s *Service) Verify(ctx context.Context, domain string, requirements []ProviderRequirement) Report {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	key := domain + "|"
	for _, req := range requirements {
		key += req.Name + ":" + req.SPFMechanism + ":" + req.SPFHost + ";"
		for _, d := range req.DKIM {
			key += d.Selector + ":" + d.Type + ":" + d.Expected + ","
		}
	}
	now := s.now().UTC()
	s.mu.Lock()
	if item, ok := s.cache[key]; ok && now.Sub(item.at) < s.ttl {
		s.mu.Unlock()
		return item.report
	}
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	spf, cost := s.verifySPF(ctx, domain, "", "")
	report := Report{Domain: domain, CheckedAt: now, DMARC: s.verifyDMARC(ctx, domain)}
	for _, req := range requirements {
		v := spf
		v.Provider = req.Name
		v.Cost = cost
		if v.Status == Pass && req.SPFMechanism != "" && !containsMechanism(spf.Detail, req.SPFMechanism) {
			v.Status, v.Detail, v.Record = Fail, "SPF does not authorize the required provider mechanism", "TXT "+domain+" = v=spf1 "+req.SPFMechanism+" ..."
		}
		if v.Status == Pass {
			for _, dkim := range req.DKIM {
				dv := s.verifyDKIM(ctx, domain, dkim)
				if dv.Status != Pass {
					v.Status, v.Detail, v.Record = dv.Status, dv.Detail, dv.Record
					break
				}
			}
		}
		report.Providers = append(report.Providers, v)
	}
	s.mu.Lock()
	s.cache[key] = cachedReport{report: report, at: now}
	s.mu.Unlock()
	return report
}

func (s *Service) verifySPF(ctx context.Context, domain, _, _ string) (Verdict, int) {
	if s.resolver == nil {
		return Verdict{Status: Fail, Detail: "DNS resolver unavailable"}, 0
	}
	records, err := s.resolver.LookupTXT(ctx, domain)
	if err != nil {
		return Verdict{Status: Fail, Detail: "SPF record lookup failed: " + err.Error(), Record: "TXT " + domain + " = v=spf1 ..."}, 0
	}
	var spf []string
	for _, record := range records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(record)), "v=spf1") {
			spf = append(spf, record)
		}
	}
	if len(spf) != 1 {
		return Verdict{Status: Fail, Detail: fmt.Sprintf("expected exactly one SPF record, found %d", len(spf)), Record: "TXT " + domain + " = v=spf1 ..."}, 0
	}
	cost, err := s.spfCost(ctx, spf[0], map[string]bool{domain: true})
	if err != nil {
		return Verdict{Status: Fail, Detail: err.Error(), Record: "TXT " + domain + " = " + spf[0]}, cost
	}
	if cost > 10 {
		return Verdict{Status: Fail, Detail: "SPF lookup cost exceeds 10 (cost " + strconv.Itoa(cost) + ")", Record: "TXT " + domain + " = " + spf[0]}, cost
	}
	return Verdict{Status: Pass, Detail: spf[0], Record: "TXT " + domain + " = " + spf[0]}, cost
}

func (s *Service) spfCost(ctx context.Context, record string, seen map[string]bool) (int, error) {
	cost := 0
	for _, mechanism := range strings.Fields(record)[1:] {
		name := strings.TrimPrefix(strings.TrimPrefix(mechanism, "-"), "~")
		name = strings.TrimPrefix(strings.TrimPrefix(name, "+"), "?")
		if !strings.HasPrefix(name, "include:") && !strings.HasPrefix(name, "redirect=") {
			continue
		}
		cost++
		host := strings.TrimPrefix(strings.TrimPrefix(name, "include:"), "redirect=")
		if seen[host] {
			return cost, fmt.Errorf("SPF include loop at %s", host)
		}
		seen[host] = true
		records, err := s.resolver.LookupTXT(ctx, host)
		if err != nil {
			return cost, fmt.Errorf("SPF include %s lookup failed: %w", host, err)
		}
		for _, nested := range records {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(nested)), "v=spf1") {
				nestedCost, err := s.spfCost(ctx, nested, seen)
				cost += nestedCost
				if err != nil {
					return cost, err
				}
				break
			}
		}
	}
	return cost, nil
}

func (s *Service) verifyDKIM(ctx context.Context, domain string, req DKIMRequirement) Verdict {
	host := req.Selector + "._domainkey." + domain
	if s.resolver == nil {
		return Verdict{Status: Fail, Detail: "DNS resolver unavailable", Record: "TXT " + host + " = " + req.Expected}
	}
	switch strings.ToUpper(req.Type) {
	case "TXT":
		records, err := s.resolver.LookupTXT(ctx, host)
		if err != nil {
			return Verdict{Status: Fail, Detail: "DKIM TXT lookup failed", Record: "TXT " + host + " = " + req.Expected}
		}
		for _, record := range records {
			if req.Expected == "" || strings.Contains(strings.ToLower(record), strings.ToLower(req.Expected)) {
				return Verdict{Status: Pass, Detail: "DKIM TXT selector is published"}
			}
		}
		return Verdict{Status: Fail, Detail: "DKIM TXT selector is missing or belongs to another provider", Record: "TXT " + host + " = " + req.Expected}
	case "CNAME":
		value, err := s.resolver.LookupCNAME(ctx, host)
		if err == nil && (req.Expected == "" || strings.Contains(strings.ToLower(value), strings.ToLower(req.Expected))) {
			return Verdict{Status: Pass, Detail: "DKIM CNAME selector is published"}
		}
		return Verdict{Status: Fail, Detail: "DKIM CNAME selector is missing or belongs to another provider", Record: "CNAME " + host + " = " + req.Expected}
	default:
		return Verdict{Status: Fail, Detail: "unsupported DKIM record type " + req.Type, Record: "TXT " + host + " = " + req.Expected}
	}
}

func (s *Service) verifyDMARC(ctx context.Context, domain string) Verdict {
	if s.resolver == nil {
		return Verdict{Status: Fail, Detail: "DNS resolver unavailable", Record: "TXT _dmarc." + domain + " = v=DMARC1; p=reject"}
	}
	host := "_dmarc." + domain
	records, err := s.resolver.LookupTXT(ctx, host)
	if err != nil {
		return Verdict{Status: Fail, Detail: "DMARC record is missing", Record: "TXT " + host + " = v=DMARC1; p=reject"}
	}
	for _, record := range records {
		lower := strings.ToLower(record)
		if strings.HasPrefix(strings.TrimSpace(lower), "v=dmarc1") {
			status := Pass
			if strings.Contains(lower, "p=none") {
				status = Warn
			}
			return Verdict{Status: status, Detail: record}
		}
	}
	return Verdict{Status: Fail, Detail: "DMARC record is missing", Record: "TXT " + host + " = v=DMARC1; p=reject"}
}

func containsMechanism(spf, mechanism string) bool {
	return strings.Contains(strings.ToLower(spf), strings.ToLower(mechanism))
}

// NetResolver adapts net.Resolver to the verifier seam.
type NetResolver struct{ Resolver *net.Resolver }

func (r NetResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
	return r.Resolver.LookupTXT(ctx, host)
}
func (r NetResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return r.Resolver.LookupCNAME(ctx, host)
}
