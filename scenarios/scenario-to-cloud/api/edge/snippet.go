package edge

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"scenario-to-cloud/domain"
)

// Proxy layout shared with `vrooli cloud-target edge route apply`. The main
// Caddyfile is host-owned; this owner writes only its snippet.
const (
	CaddyMainConfigPath = "/etc/caddy/Caddyfile"
	CaddyConfDir        = "/etc/caddy/conf.d"
	CaddyImportLine     = "import conf.d/*.caddy"
	snippetPrefix       = "vrooli-"
	snippetSuffix       = ".caddy"
)

// SnippetPath is the per-deployment snippet on the target.
func SnippetPath(deploymentID string) string {
	return path.Join(CaddyConfDir, snippetPrefix+deploymentID+snippetSuffix)
}

// RenderSnippet renders the site blocks for the spec's routes and nothing
// else: no global options, no imports. The ACME directory and the DNS-01
// provider are site-level tls options so the snippet never needs the global
// block, and the provider token is referenced through an environment
// placeholder the credential authority populates on the target; no secret
// is ever rendered.
func RenderSnippet(spec domain.EdgeSpec) string {
	routes := append([]domain.EdgeRoute(nil), spec.Routes...)
	sort.Slice(routes, func(i, j int) bool { return routes[i].Host < routes[j].Host })
	var b strings.Builder
	fmt.Fprintf(&b, "# vrooli deployment %s (edge spec %s); managed by scenario-to-cloud, do not edit\n", spec.DeploymentID, spec.Digest)
	for _, route := range routes {
		fmt.Fprintf(&b, "%s {\n", route.Host)
		if strings.TrimSpace(spec.ACMEEmail) != "" {
			fmt.Fprintf(&b, "  tls %s {\n", strings.TrimSpace(spec.ACMEEmail))
		} else {
			b.WriteString("  tls {\n")
		}
		fmt.Fprintf(&b, "    ca %s\n", ACMECA(spec.ACMEEnvironment))
		if spec.DNSProvider != nil && spec.DNSProvider.Provider != "" && spec.DNSProvider.EnvVar != "" {
			fmt.Fprintf(&b, "    dns %s {env.%s}\n", spec.DNSProvider.Provider, spec.DNSProvider.EnvVar)
		}
		b.WriteString("  }\n")
		fmt.Fprintf(&b, "  reverse_proxy 127.0.0.1:%d\n", route.UpstreamPort)
		b.WriteString("}\n")
	}
	return b.String()
}

// TargetSpec is the wire shape `vrooli cloud-target edge route apply`
// consumes (internal/cloudtarget.EdgeRouteSpec). It carries the rendered
// snippet so the target owner can verify scope without a second renderer.
type TargetSpec struct {
	SchemaVersion   int                `json:"schema_version"`
	DeploymentID    string             `json:"deployment_id"`
	Domain          string             `json:"domain"`
	Routes          []domain.EdgeRoute `json:"routes"`
	Snippet         string             `json:"snippet"`
	ACMEEnvironment string             `json:"acme_environment"`
}

// TargetSpecJSON renders the target verb input for a spec.
func TargetSpecJSON(spec domain.EdgeSpec) ([]byte, error) {
	return json.Marshal(TargetSpec{SchemaVersion: 1, DeploymentID: spec.DeploymentID, Domain: spec.Domain, Routes: spec.Routes, Snippet: RenderSnippet(spec), ACMEEnvironment: spec.ACMEEnvironment})
}

// RouteHosts lists the public hosts in sorted order.
func RouteHosts(spec domain.EdgeSpec) []string {
	hosts := make([]string, 0, len(spec.Routes))
	for _, route := range spec.Routes {
		hosts = append(hosts, route.Host)
	}
	sort.Strings(hosts)
	return hosts
}
