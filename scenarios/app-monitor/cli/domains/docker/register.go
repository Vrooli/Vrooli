package docker

import (
	"fmt"
	"os"
	"strings"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `docker` subcommand group covering Docker daemon info and containers.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "docker",
		Description: "Inspect Docker daemon state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "info", Description: "Docker daemon info", Run: func(args []string) error { return runInfo(core, args) }},
			{Name: "containers", Aliases: []string{"ps"}, Description: "List containers", Run: func(args []string) error { return runContainers(core, args) }},
		},
	}
}

func runInfo(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docker info")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/docker/info", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{"Docker daemon info"},
		ResultsHeading: "Fields",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s docker containers", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runContainers(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("docker containers")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/docker/containers", nil)
	if err != nil {
		return err
	}
	var containers []map[string]interface{}
	if err := support.Decode(body, &containers); err != nil {
		return err
	}

	rows := make([]string, 0, len(containers))
	for _, c := range containers {
		id, _ := c["Id"].(string)
		if id == "" {
			id, _ = c["id"].(string)
		}
		image, _ := c["Image"].(string)
		state, _ := c["State"].(string)
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %s", support.ShortID(id), formatNames(c["Names"]), image, state))
	}
	if len(rows) == 0 {
		rows = []string{"(no containers)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Containers: %d", len(containers))},
		ResultsHeading: "Containers",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s docker info", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func formatNames(value interface{}) string {
	names, ok := value.([]interface{})
	if !ok {
		return "(unnamed)"
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if s, ok := n.(string); ok {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return "(unnamed)"
	}
	return strings.Join(out, ", ")
}
