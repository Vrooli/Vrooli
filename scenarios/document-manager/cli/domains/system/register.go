package system

import (
	"fmt"
	"os"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `system` subcommand group for scenario-specific subsystem
// health checks. These hit /api/system/*-status and are distinct from the
// built-in root /health probe exposed by the status command.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "system",
		Description: "Inspect scenario subsystem status (database, vector store, AI)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "db-status", Aliases: []string{"db"}, Description: "Postgres connectivity status", Run: func(args []string) error {
				return runStatus(core, args, "system db-status", "/system/db-status")
			}},
			{Name: "vector-status", Aliases: []string{"vector"}, Description: "Qdrant vector store status", Run: func(args []string) error {
				return runStatus(core, args, "system vector-status", "/system/vector-status")
			}},
			{Name: "ai-status", Aliases: []string{"ai"}, Description: "Ollama AI service status", Run: func(args []string) error {
				return runStatus(core, args, "system ai-status", "/system/ai-status")
			}},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string, label, path string) error {
	fs := support.NewFlagSet(label)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	var status support.SystemStatus
	if err := support.Decode(body, &status); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Service: %s", status.Service), fmt.Sprintf("Status: %s", status.Status)}
	results := []string{
		fmt.Sprintf("service: %s", status.Service),
		fmt.Sprintf("status: %s", status.Status),
	}
	if status.Details != "" {
		results = append(results, fmt.Sprintf("details: %s", status.Details))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Fields",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s status", support.CLIName),
			fmt.Sprintf("%s %s --json", support.CLIName, label),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
