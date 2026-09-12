package domain

import "time"

// EdgeSpecSchemaVersion is the versioned wire shape of the declared edge
// contract (P14-O01). It is digested and carried on the plan.
const EdgeSpecSchemaVersion = "1"

// Listener visibility vocabulary, shared with the closure declaration.
const (
	EdgeVisibilityPublicViaEdge = "public_via_edge"
	EdgeVisibilityPrivate       = "private"
)

// ACME issuance environments. Staging is the default for anything that is
// not the operator's production environment; production issuance against a
// non-production deployment needs explicit authority (EXT-04).
const (
	ACMEEnvironmentStaging    = "staging"
	ACMEEnvironmentProduction = "production"
)

// EdgeRoute is one public host the edge terminates and forwards to one
// loopback listener.
type EdgeRoute struct {
	Host         string `json:"host"`
	UpstreamPort int    `json:"upstream_port"`
	ListenerID   string `json:"listener_id"`
}

// EdgePrivateListener is a listener the edge must never route: databases,
// metrics, management, and every port not declared public_via_edge.
type EdgePrivateListener struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	PortName string `json:"port_name,omitempty"`
	Port     int    `json:"port,omitempty"`
	// Reason is the policy rule that kept it private: declared_private,
	// undeclared, management, database, metrics.
	Reason string `json:"reason"`
}

// EdgeIPPolicy records the target addresses the public hosts must resolve
// to. IPv4 and IPv6 are independent: an absent family is not a failure.
type EdgeIPPolicy struct {
	IPv4 []string `json:"ipv4"`
	IPv6 []string `json:"ipv6"`
}

// EdgeDNSProvider names the DNS-01 provider and the credential descriptor
// that holds its token. The value never appears here.
type EdgeDNSProvider struct {
	Provider   string               `json:"provider"`
	Descriptor CredentialDescriptor `json:"descriptor"`
	// EnvVar is the environment variable the proxy reads the token from on
	// the target; the credential authority delivers it there.
	EnvVar string `json:"env_var"`
}

// EdgeSpec is the declared listener visibility and edge configuration
// contract for one deployment.
type EdgeSpec struct {
	SchemaVersion    string                `json:"schema_version"`
	DeploymentID     string                `json:"deployment_id"`
	ScenarioID       string                `json:"scenario_id"`
	Domain           string                `json:"domain"`
	Routes           []EdgeRoute           `json:"routes"`
	PrivateListeners []EdgePrivateListener `json:"private_listeners"`
	FirewallAllow    []int                 `json:"firewall_allow"`
	IPPolicy         EdgeIPPolicy          `json:"ip_policy"`
	ACMEEnvironment  string                `json:"acme_environment"`
	ACMEEmail        string                `json:"acme_email,omitempty"`
	DNSProvider      *EdgeDNSProvider      `json:"dns_provider,omitempty"`
	// Digest is sha256 over the canonical JSON of everything above.
	Digest string `json:"digest"`
}

// EdgeAddressBinding compares one address family of a host with the target.
type EdgeAddressBinding struct {
	Expected []string `json:"expected"`
	Observed []string `json:"observed"`
	// State is match, mismatch, absent (no record of this family) or
	// unresolved (lookup failed).
	State string `json:"state"`
}

// Address binding states.
const (
	EdgeBindingMatch      = "match"
	EdgeBindingMismatch   = "mismatch"
	EdgeBindingAbsent     = "absent"
	EdgeBindingUnresolved = "unresolved"
	EdgeBindingProxied    = "proxied"
)

// EdgeDNSBinding is the per-host DNS observation.
type EdgeDNSBinding struct {
	Host       string             `json:"host"`
	IPv4       EdgeAddressBinding `json:"ipv4"`
	IPv6       EdgeAddressBinding `json:"ipv6"`
	Match      bool               `json:"match"`
	ReasonCode string             `json:"reason_code,omitempty"`
}

// Renewal states reported for the edge certificate.
const (
	EdgeRenewalValid         = "valid"
	EdgeRenewalWindow        = "renewal_window"
	EdgeRenewalExpiringSoon  = "expiring_soon"
	EdgeRenewalExpired       = "expired"
	EdgeRenewalFailed        = "renewal_failed"
	EdgeRenewalNotObserved   = "not_observed"
	EdgeRenewalInvalidChain  = "invalid_chain"
	EdgeRenewalHostnameWrong = "hostname_mismatch"
)

// EdgeTLSState is the certificate lifecycle observation for one host.
type EdgeTLSState struct {
	Host         string `json:"host"`
	Issuer       string `json:"issuer,omitempty"`
	NotAfter     string `json:"not_after,omitempty"`
	DaysLeft     int    `json:"days_left"`
	RenewalState string `json:"renewal_state"`
	ReasonCode   string `json:"reason_code,omitempty"`
	Detail       string `json:"detail,omitempty"`
	// ACMEEnvironment is inferred from the issuer when possible.
	ACMEEnvironment string `json:"acme_environment,omitempty"`
}

// EdgeReadiness separates local readiness (the workload answers on its
// loopback listener) from external HTTPS readiness (the public host answers
// through the edge). External readiness can never be claimed while DNS does
// not bind the host to the target.
type EdgeReadiness struct {
	Local    EdgeReadinessCheck `json:"local"`
	External EdgeReadinessCheck `json:"external"`
}

// EdgeReadinessCheck is one readiness verdict.
type EdgeReadinessCheck struct {
	Status     string `json:"status"` // passed, failed, blocked, not_checked
	ReasonCode string `json:"reason_code,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// EdgeObservation is the secret-free public exposure record served at
// GET /api/v1/deployments/{id}/edge (P14-O06 step 16).
type EdgeObservation struct {
	SchemaVersion    string                `json:"schema_version"`
	DeploymentID     string                `json:"deployment_id"`
	Domain           string                `json:"domain"`
	SpecDigest       string                `json:"spec_digest"`
	Routes           []EdgeRoute           `json:"routes"`
	PrivateListeners []EdgePrivateListener `json:"private_listeners"`
	DNS              []EdgeDNSBinding      `json:"dns"`
	TLS              []EdgeTLSState        `json:"tls"`
	ACMEEnvironment  string                `json:"acme_environment"`
	Readiness        EdgeReadiness         `json:"readiness"`
	ObservedAt       time.Time             `json:"observed_at"`
	ProducerRef      string                `json:"producer_ref"`
	NextActions      []NextActionHint      `json:"next_actions,omitempty"`
}

// NextActionHint is the owner-operation pointer carried by observations and
// preflight checks in place of ad-hoc shell hints.
type NextActionHint struct {
	Owner     string `json:"owner"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	Label     string `json:"label"`
}
