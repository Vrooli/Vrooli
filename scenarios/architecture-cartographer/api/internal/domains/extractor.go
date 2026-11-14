package domains

import "context"

// DomainSourceExtractor reads one ladder rung of a scenario and reports
// the domains that rung declares.
//
// seam: DomainSourceExtractor is the substitution boundary for each ladder
// rung. Production wires DomainsDocExtractor + FolderExtractor +
// CLIGroupExtractor (and, later, an APIManifestExtractor); tests pass
// fakeExtractor. Registered in docs/internal/SEAMS.md.
type DomainSourceExtractor interface {
	// Source identifies which rung this extractor reads.
	Source() Source

	// Extract reads the scenario rooted at scenarioDir and returns the
	// domains the rung declares. A rung whose source is absent returns an
	// empty Extraction with a nil error — absence is not failure. A
	// malformed-but-present source returns an error so the ladder can
	// surface it.
	Extract(ctx context.Context, scenarioDir string) (Extraction, error)
}

var (
	_ DomainSourceExtractor = (*DomainsDocExtractor)(nil)
	_ DomainSourceExtractor = (*FolderExtractor)(nil)
	_ DomainSourceExtractor = (*CLIGroupExtractor)(nil)
)
