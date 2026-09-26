package problem

import (
	"fmt"
	"os"
	"strconv"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires the `problem` subcommand group covering LeetCode/HackerRank
// style problem mappings.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "problem",
		Description: "Search and manage problem mappings (LeetCode, HackerRank, etc.)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "search", Description: "Search problem mappings", Run: func(args []string) error { return runSearch(core, args) }},
			{Name: "stats", Description: "Show platform-level problem statistics", Run: func(args []string) error { return runStats(core, args) }},
			{Name: "recommend", Description: "Get recommended problems", Run: func(args []string) error { return runRecommend(core, args) }},
			{Name: "add", Description: "Add a new problem mapping", Run: func(args []string) error { return runAdd(core, args) }},
		},
	}
}

func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("problem search")
	platform := fs.String("platform", "", "Filter by platform (leetcode, hackerrank, ...)")
	difficulty := fs.String("difficulty", "", "Filter by difficulty")
	tag := fs.String("tag", "", "Filter by tag")
	limit := fs.Int("limit", 0, "Optional max entries")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	params := map[string]string{
		"platform":   *platform,
		"difficulty": *difficulty,
		"tag":        *tag,
	}
	if fs.NArg() > 0 {
		params["q"] = fs.Arg(0)
	}
	if *limit > 0 {
		params["limit"] = strconv.Itoa(*limit)
	}

	body, err := core.Get("/problems/search", support.BuildQuery(params))
	if err != nil {
		return err
	}
	return renderList(body, "Problem search results", "Problems", *jsonOutput)
}

func runStats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("problem stats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/problems/stats", nil)
	if err != nil {
		return err
	}
	return renderList(body, "Platform problem statistics", "Statistics", *jsonOutput)
}

func runRecommend(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("problem recommend")
	difficulty := fs.String("difficulty", "", "Target difficulty")
	tag := fs.String("tag", "", "Filter by tag")
	limit := fs.Int("limit", 0, "Optional max recommendations")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	params := map[string]string{
		"difficulty": *difficulty,
		"tag":        *tag,
	}
	if *limit > 0 {
		params["limit"] = strconv.Itoa(*limit)
	}

	body, err := core.Get("/problems/recommend", support.BuildQuery(params))
	if err != nil {
		return err
	}
	return renderList(body, "Recommended problems", "Recommendations", *jsonOutput)
}

func runAdd(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("problem add")
	bodyFile := fs.String("body-file", "", "Path to problem mapping JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/problems", nil, payload)
	if err != nil {
		return err
	}
	return renderMutation(body, "Added problem mapping", *jsonOutput)
}

func renderMutation(body []byte, title string, jsonOut bool) error {
	var data interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{title},
		Changes: renderGeneric(data),
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderList(body []byte, summary, heading string, jsonOut bool) error {
	var data interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: heading,
		Results:        renderGeneric(data),
	}
	if jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderGeneric(value interface{}) []string {
	switch v := value.(type) {
	case map[string]interface{}:
		return support.MapRows(v)
	case []interface{}:
		if len(v) == 0 {
			return []string{"(empty list)"}
		}
		rows := make([]string, 0, len(v))
		for i, item := range v {
			rows = append(rows, fmt.Sprintf("%d: %s", i, support.RenderValue(item)))
		}
		return rows
	case nil:
		return []string{"(empty payload)"}
	default:
		return []string{support.RenderValue(v)}
	}
}
