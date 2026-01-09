package enrichment

import (
	"test-genie/internal/requirements/types"
)

// Aggregator aggregates validation status to requirement level.
type Aggregator struct {
	// Configuration options can be added here
}

// NewAggregator creates a new Aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{}
}

// AggregateRequirement computes aggregated status for a requirement.
func (a *Aggregator) AggregateRequirement(req *types.Requirement) {
	if req == nil {
		return
	}

	// Collect live statuses from validations
	var liveStatuses []types.LiveStatus
	var valStatuses []types.ValidationStatus

	for _, val := range req.Validations {
		if val.LiveStatus != "" && val.LiveStatus != types.LiveUnknown {
			liveStatuses = append(liveStatuses, val.LiveStatus)
		}
		valStatuses = append(valStatuses, val.Status)
	}

	// Compute live rollup
	if len(liveStatuses) > 0 {
		req.LiveStatus = types.DeriveLiveRollup(liveStatuses)
	}

	// Compute aggregated status
	req.AggregatedStatus = types.AggregatedStatus{
		LiveRollup:        req.LiveStatus,
		DeclaredRollup:    a.deriveDeclaredRollup(req.Status, valStatuses),
		ValidationSummary: types.ComputeValidationSummary(req.Validations),
	}
}

// deriveDeclaredRollup computes the declared status rollup.
func (a *Aggregator) deriveDeclaredRollup(current types.DeclaredStatus, valStatuses []types.ValidationStatus) types.DeclaredStatus {
	if len(valStatuses) == 0 {
		return current
	}

	return types.DeriveRequirementStatus(current, valStatuses)
}

// AggregateModule computes summary for an entire module.
func (a *Aggregator) AggregateModule(module *types.RequirementModule) ModuleSummary {
	summary := ModuleSummary{
		ModuleName: module.EffectiveName(),
		FilePath:   module.FilePath,
	}

	for _, req := range module.Requirements {
		summary.Total++

		switch req.Status {
		case types.StatusComplete:
			summary.Complete++
		case types.StatusInProgress:
			summary.InProgress++
		case types.StatusPlanned:
			summary.Planned++
		case types.StatusPending:
			summary.Pending++
		case types.StatusNotImplemented:
			summary.NotImplemented++
		}

		if req.IsCriticalReq() {
			summary.Critical++
			if req.Status != types.StatusComplete {
				summary.CriticalGap++
			}
		}

		// Live status counts
		switch req.LiveStatus {
		case types.LivePassed:
			summary.LivePassed++
		case types.LiveFailed:
			summary.LiveFailed++
		case types.LiveSkipped:
			summary.LiveSkipped++
		case types.LiveNotRun:
			summary.LiveNotRun++
		}
	}

	if summary.Total > 0 {
		summary.CompletionPercent = float64(summary.Complete) / float64(summary.Total) * 100
	}

	return summary
}

// ModuleSummary contains aggregated statistics for a module.
type ModuleSummary struct {
	ModuleName        string
	FilePath          string
	Total             int
	Complete          int
	InProgress        int
	Planned           int
	Pending           int
	NotImplemented    int
	Critical          int
	CriticalGap       int
	LivePassed        int
	LiveFailed        int
	LiveSkipped       int
	LiveNotRun        int
	CompletionPercent float64
}

// AggregateAll computes overall summary across all modules.
func (a *Aggregator) AggregateAll(modules []*types.RequirementModule) OverallSummary {
	summary := OverallSummary{
		ModuleSummaries: make([]ModuleSummary, 0, len(modules)),
	}

	for _, module := range modules {
		moduleSummary := a.AggregateModule(module)
		summary.ModuleSummaries = append(summary.ModuleSummaries, moduleSummary)

		summary.Total += moduleSummary.Total
		summary.Complete += moduleSummary.Complete
		summary.InProgress += moduleSummary.InProgress
		summary.Planned += moduleSummary.Planned
		summary.Pending += moduleSummary.Pending
		summary.NotImplemented += moduleSummary.NotImplemented
		summary.Critical += moduleSummary.Critical
		summary.CriticalGap += moduleSummary.CriticalGap
		summary.LivePassed += moduleSummary.LivePassed
		summary.LiveFailed += moduleSummary.LiveFailed
		summary.LiveSkipped += moduleSummary.LiveSkipped
		summary.LiveNotRun += moduleSummary.LiveNotRun
	}

	if summary.Total > 0 {
		summary.CompletionPercent = float64(summary.Complete) / float64(summary.Total) * 100
	}

	if summary.LivePassed+summary.LiveFailed+summary.LiveSkipped > 0 {
		tested := summary.LivePassed + summary.LiveFailed + summary.LiveSkipped
		summary.PassRate = float64(summary.LivePassed) / float64(tested) * 100
	}

	return summary
}

// OverallSummary contains aggregated statistics across all modules.
type OverallSummary struct {
	Total             int
	Complete          int
	InProgress        int
	Planned           int
	Pending           int
	NotImplemented    int
	Critical          int
	CriticalGap       int
	LivePassed        int
	LiveFailed        int
	LiveSkipped       int
	LiveNotRun        int
	CompletionPercent float64
	PassRate          float64
	ModuleSummaries   []ModuleSummary
}

// IsHealthy returns true if there are no failing tests and critical gap is zero.
func (s OverallSummary) IsHealthy() bool {
	return s.LiveFailed == 0 && s.CriticalGap == 0
}

// HasFailures returns true if any tests are failing.
func (s OverallSummary) HasFailures() bool {
	return s.LiveFailed > 0
}

// HasCriticalGap returns true if any critical requirements are incomplete.
func (s OverallSummary) HasCriticalGap() bool {
	return s.CriticalGap > 0
}
