package diagnostics

import (
	"fmt"
	"os"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `diagnostics` subcommand group. Diagnostic payloads have
// varied shapes; commands decode into generic maps and project the fields the
// API exposes today. For interop we preserve the severity-grouped rendering the
// bash CLI established.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "diagnostics",
		Description: "Run diagnostic checks against a managed app",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "full", Description: "Run the full diagnostic bundle", Run: func(args []string) error { return runSimple(core, args, "full", "") }},
			{Name: "status", Description: "Scenario status diagnostic", Run: func(args []string) error { return runSimple(core, args, "status", "/status") }},
			{Name: "health", Description: "Health check diagnostic", Run: func(args []string) error { return runSimple(core, args, "health", "/health") }},
			{Name: "iframe-bridge", Description: "Iframe-bridge diagnostic", Run: func(args []string) error { return runSimple(core, args, "iframe-bridge", "/iframe-bridge") }},
			{Name: "localhost", Description: "Localhost usage diagnostic", Run: func(args []string) error { return runSimple(core, args, "localhost", "/localhost") }},
			{Name: "interop", Description: "UI interop compliance report", Run: func(args []string) error { return runInterop(core, args) }},
			{Name: "completeness", Description: "Scenario completeness score", Run: func(args []string) error { return runCompleteness(core, args) }},
		},
	}
}

func runSimple(core *cliapp.ScenarioApp, args []string, verb, suffix string) error {
	fs := support.NewFlagSet("diagnostics " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: diagnostics %s <app-id>", verb)
	}
	id := fs.Arg(0)

	body, err := core.Get("/apps/"+id+"/diagnostics"+suffix, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Diagnostic: %s for %s", verb, id)},
		Triage:    []cliapp.TriageGroup{{Heading: "Details", Items: support.MapRows(data)}},
		NextSteps: []string{fmt.Sprintf("%s diagnostics full %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runInterop(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("diagnostics interop")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: diagnostics interop <app-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/apps/"+id+"/diagnostics/interop", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	hasUI, _ := data["has_ui"].(bool)
	passCount := asInt(data["pass_count"])
	failCount := asInt(data["fail_count"])
	skipCount := asInt(data["skip_count"])
	total := asInt(data["total_count"])
	score := asFloat(data["score"])

	status := []string{}
	if total > 0 {
		status = append(status, fmt.Sprintf("App: %s", id))
	}
	if !hasUI {
		status = append(status, "No ui/ directory — interop checks not applicable")
	} else if failCount == 0 {
		status = append(status, fmt.Sprintf("Interop compliance: %.0f%% (%d/%d passed, %d skipped)", score, passCount, total, skipCount))
	} else {
		status = append(status, fmt.Sprintf("Interop compliance: %.0f%% (%d failing)", score, failCount))
	}

	var triage []cliapp.TriageGroup
	if failing := interopFailuresBySeverity(data); len(failing) > 0 {
		for _, sev := range []string{"critical", "high", "medium", "low"} {
			items, ok := failing[sev]
			if !ok {
				continue
			}
			triage = append(triage, cliapp.TriageGroup{Heading: sev, Items: items})
		}
	}

	nextSteps := []string{}
	if failCount > 0 {
		nextSteps = append(nextSteps,
			"Read the interop skill: prompt-manager skill read vrooli-ui-interop",
			fmt.Sprintf("%s diagnostics interop %s --json", support.CLIName, id),
		)
	}

	report := cliapp.OperationalReport{
		Status:    status,
		Triage:    triage,
		NextSteps: nextSteps,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runCompleteness(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("diagnostics completeness")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: diagnostics completeness <app-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/apps/"+id+"/completeness", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Completeness for %s", id)},
		ResultsHeading: "Fields",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s diagnostics full %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func interopFailuresBySeverity(data map[string]interface{}) map[string][]string {
	checks, ok := data["checks"].([]interface{})
	if !ok {
		return nil
	}
	grouped := map[string][]string{}
	for _, raw := range checks {
		check, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		passed, _ := check["passed"].(bool)
		skipped, _ := check["skipped"].(bool)
		if passed || skipped {
			continue
		}
		sev, _ := check["severity"].(string)
		if sev == "" {
			sev = "medium"
		}
		name, _ := check["name"].(string)
		message, _ := check["message"].(string)
		grouped[sev] = append(grouped[sev], fmt.Sprintf("%s — %s", name, message))
	}
	return grouped
}

func asInt(value interface{}) int {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func asFloat(value interface{}) float64 {
	if value == nil {
		return 0
	}
	if v, ok := value.(float64); ok {
		return v
	}
	return 0
}
