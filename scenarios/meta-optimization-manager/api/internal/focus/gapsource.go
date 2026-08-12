package focus

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

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

// conditionGapSource turns observed serving degradation into empirical focus
// items. It is intentionally derived from the existing Search Hub insight
// seam, so no provider or projection list is authored in Meta-Optimization
// Manager.
type conditionGapSource struct {
	reader     ProviderInsights
	population func(context.Context) (map[string]struct{}, error)
}

func NewConditionGapSource(reader ProviderInsights) GapSource {
	return &conditionGapSource{reader: reader}
}

// NewConditionGapSourceWithPopulation derives the condition population from
// the live coverage join. This keeps condition from inventing a second list of
// providers: only legs backing cells that currently resolve NOW are eligible.
func NewConditionGapSourceWithPopulation(reader ProviderInsights, population func(context.Context) (map[string]struct{}, error)) GapSource {
	return &conditionGapSource{reader: reader, population: population}
}

func (s *conditionGapSource) Axis() Axis { return AxisEmpirical }

func (s *conditionGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s == nil || s.reader == nil {
		return nil, fmt.Errorf("condition reader is not configured")
	}
	insights, err := s.reader.Insights(ctx)
	if err != nil {
		return nil, fmt.Errorf("condition telemetry unavailable: %w", err)
	}
	allowed := map[string]struct{}{}
	if s.population != nil {
		allowed, err = s.population(ctx)
		if err != nil {
			return nil, fmt.Errorf("condition population unavailable: %w", err)
		}
	}
	out := make([]Gap, 0)
	for _, insight := range insights {
		if insight.DegradationRate <= 0 || strings.TrimSpace(insight.ProviderID) == "" {
			continue
		}
		if s.population != nil {
			if _, ok := allowed[strings.ToLower(strings.TrimSpace(insight.ProviderID))]; !ok {
				continue
			}
		}
		out = append(out, Gap{
			ID:              "condition/" + insight.ProviderID,
			Axis:            AxisEmpirical,
			Title:           "provider leg is serving degraded responses",
			ProviderIDs:     []string{insight.ProviderID},
			EvidenceSource:  "search-hub",
			EvidenceLocator: "search-hub://insights/" + insight.ProviderID,
			Recurrence:      int(insight.TimesRouted * int64(insight.DegradationRate*100)),
			Notes:           []string{fmt.Sprintf("routed=%d degradation_rate=%.2f", insight.TimesRouted, insight.DegradationRate)},
		})
	}
	return out, nil
}

type maturityGapSource struct{ reader MaturityReader }

// NewMaturityGapSource exposes blocking maturity evidence on the same
// condition axis as serving degradation. It never changes coverage status or
// takes remediation action.
func NewMaturityGapSource(reader MaturityReader) GapSource {
	return &maturityGapSource{reader: reader}
}

func (s *maturityGapSource) Axis() Axis { return AxisEmpirical }

func (s *maturityGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s == nil || s.reader == nil {
		return nil, fmt.Errorf("maturity reader is not configured")
	}
	observations, err := s.reader.Maturity(ctx)
	if err != nil {
		return nil, fmt.Errorf("maturity evidence unavailable: %w", err)
	}
	out := make([]Gap, 0, len(observations))
	for _, observation := range observations {
		scenario := strings.TrimSpace(observation.Scenario)
		if scenario == "" || len(observation.BlockingCodes) == 0 {
			continue
		}
		codes := append([]string(nil), observation.BlockingCodes...)
		sort.Strings(codes)
		out = append(out, Gap{
			ID:              "condition/maturity/" + scenario,
			Axis:            AxisEmpirical,
			Title:           "search maturity has blocking findings",
			EvidenceSource:  "search-hub",
			EvidenceLocator: "search-hub://maturity/" + scenario,
			Recurrence:      len(codes),
			Notes:           []string{"blocking=" + strings.Join(codes, ",")},
		})
	}
	return out, nil
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
	var errs []error
	for _, named := range m.sources {
		if named.Source == nil {
			out = append(out, sourceAvailabilityGap(named.Name, sourceAxis(nil), errors.New("source is not configured")))
			continue
		}
		gaps, err := named.Source.DerivedGaps(ctx)
		out = append(out, gaps...)
		if err != nil {
			out = append(out, sourceAvailabilityGap(named.Name, sourceAxis(named.Source), err))
			errs = append(errs, fmt.Errorf("%s: %w", named.Name, err))
		}
	}
	return out, errors.Join(errs...)
}

// providerIDs normalizes the human-authored owner column into stable lookup
// keys. A cell may name multiple providers with '+' or ','; parenthetical
// explanations are presentation text, not provider identities.
func providerIDs(owner string) []string {
	parts := strings.FieldsFunc(owner, func(r rune) bool { return r == '+' || r == ',' })
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "`"))
		if open := strings.IndexByte(part, '('); open >= 0 {
			part = strings.TrimSpace(part[:open])
		}
		if part == "" || strings.Trim(part, "_-— ") == "" || strings.Contains(strings.ToLower(part), "none") {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
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
	reader   SpaceReader
	liveFunc func(context.Context, Projection, *spacedoc.SpaceDefinition) (map[string]spacedoc.CellStatus, error)
}

func (*spaceGapSource) Axis() Axis { return AxisCoverage }

// NewSpaceGapSource constructs the production GapSource over a SpaceReader.
func NewSpaceGapSource(r SpaceReader) GapSource { return &spaceGapSource{reader: r} }

// NewSpaceGapSourceWithLiveJoin applies an optional live numerator to the
// denominator before deriving gaps. The function is intentionally injected so
// focus does not own another cross-scenario transport client; production wires
// the coverage owner's typed joiner at the handler boundary.
func NewSpaceGapSourceWithLiveJoin(r SpaceReader, liveFunc func(context.Context, Projection, *spacedoc.SpaceDefinition) (map[string]spacedoc.CellStatus, error)) GapSource {
	return &spaceGapSource{reader: r, liveFunc: liveFunc}
}

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
		live := map[string]spacedoc.CellStatus{}
		if s.liveFunc != nil {
			if statuses, liveErr := s.liveFunc(ctx, p, def); liveErr == nil {
				live = statuses
			} else {
				errs = append(errs, fmt.Errorf("%s live join: %w", p, liveErr))
			}
		}
		for _, c := range def.Cells {
			status := c.Status
			if liveStatus, ok := live[c.ID]; ok {
				status = liveStatus
			}
			if status == spacedoc.StatusNow {
				continue // a NOW cell is not a gap
			}
			out = append(out, Gap{
				ID:           string(p) + "/" + c.ID,
				Axis:         AxisCoverage,
				Projection:   p,
				Title:        c.Question,
				Status:       status,
				SourceCellID: c.ID,
				ProviderIDs:  providerIDs(c.Owner),
				Notes:        append([]string(nil), c.Notes...),
			})
		}
	}
	return out, errors.Join(errs...)
}
