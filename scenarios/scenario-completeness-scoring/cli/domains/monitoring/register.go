package monitoring

import (
	"fmt"
	"os"

	"scenario-completeness-scoring/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `monitoring` subcommand group for collector and
// circuit-breaker health. These endpoints live under `/api/v1/health/...` and
// are distinct from the root `/health` probe, which is covered by the built-in
// `status` command in cli-core.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "monitoring",
		Description: "Inspect collector and circuit-breaker health",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "collectors", Description: "Show collector health status", Run: func(args []string) error { return runCollectors(core, args) }},
			{Name: "collector-test", Description: "Run a health probe against one collector", Run: func(args []string) error { return runCollectorTest(core, args) }},
			{Name: "breakers", Description: "Show circuit-breaker status", Run: func(args []string) error { return runBreakers(core, args) }},
			{Name: "breaker-reset", Description: "Reset a single circuit breaker, or all when no name is given", Run: func(args []string) error { return runBreakerReset(core, args) }},
		},
	}
}

func runCollectors(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring collectors")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/health/collectors", nil)
	if err != nil {
		return err
	}
	var resp support.CollectorHealthResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	status := []string{
		fmt.Sprintf("Overall collector status: %s", fallback(resp.Status, "unknown")),
		fmt.Sprintf("Total: %d", resp.Summary["total"]),
		fmt.Sprintf("Healthy: %d", resp.Summary["healthy"]),
		fmt.Sprintf("Degraded: %d", resp.Summary["degraded"]),
		fmt.Sprintf("Failed: %d", resp.Summary["failed"]),
	}
	report := cliapp.OperationalReport{
		Status: status,
		Triage: []cliapp.TriageGroup{
			{Heading: "Collectors", Items: support.JSONLines(body)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s monitoring breakers", support.CLIName),
			fmt.Sprintf("%s monitoring collector-test <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runCollectorTest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring collector-test")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: monitoring collector-test <name>")
	}
	name := fs.Arg(0)

	body, err := core.Request("POST", "/health/collectors/"+name+"/test", nil, map[string]interface{}{})
	if err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("Collector test issued: %s", name)},
		Triage: []cliapp.TriageGroup{
			{Heading: "Test result", Items: support.JSONLines(body)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s monitoring collectors", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runBreakers(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring breakers")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/health/circuit-breaker", nil)
	if err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{"Circuit-breaker status"},
		Triage: []cliapp.TriageGroup{
			{Heading: "Breakers", Items: support.JSONLines(body)},
		},
		NextSteps: []string{
			fmt.Sprintf("%s monitoring breaker-reset", support.CLIName),
			fmt.Sprintf("%s monitoring collectors", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runBreakerReset(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring breaker-reset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	path := "/health/circuit-breaker/reset"
	target := "all collectors"
	if fs.NArg() >= 1 {
		name := fs.Arg(0)
		path = "/health/circuit-breaker/" + name + "/reset"
		target = name
	}

	body, err := core.Request("POST", path, nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Circuit breaker reset: %s", target)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Circuit-breaker reset applied to %s.", target)},
		NextCommand: []string{fmt.Sprintf("%s monitoring breakers", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}
