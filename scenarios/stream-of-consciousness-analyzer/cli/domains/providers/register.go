package providers

import (
	"fmt"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const cliName = "stream-of-consciousness-analyzer"

type provider struct {
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Fallback bool   `json:"fallback"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "provider",
		Description: "Inspect suggestion providers",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List provider status", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("provider list", args)
	if err != nil {
		return err
	}
	_ = fs

	body, err := core.Get("/providers", nil)
	if err != nil {
		return err
	}

	var providers []provider
	if err := support.Unmarshal(body, &providers); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Providers configured: %d", len(providers))},
		Results: renderProviders(providers),
		RetrievalHints: []string{
			cliName + " suggestion generate <scheme-id>",
			cliName + " status",
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderProviders(items []provider) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		status := "inactive"
		if item.Active && item.Fallback {
			status = "active (fallback)"
		} else if item.Active {
			status = "active"
		}
		lines = append(lines, fmt.Sprintf("%s  %s", item.Name, status))
	}
	return lines
}
