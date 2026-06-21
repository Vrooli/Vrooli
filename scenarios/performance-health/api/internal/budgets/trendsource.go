package budgets

import (
	"context"

	"performance-health/internal/trend"
)

// trendReader is the slice of the trend store the measurement source needs: the
// newest persisted sample for a scenario.
type trendReader interface {
	Latest(ctx context.Context, scenario string) (trend.Sample, bool, error)
}

// trendMeasurementSource adapts the trend store to the MeasurementSource seam:
// it maps the newest persisted trend sample into the measured-axis shape
// CheckBudget evaluates against. The component-commit budgets read the slowest
// component recorded on the sample.
type trendMeasurementSource struct {
	reader trendReader
}

// NewTrendMeasurementSource builds the MeasurementSource that reads the newest
// persisted trend sample. It is the single adapter both the budget gate and the
// fleet grader use, so the trend→measurement mapping lives in one place.
func NewTrendMeasurementSource(reader trendReader) MeasurementSource {
	return trendMeasurementSource{reader: reader}
}

var _ MeasurementSource = trendMeasurementSource{}

func (s trendMeasurementSource) Latest(ctx context.Context, scenario string) (Measurement, bool, error) {
	sample, found, err := s.reader.Latest(ctx, scenario)
	if err != nil || !found {
		return Measurement{}, found, err
	}
	return SampleToMeasurement(sample), true, nil
}

// SampleToMeasurement maps a persisted trend sample to the budget measurement
// shape. Exported so the fleet grader shares the exact same mapping the budget
// gate uses (a single source of truth for "what counts as measured").
func SampleToMeasurement(s trend.Sample) Measurement {
	return Measurement{
		GoBuildMs:            s.GoBuildMs,
		UIBuildMs:            s.UIBuildMs,
		BundleBytes:          s.BundleBytes,
		LCPMs:                s.LCPMs,
		P95Ms:                s.P95Ms,
		StartupMs:            s.StartupMs,
		ComponentCommitAvgMs: s.SlowestComponentAvgMs,
		ComponentCommitMaxMs: s.SlowestComponentAvgMs,
		SlowestComponent:     s.SlowestComponent,
	}
}
