package focus

import (
	"context"

	"github.com/vrooli/api-core/spacedoc"
)

// SpaceReader reads a projection's denominator (the curated intended space) from
// its owner. It is the read side of the cross-scenario contract; the numerator
// is NEVER read here. Declared at the consumer (seam-discovery): production wires
// the coverage domain's exec+file-fallback reader (identical method set), tests
// fake it. Kept narrow so a fake is a few lines.
type SpaceReader interface {
	Read(ctx context.Context, p Projection) (*spacedoc.SpaceDefinition, error)
}

// derivedProjections is the canonical iteration order for gap derivation.
var derivedProjections = []Projection{ProjectionAnswer, ProjectionValidate, ProjectionGuide}

// GapSource yields the gaps derived live from the space docs (every non-NOW
// denominator cell). These are the not-yet-done cells; the focus service overlays
// the owned registry (notes/approaches) on top of them and merges in
// registry-only global gaps.
type GapSource interface {
	DerivedGaps(ctx context.Context) ([]Gap, error)
}

type spaceGapSource struct {
	reader SpaceReader
}

// NewSpaceGapSource constructs the production GapSource over a SpaceReader.
func NewSpaceGapSource(r SpaceReader) GapSource { return &spaceGapSource{reader: r} }

var _ GapSource = (*spaceGapSource)(nil)

// DerivedGaps reads each projection's denominator and emits one Gap per non-NOW
// cell. An unreachable projection is skipped (degrade gracefully) rather than
// failing the whole derivation — the registry gaps still surface.
func (s *spaceGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	var out []Gap
	for _, p := range derivedProjections {
		def, err := s.reader.Read(ctx, p)
		if err != nil {
			continue // degrade: a down owner drops only its derived gaps
		}
		for _, c := range def.Cells {
			if c.Status == spacedoc.StatusNow {
				continue // a NOW cell is not a gap
			}
			out = append(out, Gap{
				ID:           string(p) + "/" + c.ID,
				Projection:   p,
				Title:        c.Question,
				Status:       c.Status,
				SourceCellID: c.ID,
				Notes:        append([]string(nil), c.Notes...),
			})
		}
	}
	return out, nil
}
