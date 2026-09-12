// Package edge derives the declared public/private listener policy for a
// deployment, renders the owner-scoped proxy snippet the target applies, and
// evaluates the DNS, ACME, TLS and readiness lifecycle checks that decide
// whether external reach may be claimed (plan phase 14).
package edge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// Stable error codes owned by this package.
const (
	CodeDomainInvalid              = "edge_domain_invalid"
	CodeListenerPortUnknown        = "edge_listener_port_unknown"
	CodeNoPublicListener           = "edge_no_public_listener"
	CodeACMEProductionUnauthorized = "acme_production_unauthorized"
	CodeACMEEnvironmentInvalid     = "acme_environment_invalid"
)

// Management listeners every target keeps private regardless of
// declarations: SSH and the Bridge control-plane port.
const (
	ManagementSSHPort    = 22
	ManagementBridgePort = 18767
)

// Private-listener reasons.
const (
	ReasonDeclaredPrivate = "declared_private"
	ReasonUndeclared      = "undeclared"
	ReasonManagement      = "management"
	ReasonDatabase        = "database"
	ReasonMetrics         = "metrics"
	ReasonResourceOwned   = "resource_owned"
)

// FirewallAllow is the only inbound allow set the edge ever requests.
var FirewallAllow = []int{80, 443}

// ACMEAuthority is the EXT-04 gate: production issuance for a
// non-production deployment is allowed only when the operator configured it
// and declared a budget.
type ACMEAuthority struct {
	ProductionAllowed bool
	Budget            int
}

// PolicyInputs is everything Derive reads.
type PolicyInputs struct {
	DeploymentID string
	ScenarioID   string
	Environment  string
	Domain       string
	// Listeners are the closure's declared listeners; empty for a legacy
	// deployment without a closure.
	Listeners []domain.ClosureListener
	// Ports maps the manifest port names onto numbers.
	Ports domain.ManifestPorts
	// TargetHost is the target locator; an IP literal seeds the IP policy.
	TargetHost string
	// ObservedIPv4 / ObservedIPv6 are the target's public addresses when the
	// locator is a name (resolved by the caller).
	ObservedIPv4 []string
	ObservedIPv6 []string
	ACMEEmail    string
	// ACMEEnvironment is the manifest's request; empty derives it.
	ACMEEnvironment string
	ACMEAuthority   ACMEAuthority
	DNSProvider     *domain.EdgeDNSProvider
	// DatabaseResources names closure components that are databases.
	DatabaseResources map[string]bool
}

// Derive computes the edge spec. Only listeners the scenario itself declares
// public_via_edge are routed; database, metrics, management and undeclared
// ports stay private. A legacy deployment without declarations exposes only
// its "ui" port, which is what the previous whole-file Caddyfile did.
func Derive(in PolicyInputs) (domain.EdgeSpec, error) {
	domainName, err := ValidateDomain(in.Domain)
	if err != nil {
		return domain.EdgeSpec{}, err
	}
	acmeEnv, err := SelectACME(ACMEInputs{Requested: in.ACMEEnvironment, Environment: in.Environment, Authority: in.ACMEAuthority})
	if err != nil {
		return domain.EdgeSpec{}, err
	}
	spec := domain.EdgeSpec{
		SchemaVersion:    domain.EdgeSpecSchemaVersion,
		DeploymentID:     strings.TrimSpace(in.DeploymentID),
		ScenarioID:       strings.TrimSpace(in.ScenarioID),
		Domain:           domainName,
		Routes:           []domain.EdgeRoute{},
		PrivateListeners: []domain.EdgePrivateListener{},
		FirewallAllow:    append([]int(nil), FirewallAllow...),
		ACMEEnvironment:  acmeEnv,
		ACMEEmail:        strings.TrimSpace(in.ACMEEmail),
		DNSProvider:      in.DNSProvider,
	}
	spec.IPPolicy = ipPolicy(in)

	scenarioOwner := "scenario:" + spec.ScenarioID
	routedPorts := map[string]bool{}
	if len(in.Listeners) > 0 {
		listeners := append([]domain.ClosureListener(nil), in.Listeners...)
		sort.Slice(listeners, func(i, j int) bool { return listeners[i].ID < listeners[j].ID })
		for _, listener := range listeners {
			port := in.Ports[listener.PortName]
			public := listener.Visibility == domain.EdgeVisibilityPublicViaEdge
			switch {
			case public && listener.Owner == scenarioOwner:
				if port <= 0 {
					return domain.EdgeSpec{}, apierrors.New(CodeListenerPortUnknown, fmt.Sprintf("listener %s is declared public_via_edge but the manifest assigns no port to %q", listener.ID, listener.PortName)).WithDetail("listener", listener.ID)
				}
				if isManagementPort(port) {
					return domain.EdgeSpec{}, apierrors.New(CodeListenerPortUnknown, fmt.Sprintf("listener %s cannot be public: port %d is a management listener", listener.ID, port)).WithDetail("listener", listener.ID)
				}
				spec.Routes = append(spec.Routes, domain.EdgeRoute{Host: routeHost(domainName, listener.PortName), UpstreamPort: port, ListenerID: listener.ID})
				routedPorts[listener.PortName] = true
			default:
				spec.PrivateListeners = append(spec.PrivateListeners, domain.EdgePrivateListener{ID: listener.ID, Owner: listener.Owner, PortName: listener.PortName, Port: port, Reason: privateReason(listener, public, in.DatabaseResources)})
			}
		}
	} else if port := in.Ports["ui"]; port > 0 {
		spec.Routes = append(spec.Routes, domain.EdgeRoute{Host: domainName, UpstreamPort: port, ListenerID: spec.ScenarioID + "/ui"})
		routedPorts["ui"] = true
	}
	if len(spec.Routes) == 0 {
		return domain.EdgeSpec{}, apierrors.New(CodeNoPublicListener, "no listener is declared public_via_edge for "+spec.ScenarioID).WithDetail("scenario_id", spec.ScenarioID)
	}
	// Every manifest port not routed is private by construction.
	names := make([]string, 0, len(in.Ports))
	for name := range in.Ports {
		names = append(names, name)
	}
	sort.Strings(names)
	declared := map[string]bool{}
	for _, listener := range spec.PrivateListeners {
		declared[listener.PortName] = true
	}
	for _, name := range names {
		if routedPorts[name] || declared[name] {
			continue
		}
		reason := ReasonUndeclared
		if name == "metrics" {
			reason = ReasonMetrics
		}
		spec.PrivateListeners = append(spec.PrivateListeners, domain.EdgePrivateListener{ID: spec.ScenarioID + "/" + name, Owner: scenarioOwner, PortName: name, Port: in.Ports[name], Reason: reason})
	}
	spec.PrivateListeners = append(spec.PrivateListeners,
		domain.EdgePrivateListener{ID: "management/ssh", Owner: "host", Port: ManagementSSHPort, Reason: ReasonManagement},
		domain.EdgePrivateListener{ID: "management/bridge", Owner: "vrooli-bridge", Port: ManagementBridgePort, Reason: ReasonManagement},
	)
	sort.Slice(spec.Routes, func(i, j int) bool { return spec.Routes[i].Host < spec.Routes[j].Host })
	sort.Slice(spec.PrivateListeners, func(i, j int) bool { return spec.PrivateListeners[i].ID < spec.PrivateListeners[j].ID })
	digest, err := Digest(spec)
	if err != nil {
		return domain.EdgeSpec{}, err
	}
	spec.Digest = digest
	return spec, nil
}

// Digest is sha256 over the canonical JSON of the spec without its digest.
func Digest(spec domain.EdgeSpec) (string, error) {
	spec.Digest = ""
	raw, err := json.Marshal(spec)
	if err != nil {
		return "", apierrors.Internal("encode edge spec", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func routeHost(domainName, portName string) string {
	if portName == "ui" || portName == "" {
		return domainName
	}
	return portName + "." + domainName
}

func isManagementPort(port int) bool {
	return port == ManagementSSHPort || port == ManagementBridgePort
}

func privateReason(listener domain.ClosureListener, public bool, databases map[string]bool) string {
	owner := strings.TrimPrefix(listener.Owner, "resource:")
	switch {
	case strings.HasPrefix(listener.Owner, "resource:") && (databases[owner] || looksLikeDatabase(owner)):
		return ReasonDatabase
	case listener.PortName == "metrics":
		return ReasonMetrics
	case public:
		// A resource declared public_via_edge is still not routed: only the
		// scenario's own listeners can be public.
		return ReasonResourceOwned
	case listener.Visibility == domain.EdgeVisibilityPrivate:
		return ReasonDeclaredPrivate
	default:
		return ReasonUndeclared
	}
}

func looksLikeDatabase(component string) bool {
	lower := strings.ToLower(component)
	for _, marker := range []string{"postgres", "mysql", "mariadb", "redis", "mongo", "qdrant", "database"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func ipPolicy(in PolicyInputs) domain.EdgeIPPolicy {
	policy := domain.EdgeIPPolicy{IPv4: []string{}, IPv6: []string{}}
	add := func(value string) {
		ip := net.ParseIP(strings.TrimSpace(value))
		if ip == nil {
			return
		}
		if ip.To4() != nil {
			policy.IPv4 = append(policy.IPv4, ip.String())
		} else {
			policy.IPv6 = append(policy.IPv6, ip.String())
		}
	}
	add(in.TargetHost)
	for _, ip := range in.ObservedIPv4 {
		add(ip)
	}
	for _, ip := range in.ObservedIPv6 {
		add(ip)
	}
	policy.IPv4 = sortedUnique(policy.IPv4)
	policy.IPv6 = sortedUnique(policy.IPv6)
	return policy
}

func sortedUnique(in []string) []string {
	sort.Strings(in)
	out := make([]string, 0, len(in))
	for i, v := range in {
		if i == 0 || v != in[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// ValidateDomain accepts a lowercase registrable host name: no scheme, path,
// port, wildcard or IP literal, at least two labels, RFC 1123 labels.
func ValidateDomain(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.TrimSuffix(name, ".")
	invalid := func(reason string) (string, error) {
		return "", apierrors.New(CodeDomainInvalid, fmt.Sprintf("edge domain %q is not usable: %s", raw, reason)).WithDetail("domain", raw)
	}
	if name == "" {
		return invalid("empty")
	}
	if strings.ContainsAny(name, "/:?#@ \t") {
		return invalid("must be a bare host name without scheme, port or path")
	}
	if net.ParseIP(name) != nil {
		return invalid("must be a name, not an IP literal")
	}
	if strings.Contains(name, "*") {
		return invalid("wildcards are not routed")
	}
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return invalid("needs at least two labels")
	}
	if len(name) > 253 {
		return invalid("longer than 253 characters")
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return invalid("label length must be 1-63")
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return invalid("labels cannot start or end with a hyphen")
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return invalid("labels may only contain a-z, 0-9 and hyphen")
			}
		}
	}
	if tld := labels[len(labels)-1]; tld[0] >= '0' && tld[0] <= '9' {
		return invalid("top-level label cannot be numeric")
	}
	return name, nil
}

// ACMEInputs selects the issuance environment.
type ACMEInputs struct {
	// Requested is the manifest value: staging, production, or empty.
	Requested string
	// Environment is the deployment environment (production, staging, ...).
	Environment string
	Authority   ACMEAuthority
}

// ACME directory URLs per environment.
const (
	ACMEStagingCA    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	ACMEProductionCA = "https://acme-v02.api.letsencrypt.org/directory"
)

// SelectACME picks staging for repeated issuance tests and production only
// where authority exists: the operator's own production environment is that
// authority for itself; any other environment needs the EXT-04 grant
// (VROOLI_ACME_PRODUCTION_ALLOWED=true plus a declared budget) and is
// refused with acme_production_unauthorized otherwise.
func SelectACME(in ACMEInputs) (string, error) {
	requested := strings.ToLower(strings.TrimSpace(in.Requested))
	isProductionEnv := strings.EqualFold(strings.TrimSpace(in.Environment), "production")
	switch requested {
	case "":
		if isProductionEnv {
			return domain.ACMEEnvironmentProduction, nil
		}
		if in.Authority.ProductionAllowed && in.Authority.Budget > 0 {
			return domain.ACMEEnvironmentProduction, nil
		}
		return domain.ACMEEnvironmentStaging, nil
	case domain.ACMEEnvironmentStaging:
		return domain.ACMEEnvironmentStaging, nil
	case domain.ACMEEnvironmentProduction:
		if isProductionEnv || (in.Authority.ProductionAllowed && in.Authority.Budget > 0) {
			return domain.ACMEEnvironmentProduction, nil
		}
		return "", apierrors.New(CodeACMEProductionUnauthorized, "production ACME issuance for a non-production deployment requires the EXT-04 authority (VROOLI_ACME_PRODUCTION_ALLOWED=true and VROOLI_ACME_PRODUCTION_BUDGET>0); use staging for repeated issuance tests").
			WithDetail("environment", in.Environment).
			WithNextAction(apierrors.NextAction{Owner: "operator", Kind: "document", Reference: "ledgers/external-inputs.md#EXT-04", Label: "Grant a bounded production issuance budget"})
	default:
		return "", apierrors.New(CodeACMEEnvironmentInvalid, fmt.Sprintf("acme environment %q is not staging or production", in.Requested))
	}
}

// ACMECA returns the directory URL for an environment.
func ACMECA(environment string) string {
	if environment == domain.ACMEEnvironmentStaging {
		return ACMEStagingCA
	}
	return ACMEProductionCA
}

// ACMEAuthorityFromEnvironment reads the EXT-04 grant from the process
// environment. lookup is os.LookupEnv in production.
func ACMEAuthorityFromEnvironment(lookup func(string) (string, bool)) ACMEAuthority {
	var authority ACMEAuthority
	if value, ok := lookup("VROOLI_ACME_PRODUCTION_ALLOWED"); ok && strings.EqualFold(strings.TrimSpace(value), "true") {
		authority.ProductionAllowed = true
	}
	if value, ok := lookup("VROOLI_ACME_PRODUCTION_BUDGET"); ok {
		var budget int
		if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &budget); err == nil && budget > 0 {
			authority.Budget = budget
		}
	}
	return authority
}
