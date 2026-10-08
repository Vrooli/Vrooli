package edge

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
)

// Reason codes produced by the lifecycle checks.
const (
	ReasonDNSMatch          = "dns_match"
	ReasonDNSProxied        = "dns_proxied"
	ReasonDNSUnresolved     = "dns_unresolved"
	ReasonDNSMismatch       = "dns_mismatch"
	ReasonDNSIPv6Mismatch   = "dns_ipv6_mismatch"
	ReasonDNSNoRecord       = "dns_no_record"
	ReasonTargetUnobserved  = "target_address_unobserved"
	ReasonEdgePortBlocked   = "edge_port_blocked"
	ReasonEdgePortConflict  = "edge_port_conflict"
	ReasonEdgePortOwned     = "edge_port_owned_by_edge"
	ReasonFirewallClosed    = "edge_firewall_closed"
	ReasonLocalUnready      = "local_readiness_failed"
	ReasonExternalUnready   = "external_readiness_failed"
	ReasonExternalBlocked   = "external_readiness_blocked_by_dns"
	ReasonRenewalFailed     = "renewal_failed"
	ReasonCertExpiringSoon  = "certificate_expiring_soon"
	ReasonCertRenewalWindow = "certificate_renewal_window"
	ReasonCertExpired       = "certificate_expired"
	ReasonCertInvalid       = "certificate_invalid"
	ReasonTLSNotObserved    = "tls_not_observed"
)

// Certificate thresholds. The warning fires no later than 14 days before
// expiry (P14-A04); the renewal window mirrors the ACME client's own 30-day
// horizon so an unrenewed certificate is visible before it becomes urgent.
const (
	ExpiryWarningDays  = 14
	RenewalWindowDays  = 30
	DefaultJournalTail = 200
)

// BindDNS compares one host's records with the target's addresses, one
// address family at a time. IPv6 is never required: a host without AAAA
// records binds on IPv4 alone, and an IPv6-only target binds on AAAA alone.
// A Cloudflare-proxied host binds through the proxy.
func BindDNS(host string, observed []string, lookupErr error, policy domain.EdgeIPPolicy) domain.EdgeDNSBinding {
	binding := domain.EdgeDNSBinding{Host: host}
	v4, v6 := splitFamilies(observed)
	binding.IPv4 = domain.EdgeAddressBinding{Expected: nonNil(policy.IPv4), Observed: v4}
	binding.IPv6 = domain.EdgeAddressBinding{Expected: nonNil(policy.IPv6), Observed: v6}
	if lookupErr != nil {
		binding.IPv4.State, binding.IPv6.State = domain.EdgeBindingUnresolved, domain.EdgeBindingUnresolved
		binding.ReasonCode = ReasonDNSUnresolved
		return binding
	}
	if len(observed) > 0 && dns.AreCloudflareIPs(observed) {
		binding.IPv4.State, binding.IPv6.State = domain.EdgeBindingProxied, domain.EdgeBindingProxied
		binding.Match = true
		binding.ReasonCode = ReasonDNSProxied
		return binding
	}
	if len(policy.IPv4) == 0 && len(policy.IPv6) == 0 {
		binding.IPv4.State, binding.IPv6.State = domain.EdgeBindingUnresolved, domain.EdgeBindingUnresolved
		binding.ReasonCode = ReasonTargetUnobserved
		return binding
	}
	binding.IPv4.State = familyState(v4, policy.IPv4)
	binding.IPv6.State = familyState(v6, policy.IPv6)
	switch {
	case binding.IPv4.State == domain.EdgeBindingMismatch:
		binding.ReasonCode = ReasonDNSMismatch
	case binding.IPv6.State == domain.EdgeBindingMismatch:
		binding.ReasonCode = ReasonDNSIPv6Mismatch
	case binding.IPv4.State == domain.EdgeBindingMatch || binding.IPv6.State == domain.EdgeBindingMatch:
		binding.Match = true
		binding.ReasonCode = ReasonDNSMatch
	default:
		binding.ReasonCode = ReasonDNSNoRecord
	}
	return binding
}

func familyState(observed, expected []string) string {
	switch {
	case len(observed) == 0:
		return domain.EdgeBindingAbsent
	case len(expected) == 0:
		// A record exists for a family the target does not serve: clients
		// preferring it would reach the wrong host.
		return domain.EdgeBindingMismatch
	}
	want := map[string]bool{}
	for _, ip := range expected {
		want[ip] = true
	}
	for _, ip := range observed {
		if want[ip] {
			return domain.EdgeBindingMatch
		}
	}
	return domain.EdgeBindingMismatch
}

func splitFamilies(ips []string) ([]string, []string) {
	v4, v6 := []string{}, []string{}
	for _, raw := range ips {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			v4 = append(v4, ip.String())
		} else {
			v6 = append(v6, ip.String())
		}
	}
	sort.Strings(v4)
	sort.Strings(v6)
	return v4, v6
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

// PortObservation is one edge port as preflight saw it on the target.
type PortObservation struct {
	Port    int
	Process string
	Service string
}

// PortIssue is a typed edge-port finding.
type PortIssue struct {
	Port       int    `json:"port"`
	ReasonCode string `json:"reason_code"`
	Process    string `json:"process,omitempty"`
}

// ClassifyEdgePorts distinguishes a port held by the edge proxy itself
// (expected) from one held by a conflicting service (nginx, apache, an
// application bound to 80) and from one unreachable from outside
// (blocked by a firewall or security group).
func ClassifyEdgePorts(bound []PortObservation, unreachable []int) []PortIssue {
	var issues []PortIssue
	for _, b := range bound {
		if b.Port != 80 && b.Port != 443 {
			continue
		}
		if strings.EqualFold(b.Process, "caddy") || strings.EqualFold(b.Service, "caddy") {
			issues = append(issues, PortIssue{Port: b.Port, ReasonCode: ReasonEdgePortOwned, Process: b.Process})
			continue
		}
		issues = append(issues, PortIssue{Port: b.Port, ReasonCode: ReasonEdgePortConflict, Process: b.Process})
	}
	for _, port := range unreachable {
		issues = append(issues, PortIssue{Port: port, ReasonCode: ReasonEdgePortBlocked})
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Port != issues[j].Port {
			return issues[i].Port < issues[j].Port
		}
		return issues[i].ReasonCode < issues[j].ReasonCode
	})
	return issues
}

// HasConflict reports whether any issue is a conflicting service.
func HasConflict(issues []PortIssue) bool {
	for _, issue := range issues {
		if issue.ReasonCode == ReasonEdgePortConflict {
			return true
		}
	}
	return false
}

// Readiness combines the local and external probes. External readiness is
// blocked, not merely failed, while DNS does not bind the host to the
// target: a wrong record can never yield an external-ready claim.
func Readiness(localErr, externalErr error, dnsBound bool) domain.EdgeReadiness {
	out := domain.EdgeReadiness{Local: domain.EdgeReadinessCheck{Status: "passed"}, External: domain.EdgeReadinessCheck{Status: "passed"}}
	if localErr != nil {
		out.Local = domain.EdgeReadinessCheck{Status: "failed", ReasonCode: ReasonLocalUnready, Detail: localErr.Error()}
	}
	switch {
	case !dnsBound:
		out.External = domain.EdgeReadinessCheck{Status: "blocked", ReasonCode: ReasonExternalBlocked, Detail: "DNS does not bind the public host to this target; external readiness cannot be claimed"}
	case externalErr != nil:
		out.External = domain.EdgeReadinessCheck{Status: "failed", ReasonCode: ReasonExternalUnready, Detail: externalErr.Error()}
	}
	return out
}

// ExternallyReady is the only predicate a caller may use to claim public
// reach.
func ExternallyReady(r domain.EdgeReadiness) bool {
	return r.Local.Status == "passed" && r.External.Status == "passed"
}

// CertificateFacts is the subset of a TLS probe the lifecycle check reads.
type CertificateFacts struct {
	Present         bool
	Valid           bool
	ValidationError string
	Issuer          string
	NotAfter        time.Time
	SANs            []string
}

// RenewalEvidence is what the proxy journal says about issuance.
type RenewalEvidence struct {
	Observed    bool   `json:"observed"`
	Failed      bool   `json:"failed"`
	LastError   string `json:"last_error,omitempty"`
	LastSuccess string `json:"last_success,omitempty"`
}

// ObserveTLS derives the renewal state for one host from the live
// certificate and the journal evidence.
func ObserveTLS(host string, cert CertificateFacts, journal RenewalEvidence, now time.Time) domain.EdgeTLSState {
	state := domain.EdgeTLSState{Host: host}
	if !cert.Present {
		state.RenewalState = domain.EdgeRenewalNotObserved
		state.ReasonCode = ReasonTLSNotObserved
		if journal.Failed {
			state.RenewalState = domain.EdgeRenewalFailed
			state.ReasonCode = ReasonRenewalFailed
			state.Detail = journal.LastError
		}
		return state
	}
	state.Issuer = cert.Issuer
	state.ACMEEnvironment = acmeEnvironmentFromIssuer(cert.Issuer)
	if !cert.NotAfter.IsZero() {
		state.NotAfter = cert.NotAfter.UTC().Format(time.RFC3339)
		state.DaysLeft = int(cert.NotAfter.Sub(now).Hours() / 24)
	}
	switch {
	case !cert.Valid && !hostnameCovered(host, cert.SANs):
		state.RenewalState = domain.EdgeRenewalHostnameWrong
		state.ReasonCode = ReasonCertInvalid
		state.Detail = cert.ValidationError
	case !cert.NotAfter.IsZero() && !cert.NotAfter.After(now):
		state.RenewalState = domain.EdgeRenewalExpired
		state.ReasonCode = ReasonCertExpired
	case journal.Failed:
		state.RenewalState = domain.EdgeRenewalFailed
		state.ReasonCode = ReasonRenewalFailed
		state.Detail = journal.LastError
	case !cert.Valid:
		state.RenewalState = domain.EdgeRenewalInvalidChain
		state.ReasonCode = ReasonCertInvalid
		state.Detail = cert.ValidationError
	case state.DaysLeft <= ExpiryWarningDays:
		state.RenewalState = domain.EdgeRenewalExpiringSoon
		state.ReasonCode = ReasonCertExpiringSoon
	case state.DaysLeft <= RenewalWindowDays:
		state.RenewalState = domain.EdgeRenewalWindow
		state.ReasonCode = ReasonCertRenewalWindow
	default:
		state.RenewalState = domain.EdgeRenewalValid
	}
	return state
}

// TLSNeedsAttention says whether the state should surface as a health
// warning (or failure) with a next action.
func TLSNeedsAttention(state domain.EdgeTLSState) (warn bool, fail bool) {
	switch state.RenewalState {
	case domain.EdgeRenewalExpired, domain.EdgeRenewalHostnameWrong, domain.EdgeRenewalInvalidChain:
		return false, true
	case domain.EdgeRenewalFailed, domain.EdgeRenewalExpiringSoon:
		return true, false
	default:
		return false, false
	}
}

func hostnameCovered(host string, sans []string) bool {
	host = strings.ToLower(host)
	for _, san := range sans {
		san = strings.ToLower(san)
		if san == host {
			return true
		}
		if strings.HasPrefix(san, "*.") && strings.HasSuffix(host, san[1:]) && strings.Count(host, ".") == strings.Count(san, ".") {
			return true
		}
	}
	return false
}

func acmeEnvironmentFromIssuer(issuer string) string {
	lower := strings.ToLower(issuer)
	switch {
	case strings.Contains(lower, "staging"):
		return domain.ACMEEnvironmentStaging
	case strings.Contains(lower, "let's encrypt") || strings.Contains(lower, "r1") || strings.Contains(lower, "e1") || strings.Contains(lower, "zerossl"):
		return domain.ACMEEnvironmentProduction
	default:
		return ""
	}
}

// JournalArgv is the bounded read of the proxy journal. Argv only.
func JournalArgv(lines int) []string {
	if lines <= 0 || lines > 1000 {
		lines = DefaultJournalTail
	}
	return []string{"journalctl", "-u", "caddy", "--no-pager", "-n", strconv.Itoa(lines), "-o", "cat"}
}

var (
	journalFailure = regexp.MustCompile(`(?i)("level":"error"|\berror\b).*(obtain|renew|issu|challenge|acme|certificate)`)
	journalSuccess = regexp.MustCompile(`(?i)certificate obtained successfully|successfully renewed|certificate renewed`)
	journalMessage = regexp.MustCompile(`"(?:msg|error)":"((?:[^"\\]|\\.)*)"`)
)

// ParseCaddyJournal reads the bounded journal tail. A failure followed by a
// later success is not a failure. Token-like values are never copied into
// the evidence: only the msg/error fields of matching lines are retained.
func ParseCaddyJournal(text string) RenewalEvidence {
	evidence := RenewalEvidence{Observed: strings.TrimSpace(text) != ""}
	for _, line := range strings.Split(text, "\n") {
		switch {
		case journalSuccess.MatchString(line):
			evidence.Failed = false
			evidence.LastError = ""
			evidence.LastSuccess = extractMessage(line)
		case journalFailure.MatchString(line):
			evidence.Failed = true
			evidence.LastError = extractMessage(line)
		}
	}
	return evidence
}

func extractMessage(line string) string {
	matches := journalMessage.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 200 {
			trimmed = trimmed[:200]
		}
		return trimmed
	}
	parts := make([]string, 0, len(matches))
	for _, m := range matches {
		parts = append(parts, m[1])
	}
	joined := strings.Join(parts, ": ")
	if len(joined) > 300 {
		joined = joined[:300]
	}
	return joined
}

// Next actions name owner operations, never shell repair.

// NextActionFirewall opens the edge ports through the target owner.
func NextActionFirewall(port int) domain.NextActionHint {
	return domain.NextActionHint{Owner: "vrooli", Kind: "command", Reference: fmt.Sprintf(`vrooli cloud-target host repair --action edge.ufw.allow --subject '{"edge":{"port":%d}}'`, port), Label: fmt.Sprintf("Allow inbound %d/tcp through the privilege broker", port)}
}

// NextActionFreePort stops the conflicting scenario through the lifecycle
// owner when it is a Vrooli workload, and names the unit otherwise.
func NextActionFreePort(issue PortIssue) domain.NextActionHint {
	label := fmt.Sprintf("Port %d is held by %s; the edge proxy needs it", issue.Port, issue.Process)
	if issue.Process == "" {
		label = fmt.Sprintf("Port %d is held by another process; the edge proxy needs it", issue.Port)
	}
	return domain.NextActionHint{Owner: "vrooli", Kind: "command", Reference: `vrooli cloud-target host repair --action process.stop.scoped --subject '{"process":{"scenario":"<scenario-id>","workdir":"<deployment-workdir>"}}'`, Label: label}
}

// NextActionRouteApply re-applies the deployment snippet transactionally.
func NextActionRouteApply(deploymentID string) domain.NextActionHint {
	return domain.NextActionHint{Owner: "scenario-to-cloud", Kind: "command", Reference: "scenario-to-cloud edge route apply " + deploymentID, Label: "Re-apply the edge route (validated, rolled back on failure)"}
}

// NextActionDNS points at the DNS observation.
func NextActionDNS(deploymentID string) domain.NextActionHint {
	return domain.NextActionHint{Owner: "scenario-to-cloud", Kind: "endpoint", Reference: "/api/v1/deployments/" + deploymentID + "/edge", Label: "Inspect DNS binding and the target addresses"}
}

// NextActionRenew triggers a certificate renewal attempt.
func NextActionRenew(deploymentID string) domain.NextActionHint {
	return domain.NextActionHint{Owner: "scenario-to-cloud", Kind: "command", Reference: "scenario-to-cloud edge tls-renew " + deploymentID, Label: "Attempt certificate renewal and read the proxy journal"}
}
