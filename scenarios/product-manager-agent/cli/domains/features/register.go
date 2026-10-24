package features

import (
	"fmt"
	"os"

	"product-manager-agent/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `features` subcommand group covering list/create/update
// and the two prioritization endpoints (`prioritize`, `rice`). Creation and
// updates take a `--body-file` since the payload is nested.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "features",
		Description: "Manage and prioritize product features",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List features", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Aliases: []string{"add"}, Description: "Create a feature (--body-file PATH)", Run: func(args []string) error { return runMutate(core, args, "POST", "create") }},
			{Name: "update", Description: "Update a feature (--body-file PATH)", Run: func(args []string) error { return runMutate(core, args, "PUT", "update") }},
			{Name: "rice", Description: "Compute RICE scores from a feature array (--body-file PATH)", Run: func(args []string) error { return runRICE(core, args) }},
			{Name: "prioritize", Description: "Prioritize features by strategy (--body-file PATH)", Run: func(args []string) error { return runPrioritize(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("features list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/features", nil)
	if err != nil {
		return err
	}
	var items []support.Feature
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Features: %d", len(items))},
		ResultsHeading: "Features",
		Results:        featureRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s features rice --body-file features.json", support.CLIName),
			fmt.Sprintf("%s features prioritize --body-file request.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runMutate(core *cliapp.ScenarioApp, args []string, method, verb string) error {
	fs := support.NewFlagSet("features " + verb)
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the feature payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	respBody, err := core.Request(method, "/features", nil, payload)
	if err != nil {
		return err
	}
	var f support.Feature
	_ = support.Decode(respBody, &f)

	message := support.EnvelopeMessage(respBody)
	if message == "" {
		message = fmt.Sprintf("Feature %sd", verb)
	}

	report := cliapp.MutationReport{
		Result: []string{message},
		Changes: []string{
			fmt.Sprintf("Feature: %s", support.JoinNonEmpty(" ", f.Name, "("+support.ShortID(f.ID)+")")),
			fmt.Sprintf("Score: %.2f (priority %s)", f.Score, f.Priority),
		},
		NextCommand: []string{
			fmt.Sprintf("%s features list", support.CLIName),
			fmt.Sprintf("%s dashboard", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRICE(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("features rice")
	bodyFile := fs.String("body-file", "", "Path to JSON array of features")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/features/rice", nil, payload)
	if err != nil {
		return err
	}
	var resp support.PrioritizeResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("RICE scored %d features", resp.Total)},
		ResultsHeading: "Prioritized",
		Results:        featureRows(resp.PrioritizedFeatures),
		RetrievalHints: []string{fmt.Sprintf("%s features prioritize --body-file request.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPrioritize(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("features prioritize")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"features":[...],"strategy":"rice|value|effort"}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/features/prioritize", nil, payload)
	if err != nil {
		return err
	}
	var items []support.Feature
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Prioritized %d features", len(items))},
		ResultsHeading: "Prioritized",
		Results:        featureRows(items),
		RetrievalHints: []string{fmt.Sprintf("%s features rice --body-file features.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func featureRows(items []support.Feature) []string {
	if len(items) == 0 {
		return []string{"(no features)"}
	}
	rows := make([]string, 0, len(items))
	for _, f := range items {
		rows = append(rows, fmt.Sprintf("%s (%s) | score=%.2f | priority=%s | reach=%d impact=%d conf=%.2f effort=%d",
			f.Name, support.ShortID(f.ID), f.Score, f.Priority, f.Reach, f.Impact, f.Confidence, f.Effort))
	}
	return rows
}
