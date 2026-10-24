package types

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"test-data-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `test-data-generator types` as a flat command since the
// `GET /api/types` surface is a single read-only listing.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Catalog",
		Commands: []cliapp.Command{
			{
				Name:        "types",
				Description: "List available data types and their fields",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("types")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/types", nil)
	if err != nil {
		return err
	}
	var resp support.TypesResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	keys := make([]string, 0, len(resp.Definitions))
	for k := range resp.Definitions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	results := make([]string, 0, len(keys))
	for _, k := range keys {
		def := resp.Definitions[k]
		line := k
		if def.Description != "" {
			line = fmt.Sprintf("%s: %s", k, def.Description)
		}
		if len(def.Fields) > 0 {
			line = fmt.Sprintf("%s [fields: %s]", line, strings.Join(def.Fields, ", "))
		}
		results = append(results, line)
	}
	if len(results) == 0 {
		results = []string{"(no data types exposed by API)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Available data types: %d", len(keys))},
		ResultsHeading: "Types",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s generate users --count 5", support.CLIName),
			fmt.Sprintf("%s generate custom --schema '{\"id\":\"uuid\"}' --count 3", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
