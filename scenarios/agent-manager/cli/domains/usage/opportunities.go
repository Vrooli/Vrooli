package usage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// opportunitiesMeasure is the registry measure `usage opportunities` reads.
const opportunitiesMeasure = "usage.opportunities"

// opportunitiesFetch executes the measure with the given parameters and
// returns the registry's JSON body; tests replace it.
type opportunitiesFetch func(cliapp.RunContext, map[string]string) ([]byte, error)

func registryFetch(ctx cliapp.RunContext, params map[string]string) ([]byte, error) {
	return ctx.Core().RequestRoot("POST", "measures/execute", nil, map[string]any{"measure": opportunitiesMeasure, "params": params})
}

// opportunityRow is one row of the measure: the coverage row (kind
// "coverage") or one opportunity. Every cell is a string on the wire.
type opportunityRow map[string]string

func (r opportunityRow) number(key string) int64 {
	n, _ := strconv.ParseInt(r[key], 10, 64)
	return n
}

func (r opportunityRow) lines(key string) []string {
	if r[key] == "" {
		return nil
	}
	return strings.Split(r[key], "\n")
}

func opportunitiesCommand(fetch opportunitiesFetch) cliapp.Command {
	return cliapp.Command{
		Name: "opportunities", NeedsAPI: true,
		Description: "What should become a program, CLI command, flag or smaller skill? [--since RFC3339|YYYY-MM-DD] [--json]",
		Usage:       "agent-manager usage opportunities [--since RFC3339|YYYY-MM-DD] [--json]",
		HelpText:    "Lists, grouped by the owner who can act (owners in the order of their best-ranked opportunity) and ranked by calls saved, what agents repeatedly do by hand in the window (default the last 7 days): Vrooli command sequences seen in three or more sessions, matched against the program library (adopt an existing program, or write one); direct sqlite3, HTTP or state-directory reads of a scenario's internals that want a CLI command or RPC; JSON fields agents cut out with jq that want a focused flag; skills read again while still in context. Each names its evidence runs, example commands and the check that shows the change worked. The coverage line says what was read: an empty list means nothing only beside it. Candidates are records for the owner, never built automatically. --json prints the measure rows (`measures get usage.opportunities`).",
		Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "since", Description: "RFC 3339 or YYYY-MM-DD start of the window (default 7 days ago)"}}},
		RunCtx: func(ctx cliapp.RunContext) error {
			params := map[string]string{}
			if since := strings.TrimSpace(ctx.Flag("since")); since != "" {
				params["since"] = since
			}
			body, err := fetch(ctx, params)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var result struct {
				Fields []opportunityRow `json:"fields"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("decode opportunities: %w", err)
			}
			return ctx.RenderList(opportunitiesReport(result.Fields))
		},
	}
}

// opportunitiesReport renders the coverage first, then the opportunities
// grouped by owner, in the measure's rank order: an owner comes where its
// best-ranked opportunity does. It adds up nothing the measure did not.
func opportunitiesReport(rows []opportunityRow) cliapp.ListReport {
	report := cliapp.ListReport{ResultsHeading: "Opportunities", ListShaped: true}
	var owners []string
	byOwner := map[string][]opportunityRow{}
	for _, row := range rows {
		if row["kind"] == "coverage" {
			report.Summary = append(report.Summary, "Coverage: "+row["sentence"])
			continue
		}
		owner := row["owner"]
		if _, ok := byOwner[owner]; !ok {
			owners = append(owners, owner)
		}
		byOwner[owner] = append(byOwner[owner], row)
		report.ResultCount++
	}
	if report.ResultCount == 0 {
		report.Summary = append(report.Summary, "No pattern met its threshold (three or more sessions) in what the coverage line says was seen.")
	}
	for _, owner := range owners {
		report.Results = append(report.Results, owner)
		for _, row := range byOwner[owner] {
			report.Results = append(report.Results, opportunityLines(row)...)
		}
	}
	report.RetrievalHints = []string{"agent-manager measures get usage.opportunities --param since=<date>  # re-run after a change to verify it", "agent-manager usage friction  # failures, polling and heavy output of the same commands"}
	return report
}

func opportunityLines(row opportunityRow) []string {
	label := row["kind"]
	if row["verdict"] != "" {
		label += " · " + row["verdict"]
	}
	lines := []string{fmt.Sprintf("  [%s] %s", label, row["subject"]), "     " + row["action"]}
	impact := fmt.Sprintf("     %d %s (%s)", row.number("calls_saved"), row["saved_label"], row["saved_basis"])
	switch {
	case row["verdict"] == "independent" || row["verdict"] == "unmeasured" || row["verdict"] == "undetermined":
		// The API claims no savings for these; none are shown.
		impact = "     no program savings claimed"
	case row["mandated"] == "true":
		// A mandated ritual keeps its work: only the round-trips and tokens go.
		impact = fmt.Sprintf("     %d round-trips saved (the mandated ritual's work stays)", row.number("calls_saved"))
	}
	if tokens := row.number("tokens_saved"); tokens > 0 {
		impact += ", up to " + compact(tokens) + " result tokens"
	}
	lines = append(lines, fmt.Sprintf("%s · %d sessions · %d occurrences", impact, row.number("sessions"), row.number("occurrences")))
	if row["program"] != "" {
		lines = append(lines, fmt.Sprintf("     program %s (%s match): %d program runs vs %d manual", row["program"], row["program_basis"], row.number("program_runs"), row.number("occurrences")))
	}
	for _, detail := range row.lines("details") {
		lines = append(lines, "     touches "+detail)
	}
	for _, example := range row.lines("examples") {
		lines = append(lines, "     e.g. "+example)
	}
	if runs := row.lines("run_ids"); len(runs) > 0 {
		lines = append(lines, "     runs "+strings.Join(runs, ", "))
	}
	return append(lines, "     verify: "+row["verify"])
}
