package manifest

import (
	"fmt"
	"sort"
	"strings"
)

// Validate runs structural validation against the manifest. Returns
// every diagnostic the validator finds; severity Error means the
// manifest is unusable. Phase 4 deepens the cross-reference checks
// (e.g., allowed_dependencies points to a declared domain) — Phase 2
// covers the load-bearing minimum so service tests can exercise the
// error path.
func Validate(m ManifestDefinition) []Diagnostic {
	var out []Diagnostic
	if strings.TrimSpace(m.Scenario) == "" {
		out = append(out, Diagnostic{
			Severity: DiagnosticSeverityError,
			Path:     "scenario",
			Message:  "scenario is required",
			Code:     "MANIFEST_SCENARIO_REQUIRED",
		})
	}
	if m.Version != ManifestVersionUnspecified && m.Version != ManifestVersionV1 {
		out = append(out, Diagnostic{
			Severity: DiagnosticSeverityError,
			Path:     "version",
			Message:  fmt.Sprintf("unknown manifest_version %q", m.Version),
			Code:     "MANIFEST_UNKNOWN_VERSION",
		})
	}

	seenDomain := make(map[string]int, len(m.Domains))
	declared := make(map[string]struct{}, len(m.Domains))
	for i, d := range m.Domains {
		path := fmt.Sprintf("domains[%d]", i)
		if strings.TrimSpace(d.Name) == "" {
			out = append(out, Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     path + ".name",
				Message:  "domain name is required",
				Code:     "MANIFEST_DOMAIN_NAME_REQUIRED",
			})
			continue
		}
		if prev, ok := seenDomain[d.Name]; ok {
			out = append(out, Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     path + ".name",
				Message:  fmt.Sprintf("duplicate domain name %q (first at domains[%d])", d.Name, prev),
				Code:     "MANIFEST_DOMAIN_DUPLICATE",
			})
		}
		seenDomain[d.Name] = i
		declared[d.Name] = struct{}{}
		if len(d.Paths) == 0 {
			out = append(out, Diagnostic{
				Severity: DiagnosticSeverityWarn,
				Path:     path + ".paths",
				Message:  fmt.Sprintf("domain %q declares no paths; nothing will be assigned to it", d.Name),
				Code:     "MANIFEST_DOMAIN_NO_PATHS",
			})
		}
	}

	for i, d := range m.Domains {
		for j, dep := range d.AllowedDependencies {
			if _, ok := declared[dep]; ok {
				continue
			}
			out = append(out, Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     fmt.Sprintf("domains[%d].allowed_dependencies[%d]", i, j),
				Message:  fmt.Sprintf("references unknown domain %q", dep),
				Code:     "MANIFEST_UNKNOWN_DOMAIN_REF",
			})
		}
	}

	for i, t := range m.Thresholds {
		if t.MinValue < 0 || t.MinValue > 1 {
			out = append(out, Diagnostic{
				Severity: DiagnosticSeverityError,
				Path:     fmt.Sprintf("thresholds[%d].min_value", i),
				Message:  fmt.Sprintf("threshold %q min_value %.3f outside [0,1]", t.Tier, t.MinValue),
				Code:     "MANIFEST_THRESHOLD_OUT_OF_RANGE",
			})
		}
	}

	// Stable diagnostic order for test golden files.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
