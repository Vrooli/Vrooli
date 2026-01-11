package dns

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"

	"scenario-to-cloud/domain"
)

// PreflightChecks evaluates DNS validation checks for a deployment preflight.
func PreflightChecks(ctx context.Context, svc Service, domainName, vpsHost string, policy domain.DNSPolicy) []domain.PreflightCheck {
	eval := Evaluate(ctx, svc, domainName, vpsHost)
	return PreflightChecksFromEvaluation(eval, policy)
}

// PreflightChecksFromEvaluation maps a DNS evaluation to preflight checks.
func PreflightChecksFromEvaluation(eval Evaluation, policy domain.DNSPolicy) []domain.PreflightCheck {
	checks := make([]domain.PreflightCheck, 0, 5)
	fail := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightFail,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}
	pass := func(id, title, details string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightPass,
			Details: details,
			Data:    data,
		})
	}
	warn := func(id, title, details, hint string, data map[string]string) {
		checks = append(checks, domain.PreflightCheck{
			ID:      id,
			Title:   title,
			Status:  domain.PreflightWarn,
			Details: details,
			Hint:    hint,
			Data:    data,
		})
	}

	if policy == domain.DNSPolicySkip {
		skippedHint := "DNS checks skipped by policy. Set edge.dns_policy to required to enforce DNS validation."
		warn(domain.PreflightDNSVPSHostID, "Resolve VPS host", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSEdgeApexID, "Apex domain", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSEdgeWWWID, "WWW domain", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSDoOriginID, "Origin domain", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSOGWorkerID, "OG worker readiness", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSEdgeIPv6ID, "Edge IPv6 records", "DNS checks skipped by policy", skippedHint, nil)
		return checks
	}

	recordDNSIssue := func(id, title, details, hint string, data map[string]string) {
		if policy == domain.DNSPolicyWarn {
			warn(id, title, details, hint, data)
			return
		}
		fail(id, title, details, hint, data)
	}

	if eval.VPS.Error != nil {
		recordDNSIssue(
			domain.PreflightDNSVPSHostID,
			"Resolve VPS host",
			fmt.Sprintf("Unable to resolve VPS host %q", eval.VPS.Host),
			eval.VPS.Error.Message,
			nil,
		)
	} else {
		pass(domain.PreflightDNSVPSHostID, "Resolve VPS host", "Resolved VPS host", map[string]string{"ips": strings.Join(eval.VPS.IPs, ",")})
	}

	for _, status := range eval.Statuses {
		title := ""
		id := ""
		switch status.Role {
		case "apex":
			id = domain.PreflightDNSEdgeApexID
			title = "Apex domain"
		case "www":
			id = domain.PreflightDNSEdgeWWWID
			title = "WWW domain"
		case "origin":
			id = domain.PreflightDNSDoOriginID
			title = "Origin domain"
		case "edge":
			id = domain.PreflightDNSEdgeDomainID
			title = "Edge domain"
		default:
			continue
		}

		if status.Lookup.Error != nil {
			recordDNSIssue(
				id,
				title,
				fmt.Sprintf("Unable to resolve %s (%q)", title, status.Lookup.Host),
				status.Lookup.Error.Message,
				nil,
			)
			continue
		}

		if status.AllowProxy && status.Proxied {
			pass(
				id,
				title,
				fmt.Sprintf("%s resolves to Cloudflare proxy", status.Lookup.Host),
				map[string]string{"ips": strings.Join(status.Lookup.IPs, ","), "proxied": "cloudflare"},
			)
			continue
		}

		if eval.VPS.Error != nil {
			recordDNSIssue(
				id,
				title,
				fmt.Sprintf("Unable to verify %s (VPS host unresolved)", status.Lookup.Host),
				eval.VPS.Error.Message,
				nil,
			)
			continue
		}

		if status.PointsToVPS {
			pass(
				id,
				title,
				fmt.Sprintf("%s resolves to the VPS", status.Lookup.Host),
				map[string]string{"ips": strings.Join(status.Lookup.IPs, ","), "vps_ips": strings.Join(eval.VPS.IPs, ",")},
			)
			continue
		}

		hint := ""
		if len(eval.VPS.IPs) > 0 {
			hint, _ = BuildARecordHint(status.Lookup.Host, eval.VPS.IPs[0])
		}
		recordDNSIssue(
			id,
			title,
			fmt.Sprintf("%s resolves to %s, not your VPS (%s)", status.Lookup.Host, strings.Join(status.Lookup.IPs, ", "), strings.Join(eval.VPS.IPs, ", ")),
			hint,
			map[string]string{"vps_ips": strings.Join(eval.VPS.IPs, ","), "edge_ips": strings.Join(status.Lookup.IPs, ","), "domain": status.Lookup.Host},
		)
	}

	ogWorkerHint := "To enable OG worker routing, proxy both apex and www through Cloudflare and point do-origin to the VPS (DNS-only A record)."
	apexStatus, _ := eval.StatusForRole("apex")
	wwwStatus, hasWWW := eval.StatusForRole("www")
	doOriginStatus, _ := eval.StatusForRole("origin")
	apexProxied := apexStatus.Proxied
	wwwProxied := false
	if hasWWW {
		wwwProxied = wwwStatus.Proxied
	}

	edgeIPv6Issues := []string{}
	edgeIPv6Data := map[string]string{}
	vpsIPv6 := filterIPv6(eval.VPS.IPs)
	for _, status := range eval.Statuses {
		if status.Role != "apex" && status.Role != "www" {
			continue
		}
		if status.Proxied {
			continue
		}
		edgeIPv6 := filterIPv6(status.Lookup.IPs)
		if len(edgeIPv6) == 0 {
			continue
		}
		if eval.VPS.Error != nil || len(vpsIPv6) == 0 || !intersects(edgeIPv6, vpsIPv6) {
			edgeIPv6Issues = append(edgeIPv6Issues, fmt.Sprintf("%s -> %s", status.Host, strings.Join(edgeIPv6, ", ")))
			edgeIPv6Data[status.Host] = strings.Join(edgeIPv6, ",")
		}
	}

	if len(edgeIPv6Issues) > 0 {
		warn(
			domain.PreflightDNSEdgeIPv6ID,
			"Edge IPv6 records",
			fmt.Sprintf("IPv6 records detected without matching VPS IPv6: %s", strings.Join(edgeIPv6Issues, " | ")),
			"Remove AAAA records or configure IPv6 on the VPS to match. Clients preferring IPv6 may time out if AAAA points to a non-serving host.",
			edgeIPv6Data,
		)
	} else {
		pass(
			domain.PreflightDNSEdgeIPv6ID,
			"Edge IPv6 records",
			"No mismatched IPv6 records detected",
			nil,
		)
	}
	if eval.WWWDomain == eval.ApexDomain {
		wwwProxied = apexProxied
	}
	if !apexProxied || !wwwProxied {
		warn(
			domain.PreflightDNSOGWorkerID,
			"OG worker readiness",
			"Apex and www should be proxied through Cloudflare for OG routing",
			ogWorkerHint,
			nil,
		)
	} else if doOriginStatus.Lookup.Error != nil || !doOriginStatus.PointsToVPS {
		warn(
			domain.PreflightDNSOGWorkerID,
			"OG worker readiness",
			fmt.Sprintf("%s should point to the VPS for OG routing", doOriginStatus.Host),
			ogWorkerHint,
			map[string]string{"domain": doOriginStatus.Host},
		)
	} else {
		pass(
			domain.PreflightDNSOGWorkerID,
			"OG worker readiness",
			"OG worker routing prerequisites look good",
			nil,
		)
	}

	return checks
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
