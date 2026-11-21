package dns

import (
	"context"
	"fmt"
	"strings"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"scenario-to-cloud/domain"

	"golang.org/x/net/publicsuffix"
)

// PreflightChecks evaluates DNS validation checks for a deployment preflight.
func PreflightChecks(ctx context.Context, svc Service, domainName, vpsHost string, policy domain.DNSPolicy) []domain.PreflightCheck {
	eval := Evaluate(ctx, svc, domainName, vpsHost)
	return PreflightChecksFromEvaluation(eval, policy)
}

// MailDNSChecks verifies reviewed provider requirements without forcing
// web-only deployments to declare a mail configuration.
func MailDNSChecks(ctx context.Context, domainName string, requirements []maildns.ProviderRequirement, policy domain.DNSPolicy) []domain.PreflightCheck {
	if len(requirements) == 0 {
		return nil
	}
	return MailDNSChecksFromReport(VerifyMailDNS(ctx, domainName, requirements), policy)
}

// MailDNSChecksFromReport converts a verified mail-DNS report into deployment
// checks. Keeping this mapping separate makes the required-versus-warn policy
// explicit and testable without depending on the live DNS resolver.
func MailDNSChecksFromReport(report maildns.Report, policy domain.DNSPolicy) []domain.PreflightCheck {
	checks := make([]domain.PreflightCheck, 0, len(report.Providers))
	for _, verdict := range report.Providers {
		status := domain.PreflightFail
		if verdict.Status == maildns.Pass {
			status = domain.PreflightPass
		} else if policy != domain.DNSPolicyRequired {
			status = domain.PreflightWarn
		}
		checks = append(checks, domain.PreflightCheck{
			ID: domain.PreflightDNSMailID, Title: "Mail DNS authorization", Status: status,
			Details: fmt.Sprintf("%s: %s", verdict.Provider, verdict.Detail),
			Hint:    "Publish the provider's SPF/DKIM records, then rerun preflight.",
			Data:    map[string]string{"provider": verdict.Provider, "record": verdict.Record, "lookup_cost": fmt.Sprint(verdict.Cost)},
		})
	}
	return checks
}

// PreflightChecksFromEvaluation maps a DNS evaluation to preflight checks.
func PreflightChecksFromEvaluation(eval Evaluation, policy domain.DNSPolicy) []domain.PreflightCheck {
	builder := preflightCheckBuilder{checks: make([]domain.PreflightCheck, 0, 5), policy: policy}
	if policy == domain.DNSPolicySkip {
		builder.appendSkippedChecks()
		return builder.checks
	}
	builder.appendVPSHost(eval)
	builder.appendDomainStatuses(eval)
	builder.appendEdgeIPv6(eval)
	builder.appendOGWorker(eval)
	return builder.checks
}

type preflightCheckBuilder struct {
	checks []domain.PreflightCheck
	policy domain.DNSPolicy
}

func (b *preflightCheckBuilder) fail(id, title, details, hint string, data map[string]string) {
	b.checks = append(b.checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightFail, Details: details, Hint: hint, Data: data})
}

func (b *preflightCheckBuilder) pass(id, title, details string, data map[string]string) {
	b.checks = append(b.checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightPass, Details: details, Data: data})
}

func (b *preflightCheckBuilder) warn(id, title, details, hint string, data map[string]string) {
	b.checks = append(b.checks, domain.PreflightCheck{ID: id, Title: title, Status: domain.PreflightWarn, Details: details, Hint: hint, Data: data})
}

func (b *preflightCheckBuilder) issue(id, title, details, hint string, data map[string]string) {
	if b.policy == domain.DNSPolicyWarn {
		b.warn(id, title, details, hint, data)
		return
	}
	b.fail(id, title, details, hint, data)
}

func (b *preflightCheckBuilder) appendSkippedChecks() {
	hint := "DNS checks skipped by policy. Set edge.dns_policy to required to enforce DNS validation."
	b.warn(domain.PreflightDNSVPSHostID, "Resolve VPS host", "DNS checks skipped by policy", hint, nil)
	b.warn(domain.PreflightDNSEdgeApexID, "Apex domain", "DNS checks skipped by policy", hint, nil)
	b.warn(domain.PreflightDNSEdgeWWWID, "WWW domain", "DNS checks skipped by policy", hint, nil)
	b.warn(domain.PreflightDNSDoOriginID, "Origin domain", "DNS checks skipped by policy", hint, nil)
	b.warn(domain.PreflightDNSOGWorkerID, "OG worker readiness", "DNS checks skipped by policy", hint, nil)
	b.warn(domain.PreflightDNSEdgeIPv6ID, "Edge IPv6 records", "DNS checks skipped by policy", hint, nil)
}

func (b *preflightCheckBuilder) appendVPSHost(eval Evaluation) {
	if eval.VPS.Error != nil {
		b.issue(domain.PreflightDNSVPSHostID, "Resolve VPS host", fmt.Sprintf("Unable to resolve VPS host %q", eval.VPS.Host), eval.VPS.Error.Message, nil)
		return
	}
	b.pass(domain.PreflightDNSVPSHostID, "Resolve VPS host", "Resolved VPS host", map[string]string{"ips": strings.Join(eval.VPS.IPs, ",")})
}

func (b *preflightCheckBuilder) appendDomainStatuses(eval Evaluation) {
	for _, status := range eval.Statuses {
		b.appendDomainStatus(eval, status)
	}
}

func (b *preflightCheckBuilder) appendDomainStatus(eval Evaluation, status DomainStatus) {
	id, title, ok := preflightIdentityForRole(status.Role)
	if !ok {
		return
	}
	if status.Lookup.Error != nil {
		b.issue(id, title, fmt.Sprintf("Unable to resolve %s (%q)", title, status.Lookup.Host), status.Lookup.Error.Message, nil)
		return
	}
	if status.AllowProxy && status.Proxied {
		b.pass(id, title, fmt.Sprintf("%s resolves to Cloudflare proxy", status.Lookup.Host), map[string]string{"ips": strings.Join(status.Lookup.IPs, ","), "proxied": "cloudflare"})
		return
	}
	if eval.VPS.Error != nil {
		b.issue(id, title, fmt.Sprintf("Unable to verify %s (VPS host unresolved)", status.Lookup.Host), eval.VPS.Error.Message, nil)
		return
	}
	if status.PointsToVPS {
		b.pass(id, title, fmt.Sprintf("%s resolves to the VPS", status.Lookup.Host), map[string]string{"ips": strings.Join(status.Lookup.IPs, ","), "vps_ips": strings.Join(eval.VPS.IPs, ",")})
		return
	}
	hint := ""
	if len(eval.VPS.IPs) > 0 {
		hint, _ = BuildARecordHint(status.Lookup.Host, eval.VPS.IPs[0])
	}
	b.issue(id, title, fmt.Sprintf("%s resolves to %s, not your VPS (%s)", status.Lookup.Host, strings.Join(status.Lookup.IPs, ", "), strings.Join(eval.VPS.IPs, ", ")), hint, map[string]string{"vps_ips": strings.Join(eval.VPS.IPs, ","), "edge_ips": strings.Join(status.Lookup.IPs, ","), "domain": status.Lookup.Host})
}

func preflightIdentityForRole(role string) (string, string, bool) {
	switch role {
	case "apex":
		return domain.PreflightDNSEdgeApexID, "Apex domain", true
	case "www":
		return domain.PreflightDNSEdgeWWWID, "WWW domain", true
	case "origin":
		return domain.PreflightDNSDoOriginID, "Origin domain", true
	case "edge":
		return domain.PreflightDNSEdgeDomainID, "Edge domain", true
	default:
		return "", "", false
	}
}

func (b *preflightCheckBuilder) appendEdgeIPv6(eval Evaluation) {
	issues := []string{}
	data := map[string]string{}
	vpsIPv6 := filterIPv6(eval.VPS.IPs)
	for _, status := range eval.Statuses {
		if status.Role != "apex" && status.Role != "www" || status.Proxied {
			continue
		}
		edgeIPv6 := filterIPv6(status.Lookup.IPs)
		if len(edgeIPv6) == 0 || eval.VPS.Error == nil && len(vpsIPv6) > 0 && intersects(edgeIPv6, vpsIPv6) {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s -> %s", status.Host, strings.Join(edgeIPv6, ", ")))
		data[status.Host] = strings.Join(edgeIPv6, ",")
	}
	if len(issues) == 0 {
		b.pass(domain.PreflightDNSEdgeIPv6ID, "Edge IPv6 records", "No mismatched IPv6 records detected", nil)
		return
	}
	b.warn(domain.PreflightDNSEdgeIPv6ID, "Edge IPv6 records", fmt.Sprintf("IPv6 records detected without matching VPS IPv6: %s", strings.Join(issues, " | ")), "Remove AAAA records or configure IPv6 on the VPS to match. Clients preferring IPv6 may time out if AAAA points to a non-serving host.", data)
}

func (b *preflightCheckBuilder) appendOGWorker(eval Evaluation) {
	hint := "To enable OG worker routing, proxy both apex and www through Cloudflare and point do-origin to the VPS (DNS-only A record)."
	apexStatus, _ := eval.StatusForRole("apex")
	wwwStatus, hasWWW := eval.StatusForRole("www")
	doOriginStatus, _ := eval.StatusForRole("origin")
	apexProxied := apexStatus.Proxied
	wwwProxied := hasWWW && wwwStatus.Proxied
	if eval.WWWDomain == eval.ApexDomain {
		wwwProxied = apexProxied
	}
	if !apexProxied || !wwwProxied {
		b.warn(domain.PreflightDNSOGWorkerID, "OG worker readiness", "Apex and www should be proxied through Cloudflare for OG routing", hint, nil)
		return
	}
	if doOriginStatus.Lookup.Error != nil || !doOriginStatus.PointsToVPS {
		b.warn(domain.PreflightDNSOGWorkerID, "OG worker readiness", fmt.Sprintf("%s should point to the VPS for OG routing", doOriginStatus.Host), hint, map[string]string{"domain": doOriginStatus.Host})
		return
	}
	b.pass(domain.PreflightDNSOGWorkerID, "OG worker readiness", "OG worker routing prerequisites look good", nil)
}

func filterIPv6(ips []string) []string {
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			out = append(out, ip)
		}
	}
	return out
}

// ProxyModeCheck evaluates Cloudflare proxy mode vs ACME issuance readiness.
func ProxyModeCheck(eval Evaluation, policy domain.DNSPolicy, dns01Token string) domain.PreflightCheck {
	proxiedDomains := []string{}
	for _, status := range eval.Statuses {
		if status.Role != "apex" && status.Role != "www" && status.Role != "edge" {
			continue
		}
		if status.Proxied {
			proxiedDomains = append(proxiedDomains, status.Host)
		}
	}

	build := func(status domain.PreflightCheckStatus, details, hint string, data map[string]string) domain.PreflightCheck {
		return domain.PreflightCheck{
			ID:      domain.PreflightDNSProxyModeID,
			Title:   "Proxy mode vs issuance",
			Status:  status,
			Details: details,
			Hint:    hint,
			Data:    data,
		}
	}

	if policy == domain.DNSPolicySkip {
		return build(
			domain.PreflightWarn,
			"DNS checks skipped by policy",
			"Set edge.dns_policy to required to enforce DNS validation.",
			nil,
		)
	}
	if len(proxiedDomains) == 0 {
		return build(domain.PreflightPass, "No Cloudflare proxy detected for edge/apex/www", "", nil)
	}
	if strings.TrimSpace(dns01Token) != "" {
		return build(
			domain.PreflightPass,
			"Cloudflare proxy detected and DNS-01 credentials provided",
			"",
			map[string]string{"proxied_domains": strings.Join(proxiedDomains, ","), "dns01": "enabled"},
		)
	}

	details := "Cloudflare proxy detected without DNS-01 credentials"
	hint := "Disable proxying (DNS-only) during issuance or provide a Cloudflare API token for DNS-01."
	data := map[string]string{"proxied_domains": strings.Join(proxiedDomains, ",")}
	if policy == domain.DNSPolicyWarn {
		return build(domain.PreflightWarn, details, hint, data)
	}
	return build(domain.PreflightFail, details, hint, data)
}

// BaseDomain returns the canonical base domain for edge DNS checks.
func BaseDomain(domainName string) string {
	normalized := normalizeHost(domainName)
	if normalized == "" {
		return normalized
	}
	if apex, err := publicsuffix.EffectiveTLDPlusOne(normalized); err == nil {
		if normalized == apex || normalized == "www."+apex {
			return apex
		}
	}
	if strings.HasPrefix(normalized, "www.") {
		return strings.TrimPrefix(normalized, "www.")
	}
	return normalized
}
