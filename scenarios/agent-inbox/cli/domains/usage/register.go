package usage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"agent-inbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "usage",
		Description: "Usage and cost summaries",
		Subcommands: []cliapp.Command{
			{Name: "summary", NeedsAPI: true, Description: "Show aggregate usage stats", Run: func(args []string) error { return runSummary(core, args) }},
			{Name: "chat", NeedsAPI: true, Description: "Show usage stats for one chat", Run: func(args []string) error { return runChat(core, args) }},
		},
	}
}

func runSummary(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("usage summary")
	start := fs.String("start", "", "Start date YYYY-MM-DD")
	end := fs.String("end", "", "End date YYYY-MM-DD")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*start) != "" {
		query.Set("start", strings.TrimSpace(*start))
	}
	if strings.TrimSpace(*end) != "" {
		query.Set("end", strings.TrimSpace(*end))
	}
	body, err := core.Get("/usage", query)
	if err != nil {
		return err
	}
	return renderUsageMap(body, *jsonOutput, "Aggregate usage", []string{support.CLIName + " usage chat <chat-id>"})
}

func runChat(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("usage chat")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: usage chat <chat-id> [--json]")
	}
	id := fs.Arg(0)
	body, err := core.Get("/chats/"+id+"/usage", nil)
	if err != nil {
		return err
	}
	return renderUsageMap(body, *jsonOutput, "Chat usage", []string{support.CLIName + " chat get " + id})
}

func renderUsageMap(body []byte, jsonOutput bool, heading string, hints []string) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	results := make([]string, 0, len(keys))
	for _, key := range keys {
		results = append(results, fmt.Sprintf("%s: %v", key, payload[key]))
	}

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: hints,
	}
	return support.PrintList(jsonOutput, report)
}
