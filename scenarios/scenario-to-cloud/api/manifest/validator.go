package manifest

import (
	"fmt"
	"sort"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/vps"
)

// HasBlockingIssues returns true if any issue has error severity.
func HasBlockingIssues(issues []domain.ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Severity == domain.SeverityError {
			return true
		}
	}
	return false
}

// ValidateAndNormalize validates a domain.CloudManifest and returns a normalized copy
// along with any validation issues found.
func ValidateAndNormalize(in domain.CloudManifest) (domain.CloudManifest, []domain.ValidationIssue) {
	out := in
	var issues []domain.ValidationIssue

	if strings.TrimSpace(out.Version) == "" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "version",
			Message:  "manifest version is required",
			Hint:     "Set version to a stable schema version like '1.0.0'.",
			Severity: domain.SeverityError,
		})
	}

	if strings.TrimSpace(out.Target.Type) == "" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "target.type",
			Message:  "target type is required",
			Hint:     "For P0, target.type must be 'vps'.",
			Severity: domain.SeverityError,
		})
	} else if out.Target.Type != "vps" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "target.type",
			Message:  fmt.Sprintf("unsupported target.type %q", out.Target.Type),
			Hint:     "For P0, only 'vps' is supported.",
			Severity: domain.SeverityError,
		})
	}

	if out.Target.Type == "vps" {
		if out.Target.VPS == nil {
			issues = append(issues, domain.ValidationIssue{
				Path:     "target.vps",
				Message:  "target.vps is required when target.type is 'vps'",
				Hint:     "Provide at least {\"host\":\"<ip-or-hostname>\"} and SSH auth info.",
				Severity: domain.SeverityError,
			})
		} else {
			if strings.TrimSpace(out.Target.VPS.Host) == "" {
				issues = append(issues, domain.ValidationIssue{
					Path:     "target.vps.host",
					Message:  "VPS host is required",
					Hint:     "Set to the VPS public IP (recommended) or DNS name.",
					Severity: domain.SeverityError,
				})
			}
			if out.Target.VPS.Port == 0 {
				out.Target.VPS.Port = 22
			}
			if strings.TrimSpace(out.Target.VPS.User) == "" {
				out.Target.VPS.User = "root"
			}
			if strings.TrimSpace(out.Target.VPS.Workdir) == "" {
				out.Target.VPS.Workdir = domain.DefaultVPSWorkdir
			}
		}
	}

	if strings.TrimSpace(out.Scenario.ID) == "" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "scenario.id",
			Message:  "scenario id is required",
			Hint:     "Example: 'landing-page-business-suite'.",
			Severity: domain.SeverityError,
		})
	}

	// Dependencies snapshot (scenario-dependency-analyzer) should be explicit and include the target scenario.
	out.Dependencies.Scenarios = StableUniqueStrings(out.Dependencies.Scenarios)
	out.Dependencies.Resources = StableUniqueStrings(out.Dependencies.Resources)
	out = mergeRequiredResources(out, &issues)
	if len(out.Dependencies.Scenarios) == 0 {
		issues = append(issues, domain.ValidationIssue{
			Path:     "dependencies.scenarios",
			Message:  "dependencies.scenarios is required",
			Hint:     "deployment-manager must include the scenario-dependency-analyzer result snapshot.",
			Severity: domain.SeverityError,
		})
	} else if out.Scenario.ID != "" && !Contains(out.Dependencies.Scenarios, out.Scenario.ID) {
		issues = append(issues, domain.ValidationIssue{
			Path:     "dependencies.scenarios",
			Message:  "dependencies.scenarios must include scenario.id",
			Hint:     "Ensure the analyzer graph includes the target scenario itself.",
			Severity: domain.SeverityError,
		})
	}

	if strings.TrimSpace(out.Dependencies.Analyzer.Tool) == "" {
		out.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"
		issues = append(issues, domain.ValidationIssue{
			Path:     "dependencies.analyzer.tool",
			Message:  "dependencies.analyzer.tool should be set",
			Hint:     "Set to 'scenario-dependency-analyzer' for traceability.",
			Severity: domain.SeverityWarn,
		})
	}

	// Bundle inclusions should be explicit for deterministic packaging.
	out.Bundle.Scenarios = StableUniqueStrings(out.Bundle.Scenarios)
	out.Bundle.Resources = StableUniqueStrings(out.Bundle.Resources)
	if len(out.Bundle.Scenarios) == 0 && len(out.Dependencies.Scenarios) > 0 {
		out.Bundle.Scenarios = append([]string(nil), out.Dependencies.Scenarios...)
	}
	if len(out.Bundle.Resources) == 0 && len(out.Dependencies.Resources) > 0 {
		out.Bundle.Resources = append([]string(nil), out.Dependencies.Resources...)
	}

	if out.Scenario.ID != "" && !Contains(out.Bundle.Scenarios, out.Scenario.ID) {
		issues = append(issues, domain.ValidationIssue{
			Path:     "bundle.scenarios",
			Message:  "bundle.scenarios must include scenario.id",
			Hint:     "deployment-manager should include the full scenario list being packaged, including the target scenario.",
			Severity: domain.SeverityError,
		})
	}

	// Initialize ports map if nil
	if out.Ports == nil {
		out.Ports = make(domain.ManifestPorts)
	}
	// Set defaults for standard ports if not specified
	if out.Ports["ui"] == 0 {
		out.Ports["ui"] = 3000
	}
	if out.Ports["api"] == 0 {
		out.Ports["api"] = 3001
	}
	if out.Ports["ws"] == 0 {
		out.Ports["ws"] = 3002
	}
	if invalid := findInvalidPorts(out.Ports); len(invalid) > 0 {
		for _, p := range invalid {
			issues = append(issues, domain.ValidationIssue{
				Path:     fmt.Sprintf("ports.%s", p.Name),
				Message:  "port must be between 1 and 65535",
				Hint:     fmt.Sprintf("Set ports.%s to a valid TCP port.", p.Name),
				Severity: domain.SeverityError,
			})
		}
	}
	if duplicates := findDuplicatePorts(out.Ports); len(duplicates) > 0 {
		issues = append(issues, domain.ValidationIssue{
			Path:     "ports",
			Message:  "ports must be distinct",
			Hint:     fmt.Sprintf("Choose distinct ports; duplicates: %s.", strings.Join(duplicates, ", ")),
			Severity: domain.SeverityError,
		})
	}

	if strings.TrimSpace(out.Edge.Domain) == "" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "edge.domain",
			Message:  "edge.domain is required",
			Hint:     "DNS must already point to the VPS for Let's Encrypt HTTP-01.",
			Severity: domain.SeverityError,
		})
	} else if !looksLikeDomain(out.Edge.Domain) {
		issues = append(issues, domain.ValidationIssue{
			Path:     "edge.domain",
			Message:  "edge.domain must be a hostname (no scheme, no path)",
			Hint:     "Example: 'example.com' or 'app.example.com' (not 'https://example.com').",
			Severity: domain.SeverityError,
		})
	}

	if strings.TrimSpace(string(out.Edge.DNSPolicy)) == "" {
		out.Edge.DNSPolicy = domain.DNSPolicyRequired
	} else if out.Edge.DNSPolicy != domain.DNSPolicyRequired &&
		out.Edge.DNSPolicy != domain.DNSPolicyWarn &&
		out.Edge.DNSPolicy != domain.DNSPolicySkip {
		issues = append(issues, domain.ValidationIssue{
			Path:     "edge.dns_policy",
			Message:  "edge.dns_policy must be required, warn, or skip",
			Hint:     "Use 'required' to block on DNS, 'warn' to allow warnings, or 'skip' to defer DNS checks.",
			Severity: domain.SeverityError,
		})
	}

	if !out.Edge.Caddy.Enabled {
		issues = append(issues, domain.ValidationIssue{
			Path:     "edge.caddy.enabled",
			Message:  "Caddy should be enabled for P0",
			Hint:     "Set edge.caddy.enabled=true to provision TLS via Let's Encrypt.",
			Severity: domain.SeverityWarn,
		})
	}

	if out.Edge.Caddy.Enabled && strings.TrimSpace(out.Edge.Caddy.Email) == "" {
		issues = append(issues, domain.ValidationIssue{
			Path:     "edge.caddy.email",
			Message:  "edge.caddy.email should be set when Caddy is enabled",
			Hint:     "Set to an ops email for Let's Encrypt expiry and incident contacts (recommended).",
			Severity: domain.SeverityWarn,
		})
	}

	if !out.Bundle.IncludePackages {
		issues = append(issues, domain.ValidationIssue{
			Path:     "bundle.include_packages",
			Message:  "packages/ must be included for P0",
			Hint:     "Set bundle.include_packages=true (P0 vendors all workspace packages).",
			Severity: domain.SeverityError,
		})
	}

	if !out.Bundle.IncludeAutoheal {
		issues = append(issues, domain.ValidationIssue{
			Path:     "bundle.include_autoheal",
			Message:  "vrooli-autoheal must be included for P0",
			Hint:     "Set bundle.include_autoheal=true so the mini-Vrooli install keeps Tier-1 health guarantees.",
			Severity: domain.SeverityError,
		})
	}

	if out.Bundle.IncludeAutoheal && !Contains(out.Bundle.Scenarios, "vrooli-autoheal") {
		out.Bundle.Scenarios = StableUniqueStrings(append(out.Bundle.Scenarios, "vrooli-autoheal"))
		issues = append(issues, domain.ValidationIssue{
			Path:     "bundle.scenarios",
			Message:  "vrooli-autoheal was missing and has been added automatically",
			Hint:     "deployment-manager should add vrooli-autoheal as a hard dependency for VPS deployments so the exported manifest is explicit.",
			Severity: domain.SeverityWarn,
		})
	}

	return out, stableIssues(issues)
}

// mergeRequiredResources adds any required resources from the scenario's service.json
// that are missing from the manifest.
func mergeRequiredResources(out domain.CloudManifest, issues *[]domain.ValidationIssue) domain.CloudManifest {
	if out.Scenario.ID == "" {
		return out
	}
	required, err := vps.RequiredResourcesForScenario(out.Scenario.ID)
	if err != nil {
		if issues != nil {
			*issues = append(*issues, domain.ValidationIssue{
				Path:     "dependencies.resources",
				Message:  "unable to verify required resources from service.json",
				Hint:     err.Error(),
				Severity: domain.SeverityWarn,
			})
		}
		return out
	}
	if len(required) == 0 {
		return out
	}
	var missing []string
	for _, res := range required {
		if !Contains(out.Dependencies.Resources, res) {
			missing = append(missing, res)
		}
	}
	if len(missing) == 0 {
		return out
	}
	out.Dependencies.Resources = StableUniqueStrings(append(out.Dependencies.Resources, missing...))
	if issues != nil {
		*issues = append(*issues, domain.ValidationIssue{
			Path:     "dependencies.resources",
			Message:  "required resources were missing and have been added",
			Hint:     "Ensure scenario-dependency-analyzer includes dependencies.resources from .vrooli/service.json when exporting manifests.",
			Severity: domain.SeverityWarn,
		})
	}
	return out
}

// looksLikeDomain returns true if the string appears to be a valid domain name.
func looksLikeDomain(d string) bool {
	d = strings.TrimSpace(d)
	if d == "" {
		return false
	}
	if strings.Contains(d, "://") || strings.Contains(d, "/") {
		return false
	}
	if strings.HasPrefix(d, ".") || strings.HasSuffix(d, ".") {
		return false
	}
	host := strings.TrimSuffix(d, ".")
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

// findDuplicatePorts returns descriptions of any duplicate port assignments.
func findDuplicatePorts(p domain.ManifestPorts) []string {
	seen := map[int]string{}
	var duplicates []string
	// Sort keys for deterministic output
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		value := p[name]
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

// findInvalidPorts returns ports outside the valid TCP port range.
func findInvalidPorts(p domain.ManifestPorts) []invalidPort {
	var invalid []invalidPort
	for name, value := range p {
		if value <= 0 || value > 65535 {
			invalid = append(invalid, invalidPort{Name: name, Value: value})
		}
	}
	sort.Slice(invalid, func(i, j int) bool { return invalid[i].Name < invalid[j].Name })
	return invalid
}

// stableIssues sorts issues for deterministic output (errors before warnings, then by path).
func stableIssues(in []domain.ValidationIssue) []domain.ValidationIssue {
	out := append([]domain.ValidationIssue(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// StableUniqueStrings returns a sorted, deduplicated copy of the input slice.
func StableUniqueStrings(in []string) []string {
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

// Contains returns true if value is in the slice.
func Contains(in []string, value string) bool {
	for _, v := range in {
		if v == value {
			return true
		}
	}
	return false
}
