package scenariocli

import (
	"fmt"
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	scenarioTimingsParameterA = 100
	scenarioTimingsParameterB = 1000
)

type TimingsRequest struct {
	Scenario string
	Since    time.Time
	JSON     bool
}

type TimingsResponse struct {
	Rows []scenarioruntime.StartTimingSummary `json:"rows"`
}

func ParseTimingsRequest(globalsJSON bool, args []string) (TimingsRequest, error) {
	spec := commandSpec(CommandTimings)
	parsed, err := commandtree.ParseArgs("scenario timings", commandHelpText(CommandTimings), spec.Args, args)
	if err != nil {
		return TimingsRequest{}, err
	}
	var since time.Time
	if raw := parsed.FlagValue("--since"); raw != "" {
		parsedSince, parseErr := time.Parse("2006-01-02", raw)
		if parseErr != nil {
			parsedSince, parseErr = time.Parse(time.RFC3339, raw)
		}
		if parseErr != nil {
			return TimingsRequest{}, fmt.Errorf("invalid --since %q: use YYYY-MM-DD or RFC3339", raw)
		}
		since = parsedSince
	}
	return TimingsRequest{
		Scenario: parsed.FlagValue("--scenario"),
		Since:    since,
		JSON:     globalsJSON || parsed.HasFlag("--json"),
	}, nil
}

func RenderTimingsResponse(w io.Writer, format cliout.Format, resp TimingsResponse) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		if len(resp.Rows) == 0 {
			_, err := fmt.Fprintln(w, "No retained terminal start-operation timings.")
			return err
		}
		if _, err := fmt.Fprintln(w, "Scope       Operation  Step          Count  Mean     P50      P90      Total    Share"); err != nil {
			return err
		}
		for _, row := range resp.Rows {
			if _, err := fmt.Fprintf(w, "%-11s %-10s %-12s %5d  %-8s %-8s %-8s %-8s %5.1f%%\n", row.Scenario, row.Operation, row.Step, row.Count, formatTimingMS(row.MeanMS), formatTimingMS(row.P50MS), formatTimingMS(row.P90MS), formatTimingMS(row.TotalMS), row.Share*scenarioTimingsParameterA); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(w, "Note: statistics cover only the retained terminal-operation tail (currently the newest five per scenario and variant).")
		return err
	})
}

func formatTimingMS(value float64) string {
	if value >= scenarioTimingsParameterB {
		return fmt.Sprintf("%.1fs", value/scenarioTimingsParameterB)
	}
	return fmt.Sprintf("%.0fms", value)
}
