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
	"time"
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
	// Archetype is the optional primary archetype (e.g., "service",
	// "reporting"); drives archetype-aware heuristic exemptions.
	Archetype string
	// Provenance lists every source that declared a domain with this name.
	Provenance []Source
}

// DomainDeclaration is one ladder rung's raw view of the domain set,
// retained so convergence reporting can show where surfaces disagree.
type DomainDeclaration struct {
	Source        Source
	DomainNames   []string
	Authoritative bool
}

// DerivedDomainMap is the canonical resolved domain map for a scenario.
type DerivedDomainMap struct {
	Scenario        string
	Domains         []DerivedDomain
	SharedSubstrate []string
	NonDomains      []string
	Authority       Source
	Declarations    []DomainDeclaration
	DerivedAt       time.Time
}

// DomainFor returns the name of the domain whose paths cover the given
// repo-relative path, or "" when no domain owns it. The first matching
// domain in Domains order wins; callers keep Domains sorted by name for
// determinism.
func (m DerivedDomainMap) DomainFor(path string) string {
	for _, d := range m.Domains {
		for _, p := range d.Paths {
			if PathMatches(path, p) {
				return d.Name
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
	for _, p := range m.SharedSubstrate {
		if PathMatches(path, p) {
			return true
		}
	}
	return false
}

// ExtractedDomain is one source's raw declaration of a single domain,
// before ladder resolution. Folder/CLI extractors leave Glossary and
// Archetype empty; only the DomainsDoc extractor populates them.
type ExtractedDomain struct {
	Name      string
	Paths     []string
	Glossary  []string
	Archetype string
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
