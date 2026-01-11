package dns

import (
	"context"

	"scenario-to-cloud/domain"
)

// DomainStatus captures DNS status for a specific domain role.
type DomainStatus struct {
	Role        string
	Host        string
	Lookup      domain.DNSLookupResult
	AllowProxy  bool
	Proxied     bool
	PointsToVPS bool
}

// Evaluation aggregates DNS lookups for edge-related domains and the VPS host.
type Evaluation struct {
	EdgeDomain   string
	BaseDomain   string
	ApexDomain   string
	WWWDomain    string
	OriginDomain string
	VPS          domain.DNSLookupResult
	Statuses     []DomainStatus
}

// Evaluate resolves edge-related DNS records and compares them to the VPS host.
func Evaluate(ctx context.Context, svc Service, domainName, vpsHost string) Evaluation {
	edgeDomain := normalizeHost(domainName)
	baseDomain := BaseDomain(edgeDomain)
	apexDomain := baseDomain
	wwwDomain := "www." + baseDomain
	originDomain := "do-origin." + baseDomain
	vpsLookup := svc.ResolveHost(ctx, vpsHost)

	statuses := make([]DomainStatus, 0, 4)
	addStatus := func(role, host string, allowProxy bool) {
		lookup := svc.ResolveHost(ctx, host)
		status := DomainStatus{
			Role:       role,
			Host:       host,
			Lookup:     lookup,
			AllowProxy: allowProxy,
			Proxied:    areCloudflareIPs(lookup.IPs),
		}
		if vpsLookup.Error == nil && lookup.Error == nil {
			status.PointsToVPS = intersects(vpsLookup.IPs, lookup.IPs)
		}
		statuses = append(statuses, status)
	}

	addStatus("apex", apexDomain, true)
	if wwwDomain != apexDomain {
		addStatus("www", wwwDomain, true)
	}
	addStatus("origin", originDomain, false)
	if edgeDomain != "" && edgeDomain != apexDomain && edgeDomain != wwwDomain && edgeDomain != originDomain {
		addStatus("edge", edgeDomain, true)
	}

	return Evaluation{
		EdgeDomain:   edgeDomain,
		BaseDomain:   baseDomain,
		ApexDomain:   apexDomain,
		WWWDomain:    wwwDomain,
		OriginDomain: originDomain,
		VPS:          vpsLookup,
		Statuses:     statuses,
	}
}

func (e Evaluation) StatusForRole(role string) (DomainStatus, bool) {
	for _, status := range e.Statuses {
		if status.Role == role {
			return status, true
		}
	}
	return DomainStatus{}, false
}
