package service

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"tunnel-manager/domain"

	"gopkg.in/yaml.v3"
)

// RouteService handles route manifest CRUD operations with validation.
type RouteService struct {
	store RouteStore
}

func NewRouteService(s RouteStore) *RouteService {
	return &RouteService{store: s}
}

func (rs *RouteService) List() ([]domain.Route, error) {
	routes, err := rs.store.List()
	if err != nil {
		return nil, domain.ErrInternal("list routes", err)
	}
	return routes, nil
}

func (rs *RouteService) GetByID(id int) (*domain.Route, error) {
	r, err := rs.store.GetByID(id)
	if err != nil {
		return nil, domain.ErrInternal("get route", err)
	}
	if r == nil {
		return nil, domain.ErrNotFound("route not found")
	}
	return r, nil
}

func (rs *RouteService) Create(in domain.RouteInput) (*domain.Route, error) {
	if err := validateRouteInput(in, false); err != nil {
		return nil, err
	}
	healthPath := "/health"
	if in.HealthPath != "" {
		healthPath = in.HealthPath
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	r, err := rs.store.Create(in.Subdomain, in.ScenarioName, in.LocalPort, healthPath, in.PublicURL, enabled)
	if err != nil {
		return nil, domain.ErrInternal("create route", err)
	}
	return r, nil
}

func (rs *RouteService) Update(id int, in domain.RouteInput) (*domain.Route, error) {
	if err := validateRouteInput(in, true); err != nil {
		return nil, err
	}
	existing, err := rs.store.GetByID(id)
	if err != nil {
		return nil, domain.ErrInternal("get route for update", err)
	}
	if existing == nil {
		return nil, domain.ErrNotFound("route not found")
	}

	mergeRouteFields(existing, in)

	r, err := rs.store.Update(id, existing.Subdomain, existing.ScenarioName, existing.LocalPort, existing.HealthPath, existing.PublicURL, existing.Enabled)
	if err != nil {
		return nil, domain.ErrInternal("update route", err)
	}
	return r, nil
}

func (rs *RouteService) Delete(id int) error {
	return rs.store.Delete(id)
}

// SeedFromConfig reads the cloudflared config file and seeds routes that don't already exist.
// Returns the number of routes seeded and any error.
func (rs *RouteService) SeedFromConfig(configPath string) (int, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("read cloudflared config: %w", err)
	}

	var cfg domain.CloudflaredConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return 0, fmt.Errorf("parse cloudflared config: %w", err)
	}

	existing, err := rs.List()
	if err != nil {
		return 0, fmt.Errorf("list existing routes: %w", err)
	}

	// Build set of existing subdomains for dedup
	existingSubdomains := make(map[string]bool)
	for _, r := range existing {
		existingSubdomains[r.Subdomain] = true
	}

	seeded := 0
	for _, ingress := range cfg.Ingress {
		// Skip catch-all (no hostname) rules
		if ingress.Hostname == "" {
			continue
		}
		// Skip http_status rules (e.g., "http_status:404")
		if strings.HasPrefix(ingress.Service, "http_status:") {
			continue
		}

		subdomain := extractSubdomain(ingress.Hostname)
		if subdomain == "" {
			continue
		}

		// Skip if already exists
		if existingSubdomains[subdomain] {
			continue
		}

		port := extractPortFromService(ingress.Service)
		if port == 0 {
			slog.Warn("seed: skipping route, cannot extract port", "hostname", ingress.Hostname, "service", ingress.Service)
			continue
		}

		scenarioName := subdomain // best guess: subdomain matches scenario name

		_, err := rs.Create(domain.RouteInput{
			Subdomain:    subdomain,
			ScenarioName: scenarioName,
			LocalPort:    port,
			PublicURL:    "https://" + ingress.Hostname,
		})
		if err != nil {
			slog.Warn("seed: failed to create route", "subdomain", subdomain, "error", err)
			continue
		}
		seeded++
	}

	return seeded, nil
}

// mergeRouteFields applies non-zero fields from in onto existing.
func mergeRouteFields(existing *domain.Route, in domain.RouteInput) {
	if in.Subdomain != "" {
		existing.Subdomain = in.Subdomain
	}
	if in.ScenarioName != "" {
		existing.ScenarioName = in.ScenarioName
	}
	if in.LocalPort != 0 {
		existing.LocalPort = in.LocalPort
	}
	if in.HealthPath != "" {
		existing.HealthPath = in.HealthPath
	}
	if in.PublicURL != "" {
		existing.PublicURL = in.PublicURL
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
}

func validateRouteInput(in domain.RouteInput, isUpdate bool) error {
	if !isUpdate {
		if in.Subdomain == "" {
			return domain.ErrValidation("subdomain is required")
		}
		if in.ScenarioName == "" {
			return domain.ErrValidation("scenario_name is required")
		}
		if in.LocalPort == 0 {
			return domain.ErrValidation("local_port is required")
		}
	}
	if in.LocalPort != 0 && (in.LocalPort < 1 || in.LocalPort > 65535) {
		return domain.ErrValidation("local_port must be between 1 and 65535")
	}
	return nil
}

// extractSubdomain gets the first subdomain component from a hostname.
// e.g., "agent-manager.itsagitime.com" -> "agent-manager"
func extractSubdomain(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) < 3 {
		return "" // not a subdomain (e.g., "example.com")
	}
	return parts[0]
}

// extractPortFromService parses a service URL like "http://localhost:36238" to extract the port.
func extractPortFromService(service string) int {
	idx := strings.LastIndex(service, ":")
	if idx < 0 {
		return 0
	}
	portStr := service[idx+1:]
	if slashIdx := strings.Index(portStr, "/"); slashIdx >= 0 {
		portStr = portStr[:slashIdx]
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0
	}
	return port
}
