package dns

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-cloud/domain"
)

// PreflightChecks evaluates DNS validation checks for a deployment preflight.
func PreflightChecks(ctx context.Context, svc Service, domainName, vpsHost string, policy domain.DNSPolicy) []domain.PreflightCheck {
	checks := make([]domain.PreflightCheck, 0, 3)
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
		warn(domain.PreflightDNSEdgeDomainID, "Resolve edge domain", "DNS checks skipped by policy", skippedHint, nil)
		warn(domain.PreflightDNSPointsToVPSID, "DNS points to VPS", "DNS checks skipped by policy", skippedHint, nil)
		return checks
	}

	compare := svc.CompareDomainToVPS(ctx, domainName, vpsHost)

	recordDNSIssue := func(id, title, details, hint string, data map[string]string) {
		if policy == domain.DNSPolicyWarn {
			warn(id, title, details, hint, data)
			return
		}
		fail(id, title, details, hint, data)
	}

	if compare.VPS.Error != nil {
		recordDNSIssue(
			domain.PreflightDNSVPSHostID,
			"Resolve VPS host",
			fmt.Sprintf("Unable to resolve VPS host %q", compare.VPS.Host),
			compare.VPS.Error.Message,
			nil,
		)
	} else {
		pass(domain.PreflightDNSVPSHostID, "Resolve VPS host", "Resolved VPS host", map[string]string{"ips": strings.Join(compare.VPS.IPs, ",")})
	}

	if compare.Domain.Error != nil {
		recordDNSIssue(
			domain.PreflightDNSEdgeDomainID,
			"Resolve edge domain",
			fmt.Sprintf("Unable to resolve edge.domain %q", compare.Domain.Host),
			compare.Domain.Error.Message,
			nil,
		)
	} else {
		pass(domain.PreflightDNSEdgeDomainID, "Resolve edge domain", "Resolved edge.domain", map[string]string{"ips": strings.Join(compare.Domain.IPs, ",")})
	}

	if compare.VPS.Error == nil && compare.Domain.Error == nil {
		if compare.PointsToVPS {
			pass(domain.PreflightDNSPointsToVPSID, "DNS points to VPS", "edge.domain resolves to the VPS", nil)
		} else {
			hint := ""
			if len(compare.VPS.IPs) > 0 {
				hint, _ = BuildARecordHint(compare.Domain.Host, compare.VPS.IPs[0])
			}
			recordDNSIssue(
				domain.PreflightDNSPointsToVPSID,
				"DNS points to VPS",
				fmt.Sprintf("%s resolves to %s, not your VPS (%s)", compare.Domain.Host, strings.Join(compare.Domain.IPs, ", "), strings.Join(compare.VPS.IPs, ", ")),
				hint,
				map[string]string{"vps_ips": strings.Join(compare.VPS.IPs, ","), "edge_ips": strings.Join(compare.Domain.IPs, ","), "domain": compare.Domain.Host},
			)
		}
	}

	return checks
}
