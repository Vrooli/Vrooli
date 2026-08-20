package focus

import (
	"context"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
	internalcoverage "infrastructure-manager/internal/coverage"
)

// CoverageSource adapts the coverage domain's read model to the focus domain.
// It remains read-only and keeps cross-domain wiring out of transport handlers.
type CoverageSource struct{ Coverage *internalcoverage.Service }

func NewCoverageSource(root string) CoverageSource {
	return CoverageSource{Coverage: internalcoverage.NewService(root, nil)}
}

func (s CoverageSource) Read(ctx context.Context) ([]Finding, []GapSource, error) {
	if s.Coverage == nil {
		return nil, unavailableSources("coverage source is not configured"), nil
	}
	snapshot, err := s.Coverage.Snapshot(ctx, nil)
	if err != nil {
		return nil, unavailableSources(err.Error()), nil
	}
	findings := make([]Finding, 0)
	for projection, item := range snapshot.Projections {
		for _, cell := range item.Definition.Cells {
			if cell.Status != spacedoc.StatusMissing {
				continue
			}
			findings = append(findings, Finding{ID: string(projection) + "/" + cell.ID, Source: "coverage", CellRef: string(projection) + "/" + cell.ID, Title: cell.Question, Message: strings.Join(cell.Notes, " "), Stage: StageMeasurement, Severity: 1, ExpectedReturn: "NOW"})
		}
	}
	for _, integrity := range snapshot.Findings {
		source := "coverage-drift"
		if integrity.Code == "SPACE_UNAVAILABLE" {
			source = "source-unavailability"
		}
		findings = append(findings, Finding{ID: "integrity/" + integrity.Code + "/" + integrity.Location, Source: source, CellRef: integrity.Location, Title: integrity.Code, Message: integrity.Message, Stage: StageIntegrity, Severity: 10})
	}
	openLoopCount, coverageDriftCount := 0, 0
	for _, finding := range findings {
		if finding.Stage == StageMeasurement {
			openLoopCount++
		}
		if finding.Stage == StageIntegrity {
			coverageDriftCount++
		}
	}
	return findings, []GapSource{
		{ID: "out-of-band", Label: "out-of-band readings", Available: false, Reason: "condition source join is not configured"},
		{ID: "untrusted", Label: "untrusted readings", Available: false, Reason: "condition trust source join is not configured"},
		{ID: "open-loop", Label: "open-loop cells", Available: true, FindingCount: openLoopCount},
		{ID: "coverage-drift", Label: "coverage drift", Available: true, FindingCount: coverageDriftCount},
		{ID: "source-unavailability", Label: "source unavailability", Available: true},
	}, nil
}

func unavailableSources(reason string) []GapSource {
	return []GapSource{
		{ID: "out-of-band", Label: "out-of-band readings", Available: false, Reason: reason},
		{ID: "untrusted", Label: "untrusted readings", Available: false, Reason: reason},
		{ID: "open-loop", Label: "open-loop cells", Available: false, Reason: reason},
		{ID: "coverage-drift", Label: "coverage drift", Available: false, Reason: reason},
		{ID: "source-unavailability", Label: "source unavailability", Available: false, Reason: reason},
	}
}
