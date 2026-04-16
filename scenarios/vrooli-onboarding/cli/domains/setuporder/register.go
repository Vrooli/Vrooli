package setuporder

import (
	"fmt"
	"os"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `vrooli-onboarding setup-order` as a flat command since
// `/api/v1/setup-order` is a single read-only endpoint that returns resources
// sorted by their computed dependency order.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Setup order",
		Commands: []cliapp.Command{
			{
				Name:        "setup-order",
				Description: "Show recommended resource setup order",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("setup-order")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/setup-order", nil)
	if err != nil {
		return err
	}
	var resp support.SetupOrderResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Setup order for %d resource(s)", resp.Total)},
		ResultsHeading: "Order",
		Results:        rows(resp.SetupOrder),
		RetrievalHints: []string{
			fmt.Sprintf("%s resources list", support.CLIName),
			fmt.Sprintf("%s config generate --resources <names>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func rows(entries []support.OrderedResource) []string {
	if len(entries) == 0 {
		return []string{"(no resources in setup order)"}
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		deps := "none"
		if len(e.Dependencies) > 0 {
			deps = strings.Join(e.Dependencies, ",")
		}
		out = append(out, fmt.Sprintf("%d. %s | category=%s | deps=%s", e.Order, e.Name, e.Category, deps))
	}
	return out
}
