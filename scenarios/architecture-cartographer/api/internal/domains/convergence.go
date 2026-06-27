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
	// FindingAuthorityFallback: the trust ladder fell back to a derived
	// source (api folders, cli groups) because DOMAINS.md was missing or
	// empty. Convergence findings against this authority are tautological;
	// the real signal is that the scenario has no curated DOMAINS.md.
	FindingAuthorityFallback = "authority_fallback"
	// FindingArchetypeDrift: declared archetypes disagree with
	// high-confidence inferred archetypes.
	FindingArchetypeDrift = "archetype_drift"
)

// ConvergenceFinding is one cross-surface disagreement about the domain set.
//
// When the same info-severity Kind fires for several domains, Convergence
// rolls them up into a single finding: Domain stays empty and
// RolledUpDomains carries every affected name (sorted). Warn-severity
// findings are never rolled up — each is independently actionable.
type ConvergenceFinding struct {
	Kind     string
	Domain   string
	Severity ConvergenceSeverity
	Message  string
	// Sources are the rungs involved in the finding.
	Sources []Source
	// RolledUpDomains is non-empty when this finding represents two or
	// more identical-Kind info-severity findings merged into one row.
	RolledUpDomains []string
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
	input := convergenceInput(m)
	out := authorityFallbackFindings(m, input)
	out = append(out, missingSurfaceFindings(input)...)
	out = append(out, undeclaredFolderFindings(input)...)
	out = append(out, uiFeatureFindings(input)...)
	out = append(out, archetypeDriftFindings(m.Domains, input.authoritySource)...)

	out = rollupInfoFindings(out)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

type convergenceSets struct {
	authoritySource Source
	authoritySet    map[string]struct{}
	folders         map[string]struct{}
	cli             map[string]struct{}
	ui              map[string]struct{}
	nonDomain       map[string]struct{}
}

func convergenceInput(m DerivedDomainMap) convergenceSets {
	bySource := map[Source]map[string]struct{}{}
	var authoritySource Source
	authoritySet := map[string]struct{}{}
	for _, decl := range m.Declarations {
		set := stringSet(decl.DomainNames)
		bySource[decl.Source] = set
		if decl.Authoritative {
			authoritySource = decl.Source
			authoritySet = set
		}
	}
	return convergenceSets{
		authoritySource: authoritySource,
		authoritySet:    authoritySet,
		folders:         bySource[SourceAPIFolders],
		cli:             bySource[SourceCLIGroups],
		ui:              bySource[SourceUIFeatures],
		nonDomain:       stringSet(m.NonDomains),
	}
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func authorityFallbackFindings(m DerivedDomainMap, input convergenceSets) []ConvergenceFinding {
	if m.AuthorityConfidence != ConfidenceLow {
		return nil
	}
	return []ConvergenceFinding{{
		Kind:     FindingAuthorityFallback,
		Severity: ConvergenceInfo,
		Message:  "no curated DOMAINS.md (or api manifest); authority fell back to " + string(input.authoritySource) + " — domain map is inferred, not declared",
		Sources:  []Source{input.authoritySource},
	}}
}

func missingSurfaceFindings(input convergenceSets) []ConvergenceFinding {
	var out []ConvergenceFinding
	for name := range input.authoritySet {
		out = append(out, missingImplementationFinding(input, name)...)
		out = append(out, missingCLIGroupFinding(input, name)...)
	}
	return out
}

func missingImplementationFinding(input convergenceSets, name string) []ConvergenceFinding {
	if len(input.folders) == 0 {
		return nil
	}
	if _, ok := input.folders[name]; ok {
		return nil
	}
	return []ConvergenceFinding{{
		Kind:     FindingMissingImplementation,
		Domain:   name,
		Severity: ConvergenceWarn,
		Message:  "declared in the authority source but no api/internal/ folder implements it",
		Sources:  []Source{input.authoritySource, SourceAPIFolders},
	}}
}

func missingCLIGroupFinding(input convergenceSets, name string) []ConvergenceFinding {
	if len(input.cli) == 0 || isCLICoreBuiltinDomain(name) {
		return nil
	}
	if _, ok := input.cli[name]; ok {
		return nil
	}
	return []ConvergenceFinding{{
		Kind:     FindingMissingCLIGroup,
		Domain:   name,
		Severity: ConvergenceInfo,
		Message:  "declared domain has no cli/manifest.json command group",
		Sources:  []Source{input.authoritySource, SourceCLIGroups},
	}}
}

func undeclaredFolderFindings(input convergenceSets) []ConvergenceFinding {
	var out []ConvergenceFinding
	for name := range input.folders {
		if _, ok := input.authoritySet[name]; ok {
			continue
		}
		if _, ok := input.nonDomain[name]; ok {
			continue
		}
		out = append(out, ConvergenceFinding{
			Kind:     FindingUndeclaredFolder,
			Domain:   name,
			Severity: ConvergenceWarn,
			Message:  "an api/internal/ folder exists for a domain the authority source never declared",
			Sources:  []Source{SourceAPIFolders, input.authoritySource},
		})
	}
	return out
}

func uiFeatureFindings(input convergenceSets) []ConvergenceFinding {
	var out []ConvergenceFinding
	for name := range input.ui {
		if _, ok := input.nonDomain[name]; ok {
			continue
		}
		if _, ok := input.authoritySet[name]; ok {
			continue
		}
		out = append(out, ConvergenceFinding{
			Kind:     FindingUIFeatureNoDomain,
			Domain:   name,
			Severity: ConvergenceInfo,
			Message:  "a ui/src/features/ folder maps to no declared domain (advisory coverage)",
			Sources:  []Source{SourceUIFeatures, input.authoritySource},
		})
	}
	return out
}

func archetypeDriftFindings(domains []DerivedDomain, authoritySource Source) []ConvergenceFinding {
	var out []ConvergenceFinding
	for _, domain := range domains {
		if !archetypeDrifts(domain) {
			continue
		}
		out = append(out, ConvergenceFinding{
			Kind:     FindingArchetypeDrift,
			Domain:   domain.Name,
			Severity: ConvergenceWarn,
			Message:  "declared archetype set does not include any high-confidence inferred archetype",
			Sources:  []Source{authoritySource},
		})
	}
	return out
}

func archetypeDrifts(domain DerivedDomain) bool {
	declared := map[string]struct{}{}
	var inferred []string
	for _, archetype := range domain.Archetypes {
		switch archetype.Source {
		case ArchetypeSourceDeclared:
			if archetype.Name != "" {
				declared[archetype.Name] = struct{}{}
			}
		case ArchetypeSourceInferred:
			if archetype.Confidence >= 0.8 && archetype.Name != "" {
				inferred = append(inferred, archetype.Name)
			}
		}
	}
	if len(declared) == 0 || len(inferred) == 0 {
		return false
	}
	for _, name := range inferred {
		if _, ok := declared[name]; ok {
			return false
		}
	}
	return true
}

// cliCoreBuiltinDomains is the set of domain names that cli-core ships
// command groups for out-of-the-box. A scenario that declares such a
// domain but has no per-scenario manifest entry intentionally inherits
// the cli-core implementation — that's not drift. Keep this list narrow;
// adding a name silences a real warn-equivalent signal.
var cliCoreBuiltinDomains = map[string]struct{}{
	"health":  {},
	"status":  {},
	"version": {},
	"help":    {},
}

func isCLICoreBuiltinDomain(name string) bool {
	_, ok := cliCoreBuiltinDomains[name]
	return ok
}

// rollupInfoFindings merges info-severity findings of the same Kind into
// one row with RolledUpDomains listing every affected name. Findings
// without a Domain (the authority_fallback signal) are left untouched
// because there is no domain to roll up. Warn-severity findings are
// never merged.
func rollupInfoFindings(in []ConvergenceFinding) []ConvergenceFinding {
	byKind := map[string][]int{}
	for i, f := range in {
		if f.Severity != ConvergenceInfo || f.Domain == "" {
			continue
		}
		byKind[f.Kind] = append(byKind[f.Kind], i)
	}

	drop := map[int]bool{}
	var merged []ConvergenceFinding
	for kind, idxs := range byKind {
		if len(idxs) < 2 {
			continue
		}
		domains := make([]string, 0, len(idxs))
		for _, i := range idxs {
			domains = append(domains, in[i].Domain)
			drop[i] = true
		}
		sort.Strings(domains)
		// Use the first source list (they're identical across same-kind
		// findings emitted by the same surface comparison) and a message
		// derived from the original — the rendering layer cites the
		// rolled-up domain list explicitly.
		first := in[idxs[0]]
		merged = append(merged, ConvergenceFinding{
			Kind:            kind,
			Severity:        ConvergenceInfo,
			Message:         first.Message,
			Sources:         append([]Source(nil), first.Sources...),
			RolledUpDomains: domains,
		})
	}

	if len(drop) == 0 {
		return in
	}
	out := make([]ConvergenceFinding, 0, len(in)-len(drop)+len(merged))
	for i, f := range in {
		if drop[i] {
			continue
		}
		out = append(out, f)
	}
	out = append(out, merged...)
	return out
}
