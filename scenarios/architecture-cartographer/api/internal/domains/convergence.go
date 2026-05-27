package domains

import "sort"

// ConvergenceSeverity grades how much a convergence finding matters.
// Findings are advisory signal, never hard gates.
type ConvergenceSeverity string

const (
	// ConvergenceInfo is coverage signal (e.g., an advisory UI gap).
	ConvergenceInfo ConvergenceSeverity = "info"
	// ConvergenceWarn is a real cross-surface disagreement worth fixing
	// (a declared domain with no implementation, or vice versa).
	ConvergenceWarn ConvergenceSeverity = "warn"
)

// Convergence finding kinds.
const (
	// FindingMissingImplementation: a domain is declared by the authority
	// (DOMAINS.md) but has no api/internal folder.
	FindingMissingImplementation = "missing_implementation"
	// FindingUndeclaredFolder: an api/internal folder exists for a domain
	// the authority never declared.
	FindingUndeclaredFolder = "undeclared_folder"
	// FindingMissingCLIGroup: an authoritative domain has no cli/manifest.json
	// command group.
	FindingMissingCLIGroup = "missing_cli_group"
	// FindingUIFeatureNoDomain: a ui/src/features/ folder maps to no
	// declared domain (advisory coverage).
	FindingUIFeatureNoDomain = "ui_feature_no_domain"
)

// ConvergenceFinding is one cross-surface disagreement about the domain set.
type ConvergenceFinding struct {
	Kind     string
	Domain   string
	Severity ConvergenceSeverity
	Message  string
	// Sources are the rungs involved in the finding.
	Sources []Source
}

// Convergence compares the authority rung against the lower rungs (and the
// advisory UI rung) and reports where surfaces disagree on the domain set.
// It is pure: every input is already in the DerivedDomainMap's per-source
// Declarations. Findings are sorted (kind, domain) for determinism.
//
// To avoid false positives on scenarios that simply do not expose a given
// surface, a "missing on surface X" finding is only emitted when surface X
// declared at least one domain (i.e., the surface exists at all).
func Convergence(m DerivedDomainMap) []ConvergenceFinding {
	bySource := map[Source]map[string]struct{}{}
	var authoritySource Source
	authoritySet := map[string]struct{}{}
	for _, decl := range m.Declarations {
		set := make(map[string]struct{}, len(decl.DomainNames))
		for _, n := range decl.DomainNames {
			set[n] = struct{}{}
		}
		bySource[decl.Source] = set
		if decl.Authoritative {
			authoritySource = decl.Source
			authoritySet = set
		}
	}

	folders := bySource[SourceAPIFolders]
	cli := bySource[SourceCLIGroups]
	ui := bySource[SourceUIFeatures]

	// Folders/features the authority explicitly declared as non-domains
	// (infrastructure) are not drift — exclude them from the folder/UI
	// checks below.
	nonDomain := make(map[string]struct{}, len(m.NonDomains))
	for _, n := range m.NonDomains {
		nonDomain[n] = struct{}{}
	}

	var out []ConvergenceFinding

	// Authority domain present, but missing on an implementing surface.
	for name := range authoritySet {
		if len(folders) > 0 {
			if _, ok := folders[name]; !ok {
				out = append(out, ConvergenceFinding{
					Kind:     FindingMissingImplementation,
					Domain:   name,
					Severity: ConvergenceWarn,
					Message:  "declared in the authority source but no api/internal/ folder implements it",
					Sources:  []Source{authoritySource, SourceAPIFolders},
				})
			}
		}
		if len(cli) > 0 {
			if _, ok := cli[name]; !ok {
				out = append(out, ConvergenceFinding{
					Kind:     FindingMissingCLIGroup,
					Domain:   name,
					Severity: ConvergenceInfo,
					Message:  "declared domain has no cli/manifest.json command group",
					Sources:  []Source{authoritySource, SourceCLIGroups},
				})
			}
		}
	}

	// Folder present, but the authority never declared it.
	for name := range folders {
		if _, ok := authoritySet[name]; ok {
			continue
		}
		if _, ok := nonDomain[name]; ok {
			continue // explicitly declared infrastructure
		}
		out = append(out, ConvergenceFinding{
			Kind:     FindingUndeclaredFolder,
			Domain:   name,
			Severity: ConvergenceWarn,
			Message:  "an api/internal/ folder exists for a domain the authority source never declared",
			Sources:  []Source{SourceAPIFolders, authoritySource},
		})
	}

	// UI feature maps to no declared domain (advisory).
	for name := range ui {
		if _, ok := nonDomain[name]; ok {
			continue
		}
		if _, ok := authoritySet[name]; !ok {
			out = append(out, ConvergenceFinding{
				Kind:     FindingUIFeatureNoDomain,
				Domain:   name,
				Severity: ConvergenceInfo,
				Message:  "a ui/src/features/ folder maps to no declared domain (advisory coverage)",
				Sources:  []Source{SourceUIFeatures, authoritySource},
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}
