// Package domains derives a scenario's intended domain map from sources
// that already exist on disk, replacing the deleted per-scenario
// architecture manifest. Derivation walks a trust ladder — the highest
// available rung defines the expected domain set; lower rungs feed
// provenance and (in the conflicts domain) cross-surface convergence.
//
// The package is pure and stateless: it reads a scenario directory and
// returns a DerivedDomainMap. It owns no SQLite table.
package domains

import (
	"fmt"
	"strings"
	"time"

	"architecture-cartographer/internal/archetype"
)

// Source identifies which ladder rung contributed a domain declaration.
// The order of the constants does NOT imply trust order; trust order is a
// control-surface lever resolved by the ladder (see config).
type Source string

const (
	// SourceUnspecified is the zero value.
	SourceUnspecified Source = ""
	// SourceAPIManifest is the future API manifest (top rung). Reserved for
	// an APIManifestExtractor that ships with api-health; no extractor emits
	// this yet.
	SourceAPIManifest Source = "api_manifest"
	// SourceDomainsDoc is the structured Domain Inventory in
	// docs/concepts/DOMAINS.md.
	SourceDomainsDoc Source = "domains_doc"
	// SourceAPIFolders is the de-facto domains implied by
	// api/internal/<domain>/ package folders.
	SourceAPIFolders Source = "api_folders"
	// SourceCLIGroups is the command groups declared in cli/manifest.json.
	SourceCLIGroups Source = "cli_groups"
	// SourceUIFeatures is the UI feature folders — advisory coverage only,
	// never authoritative.
	SourceUIFeatures Source = "ui_features"
)

// Advisory reports whether a source contributes coverage signal only and
// must never be selected as the authority rung. UI features are advisory:
// a scenario's UI must not define the canonical domain set.
func (s Source) Advisory() bool {
	return s == SourceUIFeatures
}

// DerivedDomain is one domain in the resolved map.
type DerivedDomain struct {
	// Name is the domain name (e.g., "graph").
	Name string
	// Paths are the repo-relative path globs/prefixes the domain owns,
	// taken from the authority rung.
	Paths []string
	// Glossary is the optional canonical vocabulary (type/function names)
	// for the symbol-glossary signal.
	Glossary []string
	// Responsibility is the human-authored semantic anchor for the domain.
	Responsibility string
	// Purpose captures why the capability exists for users/operators.
	Purpose string
	// OwnsData captures the authored data ownership note.
	OwnsData string
	// SecondaryTraits are additional declared archetype traits.
	SecondaryTraits []string
	// Surfaces are derived from code evidence. The slice walker populates
	// this in a later phase; the field lives here now so the contract is
	// ready before evidence is wired.
	Surfaces []string
	// Archetypes is the declared/inferred archetype set.
	Archetypes []DomainArchetype
	// Provenance lists every source that declared a domain with this name.
	Provenance []Source
}

type ArchetypeSource string

const (
	ArchetypeSourceUnspecified ArchetypeSource = ""
	ArchetypeSourceDeclared    ArchetypeSource = "declared"
	ArchetypeSourceInferred    ArchetypeSource = "inferred"
)

type DomainArchetype struct {
	// Name is the canonical archetype (archetype.Name vocabulary), empty when a
	// declared label does not map onto the canonical set.
	Name       string
	Source     ArchetypeSource
	Confidence float64
	Evidence   []string
	// DeclaredLabel preserves the original DOMAINS.md text when Source is
	// Declared and the label is not canonical (Name is then empty). Empty
	// otherwise. Drives honest drift reporting instead of silent coercion.
	DeclaredLabel string
}

// PrimaryArchetype returns the effective declared archetype role, falling back
// to the first archetype in the set. The returned value is the canonical name
// when the declared label maps onto the fixed vocabulary, otherwise the raw
// declared label — because the zone/layering "coordinating roles" vocabulary
// (provider, aggregation, infrastructure, composition-root) is a deliberate
// superset of the canonical Q20 archetypes, and those detectors must still see
// a non-canonical declared role.
func (a DomainArchetype) effectiveRole() string {
	if a.Name != "" {
		return a.Name
	}
	return a.DeclaredLabel
}

func (d DerivedDomain) PrimaryArchetype() string {
	for _, archetype := range d.Archetypes {
		if archetype.Source == ArchetypeSourceDeclared {
			if role := archetype.effectiveRole(); role != "" {
				return role
			}
		}
	}
	if len(d.Archetypes) > 0 {
		return d.Archetypes[0].effectiveRole()
	}
	return ""
}

func DeclaredArchetypes(names ...string) []DomainArchetype {
	out := make([]DomainArchetype, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || name == "-" || name == "—" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		da := DomainArchetype{
			Source:     ArchetypeSourceDeclared,
			Confidence: 1,
			Evidence:   []string{DomainsDocPath},
		}
		// Normalize onto the canonical vocabulary; preserve the raw label when it
		// does not map so drift reporting can compare authored vs canonical.
		if canon, ok := archetype.Normalize(name); ok {
			da.Name = string(canon)
		} else {
			da.DeclaredLabel = name
		}
		out = append(out, da)
	}
	return out
}

// DomainDeclaration is one ladder rung's raw view of the domain set,
// retained so convergence reporting can show where surfaces disagree.
type DomainDeclaration struct {
	Source        Source
	DomainNames   []string
	Authoritative bool
}

// AuthorityConfidence reports how trustworthy the resolved authority is.
//
//   - ConfidenceHigh: authority came from a deliberately-curated source
//     (the structured DOMAINS.md). Convergence findings against it are
//     real drift signals.
//   - ConfidenceLow: authority fell back to a derived source (api
//     folders, cli groups). The "ground truth" is itself inferred, so
//     convergence against it is tautological — a missing DOMAINS.md is
//     the bigger story.
type AuthorityConfidence string

const (
	ConfidenceUnspecified AuthorityConfidence = ""
	ConfidenceHigh        AuthorityConfidence = "high"
	ConfidenceLow         AuthorityConfidence = "low"
	// ConfidenceMissing signals that NO ladder rung declared any domain
	// for the scenario (ErrNoAuthority was raised and caught). The audit
	// gate treats this the same as ConfidenceLow but renders a different
	// remediation message ("write DOMAINS.md" rather than "promote to
	// DOMAINS.md").
	ConfidenceMissing AuthorityConfidence = "missing"
)

// DerivedDomainMap is the canonical resolved domain map for a scenario.
type DerivedDomainMap struct {
	Scenario            string
	Domains             []DerivedDomain
	SharedSubstrate     []string
	NonDomains          []string
	Authority           Source
	AuthorityConfidence AuthorityConfidence
	Declarations        []DomainDeclaration
	DerivedAt           time.Time
	// Warnings forwards non-fatal extraction issues (e.g. silently
	// skipped DOMAINS.md rows) so the conflicts pipeline can surface
	// them. Aggregated across every rung in source order.
	Warnings []ExtractionWarning
}

// DomainDraft is an operator-facing proposal for a curated DOMAINS.md.
// It is evidence-only and never writes authority to disk.
type DomainDraft struct {
	Scenario string
	Domains  []ProposedDomain
	Markdown string
}

// ProposedDomain is one row in a DomainDraft.
type ProposedDomain struct {
	Name       string
	Paths      []string
	Archetypes []DomainArchetype
	Glossary   []string
	Confidence string
	Evidence   []string
}

// DomainFor returns the name of the domain whose paths cover the given
// repo-relative path, or "" when no domain owns it. The first matching
// domain in Domains order wins; callers keep Domains sorted by name for
// determinism.
func (m DerivedDomainMap) DomainFor(path string) string {
	candidates := m.pathCandidates(path)
	for _, d := range m.Domains {
		for _, p := range d.Paths {
			for _, candidate := range candidates {
				if PathMatches(candidate, p) {
					return d.Name
				}
			}
		}
	}
	return ""
}

// Names returns the resolved domain names in order.
func (m DerivedDomainMap) Names() []string {
	out := make([]string, 0, len(m.Domains))
	for _, d := range m.Domains {
		out = append(out, d.Name)
	}
	return out
}

// IsSharedSubstrate reports whether the path falls under a declared
// shared-substrate prefix.
func (m DerivedDomainMap) IsSharedSubstrate(path string) bool {
	candidates := m.pathCandidates(path)
	for _, p := range m.SharedSubstrate {
		for _, candidate := range candidates {
			if PathMatches(candidate, p) {
				return true
			}
		}
	}
	return false
}

func (m DerivedDomainMap) pathCandidates(path string) []string {
	path = NormalizePath(path)
	if path == "" {
		return nil
	}
	candidates := []string{path}
	if m.Scenario != "" && !isRepoRootRelative(path) {
		rebased := rebaseToRepoRoot(m.Scenario, path)
		if rebased != path {
			candidates = append(candidates, rebased)
		}
	}
	return candidates
}

// ExtractedDomain is one source's raw declaration of a single domain,
// before ladder resolution. Folder/CLI extractors leave Glossary and
// Archetype empty; only the DomainsDoc extractor populates them.
type ExtractedDomain struct {
	Name            string
	Paths           []string
	Glossary        []string
	Responsibility  string
	Purpose         string
	OwnsData        string
	SecondaryTraits []string
	Archetypes      []DomainArchetype
}

// Extraction is the full output of one extractor for one scenario.
type Extraction struct {
	Source Source
	// Domains are the domains this rung declared (may be empty when the
	// source is absent — that is not an error).
	Domains []ExtractedDomain
	// SharedSubstrate / NonDomains are only populated by sources that can
	// express them (currently the DomainsDoc extractor).
	SharedSubstrate []string
	NonDomains      []string
	// Warnings are non-fatal extraction issues — rows that were skipped
	// for structural reasons (wrong column count, missing required cell)
	// the operator should know about. The ladder aggregates these; the
	// conflicts pipeline surfaces them as `domains_doc_parse_warning`
	// findings so silent drops can no longer hide.
	Warnings []ExtractionWarning
}

// ExtractionWarning describes one skipped/abnormal row from a structured
// source (today: DOMAINS.md). Kept narrow on purpose — the conflicts
// pipeline owns the human-facing rendering.
type ExtractionWarning struct {
	Kind    string // e.g., "domains_doc.row_shape"
	Path    string // source-relative path, e.g., "docs/concepts/DOMAINS.md"
	Line    int    // 1-based line number when known, 0 when not applicable
	Summary string // one-line operator-facing description
}

// ErrScenarioNotFound is the typed sentinel returned when a scenario's
// directory cannot be located.
type ErrScenarioNotFound struct {
	Scenario string
}

func (e ErrScenarioNotFound) Error() string {
	return fmt.Sprintf("scenario %q not found", e.Scenario)
}

// ErrNoAuthority is the typed sentinel returned when no ladder rung
// declared any domain for the scenario (nothing to derive).
type ErrNoAuthority struct {
	Scenario string
}

func (e ErrNoAuthority) Error() string {
	return fmt.Sprintf("no domain declarations found for scenario %q on any ladder rung", e.Scenario)
}
