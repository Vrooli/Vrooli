package deployment

import (
	"context"
	"errors"
	"fmt"
	"net"
	neturl "net/url"
	"slices"
	"strings"
	"time"

	internalmanifest "scenario-to-cloud/cli/internal/manifest"
)

// ManifestSelector captures the identifying fields used to map a manifest to a deployment.
type ManifestSelector struct {
	ScenarioID string `json:"scenario_id,omitempty"`
	Host       string `json:"host,omitempty"`
	Domain     string `json:"domain,omitempty"`
	Target     string `json:"target,omitempty"`
}

var lookupIPAddresses = func(host string) ([]string, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.IP == nil {
			continue
		}
		ips = append(ips, addr.IP.String())
	}
	return ips, nil
}

// ReadSelectorFromManifest extracts scenario and VPS target fields from a manifest file.
func ReadSelectorFromManifest(path string) (ManifestSelector, error) {
	raw, err := internalmanifest.ReadJSONFile(path)
	if err != nil {
		return ManifestSelector{}, err
	}

	scenarioID := strings.TrimSpace(getNestedString(raw, "scenario", "id"))
	host := strings.TrimSpace(getNestedString(raw, "target", "vps", "host"))
	domain := strings.TrimSpace(getNestedString(raw, "edge", "domain"))

	if scenarioID == "" {
		return ManifestSelector{}, fmt.Errorf("manifest is missing required field: scenario.id")
	}
	if host == "" {
		return ManifestSelector{}, fmt.Errorf("manifest is missing required field: target.vps.host")
	}

	return ManifestSelector{
		ScenarioID: scenarioID,
		Host:       host,
		Domain:     domain,
	}, nil
}

// ResolveLatestBySelector returns the newest deployment matching scenario + host.
func ResolveLatestBySelector(client *Client, selector ManifestSelector) (*DeploymentSummary, error) {
	host := normalizeToken(selector.Host)
	domain := normalizeToken(selector.Domain)
	target := normalizeToken(selector.Target)
	scenarioID := strings.TrimSpace(selector.ScenarioID)
	if host == "" && domain == "" && target == "" {
		return nil, fmt.Errorf("at least one selector is required: --host, --domain, or --target")
	}

	opts := ListOptions{}
	if scenarioID != "" {
		opts.ScenarioID = scenarioID
	}

	_, listResp, err := client.List(opts)
	if err != nil {
		return nil, err
	}

	candidates := listResp.Deployments
	if domain != "" {
		if latest := latestByDomain(candidates, domain); latest != nil {
			if host == "" || normalizeToken(latest.Host) == host {
				return latest, nil
			}
		}
	}

	if host != "" {
		if latest := latestByHost(candidates, host); latest != nil {
			return latest, nil
		}
	}

	if target == "" {
		return nil, nil
	}

	// Convenience selector resolution order:
	// 1) exact domain match
	// 2) exact host match
	// 3) DNS-assisted host match (last resort)
	//
	// DNS fallback can be wrong when domains are proxied (e.g. Cloudflare) or
	// multiple deployments share the same VPS IP, so we only accept a unique match.
	if latest := latestByDomain(candidates, target); latest != nil {
		return latest, nil
	}
	if latest := latestByHost(candidates, target); latest != nil {
		return latest, nil
	}

	fallback, err := resolveByDNSFallback(candidates, target)
	if err != nil {
		return nil, err
	}
	return fallback, nil
}

func normalizeToken(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return ""
	}
	parsed, err := neturl.Parse(v)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	}
	return v
}

func latestByDomain(candidates []DeploymentSummary, domain string) *DeploymentSummary {
	return latestBy(candidates, func(candidate DeploymentSummary) bool {
		return normalizeToken(candidate.Domain) == domain
	})
}

func latestByHost(candidates []DeploymentSummary, host string) *DeploymentSummary {
	return latestBy(candidates, func(candidate DeploymentSummary) bool {
		return normalizeToken(candidate.Host) == host
	})
}

func latestBy(candidates []DeploymentSummary, predicate func(candidate DeploymentSummary) bool) *DeploymentSummary {
	var latest *DeploymentSummary
	for i := range candidates {
		candidate := candidates[i]
		if !predicate(candidate) {
			continue
		}
		if latest == nil || candidate.CreatedAt.After(latest.CreatedAt) {
			copy := candidate
			latest = &copy
		}
	}
	return latest
}

func resolveByDNSFallback(candidates []DeploymentSummary, target string) (*DeploymentSummary, error) {
	if target == "" {
		return nil, nil
	}
	if net.ParseIP(target) != nil {
		return nil, nil
	}

	ips, err := lookupIPAddresses(target)
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("resolve target %q via DNS: %w", target, err)
	}
	if len(ips) == 0 {
		return nil, nil
	}

	normalizedIPs := make([]string, 0, len(ips))
	for _, ip := range ips {
		ipNorm := normalizeToken(ip)
		if ipNorm != "" {
			normalizedIPs = append(normalizedIPs, ipNorm)
		}
	}

	matches := make([]DeploymentSummary, 0)
	for i := range candidates {
		candidate := candidates[i]
		if slices.Contains(normalizedIPs, normalizeToken(candidate.Host)) {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		copy := matches[0]
		return &copy, nil
	default:
		return nil, fmt.Errorf(
			"target %q resolved to shared host IP(s); selector is ambiguous (matched %d deployments). Use --domain or --host explicitly",
			target,
			len(matches),
		)
	}
}

func getNestedString(m map[string]interface{}, path ...string) string {
	var current interface{} = m
	for i, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		value, ok := obj[key]
		if !ok {
			return ""
		}
		if i == len(path)-1 {
			s, ok := value.(string)
			if !ok {
				return ""
			}
			return s
		}
		current = value
	}
	return ""
}

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return "n/a"
	}
	return t.UTC().Format(time.RFC3339)
}
