package resourcecli

import (
	"fmt"
	"io"

	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/internal/cliout"
)

// WriteUpstreamCheck renders the aggregate coding-agent upstream-check report.
// JSON uses the standard success envelope (read-only/agent-safe; never a proto
// contract because the per-resource Reports are owned by the resource CLIs).
func WriteUpstreamCheck(w io.Writer, format cliout.Format, agg upstreamcheck.AggregateReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteSuccessJSON(w, "upstream_check", agg)
	}, func(w io.Writer) error {
		for _, rep := range agg.Resources {
			fmt.Fprintf(w, "%-14s installed=%-10s latest=%-10s %s\n", rep.Name, orUnknown(rep.Installed), orUnknown(rep.Latest), rep.Status)
			if rep.Note != "" {
				fmt.Fprintf(w, "  note: %s\n", rep.Note)
			}
		}
		if len(agg.Artifacts) > 0 {
			fmt.Fprintln(w, "\nresource acquisition liveness:")
			for _, finding := range agg.Artifacts {
				state := "reachable"
				if !finding.Reachable {
					state = "unreachable"
				}
				if finding.Stale {
					state += ", stale-after-two-failures"
				}
				fmt.Fprintf(w, "%-14s target=%-3d %-30s %s (%s)\n", finding.Resource, finding.Target, state, finding.Reference, finding.CheckedAt)
				if finding.Note != "" {
					fmt.Fprintf(w, "  note: %s\n", finding.Note)
				}
			}
		}
		if len(agg.Behind) > 0 {
			fmt.Fprintf(w, "\nbehind: %v — run `vrooli resource install <name>` (or the resource's `update`) to catch up\n", agg.Behind)
		}
		if len(agg.Unknown) > 0 {
			fmt.Fprintf(w, "unknown: %v — could not resolve installed/latest version\n", agg.Unknown)
		}
		return nil
	})
}

func orUnknown(s string) string {
	if s == "" {
		return "(unknown)"
	}
	return s
}
