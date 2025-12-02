// Package enrichment attaches live test status to requirements.
package enrichment

import (
	"context"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// Enricher enriches requirements with live test status.
type Enricher interface {
	// Enrich attaches live status to requirements and validations.
	Enrich(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle) error

	// ComputeSummary calculates aggregate statistics.
	ComputeSummary(modules []*types.RequirementModule) Summary
}

// Summary contains aggregate statistics.
type Summary struct {
	Total            int
	ByDeclaredStatus map[types.DeclaredStatus]int
	ByLiveStatus     map[types.LiveStatus]int
	ByCriticality    map[types.Criticality]int
	CriticalityGap   int // P0/P1 requirements not complete
	ValidationStats  ValidationStats
}

// ValidationStats contains validation-level statistics.
type ValidationStats struct {
	Total          int
	Implemented    int
	Failing        int
	Planned        int
	NotImplemented int
}

// enricher implements Enricher.
type enricher struct {
	matcher    *Matcher
	aggregator *Aggregator
	hierarchy  *HierarchyResolver
}

// New creates an Enricher.
func New() Enricher {
	return &enricher{
		matcher:    NewMatcher(),
		aggregator: NewAggregator(),
		hierarchy:  NewHierarchyResolver(),
	}
}

// Enrich attaches live status to requirements and validations.
func (e *enricher) Enrich(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle) error {
	if index == nil || evidence == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Phase 1: Match evidence to validations
	for _, module := range index.Modules {
		for i := range module.Requirements {
			req := &module.Requirements[i]
			e.enrichRequirement(ctx, req, evidence)
		}
	}

	// Phase 2: Aggregate validation status to requirement level
	for _, module := range index.Modules {
		for i := range module.Requirements {
			req := &module.Requirements[i]
			e.aggregator.AggregateRequirement(req)
		}
	}

	// Phase 3: Resolve parent/child hierarchy status
	e.hierarchy.ResolveHierarchy(ctx, index)

	return nil
}

// enrichRequirement enriches a single requirement with evidence.
func (e *enricher) enrichRequirement(ctx context.Context, req *types.Requirement, evidence *types.EvidenceBundle) {
	if req == nil || evidence == nil {
		return
	}

	// Match evidence to validations
	for i := range req.Validations {
		val := &req.Validations[i]
		e.enrichValidation(val, req.ID, evidence)
	}

	// Also check for evidence directly mapped to requirement ID
	directEvidence := e.matcher.FindDirectEvidence(req.ID, evidence)
	if len(directEvidence) > 0 {
		// Apply direct evidence to requirement's live status
		statuses := make([]types.LiveStatus, len(directEvidence))
		for i, ev := range directEvidence {
			statuses[i] = ev.Status
		}
		req.LiveStatus = types.DeriveLiveRollup(statuses)
	}
}

// enrichValidation enriches a single validation with evidence.
func (e *enricher) enrichValidation(val *types.Validation, reqID string, evidence *types.EvidenceBundle) {
	if val == nil || evidence == nil {
		return
	}

	match := e.matcher.FindBestMatch(val, reqID, evidence)
	if match != nil {
		val.LiveStatus = match.Status
		val.LiveDetails = &types.LiveDetails{
			Timestamp:       match.UpdatedAt,
			DurationSeconds: match.DurationSeconds,
			Evidence:        match.Evidence,
			SourcePath:      match.SourcePath,
		}

		// Update validation status based on live status
		val.Status = types.DeriveValidationStatus(match.Status)
	}
}

// ComputeSummary calculates aggregate statistics.
func (e *enricher) ComputeSummary(modules []*types.RequirementModule) Summary {
	summary := Summary{
		ByDeclaredStatus: make(map[types.DeclaredStatus]int),
		ByLiveStatus:     make(map[types.LiveStatus]int),
		ByCriticality:    make(map[types.Criticality]int),
	}

	for _, module := range modules {
		for _, req := range module.Requirements {
			summary.Total++

			// Count by declared status
			summary.ByDeclaredStatus[req.Status]++

			// Count by live status
			summary.ByLiveStatus[req.LiveStatus]++

			// Count by criticality
			if req.Criticality != "" {
				summary.ByCriticality[req.Criticality]++
			}

			// Count criticality gap
			if req.IsCriticalReq() && req.Status != types.StatusComplete {
				summary.CriticalityGap++
			}

			// Count validation stats
			for _, val := range req.Validations {
				summary.ValidationStats.Total++
				switch val.Status {
				case types.ValStatusImplemented:
					summary.ValidationStats.Implemented++
				case types.ValStatusFailing:
					summary.ValidationStats.Failing++
				case types.ValStatusPlanned:
					summary.ValidationStats.Planned++
				case types.ValStatusNotImplemented:
					summary.ValidationStats.NotImplemented++
				}
			}
		}
	}

	return summary
}

// EnrichOptions configures enrichment behavior.
type EnrichOptions struct {
	// IncludeManual includes manual validation evidence.
	IncludeManual bool

	// PreferRecent prefers more recent evidence when multiple matches exist.
	PreferRecent bool

	// RollupHierarchy rolls up status from children to parents.
	RollupHierarchy bool
}

// DefaultEnrichOptions returns default enrichment options.
func DefaultEnrichOptions() EnrichOptions {
	return EnrichOptions{
		IncludeManual:   true,
		PreferRecent:    true,
		RollupHierarchy: true,
	}
}
