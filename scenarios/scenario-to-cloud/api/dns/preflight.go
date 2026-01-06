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
		return checks
	}

	vpsLookup := svc.ResolveHost(ctx, vpsHost)
	baseDomain := BaseDomain(domainName)
	apexDomain := baseDomain
	wwwDomain := "www." + baseDomain
	doOriginDomain := "do-origin." + baseDomain

	type domainStatus struct {
		role        string
		host        string
		lookup      domain.DNSLookupResult
		proxied     bool
		pointsToVPS bool
	}

	statuses := make([]domainStatus, 0, 3)
	apexStatus := domainStatus{
		role:   "apex",
		host:   apexDomain,
		lookup: svc.ResolveHost(ctx, apexDomain),
	}
	apexStatus.proxied = areCloudflareIPs(apexStatus.lookup.IPs)
	if vpsLookup.Error == nil && apexStatus.lookup.Error == nil {
		apexStatus.pointsToVPS = intersects(vpsLookup.IPs, apexStatus.lookup.IPs)
	}
	statuses = append(statuses, apexStatus)

	if wwwDomain != apexDomain {
		wwwStatus := domainStatus{
			role:   "www",
			host:   wwwDomain,
			lookup: svc.ResolveHost(ctx, wwwDomain),
		}
		wwwStatus.proxied = areCloudflareIPs(wwwStatus.lookup.IPs)
		if vpsLookup.Error == nil && wwwStatus.lookup.Error == nil {
			wwwStatus.pointsToVPS = intersects(vpsLookup.IPs, wwwStatus.lookup.IPs)
		}
		statuses = append(statuses, wwwStatus)
	}

	doOriginStatus := domainStatus{
		role:   "origin",
		host:   doOriginDomain,
		lookup: svc.ResolveHost(ctx, doOriginDomain),
	}
	doOriginStatus.proxied = areCloudflareIPs(doOriginStatus.lookup.IPs)
	if vpsLookup.Error == nil && doOriginStatus.lookup.Error == nil {
		doOriginStatus.pointsToVPS = intersects(vpsLookup.IPs, doOriginStatus.lookup.IPs)
	}
	statuses = append(statuses, doOriginStatus)

	recordDNSIssue := func(id, title, details, hint string, data map[string]string) {
		if policy == domain.DNSPolicyWarn {
			warn(id, title, details, hint, data)
			return
		}
		fail(id, title, details, hint, data)
	}

	if vpsLookup.Error != nil {
		recordDNSIssue(
			domain.PreflightDNSVPSHostID,
			"Resolve VPS host",
			fmt.Sprintf("Unable to resolve VPS host %q", vpsLookup.Host),
			vpsLookup.Error.Message,
			nil,
		)
	} else {
		pass(domain.PreflightDNSVPSHostID, "Resolve VPS host", "Resolved VPS host", map[string]string{"ips": strings.Join(vpsLookup.IPs, ",")})
	}

	for _, status := range statuses {
		title := ""
		id := ""
		allowProxy := false
		switch status.role {
		case "apex":
			id = domain.PreflightDNSEdgeApexID
			title = "Apex domain"
			allowProxy = true
		case "www":
			id = domain.PreflightDNSEdgeWWWID
			title = "WWW domain"
			allowProxy = true
		case "origin":
			id = domain.PreflightDNSDoOriginID
			title = "Origin domain"
		default:
			continue
		}

		if status.lookup.Error != nil {
			recordDNSIssue(
				id,
				title,
				fmt.Sprintf("Unable to resolve %s (%q)", title, status.lookup.Host),
				status.lookup.Error.Message,
				nil,
			)
			continue
		}

		if allowProxy && status.proxied {
			pass(
				id,
				title,
				fmt.Sprintf("%s resolves to Cloudflare proxy", status.lookup.Host),
				map[string]string{"ips": strings.Join(status.lookup.IPs, ","), "proxied": "cloudflare"},
			)
			continue
		}

		if vpsLookup.Error != nil {
			recordDNSIssue(
				id,
				title,
				fmt.Sprintf("Unable to verify %s (VPS host unresolved)", status.lookup.Host),
				vpsLookup.Error.Message,
				nil,
			)
			continue
		}

		if status.pointsToVPS {
			pass(
				id,
				title,
				fmt.Sprintf("%s resolves to the VPS", status.lookup.Host),
				map[string]string{"ips": strings.Join(status.lookup.IPs, ","), "vps_ips": strings.Join(vpsLookup.IPs, ",")},
			)
			continue
		}

		hint := ""
		if len(vpsLookup.IPs) > 0 {
			hint, _ = BuildARecordHint(status.lookup.Host, vpsLookup.IPs[0])
		}
		recordDNSIssue(
			id,
			title,
			fmt.Sprintf("%s resolves to %s, not your VPS (%s)", status.lookup.Host, strings.Join(status.lookup.IPs, ", "), strings.Join(vpsLookup.IPs, ", ")),
			hint,
			map[string]string{"vps_ips": strings.Join(vpsLookup.IPs, ","), "edge_ips": strings.Join(status.lookup.IPs, ","), "domain": status.lookup.Host},
		)
	}

	ogWorkerHint := "To enable OG worker routing, proxy both apex and www through Cloudflare and point do-origin to the VPS (DNS-only A record)."
	apexProxied := apexStatus.proxied
	wwwProxied := false
	for _, status := range statuses {
		if status.role == "www" {
			wwwProxied = status.proxied
			break
		}
	}
	if wwwDomain == apexDomain {
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
	} else if doOriginStatus.lookup.Error != nil || !doOriginStatus.pointsToVPS {
		warn(
			domain.PreflightDNSOGWorkerID,
			"OG worker readiness",
			fmt.Sprintf("%s should point to the VPS for OG routing", doOriginStatus.host),
			ogWorkerHint,
			map[string]string{"domain": doOriginStatus.host},
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
