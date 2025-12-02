package reporting

import (
	"context"
	"encoding/json"
	"io"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// TraceRenderer renders full traceability reports.
type TraceRenderer struct{}

// NewTraceRenderer creates a new TraceRenderer.
func NewTraceRenderer() *TraceRenderer {
	return &TraceRenderer{}
}

// TraceReport is the full traceability data structure.
type TraceReport struct {
	Summary      TraceSummary       `json:"summary"`
	Requirements []TraceRequirement `json:"requirements"`
	Coverage     TraceCoverage      `json:"coverage"`
}

// TraceSummary contains high-level trace information.
type TraceSummary struct {
	Total             int     `json:"total"`
	WithValidations   int     `json:"with_validations"`
	WithoutValidations int    `json:"without_validations"`
	TraceCoverage     float64 `json:"trace_coverage"`
}

// TraceRequirement contains full requirement trace.
type TraceRequirement struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Module      string            `json:"module"`
	FilePath    string            `json:"file_path"`
	Status      string            `json:"status"`
	LiveStatus  string            `json:"live_status,omitempty"`
	Criticality string            `json:"criticality,omitempty"`
	PRDRef      string            `json:"prd_ref,omitempty"`
	Children    []string          `json:"children,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Validations []TraceValidation `json:"validations,omitempty"`
	Evidence    []TraceEvidence   `json:"evidence,omitempty"`
}

// TraceValidation contains validation trace information.
type TraceValidation struct {
	Type       string `json:"type"`
	Ref        string `json:"ref,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Status     string `json:"status"`
	LiveStatus string `json:"live_status,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// TraceEvidence contains evidence trace information.
type TraceEvidence struct {
	Source    string `json:"source"`
	Phase     string `json:"phase,omitempty"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp,omitempty"`
	Path      string `json:"path,omitempty"`
}

// TraceCoverage contains coverage breakdown by various dimensions.
type TraceCoverage struct {
	ByPhase       map[string]PhaseCoverage       `json:"by_phase"`
	ByCriticality map[string]CriticalityCoverage `json:"by_criticality"`
}

// PhaseCoverage contains coverage for a single phase.
type PhaseCoverage struct {
	Total      int     `json:"total"`
	Covered    int     `json:"covered"`
	Coverage   float64 `json:"coverage"`
	Passing    int     `json:"passing"`
	Failing    int     `json:"failing"`
}

// CriticalityCoverage contains coverage for a criticality level.
type CriticalityCoverage struct {
	Total      int     `json:"total"`
	Complete   int     `json:"complete"`
	Coverage   float64 `json:"coverage"`
	Gap        int     `json:"gap"`
}

// Render writes a trace report.
func (r *TraceRenderer) Render(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts Options, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	report := r.buildTraceReport(index, summary, opts)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// buildTraceReport builds the complete trace report.
func (r *TraceRenderer) buildTraceReport(index *parsing.ModuleIndex, summary enrichment.Summary, opts Options) TraceReport {
	report := TraceReport{
		Coverage: TraceCoverage{
			ByPhase:       make(map[string]PhaseCoverage),
			ByCriticality: make(map[string]CriticalityCoverage),
		},
	}

	var withValidations, withoutValidations int
	phaseStats := make(map[string]*PhaseCoverage)
	critStats := make(map[string]*CriticalityCoverage)

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			traceReq := TraceRequirement{
				ID:          req.ID,
				Title:       req.Title,
				Module:      module.EffectiveName(),
				FilePath:    module.FilePath,
				Status:      string(req.Status),
				LiveStatus:  string(req.LiveStatus),
				Criticality: string(req.Criticality),
				PRDRef:      req.PRDRef,
				Children:    req.Children,
				DependsOn:   req.DependsOn,
			}

			// Add validations
			for _, val := range req.Validations {
				if opts.Phase != "" && val.Phase != opts.Phase {
					continue
				}
				traceReq.Validations = append(traceReq.Validations, TraceValidation{
					Type:       string(val.Type),
					Ref:        val.Ref,
					WorkflowID: val.WorkflowID,
					Phase:      val.Phase,
					Status:     string(val.Status),
					LiveStatus: string(val.LiveStatus),
					Notes:      val.Notes,
				})

				// Track phase coverage
				phase := val.Phase
				if phase == "" {
					phase = "unspecified"
				}
				if phaseStats[phase] == nil {
					phaseStats[phase] = &PhaseCoverage{}
				}
				phaseStats[phase].Total++
				if val.Status == types.ValStatusImplemented {
					phaseStats[phase].Covered++
					phaseStats[phase].Passing++
				} else if val.Status == types.ValStatusFailing {
					phaseStats[phase].Covered++
					phaseStats[phase].Failing++
				}
			}

			// Track validation coverage
			if len(req.Validations) > 0 {
				withValidations++
			} else {
				withoutValidations++
			}

			// Track criticality coverage
			crit := string(req.Criticality)
			if crit == "" {
				crit = "unspecified"
			}
			if critStats[crit] == nil {
				critStats[crit] = &CriticalityCoverage{}
			}
			critStats[crit].Total++
			if req.Status == types.StatusComplete {
				critStats[crit].Complete++
			} else if req.IsCriticalReq() {
				critStats[crit].Gap++
			}

			report.Requirements = append(report.Requirements, traceReq)
		}
	}

	// Calculate summary
	total := withValidations + withoutValidations
	report.Summary = TraceSummary{
		Total:              total,
		WithValidations:    withValidations,
		WithoutValidations: withoutValidations,
	}
	if total > 0 {
		report.Summary.TraceCoverage = float64(withValidations) / float64(total) * 100
	}

	// Calculate phase coverage
	for phase, stats := range phaseStats {
		if stats.Total > 0 {
			stats.Coverage = float64(stats.Covered) / float64(stats.Total) * 100
		}
		report.Coverage.ByPhase[phase] = *stats
	}

	// Calculate criticality coverage
	for crit, stats := range critStats {
		if stats.Total > 0 {
			stats.Coverage = float64(stats.Complete) / float64(stats.Total) * 100
		}
		report.Coverage.ByCriticality[crit] = *stats
	}

	return report
}
