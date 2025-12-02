// Package reporting generates requirement reports in various formats.
package reporting

import (
	"context"
	"io"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// Reporter generates requirement reports.
type Reporter interface {
	// Generate writes a report to the provided writer.
	Generate(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts Options, w io.Writer) error
}

// Options configures report generation.
type Options struct {
	// Format specifies the output format.
	Format Format

	// IncludePending includes pending requirements.
	IncludePending bool

	// IncludeValidations includes validation details.
	IncludeValidations bool

	// Phase filters to a specific phase (for phase-inspect mode).
	Phase string

	// Verbose includes additional details.
	Verbose bool
}

// Format specifies report output format.
type Format string

const (
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
	FormatTrace    Format = "trace"
	FormatSummary  Format = "summary"
)

// DefaultOptions returns default report options.
func DefaultOptions() Options {
	return Options{
		Format:             FormatJSON,
		IncludePending:     true,
		IncludeValidations: true,
		Verbose:            false,
	}
}

// reporter implements Reporter.
type reporter struct {
	jsonRenderer     *JSONRenderer
	markdownRenderer *MarkdownRenderer
	traceRenderer    *TraceRenderer
}

// New creates a Reporter.
func New() Reporter {
	return &reporter{
		jsonRenderer:     NewJSONRenderer(),
		markdownRenderer: NewMarkdownRenderer(),
		traceRenderer:    NewTraceRenderer(),
	}
}

// Generate writes a report to the provided writer.
func (r *reporter) Generate(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts Options, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch opts.Format {
	case FormatJSON:
		return r.jsonRenderer.Render(ctx, index, summary, opts, w)
	case FormatMarkdown:
		return r.markdownRenderer.Render(ctx, index, summary, opts, w)
	case FormatTrace:
		return r.traceRenderer.Render(ctx, index, summary, opts, w)
	case FormatSummary:
		return r.jsonRenderer.RenderSummary(ctx, summary, w)
	default:
		return r.jsonRenderer.Render(ctx, index, summary, opts, w)
	}
}

// ReportData represents the complete report data structure.
type ReportData struct {
	Summary    SummaryData    `json:"summary"`
	Modules    []ModuleData   `json:"modules,omitempty"`
	Statistics StatisticsData `json:"statistics"`
}

// SummaryData contains high-level summary information.
type SummaryData struct {
	Total            int     `json:"total"`
	Complete         int     `json:"complete"`
	InProgress       int     `json:"in_progress"`
	Pending          int     `json:"pending"`
	CompletionRate   float64 `json:"completion_rate"`
	CriticalGap      int     `json:"critical_gap"`
	LivePassed       int     `json:"live_passed"`
	LiveFailed       int     `json:"live_failed"`
	PassRate         float64 `json:"pass_rate"`
}

// ModuleData contains module-level information.
type ModuleData struct {
	Name         string            `json:"name"`
	FilePath     string            `json:"file_path"`
	Requirements []RequirementData `json:"requirements,omitempty"`
	Summary      SummaryData       `json:"summary"`
}

// RequirementData contains requirement-level information.
type RequirementData struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	LiveStatus  string           `json:"live_status,omitempty"`
	Criticality string           `json:"criticality,omitempty"`
	Validations []ValidationData `json:"validations,omitempty"`
}

// ValidationData contains validation-level information.
type ValidationData struct {
	Type       string `json:"type"`
	Ref        string `json:"ref,omitempty"`
	Phase      string `json:"phase,omitempty"`
	Status     string `json:"status"`
	LiveStatus string `json:"live_status,omitempty"`
}

// StatisticsData contains detailed statistics.
type StatisticsData struct {
	ByStatus      map[string]int `json:"by_status"`
	ByCriticality map[string]int `json:"by_criticality"`
	ByPhase       map[string]int `json:"by_phase"`
	Validations   ValidationStats `json:"validations"`
}

// ValidationStats contains validation statistics.
type ValidationStats struct {
	Total          int `json:"total"`
	Implemented    int `json:"implemented"`
	Failing        int `json:"failing"`
	Planned        int `json:"planned"`
	NotImplemented int `json:"not_implemented"`
}

// BuildReportData creates ReportData from index and summary.
func BuildReportData(index *parsing.ModuleIndex, summary enrichment.Summary, opts Options) ReportData {
	data := ReportData{
		Summary: SummaryData{
			Total:       summary.Total,
			CriticalGap: summary.CriticalityGap,
		},
		Statistics: StatisticsData{
			ByStatus:      make(map[string]int),
			ByCriticality: make(map[string]int),
			ByPhase:       make(map[string]int),
			Validations: ValidationStats{
				Total:          summary.ValidationStats.Total,
				Implemented:    summary.ValidationStats.Implemented,
				Failing:        summary.ValidationStats.Failing,
				Planned:        summary.ValidationStats.Planned,
				NotImplemented: summary.ValidationStats.NotImplemented,
			},
		},
	}

	// Convert status counts
	for status, count := range summary.ByDeclaredStatus {
		data.Statistics.ByStatus[string(status)] = count
		switch status {
		case types.StatusComplete:
			data.Summary.Complete = count
		case types.StatusInProgress:
			data.Summary.InProgress = count
		case types.StatusPending:
			data.Summary.Pending = count
		}
	}

	// Convert live status counts
	for status, count := range summary.ByLiveStatus {
		switch status {
		case types.LivePassed:
			data.Summary.LivePassed = count
		case types.LiveFailed:
			data.Summary.LiveFailed = count
		}
	}

	// Calculate rates
	if data.Summary.Total > 0 {
		data.Summary.CompletionRate = float64(data.Summary.Complete) / float64(data.Summary.Total) * 100
	}
	tested := data.Summary.LivePassed + data.Summary.LiveFailed
	if tested > 0 {
		data.Summary.PassRate = float64(data.Summary.LivePassed) / float64(tested) * 100
	}

	// Convert criticality counts
	for crit, count := range summary.ByCriticality {
		data.Statistics.ByCriticality[string(crit)] = count
	}

	// Build module data
	if index != nil {
		for _, module := range index.Modules {
			moduleData := buildModuleData(module, opts)
			data.Modules = append(data.Modules, moduleData)
		}
	}

	return data
}

// buildModuleData creates ModuleData from a module.
func buildModuleData(module *types.RequirementModule, opts Options) ModuleData {
	data := ModuleData{
		Name:     module.EffectiveName(),
		FilePath: module.FilePath,
	}

	var complete, inProgress, pending, passed, failed int

	for _, req := range module.Requirements {
		if !opts.IncludePending && req.Status == types.StatusPending {
			continue
		}

		reqData := RequirementData{
			ID:          req.ID,
			Title:       req.Title,
			Status:      string(req.Status),
			LiveStatus:  string(req.LiveStatus),
			Criticality: string(req.Criticality),
		}

		if opts.IncludeValidations {
			for _, val := range req.Validations {
				if opts.Phase != "" && val.Phase != opts.Phase {
					continue
				}
				reqData.Validations = append(reqData.Validations, ValidationData{
					Type:       string(val.Type),
					Ref:        val.Ref,
					Phase:      val.Phase,
					Status:     string(val.Status),
					LiveStatus: string(val.LiveStatus),
				})
			}
		}

		data.Requirements = append(data.Requirements, reqData)

		// Count for module summary
		switch req.Status {
		case types.StatusComplete:
			complete++
		case types.StatusInProgress:
			inProgress++
		case types.StatusPending:
			pending++
		}

		switch req.LiveStatus {
		case types.LivePassed:
			passed++
		case types.LiveFailed:
			failed++
		}
	}

	total := len(data.Requirements)
	data.Summary = SummaryData{
		Total:      total,
		Complete:   complete,
		InProgress: inProgress,
		Pending:    pending,
		LivePassed: passed,
		LiveFailed: failed,
	}

	if total > 0 {
		data.Summary.CompletionRate = float64(complete) / float64(total) * 100
	}

	return data
}
