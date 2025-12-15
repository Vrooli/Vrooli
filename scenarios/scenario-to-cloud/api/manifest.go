package main

import (
	"fmt"
	"sort"
	"strings"
)

type ValidationIssueSeverity string

const (
	SeverityError ValidationIssueSeverity = "error"
	SeverityWarn  ValidationIssueSeverity = "warn"
)

type ValidationIssue struct {
	Path     string                  `json:"path"`
	Message  string                  `json:"message"`
	Hint     string                  `json:"hint,omitempty"`
	Severity ValidationIssueSeverity `json:"severity"`
}

func hasBlockingIssues(issues []ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

type CloudManifest struct {
	Version      string               `json:"version"`
	Environment  string               `json:"environment,omitempty"`
	Target       ManifestTarget       `json:"target"`
	Scenario     ManifestScenario     `json:"scenario"`
	Dependencies ManifestDependencies `json:"dependencies"`
	Bundle       ManifestBundle       `json:"bundle"`
	Ports        ManifestPorts        `json:"ports"`
	Edge         ManifestEdge         `json:"edge"`
}

type ManifestTarget struct {
	Type string       `json:"type"`
	VPS  *ManifestVPS `json:"vps,omitempty"`
}

type ManifestVPS struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path,omitempty"`
	Workdir string `json:"workdir,omitempty"`
}

type ManifestScenario struct {
	ID  string `json:"id"`
	Ref string `json:"ref,omitempty"`
}

type ManifestDependencies struct {
	Scenarios []string `json:"scenarios,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Analyzer  struct {
		Tool        string `json:"tool,omitempty"`
		Fingerprint string `json:"fingerprint,omitempty"`
		GeneratedAt string `json:"generated_at,omitempty"`
	} `json:"analyzer,omitempty"`
}

type ManifestBundle struct {
	IncludePackages bool     `json:"include_packages"`
	IncludeAutoheal bool     `json:"include_autoheal"`
	Scenarios       []string `json:"scenarios,omitempty"`
	Resources       []string `json:"resources,omitempty"`
}

type ManifestPorts struct {
	UI  int `json:"ui,omitempty"`
	API int `json:"api,omitempty"`
	WS  int `json:"ws,omitempty"`
}

type ManifestEdge struct {
	Domain string        `json:"domain"`
	Caddy  ManifestCaddy `json:"caddy"`
}

type ManifestCaddy struct {
	Enabled bool   `json:"enabled"`
	Email   string `json:"email,omitempty"`
}

type ManifestValidateResponse struct {
	Valid      bool              `json:"valid"`
	Issues     []ValidationIssue `json:"issues,omitempty"`
	Manifest   CloudManifest     `json:"manifest"`
	Timestamp  string            `json:"timestamp"`
	SchemaHint string            `json:"schema_hint,omitempty"`
}

func ValidateAndNormalizeManifest(in CloudManifest) (CloudManifest, []ValidationIssue) {
	out := in
	var issues []ValidationIssue

	if strings.TrimSpace(out.Version) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "version",
			Message:  "manifest version is required",
			Hint:     "Set version to a stable schema version like '1.0.0'.",
			Severity: SeverityError,
		})
	}

	if strings.TrimSpace(out.Target.Type) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "target.type",
			Message:  "target type is required",
			Hint:     "For P0, target.type must be 'vps'.",
			Severity: SeverityError,
		})
	} else if out.Target.Type != "vps" {
		issues = append(issues, ValidationIssue{
			Path:     "target.type",
			Message:  fmt.Sprintf("unsupported target.type %q", out.Target.Type),
			Hint:     "For P0, only 'vps' is supported.",
			Severity: SeverityError,
		})
	}

	if out.Target.Type == "vps" {
		if out.Target.VPS == nil {
			issues = append(issues, ValidationIssue{
				Path:     "target.vps",
				Message:  "target.vps is required when target.type is 'vps'",
				Hint:     "Provide at least {\"host\":\"<ip-or-hostname>\"} and SSH auth info.",
				Severity: SeverityError,
			})
		} else {
			if strings.TrimSpace(out.Target.VPS.Host) == "" {
				issues = append(issues, ValidationIssue{
					Path:     "target.vps.host",
					Message:  "VPS host is required",
					Hint:     "Set to the VPS public IP (recommended) or DNS name.",
					Severity: SeverityError,
				})
			}
			if out.Target.VPS.Port == 0 {
				out.Target.VPS.Port = 22
			}
			if strings.TrimSpace(out.Target.VPS.User) == "" {
				out.Target.VPS.User = "root"
			}
			if strings.TrimSpace(out.Target.VPS.Workdir) == "" {
				out.Target.VPS.Workdir = "/root/Vrooli"
			}
		}
	}

	if strings.TrimSpace(out.Scenario.ID) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "scenario.id",
			Message:  "scenario id is required",
			Hint:     "Example: 'landing-page-business-suite'.",
			Severity: SeverityError,
		})
	}

	// Dependencies snapshot (scenario-dependency-analyzer) should be explicit and include the target scenario.
	out.Dependencies.Scenarios = stableUniqueStrings(out.Dependencies.Scenarios)
	out.Dependencies.Resources = stableUniqueStrings(out.Dependencies.Resources)
	if len(out.Dependencies.Scenarios) == 0 {
		issues = append(issues, ValidationIssue{
			Path:     "dependencies.scenarios",
			Message:  "dependencies.scenarios is required",
			Hint:     "deployment-manager must include the scenario-dependency-analyzer result snapshot.",
			Severity: SeverityError,
		})
	} else if out.Scenario.ID != "" && !contains(out.Dependencies.Scenarios, out.Scenario.ID) {
		issues = append(issues, ValidationIssue{
			Path:     "dependencies.scenarios",
			Message:  "dependencies.scenarios must include scenario.id",
			Hint:     "Ensure the analyzer graph includes the target scenario itself.",
			Severity: SeverityError,
		})
	}

	if strings.TrimSpace(out.Dependencies.Analyzer.Tool) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "dependencies.analyzer.tool",
			Message:  "dependencies.analyzer.tool should be set",
			Hint:     "Set to 'scenario-dependency-analyzer' for traceability.",
			Severity: SeverityWarn,
		})
	}

	// Bundle inclusions should be explicit for deterministic packaging.
	out.Bundle.Scenarios = stableUniqueStrings(out.Bundle.Scenarios)
	out.Bundle.Resources = stableUniqueStrings(out.Bundle.Resources)
	if len(out.Bundle.Scenarios) == 0 && len(out.Dependencies.Scenarios) > 0 {
		out.Bundle.Scenarios = append([]string(nil), out.Dependencies.Scenarios...)
	}
	if len(out.Bundle.Resources) == 0 && len(out.Dependencies.Resources) > 0 {
		out.Bundle.Resources = append([]string(nil), out.Dependencies.Resources...)
	}

	if out.Scenario.ID != "" && !contains(out.Bundle.Scenarios, out.Scenario.ID) {
		issues = append(issues, ValidationIssue{
			Path:     "bundle.scenarios",
			Message:  "bundle.scenarios must include scenario.id",
			Hint:     "deployment-manager should include the full scenario list being packaged, including the target scenario.",
			Severity: SeverityError,
		})
	}

	if out.Ports.UI == 0 {
		out.Ports.UI = 3000
	}
	if out.Ports.API == 0 {
		out.Ports.API = 3001
	}
	if out.Ports.WS == 0 {
		out.Ports.WS = 3002
	}
	if invalid := findInvalidPorts(out.Ports); len(invalid) > 0 {
		for _, p := range invalid {
			issues = append(issues, ValidationIssue{
				Path:     fmt.Sprintf("ports.%s", p.Name),
				Message:  "port must be between 1 and 65535",
				Hint:     fmt.Sprintf("Set ports.%s to a valid TCP port.", p.Name),
				Severity: SeverityError,
			})
		}
	}
	if duplicates := findDuplicatePorts(out.Ports); len(duplicates) > 0 {
		issues = append(issues, ValidationIssue{
			Path:     "ports",
			Message:  "ports must be distinct",
			Hint:     fmt.Sprintf("Choose distinct ports; duplicates: %s.", strings.Join(duplicates, ", ")),
			Severity: SeverityError,
		})
	}

	if strings.TrimSpace(out.Edge.Domain) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "edge.domain",
			Message:  "edge.domain is required",
			Hint:     "DNS must already point to the VPS for Let’s Encrypt HTTP-01.",
			Severity: SeverityError,
		})
	} else if !looksLikeDomain(out.Edge.Domain) {
		issues = append(issues, ValidationIssue{
			Path:     "edge.domain",
			Message:  "edge.domain must be a hostname (no scheme, no path)",
			Hint:     "Example: 'example.com' or 'app.example.com' (not 'https://example.com').",
			Severity: SeverityError,
		})
	}

	if !out.Edge.Caddy.Enabled {
		issues = append(issues, ValidationIssue{
			Path:     "edge.caddy.enabled",
			Message:  "Caddy should be enabled for P0",
			Hint:     "Set edge.caddy.enabled=true to provision TLS via Let’s Encrypt.",
			Severity: SeverityWarn,
		})
	}

	if out.Edge.Caddy.Enabled && strings.TrimSpace(out.Edge.Caddy.Email) == "" {
		issues = append(issues, ValidationIssue{
			Path:     "edge.caddy.email",
			Message:  "edge.caddy.email should be set when Caddy is enabled",
			Hint:     "Set to an ops email for Let's Encrypt expiry and incident contacts (recommended).",
			Severity: SeverityWarn,
		})
	}

	if !out.Bundle.IncludePackages {
		issues = append(issues, ValidationIssue{
			Path:     "bundle.include_packages",
			Message:  "packages/ must be included for P0",
			Hint:     "Set bundle.include_packages=true (P0 vendors all workspace packages).",
			Severity: SeverityError,
		})
	}

	if !out.Bundle.IncludeAutoheal {
		issues = append(issues, ValidationIssue{
			Path:     "bundle.include_autoheal",
			Message:  "vrooli-autoheal must be included for P0",
			Hint:     "Set bundle.include_autoheal=true so the mini-Vrooli install keeps Tier-1 health guarantees.",
			Severity: SeverityError,
		})
	}

	if out.Bundle.IncludeAutoheal && !contains(out.Bundle.Scenarios, "vrooli-autoheal") {
		out.Bundle.Scenarios = stableUniqueStrings(append(out.Bundle.Scenarios, "vrooli-autoheal"))
		issues = append(issues, ValidationIssue{
			Path:     "bundle.scenarios",
			Message:  "vrooli-autoheal was missing and has been added automatically",
			Hint:     "deployment-manager should add vrooli-autoheal as a hard dependency for VPS deployments so the exported manifest is explicit.",
			Severity: SeverityWarn,
		})
	}

	return out, stableIssues(issues)
}

func looksLikeDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}
	if strings.Contains(domain, "://") || strings.Contains(domain, "/") {
		return false
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	host := strings.TrimSuffix(domain, ".")
	if len(host) > 253 {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	return true
}

func findDuplicatePorts(p ManifestPorts) []string {
	seen := map[int]string{}
	var duplicates []string
	for name, value := range map[string]int{"ui": p.UI, "api": p.API, "ws": p.WS} {
		if value <= 0 || value > 65535 {
			continue
		}
		if existing, ok := seen[value]; ok {
			duplicates = append(duplicates, fmt.Sprintf("%s=%d (dup of %s)", name, value, existing))
			continue
		}
		seen[value] = name
	}
	return duplicates
}

type invalidPort struct {
	Name  string
	Value int
}

func findInvalidPorts(p ManifestPorts) []invalidPort {
	var invalid []invalidPort
	for name, value := range map[string]int{"ui": p.UI, "api": p.API, "ws": p.WS} {
		if value <= 0 || value > 65535 {
			invalid = append(invalid, invalidPort{Name: name, Value: value})
		}
	}
	sort.Slice(invalid, func(i, j int) bool { return invalid[i].Name < invalid[j].Name })
	return invalid
}

func stableIssues(in []ValidationIssue) []ValidationIssue {
	out := append([]ValidationIssue(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func stableUniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func contains(in []string, value string) bool {
	for _, v := range in {
		if v == value {
			return true
		}
	}
	return false
}
