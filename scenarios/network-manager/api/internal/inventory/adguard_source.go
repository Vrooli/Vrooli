package inventory

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"network-manager/internal/resolver"
)

const adGuardClientsEndpoint = "/control/clients"

type AdGuardClientDiscoverySource struct {
	Backends resolver.Repository
	Secrets  resolver.SecretResolver
	HTTP     http.RoundTripper
	Timeout  time.Duration
}

func NewAdGuardClientDiscoverySource(backends resolver.Repository, secrets resolver.SecretResolver) AdGuardClientDiscoverySource {
	if secrets == nil {
		secrets = resolver.NewCredentialAuthorityResolver()
	}
	return AdGuardClientDiscoverySource{Backends: backends, Secrets: secrets, Timeout: 10 * time.Second}
}

func (s AdGuardClientDiscoverySource) Discover(ctx context.Context) ([]Observation, []string, error) {
	client, findings, err := s.client(ctx)
	if err != nil {
		return nil, findings, ErrUnsupported
	}

	payload, code, err := resolver.GetAdGuardJSON[adGuardClientsPayload](ctx, client, adGuardClientsEndpoint)
	if err != nil {
		return nil, []string{"AdGuard Home clients endpoint could not be reached; inventory was not changed."}, err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return nil, []string{"AdGuard Home rejected the configured credentials; inventory was not changed."}, ErrUnsupported
	}
	if code < 200 || code >= 300 {
		return nil, []string{fmt.Sprintf("AdGuard Home clients endpoint returned HTTP %d; inventory was not changed.", code)}, ErrUnsupported
	}

	observations := observationsFromAdGuardClients(payload)
	if len(observations) == 0 {
		return nil, []string{"AdGuard Home returned no configured or automatically discovered clients."}, nil
	}
	return observations, []string{
		fmt.Sprintf("Imported %d AdGuard Home client observation(s) without query-level DNS log data.", len(observations)),
	}, nil
}

func (s AdGuardClientDiscoverySource) client(ctx context.Context) (*resolver.AdGuardHomeClient, []string, error) {
	if s.Backends == nil {
		return nil, []string{"Resolver backend repository is unavailable; AdGuard client discovery is disabled."}, fmt.Errorf("resolver backend repository unavailable")
	}
	cfg, err := s.Backends.GetBackend(ctx, resolver.AdGuardHomeBackend)
	if err != nil {
		return nil, []string{"AdGuard Home resolver backend is not configured; no resolver client evidence was imported."}, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.CredentialRef) == "" {
		return nil, []string{"AdGuard Home resolver backend is missing base_url or credential_ref; no resolver client evidence was imported."}, fmt.Errorf("adguard backend incomplete")
	}
	secrets := s.Secrets
	if secrets == nil {
		secrets = resolver.NewCredentialAuthorityResolver()
	}
	creds, err := secrets.ResolveAdGuardCredentials(ctx, cfg)
	if err != nil {
		return nil, []string{"AdGuard Home credential could not be resolved through the credential authority; no resolver client evidence was imported."}, err
	}
	return resolver.NewAdGuardHomeClient(cfg.BaseURL, creds, s.HTTP, s.Timeout), nil, nil
}

type adGuardClientsPayload struct {
	Clients     []adGuardClientEntry `json:"clients"`
	AutoClients []adGuardClientEntry `json:"auto_clients"`
}

type adGuardClientEntry struct {
	Name   string   `json:"name"`
	IDs    []string `json:"ids"`
	IP     string   `json:"ip"`
	Source string   `json:"source"`
}

func observationsFromAdGuardClients(payload adGuardClientsPayload) []Observation {
	out := make([]Observation, 0, len(payload.Clients)+len(payload.AutoClients))
	for _, client := range payload.Clients {
		if obs, ok := observationFromAdGuardClient(client, "adguard-home/configured"); ok {
			out = append(out, obs)
		}
	}
	for _, client := range payload.AutoClients {
		if obs, ok := observationFromAdGuardClient(client, "adguard-home/auto"); ok {
			out = append(out, obs)
		}
	}
	return out
}

func observationFromAdGuardClient(client adGuardClientEntry, source string) (Observation, bool) {
	ids := cleanInventoryStrings(client.IDs)
	name := strings.TrimSpace(client.Name)
	if isAdGuardSystemAlias(name) {
		return Observation{}, false
	}
	ip := firstIP(strings.TrimSpace(client.IP), ids)
	mac := firstMAC(ids)
	resolverID := firstResolverClientID(ids)
	if source == "adguard-home/auto" && resolverID == "" {
		resolverID = firstNonIPID(ids)
	}
	obs := Observation{
		Source:           source,
		Hostname:         name,
		IPAddress:        ip,
		MACAddress:       mac,
		ResolverClientID: resolverID,
	}
	if source == "adguard-home/configured" && resolverID != "" {
		obs.StableID = "adguard-client:" + resolverID
	}
	return obs, obs.IPAddress != "" || obs.MACAddress != "" || obs.ResolverClientID != ""
}

func isAdGuardSystemAlias(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "localhost" || strings.HasPrefix(name, "ip6-")
}

func firstIP(primary string, ids []string) string {
	if ip := parseClientIP(primary); ip != "" {
		return ip
	}
	for _, id := range ids {
		if ip := parseClientIP(id); ip != "" {
			return ip
		}
	}
	return ""
}

func firstMAC(ids []string) string {
	for _, id := range ids {
		if mac, err := net.ParseMAC(id); err == nil {
			return strings.ToLower(mac.String())
		}
	}
	return ""
}

func firstResolverClientID(ids []string) string {
	for _, id := range ids {
		if parseClientIP(id) != "" || parseAnyIP(id) != nil {
			continue
		}
		if _, err := net.ParseMAC(id); err == nil {
			return strings.ToLower(id)
		}
		if strings.TrimSpace(id) != "" && !strings.Contains(id, "/") {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func firstNonIPID(ids []string) string {
	for _, id := range ids {
		if parseAnyIP(id) == nil && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func parseClientIP(value string) string {
	ip := parseAnyIP(value)
	if ip == nil || !ip.IsGlobalUnicast() {
		return ""
	}
	return ip.String()
}

func parseAnyIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "/") {
		ip, _, err := net.ParseCIDR(value)
		if err == nil {
			return ip
		}
		return nil
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip
	}
	return nil
}

func cleanInventoryStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
