package focus

import (
	"context"
	"errors"
	"fmt"

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
var derivedProjections = []Projection{ProjectionAnswer, ProjectionValidate, ProjectionGuide, ProjectionAct}

// GapSource yields the gaps derived live from the space docs (every non-NOW
// denominator cell). These are the not-yet-done cells; the focus service overlays
// the owned registry (notes/approaches) on top of them and merges in
// registry-only global gaps.
type GapSource interface {
	DerivedGaps(ctx context.Context) ([]Gap, error)
}

// NamedGapSource gives a source a stable human-facing identity for provenance
// and graceful degradation reporting.
type NamedGapSource struct {
	Name   string
	Source GapSource
}

type multiGapSource struct {
	sources []NamedGapSource
}

// NewMultiGapSource constructs a source that fans out to the named sources in
// order. A source failure becomes a visible availability gap and never removes
// gaps returned by another source.
func NewMultiGapSource(sources []NamedGapSource) GapSource {
	return &multiGapSource{sources: append([]NamedGapSource(nil), sources...)}
}

var _ GapSource = (*multiGapSource)(nil)

func (m *multiGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	var out []Gap
	for _, named := range m.sources {
		if named.Source == nil {
			out = append(out, sourceAvailabilityGap(named.Name, sourceAxis(nil), errors.New("source is not configured")))
			continue
		}
		gaps, err := named.Source.DerivedGaps(ctx)
		out = append(out, gaps...)
		if err != nil {
			out = append(out, sourceAvailabilityGap(named.Name, sourceAxis(named.Source), err))
		}
	}
	return out, nil
}

type axisProvider interface{ Axis() Axis }

func sourceAxis(source GapSource) Axis {
	if provider, ok := source.(axisProvider); ok && provider.Axis() != "" {
		return provider.Axis()
	}
	return AxisEmpirical
}

func sourceAvailabilityGap(name string, axis Axis, err error) Gap {
	return Gap{
		ID:                 "source/" + name + "/availability",
		Axis:               axis,
		Title:              fmt.Sprintf("%s evidence source unavailable", name),
		Global:             true,
		EvidenceSource:     name,
		AvailabilityReason: fmt.Sprintf("%s: %v", name, err),
	}
}

type spaceGapSource struct {
	reader SpaceReader
}

func (*spaceGapSource) Axis() Axis { return AxisCoverage }

// NewSpaceGapSource constructs the production GapSource over a SpaceReader.
func NewSpaceGapSource(r SpaceReader) GapSource { return &spaceGapSource{reader: r} }

var _ GapSource = (*spaceGapSource)(nil)

// DerivedGaps reads each projection's denominator and emits one Gap per non-NOW
// cell. An unreachable projection is skipped (degrade gracefully) rather than
// failing the whole derivation — the registry gaps still surface.
func (s *spaceGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	var out []Gap
	var errs []error
	for _, p := range derivedProjections {
		def, err := s.reader.Read(ctx, p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			continue // degrade: a down owner drops only its derived gaps
		}
		for _, c := range def.Cells {
			if c.Status == spacedoc.StatusNow {
				continue // a NOW cell is not a gap
			}
			out = append(out, Gap{
				ID:           string(p) + "/" + c.ID,
				Axis:         AxisCoverage,
				Projection:   p,
				Title:        c.Question,
				Status:       c.Status,
				SourceCellID: c.ID,
				Notes:        append([]string(nil), c.Notes...),
			})
		}
	}
	return out, errors.Join(errs...)
}
