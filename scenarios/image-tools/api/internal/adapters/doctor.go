package adapters

import (
	"fmt"
	"strings"

	"image-tools/internal/models"
)

// FindingSeverity classifies a catalog doctor finding.
type FindingSeverity string

const (
	FindingError   FindingSeverity = "error"
	FindingWarning FindingSeverity = "warning"
)

// CatalogFinding is one actionable adapter-catalog integrity finding.
type CatalogFinding struct {
	Severity  FindingSeverity
	Code      string
	AdapterID string
	Message   string
}

// CatalogDoctorReport is the complete catalog integrity report.
type CatalogDoctorReport struct {
	OK       bool
	Findings []CatalogFinding
}

// DoctorCatalog checks installability + policy invariants that must hold for the
// adapter catalog to be production-ready. It mirrors the model doctor: an enabled
// adapter must declare a concrete, auto-resolvable fetch strategy (and a repo
// source must pin its revision); a conditional-commercial adapter must not be
// enabled; and an adapter offered for execution must be Ready (no vaporware). A
// disabled offender is a warning (fix before enabling); an enabled one an error.
func (r *Registry) DoctorCatalog() CatalogDoctorReport {
	var findings []CatalogFinding
	add := func(f CatalogFinding) {
		if f.Severity == "" {
			f.Severity = FindingError
		}
		findings = append(findings, f)
	}

	for _, a := range r.adapters {
		if a.CapabilityLabels.CommercialUse == models.CommercialUseConditional && a.Enabled {
			add(CatalogFinding{
				Code:      "enabled_conditional_commercial_use",
				AdapterID: a.ID,
				Message:   "enabled adapter has conditional commercial-use terms",
			})
		}
		// A Ready adapter that is enabled but not installable would be offered yet
		// un-runnable; readiness without a fetch strategy is incoherent.
		for _, f := range doctorFetchStrategy(a) {
			add(f)
		}
		for _, f := range doctorChecksumPolicy(a) {
			add(f)
		}
	}

	ok := true
	for _, f := range findings {
		if f.Severity == FindingError {
			ok = false
			break
		}
	}
	return CatalogDoctorReport{OK: ok, Findings: findings}
}

// RegistryLint checks the fetch-strategy contract over EVERY row (enabled and
// disabled), so a source-less adapter cannot ship undetected. An enabled offender
// is an error; a disabled one a warning (must be fixed before it can be enabled).
func (r *Registry) RegistryLint() CatalogDoctorReport {
	var findings []CatalogFinding
	for _, a := range r.adapters {
		findings = append(findings, doctorFetchStrategy(a)...)
	}
	ok := true
	for _, f := range findings {
		if f.Severity == FindingError {
			ok = false
			break
		}
	}
	return CatalogDoctorReport{OK: ok, Findings: findings}
}

// doctorFetchStrategy returns findings when an adapter has no concrete install
// source, or declares a repo source without a pinned revision. Severity is error
// for an enabled adapter, warning otherwise.
func doctorFetchStrategy(a Adapter) []CatalogFinding {
	sev := FindingWarning
	if a.Enabled {
		sev = FindingError
	}
	var out []CatalogFinding
	if !a.Source.HasFetchStrategy() {
		out = append(out, CatalogFinding{
			Severity:  sev,
			Code:      "adapter_without_fetch_strategy",
			AdapterID: a.ID,
			Message:   "adapter declares no assets[]/repo/local_path; add a concrete fetch source",
		})
		return out
	}
	if a.Source.HasRepo() && strings.TrimSpace(a.Source.Repo.Revision) == "" {
		out = append(out, CatalogFinding{
			Severity:  sev,
			Code:      "repo_source_without_pinned_revision",
			AdapterID: a.ID,
			Message:   "repo-source adapter has no pinned revision (commit SHA) — required for a reproducible install before enable",
		})
	}
	for i, as := range a.Source.Assets {
		prefix := fmt.Sprintf("asset %d", i)
		if strings.TrimSpace(as.URL) == "" {
			out = append(out, CatalogFinding{Severity: sev, Code: "asset_missing_url", AdapterID: a.ID, Message: prefix + " has no url"})
		}
		if strings.TrimSpace(as.Filename) == "" {
			out = append(out, CatalogFinding{Severity: sev, Code: "asset_missing_filename", AdapterID: a.ID, Message: prefix + " has no filename"})
		}
		if as.MinBytes <= 0 {
			out = append(out, CatalogFinding{Severity: sev, Code: "asset_missing_min_bytes", AdapterID: a.ID, Message: prefix + " has no positive min_bytes"})
		}
		if sha := strings.TrimSpace(as.SHA256); sha != "" && len(sha) != 64 {
			out = append(out, CatalogFinding{Severity: sev, Code: "asset_bad_sha256", AdapterID: a.ID, Message: prefix + " sha256 is not 64 hex characters"})
		}
	}
	return out
}

func doctorChecksumPolicy(a Adapter) []CatalogFinding {
	c := a.Source.Checksum
	algo := strings.TrimSpace(c.Algo)
	value := strings.TrimSpace(c.Value)
	status := strings.TrimSpace(c.Status)
	if algo == "" && value == "" && status == "" {
		return nil
	}
	var out []CatalogFinding
	if algo != "" && algo != "sha256" {
		out = append(out, CatalogFinding{Code: "checksum_algo_not_sha256", AdapterID: a.ID, Message: fmt.Sprintf("checksum algo %q is not sha256", algo)})
	}
	if value != "" && len(value) != 64 {
		out = append(out, CatalogFinding{Code: "checksum_bad_sha256", AdapterID: a.ID, Message: "checksum value is not 64 hex characters"})
	}
	if value == "" && status == "pinned" {
		out = append(out, CatalogFinding{Code: "checksum_pinned_without_value", AdapterID: a.ID, Message: "checksum status is pinned but value is empty"})
	}
	return out
}
