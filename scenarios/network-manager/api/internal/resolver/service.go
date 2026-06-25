package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const AdGuardHomeBackend = "adguard-home"

type Service struct {
	repo         Repository
	client       AdGuardClient
	dnsInspector HostDNSInspector
}

type Config struct {
	Repo         Repository
	Client       AdGuardClient
	DNSInspector HostDNSInspector
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, client: cfg.Client, dnsInspector: cfg.DNSInspector}
	if s.client == nil {
		s.client = ConservativeAdGuardClient{}
	}
	if s.dnsInspector == nil {
		s.dnsInspector = DefaultHostDNSInspector{}
	}
	return s
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		return Status{
			Backend:           AdGuardHomeBackend,
			Status:            "not_configured",
			EnforcementStatus: "unverified",
			Warnings:          []string{"No governed AdGuard Home backend is configured."},
		}, nil
	}
	if err != nil {
		return Status{}, err
	}
	return s.statusFromClient(ctx, cfg)
}

func (s *Service) ConfigureAdGuardHome(ctx context.Context, baseURL, username, tokenRef string, dryRun bool) (Status, []string, error) {
	cfg, err := normalizeConfig(baseURL, username, tokenRef)
	if err != nil {
		return Status{}, nil, err
	}
	if dryRun {
		return Status{
			Backend:           AdGuardHomeBackend,
			Status:            "dry_run",
			BaseURL:           cfg.BaseURL,
			EnforcementStatus: "unverified",
			Warnings:          []string{"Dry run only; credentials were not stored and resolver state was not changed."},
		}, []string{"Configuration shape is valid.", "Store the credential in the Vrooli secret system and pass only its token_ref."}, nil
	}
	saved, err := s.repo.SaveBackend(ctx, cfg)
	if err != nil {
		return Status{}, nil, err
	}
	status, err := s.statusFromClient(ctx, saved)
	if err != nil {
		return Status{}, nil, err
	}
	return status, []string{"Stored AdGuard Home backend using a secret reference.", "Filtering remains unclaimed until health confirms it."}, nil
}

func (s *Service) UpdateUpstreams(ctx context.Context, upstreams []string, dryRun bool) (Status, []string, error) {
	cleaned, err := normalizeUpstreams(upstreams)
	if err != nil {
		return Status{}, nil, err
	}
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		return Status{}, nil, fmt.Errorf("configure AdGuard Home before updating upstreams")
	}
	if err != nil {
		return Status{}, nil, err
	}
	if dryRun {
		changes, err := s.client.PreviewUpstreams(ctx, cfg, cleaned)
		if err != nil {
			return Status{}, nil, err
		}
		status, err := s.statusFromClient(ctx, cfg)
		if err != nil {
			return Status{}, nil, err
		}
		return status, changes, nil
	}
	clientStatus, changes, err := s.client.UpdateUpstreams(ctx, cfg, cleaned)
	if err != nil {
		return Status{}, nil, err
	}
	if err := s.repo.UpdateUpstreams(ctx, AdGuardHomeBackend, cleaned); err != nil {
		return Status{}, nil, err
	}
	return fromClientStatus(cfg, clientStatus), changes, nil
}

func (s *Service) Health(ctx context.Context) (Status, []string, error) {
	cfg, err := s.repo.GetBackend(ctx, AdGuardHomeBackend)
	if errors.Is(err, ErrNotFound) {
		status := Status{Backend: AdGuardHomeBackend, Status: "not_configured", EnforcementStatus: "unverified", Warnings: []string{"No AdGuard Home backend is configured."}}
		return status, []string{"No backend configuration found."}, nil
	}
	if err != nil {
		return Status{}, nil, err
	}
	clientStatus, err := s.client.Check(ctx, cfg)
	if err != nil {
		return Status{}, nil, err
	}
	return fromClientStatus(cfg, clientStatus), clientStatus.Checks, nil
}

func (s *Service) AdGuardRollout(ctx context.Context) (RolloutReport, error) {
	status, checks, err := s.Health(ctx)
	if err != nil {
		return RolloutReport{}, err
	}
	dnsBindIP := firstNonEmpty(os.Getenv("ADGUARD_HOME_DNS_BIND_IP"), defaultAdGuardDNSBindIP())
	inspection := s.dnsInspector.InspectHostDNS(ctx, dnsBindIP)
	report := RolloutReport{
		DNSBindIP:      dnsBindIP,
		ResolverStatus: status,
		RouterSettings: routerSettings(dnsBindIP),
		Warnings:       append([]string{}, status.Warnings...),
	}
	report.Checks = append(report.Checks, adGuardHealthRolloutCheck(status, checks))
	report.Checks = append(report.Checks, hostDNSRolloutCheck(dnsBindIP, inspection))
	report.Checks = append(report.Checks, clientEvidenceRolloutCheck(status))
	report.Checks = append(report.Checks, routerRolloutCheck(dnsBindIP, status))
	report.Checks = append(report.Checks, ipv6RolloutCheck(dnsBindIP, inspection))
	report.Warnings = append(report.Warnings, inspection.Warnings...)
	report.Status, report.Summary, report.NextSteps = summarizeRollout(report.Checks, dnsBindIP)
	return report, nil
}

func defaultAdGuardDNSBindIP() string {
	return net.IPv4(192, 168, 1, 173).String()
}

func (s *Service) statusFromClient(ctx context.Context, cfg BackendConfig) (Status, error) {
	clientStatus, err := s.client.Check(ctx, cfg)
	if err != nil {
		return Status{}, err
	}
	status := fromClientStatus(cfg, clientStatus)
	if len(status.Upstreams) == 0 {
		upstreams, err := s.repo.GetUpstreams(ctx, AdGuardHomeBackend)
		if err != nil {
			return Status{}, err
		}
		status.Upstreams = upstreams
	}
	return status, nil
}

func normalizeConfig(baseURL, username, tokenRef string) (BackendConfig, error) {
	baseURL = firstNonEmpty(baseURL, os.Getenv("ADGUARD_HOME_BASE_URL"), os.Getenv("ADGUARD_HOME_URL"))
	username = firstNonEmpty(username, os.Getenv("ADGUARD_HOME_USERNAME"))
	tokenRef = firstNonEmpty(tokenRef, os.Getenv("ADGUARD_HOME_CREDENTIAL_REF"))
	if baseURL == "" {
		return BackendConfig{}, fmt.Errorf("base_url is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return BackendConfig{}, fmt.Errorf("base_url must be an absolute URL")
	}
	if tokenRef == "" {
		return BackendConfig{}, fmt.Errorf("token_ref is required; plaintext resolver tokens are not accepted")
	}
	return BackendConfig{Backend: AdGuardHomeBackend, BaseURL: strings.TrimRight(baseURL, "/"), Username: username, TokenRef: tokenRef}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func normalizeUpstreams(values []string) ([]string, error) {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one upstream is required")
	}
	return out, nil
}

func fromClientStatus(cfg BackendConfig, clientStatus ClientStatus) Status {
	status := clientStatus.Status
	if status == "" {
		status = "unknown"
	}
	return Status{
		Backend:             cfg.Backend,
		Status:              status,
		BaseURL:             cfg.BaseURL,
		Upstreams:           clientStatus.Upstreams,
		FilteringEnabled:    clientStatus.FilteringEnabled,
		Warnings:            clientStatus.Warnings,
		EnforcementStatus:   fallbackStatus(clientStatus.EnforcementStatus),
		EnforcementEvidence: clientStatus.EnforcementEvidence,
	}
}

func fallbackStatus(status string) string {
	if status != "" {
		return status
	}
	return "unverified"
}

func joinUpstreams(upstreams []string) string {
	if len(upstreams) == 0 {
		return "(none)"
	}
	return strings.Join(upstreams, ", ")
}

type DefaultHostDNSInspector struct{}

func (DefaultHostDNSInspector) InspectHostDNS(ctx context.Context, targetDNS string) DNSInspection {
	return inspectHostDNS(ctx, targetDNS)
}

func inspectHostDNS(ctx context.Context, targetDNS string) DNSInspection {
	var out DNSInspection
	for _, path := range hostDNSConfigPaths() {
		if err := ctx.Err(); err != nil {
			out.Warnings = append(out.Warnings, "Host DNS inspection was cancelled.")
			break
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		servers := extractResolvConfServers(string(data))
		if len(servers) == 0 {
			continue
		}
		out.Evidence = append(out.Evidence, fmt.Sprintf("Read DNS server configuration from %s.", path))
		out.Servers = append(out.Servers, servers...)
	}
	out.Servers = cleanUniqueStrings(out.Servers)
	if len(out.Servers) == 0 {
		out.Warnings = append(out.Warnings, "No host DNS servers could be detected.")
	} else if containsString(out.Servers, targetDNS) {
		out.Evidence = append(out.Evidence, fmt.Sprintf("This host is configured to use %s for DNS.", targetDNS))
	} else {
		out.Evidence = append(out.Evidence, fmt.Sprintf("Detected host DNS servers: %s.", strings.Join(out.Servers, ", ")))
	}
	return out
}

func hostDNSConfigPaths() []string {
	return []string{"/etc/resolv.conf", "/run/systemd/resolve/resolv.conf", "/run/systemd/resolve/stub-resolv.conf"}
}

func extractResolvConfServers(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			out = append(out, fields[1])
		}
	}
	return out
}

func adGuardHealthRolloutCheck(status Status, checks []string) RolloutCheck {
	if status.Status == "healthy" && status.FilteringEnabled {
		return RolloutCheck{ID: "adguard-resource", Title: "AdGuard resource protection", Status: "passed", Evidence: "AdGuard is healthy and DNS filtering is enabled.", Recommendations: checks}
	}
	return RolloutCheck{ID: "adguard-resource", Title: "AdGuard resource protection", Status: "blocked", Evidence: fmt.Sprintf("Resolver status is %s and filtering_enabled=%t.", status.Status, status.FilteringEnabled), Recommendations: []string{"Start and bootstrap the adguard-home resource before changing router DNS.", "`resource-adguard-home api-health --json` should report healthy with protection enabled."}}
}

func hostDNSRolloutCheck(dnsBindIP string, inspection DNSInspection) RolloutCheck {
	if containsString(inspection.Servers, dnsBindIP) {
		return RolloutCheck{ID: "this-host-dns", Title: "This Vrooli host DNS", Status: "passed", Evidence: fmt.Sprintf("This host is using %s for DNS.", dnsBindIP), Recommendations: []string{"Use this as a local smoke test before router rollout."}}
	}
	evidence := "This host is not confirmed to use the AdGuard DNS listener."
	if len(inspection.Servers) > 0 {
		evidence = fmt.Sprintf("Detected host DNS servers: %s.", strings.Join(inspection.Servers, ", "))
	}
	return RolloutCheck{ID: "this-host-dns", Title: "This Vrooli host DNS", Status: "manual_required", Evidence: evidence, Recommendations: []string{fmt.Sprintf("Set this host's DNS server to %s if you want local protection before router rollout.", dnsBindIP), "Re-run `network-manager resolver rollout --json` after changing DNS."}}
}

func clientEvidenceRolloutCheck(status Status) RolloutCheck {
	if status.EnforcementStatus == "client_evidence_observed" || len(status.EnforcementEvidence) > 0 {
		return RolloutCheck{ID: "client-evidence", Title: "Observed AdGuard clients", Status: "review_required", Evidence: strings.Join(status.EnforcementEvidence, " "), Recommendations: []string{"Client evidence proves some clients have used AdGuard, not that the router assigns it to every client.", "`network-manager devices refresh --json` imports this evidence without query-level DNS logs."}}
	}
	return RolloutCheck{ID: "client-evidence", Title: "Observed AdGuard clients", Status: "manual_required", Evidence: "No usable AdGuard client metadata has been observed yet.", Recommendations: []string{"After router DNS changes, renew client DHCP leases or reconnect Wi-Fi clients.", "Then run `network-manager devices refresh --json`."}}
}

func routerRolloutCheck(dnsBindIP string, status Status) RolloutCheck {
	if status.EnforcementStatus == "verified" {
		return RolloutCheck{ID: "router-dhcp", Title: "Router DHCP DNS assignment", Status: "passed", Evidence: "Router/client DNS assignment is verified.", Recommendations: []string{"Continue monitoring snapshots and resolver health."}}
	}
	return RolloutCheck{ID: "router-dhcp", Title: "Router DHCP DNS assignment", Status: "manual_required", Evidence: "Network Manager has not verified that router DHCP assigns AdGuard to household clients.", Recommendations: []string{fmt.Sprintf("Reserve this server at %s in the router DHCP/static lease table.", dnsBindIP), fmt.Sprintf("Set router DHCP IPv4 DNS server to %s.", dnsBindIP), "Renew client DHCP leases or reconnect clients, then refresh inventory and snapshots."}}
}

func ipv6RolloutCheck(dnsBindIP string, inspection DNSInspection) RolloutCheck {
	var ipv6Servers []string
	for _, server := range inspection.Servers {
		if strings.Contains(server, ":") && server != "::1" {
			ipv6Servers = append(ipv6Servers, server)
		}
	}
	if len(ipv6Servers) == 0 {
		return RolloutCheck{ID: "ipv6-rdnss", Title: "IPv6 DNS/RDNSS path", Status: "review_required", Evidence: "No non-loopback IPv6 DNS server was detected on this host.", Recommendations: []string{"Confirm the router is not advertising unmanaged IPv6 DNS to clients.", "If the router supports IPv6 DNS/RDNSS, point it at a managed AdGuard IPv6 listener or disable unmanaged IPv6 DNS advertisement until supported."}}
	}
	return RolloutCheck{ID: "ipv6-rdnss", Title: "IPv6 DNS/RDNSS path", Status: "manual_required", Evidence: fmt.Sprintf("Detected IPv6 DNS server(s): %s.", strings.Join(ipv6Servers, ", ")), Recommendations: []string{"Update router IPv6 RDNSS/DHCPv6 DNS to the managed resolver path, or disable unmanaged IPv6 DNS advertisement until managed.", fmt.Sprintf("IPv4 AdGuard DNS remains %s; do not assume IPv6 clients are covered by that alone.", dnsBindIP)}}
}

func routerSettings(dnsBindIP string) []string {
	return []string{
		fmt.Sprintf("Reserve/static lease: assign this Vrooli server %s.", dnsBindIP),
		fmt.Sprintf("Router DHCP IPv4 DNS server: %s.", dnsBindIP),
		"Router DHCP secondary DNS: leave blank or use the same managed resolver; do not add ISP DNS as fallback if filtering must be enforced.",
		"IPv6 RDNSS/DHCPv6 DNS: point to a managed AdGuard IPv6 listener when available, or disable unmanaged IPv6 DNS advertisement until managed.",
	}
}

func summarizeRollout(checks []RolloutCheck, dnsBindIP string) (string, string, []string) {
	blocked := hasCheckStatus(checks, "blocked")
	routerManual := checkStatus(checks, "router-dhcp") == "manual_required"
	hostPassed := checkStatus(checks, "this-host-dns") == "passed"
	if blocked {
		return "blocked", "AdGuard is not ready for household rollout yet.", []string{"Fix blocked checks before changing router DNS."}
	}
	if routerManual {
		steps := []string{
			fmt.Sprintf("Log into the router and reserve %s for this Vrooli server.", dnsBindIP),
			fmt.Sprintf("Set router DHCP IPv4 DNS to %s.", dnsBindIP),
			"Review IPv6 RDNSS/DHCPv6 DNS so clients cannot bypass AdGuard through unmanaged IPv6 DNS.",
			"Renew client DHCP leases or reconnect clients.",
			"Run `network-manager devices refresh --json`, `network-manager resolver rollout --json`, and a new network snapshot.",
		}
		if hostPassed {
			return "host_protected_router_manual", "AdGuard is protecting this host; household-wide enforcement still needs router DHCP/RDNSS rollout.", steps
		}
		return "ready_for_router_rollout", "AdGuard is healthy; router DHCP/RDNSS rollout is still manual and unverified.", steps
	}
	return "household_verified", "Household-wide AdGuard enforcement is verified.", []string{"Keep monitoring resolver health, device evidence, and network snapshots."}
}

func hasCheckStatus(checks []RolloutCheck, status string) bool {
	for _, check := range checks {
		if check.Status == status {
			return true
		}
	}
	return false
}

func checkStatus(checks []RolloutCheck, id string) string {
	for _, check := range checks {
		if check.ID == id {
			return check.Status
		}
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
