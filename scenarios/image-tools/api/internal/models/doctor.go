package models

import (
	"fmt"
	"strings"
)

// FindingSeverity classifies a catalog doctor finding.
type FindingSeverity string

const (
	FindingError   FindingSeverity = "error"
	FindingWarning FindingSeverity = "warning"
)

// CatalogFinding is one actionable model-catalog integrity finding.
type CatalogFinding struct {
	Severity  FindingSeverity
	Code      string
	ModelID   string
	Operation string
	Message   string
}

// CatalogDoctorReport is the complete catalog integrity report.
type CatalogDoctorReport struct {
	OK       bool
	Findings []CatalogFinding
}

// DoctorCatalog checks installability and policy invariants that must become
// true before the seed can be treated as production-ready.
func (r *Registry) DoctorCatalog() CatalogDoctorReport {
	var findings []CatalogFinding
	add := func(f CatalogFinding) {
		if f.Severity == "" {
			f.Severity = FindingError
		}
		findings = append(findings, f)
	}

	installableEnabledByOp := make(map[string]int, len(r.vocab))
	for _, m := range r.models {
		if b, blocked := r.blockByID[m.ID]; blocked {
			add(CatalogFinding{
				Code:    "seed_blocklist_overlap",
				ModelID: m.ID,
				Message: fmt.Sprintf("seed model is also blocklisted: %s", b.Reason),
			})
		}

		if m.CapabilityLabels.CommercialUse == CommercialUseConditional && m.Enabled {
			add(CatalogFinding{
				Code:    "enabled_conditional_commercial_use",
				ModelID: m.ID,
				Message: "enabled model has conditional commercial-use terms",
			})
		}

		for _, f := range doctorChecksumPolicy(m) {
			add(f)
		}

		installable := doctorModelInstallable(m, add)
		if m.Enabled && installable {
			for _, op := range m.Operations {
				installableEnabledByOp[op]++
			}
		}
	}

	for _, op := range r.vocabOrder {
		if installableEnabledByOp[op] == 0 {
			add(CatalogFinding{
				Code:      "operation_without_installable_enabled_model",
				Operation: op,
				Message:   "operation has no enabled model with resolvable install assets",
			})
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

func doctorModelInstallable(m Model, add func(CatalogFinding)) bool {
	if !m.Enabled {
		return false
	}
	if !m.RequiresWeights() || strings.TrimSpace(m.Source.LocalPath) != "" {
		return true
	}
	if len(m.Source.Assets) == 0 {
		add(CatalogFinding{
			Code:    "enabled_model_without_assets",
			ModelID: m.ID,
			Message: "enabled weight-backed model has no source.assets[] and no local_path",
		})
		return false
	}

	installable := true
	for i, a := range m.Source.Assets {
		prefix := fmt.Sprintf("asset %d", i)
		if strings.TrimSpace(a.URL) == "" {
			add(CatalogFinding{Code: "asset_missing_url", ModelID: m.ID, Message: prefix + " has no url"})
			installable = false
		}
		if strings.TrimSpace(a.Filename) == "" {
			add(CatalogFinding{Code: "asset_missing_filename", ModelID: m.ID, Message: prefix + " has no filename"})
			installable = false
		}
		if a.Kind == ArtifactGeneric {
			add(CatalogFinding{Code: "asset_missing_kind", ModelID: m.ID, Message: prefix + " has no artifact kind"})
			installable = false
		}
		if a.MinBytes <= 0 {
			add(CatalogFinding{Code: "asset_missing_min_bytes", ModelID: m.ID, Message: prefix + " has no positive min_bytes"})
			installable = false
		}
		if sha := strings.TrimSpace(a.SHA256); sha != "" && len(sha) != 64 {
			add(CatalogFinding{Code: "asset_bad_sha256", ModelID: m.ID, Message: prefix + " sha256 is not 64 hex characters"})
			installable = false
		}
	}
	return installable
}

func doctorChecksumPolicy(m Model) []CatalogFinding {
	if !m.RequiresWeights() {
		return nil
	}
	c := m.Source.Checksum
	algo := strings.TrimSpace(c.Algo)
	value := strings.TrimSpace(c.Value)
	status := strings.TrimSpace(c.Status)
	if algo == "" && value == "" && status == "" {
		return nil
	}
	var out []CatalogFinding
	if algo != "sha256" {
		out = append(out, CatalogFinding{
			Code:    "checksum_algo_not_sha256",
			ModelID: m.ID,
			Message: fmt.Sprintf("checksum algo %q is not sha256", algo),
		})
	}
	if value != "" && len(value) != 64 {
		out = append(out, CatalogFinding{
			Code:    "checksum_bad_sha256",
			ModelID: m.ID,
			Message: "checksum value is not 64 hex characters",
		})
	}
	if value == "" && status == "pinned" {
		out = append(out, CatalogFinding{
			Code:    "checksum_pinned_without_value",
			ModelID: m.ID,
			Message: "checksum status is pinned but value is empty",
		})
	}
	return out
}
