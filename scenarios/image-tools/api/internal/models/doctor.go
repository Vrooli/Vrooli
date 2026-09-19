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

		for _, f := range doctorDiffusersEditRunnable(m) {
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

// RegistryLint checks the fetch-strategy contract over EVERY row (enabled and
// disabled), so a landing-page-only stub cannot ship undetected — it is caught in
// `go test` / `models registry-lint`, not at a user's failed install. The rule:
// a weight-backed model must declare a concrete fetch strategy (assets[]/repo/
// local_path) OR be honestly marked source.manual. download_url is never a fetch
// strategy (it is documentation-only). An enabled offender is an error; a disabled
// one a warning (it must be fixed before it can be enabled).
func (r *Registry) RegistryLint() CatalogDoctorReport {
	var findings []CatalogFinding
	for _, m := range r.models {
		if f := fetchStrategyFinding(m); f != nil {
			findings = append(findings, *f)
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

// fetchStrategyFinding returns a finding when a weight-backed model has no
// concrete, auto-resolvable install source and is not honestly marked manual.
func fetchStrategyFinding(m Model) *CatalogFinding {
	if !m.RequiresWeights() || m.HasFetchStrategy() || m.Source.Manual {
		return nil
	}
	sev := FindingWarning
	if m.Enabled {
		sev = FindingError
	}
	return &CatalogFinding{
		Severity: sev,
		Code:     "model_without_fetch_strategy",
		ModelID:  m.ID,
		Message:  "weight-backed model declares no assets[]/repo/local_path and is not marked source.manual; its download_url is documentation-only and is never fetched — add a concrete fetch source or set source.manual",
	}
}

func doctorModelInstallable(m Model, add func(CatalogFinding)) bool {
	if !m.Enabled {
		return false
	}
	if !m.RequiresWeights() || strings.TrimSpace(m.Source.LocalPath) != "" {
		return true
	}
	// A repo snapshot is a valid fetch strategy, but the revision MUST be pinned
	// to an immutable commit SHA for a reproducible install + tree-manifest hash.
	if m.Source.HasRepo() {
		if strings.TrimSpace(m.Source.Repo.Revision) == "" {
			add(CatalogFinding{
				Code:    "repo_source_without_pinned_revision",
				ModelID: m.ID,
				Message: "enabled repo-source model has no pinned revision (commit SHA)",
			})
			return false
		}
		return true
	}
	if len(m.Source.Assets) == 0 {
		add(CatalogFinding{
			Code:    "enabled_model_without_assets",
			ModelID: m.ID,
			Message: "enabled weight-backed model has no source.assets[], repo, or local_path",
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

// doctorDiffusersEditRunnable enforces the "enabled ⇒ runnable" invariant for
// diffusers instruction-edit models (the ones that execute through the generic
// _diffusers runner). An enabled such model must name a registered family adapter
// that is proven (Ready), and declare its minimum runtime — so it is impossible
// to enable a model that the runner would refuse or that has no pinned runtime.
// Inpaint/outpaint diffusers models use a different sidecar and carry no family,
// so they are out of scope here.
func doctorDiffusersEditRunnable(m Model) []CatalogFinding {
	if !m.Enabled || m.Backend != BackendDiffusers || !m.ServesOperation("edit_instruct") {
		return nil
	}
	fam := strings.TrimSpace(m.Runtime.Family)
	if fam == "" {
		return []CatalogFinding{{
			Code:    "enabled_edit_model_without_family",
			ModelID: m.ID,
			Message: "enabled diffusers edit_instruct model declares no runtime.family adapter",
		}}
	}
	// validateModel guarantees a declared family is registered, so the lookup
	// always resolves here; the readiness + min_runtime checks are what this
	// invariant adds on top.
	reg, _ := DiffusersFamilyByName(fam)
	var out []CatalogFinding
	if !reg.Ready {
		out = append(out, CatalogFinding{
			Code:    "enabled_edit_model_family_not_ready",
			ModelID: m.ID,
			Message: fmt.Sprintf("runtime.family %q adapter is not yet proven runnable: %s", fam, reg.Pending),
		})
	}
	if strings.TrimSpace(m.Runtime.MinRuntime) == "" {
		out = append(out, CatalogFinding{
			Severity: FindingWarning,
			Code:     "enabled_edit_model_without_min_runtime",
			ModelID:  m.ID,
			Message:  "enabled diffusers edit_instruct model declares no runtime.min_runtime",
		})
	}
	return out
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
