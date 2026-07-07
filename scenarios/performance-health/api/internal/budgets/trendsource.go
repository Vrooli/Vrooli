package budgets

import (
	"context"

	"performance-health/internal/perfsample"
)

// SampleReader is the consumer-owned seam the measurement source needs: the
// newest persisted performance sample for a scenario (scenario-level via Latest,
// or a specific interaction flow via LatestFlow). The trend store satisfies it;
// budgets depends on the substrate DTO, not on the trend domain.
type SampleReader interface {
	Latest(ctx context.Context, scenario string) (perfsample.Sample, bool, error)
	LatestFlow(ctx context.Context, scenario, flow string) (perfsample.Sample, bool, error)
}

// sampleMeasurementSource adapts a SampleReader to the MeasurementSource seam:
// it maps the newest persisted sample into the measured-axis shape CheckBudget
// evaluates against. The component-commit budgets read the slowest component
// recorded on the sample.
type sampleMeasurementSource struct {
	reader SampleReader
}

// NewSampleMeasurementSource builds the MeasurementSource that reads the newest
// persisted performance sample. It is the single adapter both the budget gate
// and the fleet grader use, so the sample→measurement mapping lives in one place.
func NewSampleMeasurementSource(reader SampleReader) MeasurementSource {
	return sampleMeasurementSource{reader: reader}
}

var _ MeasurementSource = sampleMeasurementSource{}

func (s sampleMeasurementSource) Latest(ctx context.Context, scenario string) (Measurement, bool, error) {
	sample, found, err := s.reader.Latest(ctx, scenario)
	if err != nil || !found {
		return Measurement{}, found, err
	}
	return SampleToMeasurement(sample), true, nil
}

// LatestFlow maps the newest flow-tagged sample into the measured-axis shape,
// satisfying FlowMeasurementSource so CheckFlow can gate a single journey.
func (s sampleMeasurementSource) LatestFlow(ctx context.Context, scenario, flow string) (Measurement, bool, error) {
	sample, found, err := s.reader.LatestFlow(ctx, scenario, flow)
	if err != nil || !found {
		return Measurement{}, found, err
	}
	return SampleToMeasurement(sample), true, nil
}

var _ FlowMeasurementSource = sampleMeasurementSource{}

// SampleToMeasurement maps a persisted performance sample to the budget
// measurement shape. Exported so the fleet grader shares the exact same mapping
// the budget gate uses (a single source of truth for "what counts as measured").
func SampleToMeasurement(s perfsample.Sample) Measurement {
	return Measurement{
		GoBuildMs:            s.GoBuildMs,
		UIBuildMs:            s.UIBuildMs,
		BundleBytes:          s.BundleBytes,
		LCPMs:                s.LCPMs,
		StartupMs:            s.StartupMs,
		ComponentCommitAvgMs: s.SlowestComponentAvgMs,
		ComponentCommitMaxMs: s.SlowestComponentMaxMs,
		DrawnFPS:             s.DrawnFPS,
		DroppedFrameRate:     s.DroppedFrameRate,
		LongTaskTotalMs:      s.LongTaskTotalMs,
		LongTaskMaxMs:        s.LongTaskMaxMs,
		RasterTotalMs:        s.RasterTotalMs,
		LayoutTotalMs:        s.LayoutTotalMs,
		PaintTotalMs:         s.PaintTotalMs,
		InputEventCount:      s.InputEventCount,
		SlowestComponent:     s.SlowestComponent,
	}
}
