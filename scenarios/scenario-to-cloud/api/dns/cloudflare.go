package dns

import (
	"net"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/secrets"
)

var cloudflareCIDRs = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

var cloudflareNets = func() []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cloudflareCIDRs))
	for _, cidr := range cloudflareCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		nets = append(nets, network)
	}
	return nets
}()

func isCloudflareIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, network := range cloudflareNets {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func areCloudflareIPs(ips []string) bool {
	if len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !isCloudflareIP(ip) {
			return false
		}
	}
	return true
}

// AreCloudflareIPs reports whether all IPs belong to Cloudflare's proxy ranges.
func AreCloudflareIPs(ips []string) bool {
	return areCloudflareIPs(ips)
}

// ProviderName is the only DNS-01 provider the edge renders today.
const ProviderName = "cloudflare"

// ProviderBinding names the DNS-01 credential a manifest declares, by
// credential descriptor only. The token value is never read here: the
// credential authority delivers it to the target, where the proxy reads it
// from the environment variable the binding names. A manifest that declares
// no Cloudflare token secret has no provider binding.
func ProviderBinding(manifest domain.CloudManifest) *domain.EdgeDNSProvider {
	plan, ok := providerPlan(manifest)
	if !ok {
		return nil
	}
	descriptor := domain.CredentialDescriptor{}
	if plan.Descriptor != nil {
		descriptor = domain.CredentialDescriptor{LogicalID: strings.TrimSpace(plan.Descriptor.LogicalID), Field: strings.TrimSpace(plan.Descriptor.Field)}
	}
	if descriptor.IsZero() {
		// Legacy manifest rows carry no descriptor; the provisioning seam
		// writes the value under the deployment identity namespace and the
		// normalised field, so that is the address it will be found at.
		descriptor = domain.CredentialDescriptor{LogicalID: "vrooli/" + strings.TrimSpace(manifest.Scenario.ID), Field: secrets.CredentialField(domain.CloudflareAPITokenKey)}
	}
	return &domain.EdgeDNSProvider{Provider: ProviderName, Descriptor: descriptor, EnvVar: domain.CloudflareAPITokenKey}
}

// ProviderConfigured reports whether a value is bound for the provider
// credential in the caller-resolved secret set (keyed by target name or by
// descriptor address). It returns a fact, never the value.
func ProviderConfigured(manifest domain.CloudManifest, resolved map[string]string) bool {
	binding := ProviderBinding(manifest)
	if binding == nil || resolved == nil {
		return false
	}
	if strings.TrimSpace(resolved[binding.EnvVar]) != "" {
		return true
	}
	return strings.TrimSpace(resolved[binding.Descriptor.Address()]) != ""
}

func providerPlan(manifest domain.CloudManifest) (domain.BundleSecretPlan, bool) {
	if manifest.Secrets == nil {
		return domain.BundleSecretPlan{}, false
	}
	for _, plan := range manifest.Secrets.BundleSecrets {
		if strings.EqualFold(strings.TrimSpace(plan.Target.Name), domain.CloudflareAPITokenKey) || strings.EqualFold(strings.TrimSpace(plan.ID), domain.CloudflareAPITokenKey) {
			return plan, true
		}
	}
	return domain.BundleSecretPlan{}, false
}
