package measures

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/opportunities"
	"agent-manager/internal/usageattribution"

	measurelib "github.com/vrooli/measures-go"
)

// UsageOpportunities is what agents repeatedly do by hand that an owner can
// turn into a program, CLI command, focused flag or smaller skill.
const UsageOpportunities = "usage.opportunities"

// ProgramLibrarySource reads the program library sequences are matched to.
type ProgramLibrarySource = usageattribution.LibrarySource

// SetProgramLibrary supplies the program library for opportunity mining.
func (h *Handler) SetProgramLibrary(source ProgramLibrarySource) { h.programs = source }

var opportunitiesDefinition = Definition{
	ID: UsageOpportunities, Counts: "repeated Vrooli command sequences, direct reads of a scenario's internals, jq field extractions, and in-context skill reloads, each with an owner",
	Numerator: "calls the change removes or replaces, and result tokens those calls put in context", Denominator: "n/a: a ranked list beside its coverage row",
	SourceTable: "invocation_read_model_facts, program-runtime library",
	Limitation:  "a pattern needs three distinct runs; only runs projected at the current fact version are read; a sequence is matched to a program by called binding or by the program's owning scenario, since contracts do not declare the CLI commands they replace; token figures are upper bounds",
}

// registerOpportunities declares the measure. It is registry-only: rows are
// flat strings and no typed RPC exists, so Service and Method are empty.
func (h *Handler) registerOpportunities(registry *measurelib.Registry) error {
	return registry.Register(measurelib.MeasureDeclaration{
		Name: UsageOpportunities, Scenario: "agent-manager", Domain: "usage",
		Intent:    "Capability opportunities from agent behavior: repeated command sequences matched against the program library (adopt an existing program or write one), bypass clusters that want a CLI or RPC, JSON fields agents cut out that want a flag, and skills reloaded within a session.",
		Questions: []string{"what should become a program", "which programs are agents not using", "what do agents reach around the CLI for", "which skills are reloaded too often"},
		Params: map[string]measurelib.Param{
			"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenLast7d)},
			"since":  {Name: "since", Type: "string", Format: "RFC 3339", Description: "start of the window; overrides the window's start"},
		},
		Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: "calls_saved", Unit: "calls", SummaryTemplate: "capability opportunities ({window})"},
		Effect: measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		if h.tools == nil || h.programs == nil {
			return measurelib.MeasureResult{}, fmt.Errorf("opportunities are unavailable: tool calls or the program library are not wired")
		}
		window, err := h.resolveWindow(request, measurelib.TokenLast7d)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		if since := strings.TrimSpace(request.Params["since"]); since != "" {
			if window.From, err = time.Parse(time.RFC3339, since); err != nil {
				// A verify line says "since <date of the change>": a bare date is that.
				var dateErr error
				if window.From, dateErr = time.ParseInLocation("2006-01-02", since, time.UTC); dateErr != nil {
					return measurelib.MeasureResult{}, fmt.Errorf("since %q is not RFC 3339 or YYYY-MM-DD: %w", since, err)
				}
			}
		}
		if window.From, err = usageattribution.ToolWindowStart(window.From, window.To); err != nil {
			return measurelib.MeasureResult{}, err
		}
		tools, err := h.tools.ToolCalls(ctx, window.From, window.To)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		report := opportunities.Analyze(opportunities.Input{Calls: tools.Calls, Stale: tools.Stale, Library: h.programs.Library(ctx), Outputs: opportunities.OutputsFrom(ctx, h.tools), From: window.From, To: window.To})
		return measurelib.MeasureResult{Fields: opportunityRows(report), Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT command facts WITH stored tool features FROM invocation_read_model_facts WHERE tool_profiled = 1 AND classifier_version = <current> AND occurred_at >= %q AND occurred_at < %q; older versions counted as stale; program library from program-runtime",
			window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano))}}, nil
	})
}

// opportunityRows is the coverage row, then one row per opportunity in rank
// order. Multi-valued cells (details, runs, examples) are newline-separated.
func opportunityRows(report opportunities.Report) []map[string]string {
	n := strconv.Itoa
	c := report.Coverage
	rows := []map[string]string{{
		"kind": "coverage", "sentence": c.Sentence(), "calls": n(c.Calls), "vrooli_calls": n(c.VrooliCalls), "runs": n(c.Runs), "stale_runs": n(c.Stale.Runs), "stale_calls": n(c.Stale.Calls),
		"window_start": c.From.UTC().Format(time.RFC3339), "window_end": c.To.UTC().Format(time.RFC3339),
		"library_programs": n(len(c.Library.Programs)), "library_unavailable": c.Library.Unavailable,
	}}
	for _, o := range report.Opportunities {
		row := map[string]string{
			"kind": o.Kind, "owner": o.Owner, "subject": o.Subject, "action": o.Action, "verdict": o.Verdict,
			"sessions": n(o.Sessions), "occurrences": n(o.Occurrences), "calls_saved": n(o.CallsSaved), "saved_label": o.SavedLabel, "saved_basis": o.SavedBasis, "tokens_saved": strconv.FormatInt(o.TokensSaved, 10), "tokens_unknown": n(o.TokensUnknown),
			"dependency_measured": n(o.MeasuredSteps), "dependency_used": n(o.DependentSteps), "mandated": strconv.FormatBool(o.Mandated), "details": strings.Join(o.Details, "\n"), "run_ids": strings.Join(o.RunIDs, "\n"), "examples": strings.Join(o.Examples, "\n"), "verify": o.Verify,
		}
		if o.Program != nil {
			row["program"], row["program_basis"], row["program_runs"] = o.Program.Name, o.Program.Basis, n(o.Program.Runs)
		}
		rows = append(rows, row)
	}
	return rows
}
