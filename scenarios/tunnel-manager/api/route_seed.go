package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// CloudflaredIngress represents a single ingress rule in cloudflared config.
type CloudflaredIngress struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}

// CloudflaredConfig represents a cloudflared config.yml.
type CloudflaredConfig struct {
	Tunnel          string               `yaml:"tunnel,omitempty"`
	CredentialsFile string               `yaml:"credentials-file,omitempty"`
	WarpRouting     map[string]any       `yaml:"warp-routing,omitempty"`
	Ingress         []CloudflaredIngress `yaml:"ingress"`
}

// SeedFromConfig reads the cloudflared config file and seeds routes that don't already exist.
// Returns the number of routes seeded and any error.
func (rs *RouteService) SeedFromConfig(configPath string) (int, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, fmt.Errorf("read cloudflared config: %w", err)
	}

	var cfg CloudflaredConfig
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
			log.Printf("seed: skipping %s — cannot extract port from %q", ingress.Hostname, ingress.Service)
			continue
		}

		scenarioName := subdomain // best guess: subdomain matches scenario name

		_, err := rs.Create(RouteInput{
			Subdomain:    subdomain,
			ScenarioName: scenarioName,
			LocalPort:    port,
			PublicURL:    "https://" + ingress.Hostname,
		})
		if err != nil {
			log.Printf("seed: failed to create route for %s: %v", subdomain, err)
			continue
		}
		seeded++
	}

	return seeded, nil
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
