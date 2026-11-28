package annotations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// maxProvenancePasses bounds the client-side continuation of a refresh; each
// pass is independently time-budgeted by the server.
const maxProvenancePasses = 200

type provenanceRefresh struct {
	Policy    string `json:"policy"`
	Refreshed int    `json:"refreshed"`
	Failed    int    `json:"failed"`
	Remaining int    `json:"remaining"`
}

func provenanceCommand() cliapp.Command {
	return cliapp.Command{
		Name: "provenance", NeedsAPI: true,
		Description: "Recompute whose words annotated runs carried, after the pack's provenance policy changed",
		Usage:       "agent-manager annotations provenance [--limit N] [--json]",
		HelpText: "Each annotated run stores whether a person or an agent wrote its operator text, under the pack's runtime.provenance policy. " +
			"Editing that policy makes the stored rows stale: coverage then counts them apart instead of mixing two definitions. " +
			"This reads each stale run's facts and rewrites only its authorship row; no label is evaluated and no model is asked. " +
			"It repeats until none are stale.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "limit", Default: "0", Description: "Runs recomputed per server pass; 0 uses the server default"},
			{Name: "json", Bool: true, LocalOnly: true, Description: "Print the final result as JSON"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			limit := 0
			if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" && raw != "0" {
				parsed, err := parsePositiveInt(raw, "limit")
				if err != nil {
					return err
				}
				limit = parsed
			}
			var total provenanceRefresh
			for pass := 0; pass < maxProvenancePasses; pass++ {
				body, err := ctx.Core().Request("POST", "annotations/provenance/refresh", nil, map[string]int{"limit": limit})
				if err != nil {
					return err
				}
				var current provenanceRefresh
				if err := json.Unmarshal(body, &current); err != nil {
					return fmt.Errorf("decode provenance refresh: %w", err)
				}
				total.Policy, total.Remaining = current.Policy, current.Remaining
				total.Refreshed += current.Refreshed
				total.Failed += current.Failed
				if current.Remaining == 0 || current.Refreshed == 0 {
					break
				}
			}
			if ctx.JSON() {
				encoded, err := json.MarshalIndent(total, "", "  ")
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(ctx.Stdout(), string(encoded))
				return err
			}
			fmt.Fprintf(ctx.Stdout(), "Provenance policy %s: recomputed %d run(s), %d unreadable, %d still stale\n",
				total.Policy, total.Refreshed, total.Failed, total.Remaining)
			return nil
		},
	}
}
