package domains

import (
	"context"
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
	folder := NewFolderExtractor()
	if len(extraNonDomainFolders) > 0 {
		folder = NewFolderExtractorWithExemptions(extraNonDomainFolders)
	}
	bySource := map[Source]DomainSourceExtractor{
		SourceDomainsDoc: NewDomainsDocExtractor(),
		SourceAPIFolders: folder,
		SourceCLIGroups:  NewCLIGroupExtractor(),
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
		out = append(out, NewDomainsDocExtractor(), folder, NewCLIGroupExtractor())
	}
	// UI features are always present (advisory only).
	out = append(out, NewUIFeatureExtractor())
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

func (s *service) now() time.Time {
	if s.clock == nil {
		return time.Time{}
	}
	return s.clock.Now()
}
