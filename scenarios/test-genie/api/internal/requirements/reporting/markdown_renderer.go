package reporting

import (
	"context"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// MarkdownRenderer renders reports as Markdown.
type MarkdownRenderer struct{}

// NewMarkdownRenderer creates a new MarkdownRenderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render writes a Markdown report.
func (r *MarkdownRenderer) Render(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts Options, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Header
	fmt.Fprintln(w, "# Requirements Report")
	fmt.Fprintln(w)

	// Summary section
	r.renderSummary(summary, w)

	// Modules section
	if index != nil && len(index.Modules) > 0 {
		fmt.Fprintln(w, "## Modules")
		fmt.Fprintln(w)

		for _, module := range index.Modules {
			r.renderModule(module, opts, w)
		}
	}

	return nil
}

// renderSummary renders the summary section.
func (r *MarkdownRenderer) renderSummary(summary enrichment.Summary, w io.Writer) {
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)

	// Calculate stats
	complete := summary.ByDeclaredStatus[types.StatusComplete]
	inProgress := summary.ByDeclaredStatus[types.StatusInProgress]
	pending := summary.ByDeclaredStatus[types.StatusPending]
	passed := summary.ByLiveStatus[types.LivePassed]
	failed := summary.ByLiveStatus[types.LiveFailed]

	completionRate := float64(0)
	if summary.Total > 0 {
		completionRate = float64(complete) / float64(summary.Total) * 100
	}

	passRate := float64(0)
	tested := passed + failed
	if tested > 0 {
		passRate = float64(passed) / float64(tested) * 100
	}

	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| Total Requirements | %d |\n", summary.Total)
	fmt.Fprintf(w, "| Complete | %d |\n", complete)
	fmt.Fprintf(w, "| In Progress | %d |\n", inProgress)
	fmt.Fprintf(w, "| Pending | %d |\n", pending)
	fmt.Fprintf(w, "| Completion Rate | %.1f%% |\n", completionRate)
	fmt.Fprintf(w, "| Critical Gap | %d |\n", summary.CriticalityGap)
	fmt.Fprintf(w, "| Tests Passed | %d |\n", passed)
	fmt.Fprintf(w, "| Tests Failed | %d |\n", failed)
	fmt.Fprintf(w, "| Pass Rate | %.1f%% |\n", passRate)
	fmt.Fprintln(w)

	// Validation stats
	fmt.Fprintln(w, "### Validations")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "| Status | Count |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| Implemented | %d |\n", summary.ValidationStats.Implemented)
	fmt.Fprintf(w, "| Failing | %d |\n", summary.ValidationStats.Failing)
	fmt.Fprintf(w, "| Planned | %d |\n", summary.ValidationStats.Planned)
	fmt.Fprintf(w, "| Not Implemented | %d |\n", summary.ValidationStats.NotImplemented)
	fmt.Fprintln(w)
}

// renderModule renders a single module section.
func (r *MarkdownRenderer) renderModule(module *types.RequirementModule, opts Options, w io.Writer) {
	fmt.Fprintf(w, "### %s\n\n", module.EffectiveName())

	if len(module.Requirements) == 0 {
		fmt.Fprintln(w, "_No requirements_")
		fmt.Fprintln(w)
		return
	}

	// Requirements table
	fmt.Fprintf(w, "| ID | Title | Status | Live Status | Criticality |\n")
	fmt.Fprintf(w, "|----|-------|--------|-------------|-------------|\n")

	for _, req := range module.Requirements {
		if !opts.IncludePending && req.Status == types.StatusPending {
			continue
		}

		liveStatus := string(req.LiveStatus)
		if liveStatus == "" {
			liveStatus = "-"
		}

		criticality := string(req.Criticality)
		if criticality == "" {
			criticality = "-"
		}

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
			escapeMarkdown(req.ID),
			escapeMarkdown(truncate(req.Title, 50)),
			statusBadge(req.Status),
			liveStatusBadge(req.LiveStatus),
			criticality,
		)
	}
	fmt.Fprintln(w)

	// Optionally render validations
	if opts.IncludeValidations && opts.Verbose {
		r.renderValidations(module, opts, w)
	}
}

// renderValidations renders validations for a module.
func (r *MarkdownRenderer) renderValidations(module *types.RequirementModule, opts Options, w io.Writer) {
	var hasValidations bool
	for _, req := range module.Requirements {
		if len(req.Validations) > 0 {
			hasValidations = true
			break
		}
	}

	if !hasValidations {
		return
	}

	fmt.Fprintln(w, "#### Validations")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "| Requirement | Type | Ref | Phase | Status |\n")
	fmt.Fprintf(w, "|-------------|------|-----|-------|--------|\n")

	for _, req := range module.Requirements {
		for _, val := range req.Validations {
			if opts.Phase != "" && val.Phase != opts.Phase {
				continue
			}

			ref := val.Ref
			if ref == "" {
				ref = val.WorkflowID
			}
			if ref == "" {
				ref = "-"
			}

			phase := val.Phase
			if phase == "" {
				phase = "-"
			}

			fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
				escapeMarkdown(req.ID),
				string(val.Type),
				escapeMarkdown(truncate(ref, 40)),
				phase,
				validationStatusBadge(val.Status),
			)
		}
	}
	fmt.Fprintln(w)
}

// statusBadge returns a badge for declared status.
func statusBadge(s types.DeclaredStatus) string {
	switch s {
	case types.StatusComplete:
		return "✅ Complete"
	case types.StatusInProgress:
		return "🔄 In Progress"
	case types.StatusPlanned:
		return "📋 Planned"
	case types.StatusPending:
		return "⏳ Pending"
	case types.StatusNotImplemented:
		return "❌ Not Implemented"
	default:
		return string(s)
	}
}

// liveStatusBadge returns a badge for live status.
func liveStatusBadge(s types.LiveStatus) string {
	switch s {
	case types.LivePassed:
		return "✅ Passed"
	case types.LiveFailed:
		return "❌ Failed"
	case types.LiveSkipped:
		return "⏭️ Skipped"
	case types.LiveNotRun:
		return "⏸️ Not Run"
	default:
		return "-"
	}
}

// validationStatusBadge returns a badge for validation status.
func validationStatusBadge(s types.ValidationStatus) string {
	switch s {
	case types.ValStatusImplemented:
		return "✅ Implemented"
	case types.ValStatusFailing:
		return "❌ Failing"
	case types.ValStatusPlanned:
		return "📋 Planned"
	case types.ValStatusNotImplemented:
		return "⏳ Not Implemented"
	default:
		return string(s)
	}
}

// escapeMarkdown escapes special Markdown characters.
func escapeMarkdown(s string) string {
	// Escape pipe characters which break tables
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
