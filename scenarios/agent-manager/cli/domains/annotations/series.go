package annotations

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

type seriesPoint struct {
	Start             string   `json:"start"`
	Messages          int      `json:"messages"`
	ProvenMessages    int      `json:"proven_messages"`
	Mean              *float64 `json:"mean"`
	ProvenMean        *float64 `json:"proven_mean"`
	FrictionPositives int      `json:"friction_positives"`
}

type seriesResult struct {
	Label          string        `json:"label"`
	Bucket         string        `json:"bucket"`
	Messages       int           `json:"messages"`
	ProvenMessages int           `json:"proven_messages"`
	Points         []seriesPoint `json:"points"`
	Note           string        `json:"note"`
}

func seriesCommand() cliapp.Command {
	return cliapp.Command{
		Name: "series", NeedsAPI: true,
		Description: "Show a message-scope label (mood) over time, over human operator messages only",
		Usage:       "agent-manager annotations series --label <id> [--window 30d] [--bucket hour|day|week] [--json]",
		HelpText: "A message-scope label is answered once per human operator message, so this is a mood series over what a person wrote; workflow briefs and harness text are never subjects. " +
			"Mean is the calibrated expected score (or probability) over every message in the bucket; proven_mean averages only messages from a model whose hold-out proof clears the gate. " +
			"Until a model is proven the series is evidence of mood, not a measurement, and the output says so.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "label", Required: true, Description: "Message-scope label id (see `annotations labels`)"},
			{Name: "window", Default: "30d", Description: "Window: 24h, 7d, 30d, all or RFC3339"},
			{Name: "bucket", Default: "day", Description: "hour, day or week"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			query := url.Values{
				"label": []string{strings.TrimSpace(ctx.Flag("label"))}, "window": []string{normalizedWindow(ctx.Flag("window"))},
				"bucket": []string{strings.TrimSpace(ctx.Flag("bucket"))},
			}
			body, err := ctx.Core().Request("GET", "annotations/series", query, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var result seriesResult
			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}
			renderSeries(ctx.Stdout(), result)
			return nil
		},
	}
}

func renderSeries(out io.Writer, result seriesResult) {
	fmt.Fprintf(out, "Series %s per %s · %d human message(s), %d proven\n", result.Label, result.Bucket, result.Messages, result.ProvenMessages)
	for _, point := range result.Points {
		mean, proven := "-", "-"
		if point.Mean != nil {
			mean = fmt.Sprintf("%+.2f", *point.Mean)
		}
		if point.ProvenMean != nil {
			proven = fmt.Sprintf("%+.2f", *point.ProvenMean)
		}
		fmt.Fprintf(out, "  %s  messages=%-3d mean=%-6s proven_mean=%-6s friction=%d\n", point.Start, point.Messages, mean, proven, point.FrictionPositives)
	}
	if result.Note != "" {
		fmt.Fprintln(out, result.Note)
	}
}
