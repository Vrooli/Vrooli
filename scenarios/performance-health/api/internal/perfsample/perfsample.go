// Package perfsample holds the cross-domain performance-sample DTO. It is
// shared substrate: a business-vocabulary-free measurement record that several
// producer domains (benchmark, analysis, startup) emit and the trend read-model
// persists. Keeping the DTO here — rather than inside the trend domain — lets
// producers depend on the measurement contract without importing a sibling
// domain, so the concrete trend store can be wired once from the composition
// root (dependency inversion; see docs/internal/SEAMS.md and the layering
// detector in architecture-cartographer). Each consumer owns its own narrow
// writer/reader interface over this DTO.
package perfsample

import "time"

// Sample is one performance sample. Every axis is optional; a zero value means
// "not measured this run".
type Sample struct {
	Scenario string
	// Flow tags a sample to a specific interaction-capture flow slug. Empty means
	// a scenario-level sample (build/bundle/startup/scenario-LCP); a non-empty
	// value scopes the sample to one budgeted journey so per-flow budgets gate
	// independently of the scenario aggregate.
	Flow                  string
	CapturedAt            time.Time
	GoBuildMs             int64
	UIBuildMs             int64
	BundleBytes           int64
	LCPMs                 int64
	StartupMs             int64
	SlowestComponent      string
	SlowestComponentAvgMs float64
	SlowestComponentMaxMs float64
	DrawnFPS              float64
	DroppedFrameRate      float64
	LongTaskTotalMs       int64
	LongTaskMaxMs         float64
	RasterTotalMs         float64
	LayoutTotalMs         float64
	PaintTotalMs          float64
	InputEventCount       int64
	Note                  string
}

// HasInteractionMetrics reports whether the sample carries any frame, browser
// work, long-task, or input evidence for an interaction flow.
func (s Sample) HasInteractionMetrics() bool {
	return s.DrawnFPS > 0 || s.DroppedFrameRate > 0 || s.LongTaskTotalMs > 0 ||
		s.LongTaskMaxMs > 0 || s.RasterTotalMs > 0 || s.LayoutTotalMs > 0 ||
		s.PaintTotalMs > 0 || s.InputEventCount > 0
}
