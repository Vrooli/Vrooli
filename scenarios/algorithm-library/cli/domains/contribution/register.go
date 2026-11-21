package contribution

import (
	"fmt"
	"os"

	"algorithm-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wires the `contribution` subcommand group. All submit/review payloads
// accept arbitrary contributor-provided fields (algorithms, implementations,
// review decisions), so this domain uses `--body-file` passthrough rather than
// modelling every field as a CLI flag.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "contribution",
		Description: "Submit and review community contributions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "submit-algorithm", Description: "Submit a new algorithm contribution", Run: func(args []string) error {
				return runSubmit(core, args, "/contributions/algorithm", "Algorithm submission")
			}},
			{Name: "submit-implementation", Description: "Submit a new implementation contribution", Run: func(args []string) error {
				return runSubmit(core, args, "/contributions/implementation", "Implementation submission")
			}},
			{Name: "list", Aliases: []string{"ls"}, Description: "List pending and past contributions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "review", Description: "Approve or reject a contribution", Run: func(args []string) error { return runReview(core, args) }},
		},
	}
}

func runSubmit(core *cliapp.ScenarioApp, args []string, path, title string) error {
	fs := support.NewFlagSet("contribution submit")
	bodyFile := fs.String("body-file", "", "Path to submission JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	return renderMutation(body, title, *jsonOutput)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("contribution list")
	status := fs.String("status", "", "Filter by status (pending, approved, rejected)")
	contributor := fs.String("contributor", "", "Filter by contributor id")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	params := support.BuildQuery(map[string]string{
		"status":      *status,
		"contributor": *contributor,
	})
	body, err := core.Get("/contributions", params)
	if err != nil {
		return err
	}
	return renderList(body, "Contributions", "Entries", *jsonOutput)
}

func runReview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("contribution review")
	bodyFile := fs.String("body-file", "", "Path to review decision JSON (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: contribution review <contribution-id> --body-file <path>")
	}
	id := fs.Arg(0)

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/contributions/"+id+"/review", nil, payload)
	if err != nil {
		return err
	}
	return renderMutation(body, fmt.Sprintf("Reviewed contribution %s", id), *jsonOutput)
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
