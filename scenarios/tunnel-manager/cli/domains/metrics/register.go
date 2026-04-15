package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"

	"tunnel-manager/cli/internal/flags"
	"tunnel-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "metrics",
		Description: "Inspect latest and historical tunnel metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "latest", NeedsAPI: true, Description: "Show latest tunnel metrics", Run: func(args []string) error { return latest(deps, args) }},
			{Name: "history", NeedsAPI: true, Description: "Show metrics history", Run: func(args []string) error { return history(deps, args) }},
		},
	}
}

func latest(deps support.Dependencies, args []string) error {
	body, err := deps.ScenarioApp().Get("/metrics/latest", nil)
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var metrics map[string]any
	if err := json.Unmarshal(body, &metrics); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	results := make([]string, 0, len(keys))
	for _, key := range keys {
		results = append(results, fmt.Sprintf("%s: %v", key, metrics[key]))
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Latest metric fields: %d", len(results)),
		},
		Results:        results,
		RetrievalHints: []string{"tunnel-manager metrics history", "tunnel-manager health detailed"},
	})
}

func history(deps support.Dependencies, args []string) error {
	hours := "24"
	if v, ok := flags.StringValue(args, "hours"); ok {
		hours = v
	}

	body, err := deps.ScenarioApp().Get("/metrics/history", urlValues("hours", hours))
	if err != nil {
		return err
	}
	if flags.HasJSONOutput(args) {
		cliutil.PrintJSON(body)
		return nil
	}

	var entries []struct {
		Timestamp     string  `json:"timestamp"`
		HAConnections int     `json:"ha_connections"`
		Errors        int     `json:"errors"`
		Streams       int     `json:"streams"`
		RTT           float64 `json:"rtt_ms"`
	}
	if err := json.Unmarshal(body, &entries); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		results = append(results, fmt.Sprintf(
			"%s | ha=%d | errors=%d | streams=%d | rtt=%.1fms",
			entry.Timestamp,
			entry.HAConnections,
			entry.Errors,
			entry.Streams,
			entry.RTT,
		))
	}

	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Metrics history window: %s hours", hours),
			fmt.Sprintf("Entries: %d", len(entries)),
		},
		Results:        results,
		RetrievalHints: []string{"tunnel-manager metrics latest", "tunnel-manager probe history"},
	})
}

func urlValues(key, value string) url.Values {
	if value == "" {
		return nil
	}
	query := url.Values{}
	query.Set(key, value)
	return query
}
