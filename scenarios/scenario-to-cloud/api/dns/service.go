package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// Service centralizes DNS lookups and comparisons for scenario-to-cloud.
type Service interface {
	ResolveHost(ctx context.Context, host string) domain.DNSLookupResult
	CheckDomainReachability(ctx context.Context, domain string) domain.ReachabilityResult
	CompareDomainToVPS(ctx context.Context, domain, vpsHost string) domain.DNSComparisonResult
}

// Resolver abstracts DNS lookups for testing.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Config controls DNS service behavior.
type Config struct {
	Timeout time.Duration
}

// Option applies configuration to Service.
type Option func(*Config)

// WithTimeout sets a default DNS lookup timeout when no deadline is provided.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// DefaultService implements Service using the provided resolver.
type DefaultService struct {
	resolver Resolver
	config   Config
}

// NetResolver resolves DNS via the system resolver.
type NetResolver struct{}

func (NetResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// NewService returns a Service backed by the given resolver.
func NewService(resolver Resolver, opts ...Option) *DefaultService {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &DefaultService{
		resolver: resolver,
		config:   cfg,
	}
}

func (s *DefaultService) ResolveHost(ctx context.Context, host string) domain.DNSLookupResult {
	host = normalizeHost(host)
	if host == "" {
		return domain.DNSLookupResult{
			Host: host,
			Error: &domain.DNSLookupError{
				Kind:    domain.DNSLookupInvalidHost,
				Message: "host is empty",
			},
		}
	}
	if isIPLiteral(host) {
		ip := net.ParseIP(host)
		if ip == nil {
			return domain.DNSLookupResult{
				Host: host,
				Error: &domain.DNSLookupError{
					Kind:    domain.DNSLookupInvalidHost,
					Message: "host is not a valid IP address",
				},
			}
		}
		return domain.DNSLookupResult{Host: host, IPs: []string{ip.String()}}
	}
	ctx, cancel := s.applyTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	ips, err := s.resolver.LookupHost(ctx, host)
	if err != nil {
		return domain.DNSLookupResult{
			Host:  host,
			Error: lookupErrorFrom(err),
		}
	}
	return domain.DNSLookupResult{
		Host: host,
		IPs:  normalizeIPs(ips),
	}
}

func (s *DefaultService) CheckDomainReachability(ctx context.Context, domainName string) domain.ReachabilityResult {
	result := domain.ReachabilityResult{
		Target: domainName,
		Type:   "domain",
	}

	lookup := s.ResolveHost(ctx, domainName)
	if lookup.Error != nil {
		result.Reachable = false
		result.ErrorKind = lookup.Error.Kind
		switch lookup.Error.Kind {
		case domain.DNSLookupNotFound:
			result.Message = "Domain does not resolve (NXDOMAIN)"
			result.Hint = "DNS is not configured for this domain yet. You can proceed if you plan to configure DNS before deployment."
		case domain.DNSLookupTimeout:
			result.Message = "DNS lookup timed out"
			result.Hint = "Unable to verify DNS. You can proceed if you're confident the domain is configured correctly."
		case domain.DNSLookupInvalidHost:
			result.Message = "Invalid domain"
			result.Hint = "Check the domain formatting. You can proceed if DNS will be configured later."
		default:
			result.Message = "DNS lookup failed"
			result.Hint = "Check that the domain is valid. You can proceed if DNS will be configured later."
		}
		return result
	}

	result.Reachable = true
	result.Message = "Domain resolves to " + strings.Join(lookup.IPs, ", ")
	return result
}

func (s *DefaultService) CompareDomainToVPS(ctx context.Context, domainName, vpsHost string) domain.DNSComparisonResult {
	vpsLookup := s.ResolveHost(ctx, vpsHost)
	domainLookup := s.ResolveHost(ctx, domainName)
	pointsToVPS := false
	if vpsLookup.Error == nil && domainLookup.Error == nil {
		pointsToVPS = intersects(vpsLookup.IPs, domainLookup.IPs)
	}
	return domain.DNSComparisonResult{
		Domain:      domainLookup,
		VPS:         vpsLookup,
		PointsToVPS: pointsToVPS,
	}
}

func (s *DefaultService) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.config.Timeout <= 0 {
		return ctx, nil
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, nil
	}
	return context.WithTimeout(ctx, s.config.Timeout)
}

func lookupErrorFrom(err error) *domain.DNSLookupError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &domain.DNSLookupError{Kind: domain.DNSLookupTimeout, Message: err.Error()}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &domain.DNSLookupError{Kind: domain.DNSLookupTimeout, Message: err.Error()}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return &domain.DNSLookupError{Kind: domain.DNSLookupNotFound, Message: dnsErr.Error()}
		}
		if dnsErr.IsTimeout {
			return &domain.DNSLookupError{Kind: domain.DNSLookupTimeout, Message: dnsErr.Error()}
		}
		return &domain.DNSLookupError{Kind: domain.DNSLookupUnknown, Message: dnsErr.Error()}
	}
	return &domain.DNSLookupError{Kind: domain.DNSLookupUnknown, Message: err.Error()}
}

func BuildARecordHint(domainName, targetIP string) (string, domain.DNSARecordHint) {
	hintData := domain.DNSARecordHint{
		Domain:          domainName,
		TargetIP:        targetIP,
		Providers:       []string{"Cloudflare", "Namecheap", "GoDaddy", "DigitalOcean"},
		PropagationNote: "DNS changes can take up to 48 hours to propagate (usually 5-30 minutes).",
	}
	hint := fmt.Sprintf(
		"Update DNS A record for %s to point to %s.\n\n"+
			"Common DNS providers:\n"+
			"- Cloudflare: DNS > Records > Add A record with name '@' or subdomain, IPv4 address %s\n"+
			"- Namecheap: Domain List > Manage > Advanced DNS > Add A Record\n"+
			"- GoDaddy: DNS Management > Add > Type: A, Points to: %s\n"+
			"- DigitalOcean: Networking > Domains > Add A record\n\n"+
			"%s",
		domainName, targetIP, targetIP, targetIP, hintData.PropagationNote,
	)
	return hint, hintData
}

func normalizeIPs(ips []string) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if parsed := net.ParseIP(ip); parsed != nil {
			ip = parsed.String()
		}
		out = append(out, ip)
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	last := ""
	for i, v := range in {
		if i == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}

func intersects(a, b []string) bool {
	set := map[string]struct{}{}
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func isIPLiteral(host string) bool {
	return net.ParseIP(strings.TrimSpace(host)) != nil
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimSuffix(host, ".")
	return strings.ToLower(host)
}
