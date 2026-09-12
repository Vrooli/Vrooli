package cloudtarget

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// EdgeSpecSchemaVersion is the wire version of the route spec the cloud side
// hands to `edge route apply`.
const EdgeSpecSchemaVersion = 1

// EdgeRoute is one public host routed to one loopback upstream.
type EdgeRoute struct {
	Host         string `json:"host"`
	UpstreamPort int    `json:"upstream_port"`
	ListenerID   string `json:"listener_id"`
}

// EdgeRouteSpec is the owner-scoped edge configuration for one deployment.
// The cloud side derives it from the deployment closure's declared listeners
// and renders the Caddy site blocks; the target verifies the snippet only
// contains what the routes declare before it touches the proxy.
type EdgeRouteSpec struct {
	SchemaVersion int         `json:"schema_version"`
	DeploymentID  string      `json:"deployment_id"`
	Domain        string      `json:"domain"`
	Routes        []EdgeRoute `json:"routes"`
	// Snippet is the rendered Caddy configuration for this deployment only:
	// site blocks for the route hosts, nothing global, no imports.
	Snippet string `json:"snippet"`
	// ACMEEnvironment is "staging" or "production"; evidence for the receipt.
	ACMEEnvironment string `json:"acme_environment"`
}

var hostPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,62}$`)

// managementPorts are never valid upstreams: the deployment edge must not be
// able to route public traffic at the SSH or Bridge management listeners.
var managementPorts = map[int]string{22: "ssh", 18767: "bridge"}

// ParseEdgeRouteSpec decodes and validates a spec. It fails closed on any
// unknown field, undeclared host in the snippet, or non-loopback upstream.
func ParseEdgeRouteSpec(raw []byte) (EdgeRouteSpec, error) {
	var spec EdgeRouteSpec
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return EdgeRouteSpec{}, refuse(CodeEdgeSpecInvalid, "edge spec is not valid JSON: %v", err)
	}
	if err := spec.Validate(); err != nil {
		return EdgeRouteSpec{}, err
	}
	return spec, nil
}

// Validate applies the owner-scope rules.
func (spec EdgeRouteSpec) Validate() error {
	if spec.SchemaVersion != EdgeSpecSchemaVersion {
		return refuse(CodeEdgeSpecInvalid, "edge spec schema_version %d is not %d", spec.SchemaVersion, EdgeSpecSchemaVersion)
	}
	if err := validIdentifier("deployment id", spec.DeploymentID); err != nil {
		return err
	}
	if len(spec.Routes) == 0 {
		return refuse(CodeEdgeSpecInvalid, "edge spec declares no routes")
	}
	switch spec.ACMEEnvironment {
	case "staging", "production":
	default:
		return refuse(CodeEdgeSpecInvalid, "edge spec acme_environment %q is not staging or production", spec.ACMEEnvironment)
	}
	hosts := map[string]bool{}
	for _, route := range spec.Routes {
		host := strings.ToLower(strings.TrimSpace(route.Host))
		if !hostPattern.MatchString(host) {
			return refuse(CodeEdgeSpecInvalid, "route host %q is not a valid public host name", route.Host)
		}
		if hosts[host] {
			return refuse(CodeEdgeSpecInvalid, "route host %q is declared twice", host)
		}
		hosts[host] = true
		if route.UpstreamPort < 1 || route.UpstreamPort > 65535 {
			return refuse(CodeEdgeSpecInvalid, "route %s upstream port %d is out of range", host, route.UpstreamPort)
		}
		if name, private := managementPorts[route.UpstreamPort]; private {
			return refuse(CodeEdgePrivateListener, "route %s targets the %s management listener on port %d", host, name, route.UpstreamPort)
		}
		if strings.TrimSpace(route.ListenerID) == "" {
			return refuse(CodeEdgeSpecInvalid, "route %s names no listener", host)
		}
	}
	if strings.TrimSpace(spec.Domain) != "" && !hosts[strings.ToLower(strings.TrimSpace(spec.Domain))] {
		return refuse(CodeEdgeSpecInvalid, "domain %q has no route", spec.Domain)
	}
	return checkSnippetScope(spec.Snippet, spec.Routes)
}

// SpecDigest is the sha256 of the canonical spec (routes and snippet), so a
// replay with a different route set is refused by the receipt input digest.
func (spec EdgeRouteSpec) SpecDigest() (string, error) {
	routes := append([]EdgeRoute(nil), spec.Routes...)
	sort.Slice(routes, func(i, j int) bool { return routes[i].Host < routes[j].Host })
	return CanonicalDigest(map[string]any{
		"schema_version":   spec.SchemaVersion,
		"deployment_id":    spec.DeploymentID,
		"domain":           spec.Domain,
		"routes":           routes,
		"snippet":          spec.Snippet,
		"acme_environment": spec.ACMEEnvironment,
	})
}

var (
	siteHeaderPattern = regexp.MustCompile(`^\s*([A-Za-z0-9.*-]+(?:\s*,\s*[A-Za-z0-9.*-]+)*)\s*\{\s*$`)
	reverseProxyLine  = regexp.MustCompile(`^\s*reverse_proxy\s+(\S+)\s*$`)
)

// checkSnippetScope proves the rendered snippet stays inside the declared
// routes: every site address is a declared host, every reverse_proxy target
// is 127.0.0.1:<declared port>, and no import or global option block appears
// (either could reach configuration this deployment does not own).
func checkSnippetScope(snippet string, routes []EdgeRoute) error {
	if strings.TrimSpace(snippet) == "" {
		return refuse(CodeEdgeSpecInvalid, "edge spec snippet is empty")
	}
	declared := map[string]int{}
	for _, route := range routes {
		declared[strings.ToLower(strings.TrimSpace(route.Host))] = route.UpstreamPort
	}
	seen := map[string]bool{}
	depth := 0
	currentHosts := []string{}
	for lineNo, line := range strings.Split(snippet, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if depth == 0 {
			if trimmed == "{" {
				return refuse(CodeEdgeSnippetOutOfScope, "snippet line %d opens a global options block", lineNo+1)
			}
			match := siteHeaderPattern.FindStringSubmatch(line)
			if match == nil {
				return refuse(CodeEdgeSnippetOutOfScope, "snippet line %d is not a site block header: %q", lineNo+1, trimmed)
			}
			currentHosts = currentHosts[:0]
			for _, host := range strings.Split(match[1], ",") {
				host = strings.ToLower(strings.TrimSpace(host))
				if _, ok := declared[host]; !ok {
					return refuse(CodeEdgeSnippetOutOfScope, "snippet routes undeclared host %q", host)
				}
				seen[host] = true
				currentHosts = append(currentHosts, host)
			}
			depth++
			continue
		}
		fields := strings.Fields(trimmed)
		if fields[0] == "import" {
			return refuse(CodeEdgeSnippetOutOfScope, "snippet line %d imports external configuration", lineNo+1)
		}
		if match := reverseProxyLine.FindStringSubmatch(line); match != nil {
			target := match[1]
			hostPart, portPart, err := splitHostPort(target)
			if err != nil || hostPart != "127.0.0.1" {
				return refuse(CodeEdgeSnippetOutOfScope, "snippet line %d proxies to %q, only 127.0.0.1:<declared port> is allowed", lineNo+1, target)
			}
			port, _ := strconv.Atoi(portPart)
			allowed := false
			for _, host := range currentHosts {
				if declared[host] == port {
					allowed = true
				}
			}
			if !allowed {
				return refuse(CodeEdgeSnippetOutOfScope, "snippet line %d proxies %s to undeclared port %d", lineNo+1, strings.Join(currentHosts, ","), port)
			}
		}
		if strings.HasSuffix(trimmed, "{") {
			depth++
		}
		if trimmed == "}" {
			depth--
		}
	}
	if depth != 0 {
		return refuse(CodeEdgeSpecInvalid, "snippet has unbalanced braces")
	}
	for host := range declared {
		if !seen[host] {
			return refuse(CodeEdgeSpecInvalid, "declared route %s has no site block in the snippet", host)
		}
	}
	return nil
}

func splitHostPort(target string) (string, string, error) {
	idx := strings.LastIndex(target, ":")
	if idx <= 0 || idx == len(target)-1 {
		return "", "", fmt.Errorf("no port in %q", target)
	}
	if _, err := strconv.Atoi(target[idx+1:]); err != nil {
		return "", "", err
	}
	return target[:idx], target[idx+1:], nil
}
