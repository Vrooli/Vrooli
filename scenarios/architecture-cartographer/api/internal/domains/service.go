package domains

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Clock is the slim time seam so derivation timestamps are deterministic
// in tests. Mirrors internal/clock.Clock without importing it (this
// package stays dependency-light).
type Clock interface {
	Now() time.Time
}

// ScenarioLocator resolves a scenario name to its on-disk root directory.
//
// seam: ScenarioLocator decouples derivation from how scenario directories
// are found. Production wires a repo-root-relative locator
// (<repoRoot>/scenarios/<name>); tests pass a fake pointing at a fixture
// directory. Registered in docs/internal/SEAMS.md.
type ScenarioLocator interface {
	Locate(scenario string) (string, error)
}

// Service is the application-layer surface for domain derivation.
type Service interface {
	// ExtractDomains derives the domain map fresh from on-disk sources.
	ExtractDomains(ctx context.Context, scenario string) (DerivedDomainMap, error)
	// GetDomainMap returns the derived map. Derivation is deterministic and
	// cheap, so this currently re-derives; the separate method exists so a
	// future cache can back it without an interface change.
	GetDomainMap(ctx context.Context, scenario string) (DerivedDomainMap, error)
	// DraftDomains proposes a DOMAINS.md inventory from extracted evidence.
	// It never writes authority; callers render the draft for human review.
	DraftDomains(ctx context.Context, scenario string) (DomainDraft, error)
}

type service struct {
	locator    ScenarioLocator
	extractors []DomainSourceExtractor
	clock      Clock
}

// NewService constructs the production derivation service. The extractors
// are supplied in trust order (highest rung first); the first rung that
// declares any domain becomes the authority.
func NewService(locator ScenarioLocator, clock Clock, extractors ...DomainSourceExtractor) Service {
	return &service{locator: locator, extractors: extractors, clock: clock}
}

// DefaultExtractors returns the production ladder in trust order. The
// API-manifest rung is intentionally absent — it ships with api-health
// later and registers ahead of the DOMAINS.md rung at that point. The UI
// rung is last and advisory: it never wins authority, only feeds
// cross-surface convergence.
func DefaultExtractors() []DomainSourceExtractor {
	return ExtractorsFor(nil, nil)
}

// ExtractorsFor builds the ladder from a control-surface trust order (source
// names, highest first) and extra non-domain folder exemptions. An empty
// order falls back to the default (domains_doc → api_folders → cli_groups).
// The advisory UI rung is always appended last. Unknown/api_manifest names
// are skipped (no extractor emits api_manifest yet).
func ExtractorsFor(ladderOrder, extraNonDomainFolders []string) []DomainSourceExtractor {
	return ExtractorsForWithSurfaceProvider(ladderOrder, extraNonDomainFolders, nil)
}

func ExtractorsForWithSurfaceProvider(ladderOrder, extraNonDomainFolders []string, surfaces SurfaceProvider) []DomainSourceExtractor {
	folder := NewFolderExtractorWithSurfaceProvider(nil, surfaces)
	if len(extraNonDomainFolders) > 0 {
		folder = NewFolderExtractorWithSurfaceProvider(extraNonDomainFolders, surfaces)
	}
	bySource := map[Source]DomainSourceExtractor{
		SourceDomainsDoc: NewDomainsDocExtractor(),
		SourceAPIFolders: folder,
		SourceCLIGroups:  NewCLIGroupExtractorWithSurfaceProvider(surfaces),
	}
	order := ladderOrder
	if len(order) == 0 {
		order = []string{string(SourceDomainsDoc), string(SourceAPIFolders), string(SourceCLIGroups)}
	}
	out := make([]DomainSourceExtractor, 0, len(order)+1)
	for _, name := range order {
		if ex, ok := bySource[Source(name)]; ok {
			out = append(out, ex)
		}
	}
	if len(out) == 0 {
		// Defensive: never return an empty ladder.
		out = append(out, NewDomainsDocExtractor(), folder, NewCLIGroupExtractorWithSurfaceProvider(surfaces))
	}
	// UI features are always present (advisory only).
	out = append(out, NewUIFeatureExtractorWithSurfaceProvider(surfaces))
	return out
}

var _ Service = (*service)(nil)

func (s *service) ExtractDomains(ctx context.Context, scenario string) (DerivedDomainMap, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return DerivedDomainMap{}, ErrScenarioNotFound{Scenario: scenario}
	}
	dir, err := s.locator.Locate(scenario)
	if err != nil {
		return DerivedDomainMap{}, err
	}
	extractions, err := RunLadder(ctx, dir, s.extractors)
	if err != nil {
		return DerivedDomainMap{}, err
	}
	return Resolve(scenario, extractions, s.now())
}

func (s *service) GetDomainMap(ctx context.Context, scenario string) (DerivedDomainMap, error) {
	return s.ExtractDomains(ctx, scenario)
}

func (s *service) DraftDomains(ctx context.Context, scenario string) (DomainDraft, error) {
	m, err := s.ExtractDomains(ctx, scenario)
	if err != nil {
		return DomainDraft{}, err
	}
	return DraftFromMap(m), nil
}

// DraftFromMap converts derived ladder evidence into a markdown draft.
// The output is deliberately conservative: it preserves known glossary
// and archetype data when the authority is curated, otherwise it marks
// those cells TODO so humans ratify intent rather than inheriting guesses.
func DraftFromMap(m DerivedDomainMap) DomainDraft {
	out := DomainDraft{Scenario: m.Scenario}
	for _, d := range m.Domains {
		p := ProposedDomain{
			Name:       d.Name,
			Paths:      append([]string(nil), d.Paths...),
			Glossary:   append([]string(nil), d.Glossary...),
			Archetypes: cloneArchetypes(d.Archetypes),
			Confidence: draftConfidence(m, d),
			Evidence:   sourceEvidence(d.Provenance),
		}
		if len(p.Archetypes) == 0 {
			p.Archetypes = DeclaredArchetypes("TODO")
		}
		out.Domains = append(out.Domains, p)
	}
	out.Markdown = draftMarkdown(out)
	return out
}

func draftConfidence(m DerivedDomainMap, d DerivedDomain) string {
	if m.AuthorityConfidence == ConfidenceHigh {
		return "high"
	}
	if len(d.Provenance) > 1 {
		return "medium"
	}
	return "low"
}

func sourceEvidence(sources []Source) []string {
	if len(sources) == 0 {
		return nil
	}
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		if s == SourceUnspecified {
			continue
		}
		out = append(out, string(s))
	}
	sort.Strings(out)
	return out
}

func draftMarkdown(d DomainDraft) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Domains — %s\n\n", d.Scenario)
	b.WriteString("## Domain Inventory\n\n")
	b.WriteString("| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, domain := range d.Domains {
		fmt.Fprintf(&b, "| %s | TODO | TODO | TODO | %s | %s | %s | %s |\n",
			domain.Name,
			formatPrimaryArchetype(domain.Archetypes),
			formatSecondaryTraits(domain.Archetypes),
			formatDraftList(domain.Glossary),
			formatDraftList(domain.Paths))
	}
	return b.String()
}

func formatPrimaryArchetype(archetypes []DomainArchetype) string {
	if len(archetypes) == 0 {
		return "TODO"
	}
	for _, archetype := range archetypes {
		if archetype.Source == ArchetypeSourceDeclared && archetype.Name != "" {
			return archetype.Name
		}
	}
	return archetypes[0].Name
}

func formatSecondaryTraits(archetypes []DomainArchetype) string {
	if len(archetypes) <= 1 {
		return "—"
	}
	primary := formatPrimaryArchetype(archetypes)
	traits := make([]string, 0, len(archetypes)-1)
	for _, archetype := range archetypes {
		if archetype.Name == "" || archetype.Name == primary {
			continue
		}
		traits = append(traits, archetype.Name)
	}
	if len(traits) == 0 {
		return "—"
	}
	return strings.Join(traits, ", ")
}

func formatDraftList(values []string) string {
	if len(values) == 0 {
		return "TODO"
	}
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		quoted = append(quoted, "`"+v+"`")
	}
	if len(quoted) == 0 {
		return "TODO"
	}
	return strings.Join(quoted, ", ")
}

func (s *service) now() time.Time {
	if s.clock == nil {
		return time.Time{}
	}
	return s.clock.Now()
}
