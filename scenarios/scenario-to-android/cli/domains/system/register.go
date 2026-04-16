package system

import (
	"fmt"
	"os"

	"scenario-to-android/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `system` subcommand group for scenario-specific status
// (Android SDK readiness) and aggregate build metrics. The generic /health
// probe is served by the built-in cli-core `status` command and is not
// re-registered here.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "system",
		Description: "Inspect Android build toolchain status and aggregate metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "status",
				Description: "Show Android SDK / toolchain readiness reported by the API",
				Run:         func(args []string) error { return runStatus(core, args) },
			},
			{
				Name:        "metrics",
				Description: "Show aggregate build metrics",
				Run:         func(args []string) error { return runMetrics(core, args) },
			},
		},
	}
}

func runStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("system status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/status", nil)
	if err != nil {
		return err
	}

	var sdk support.SDKStatus
	if err := support.Decode(body, &sdk); err != nil {
		return err
	}

	readiness := "not ready"
	if sdk.Ready {
		readiness = "ready"
	}

	results := []string{
		fmt.Sprintf("Readiness: %s", readiness),
		fmt.Sprintf("Android SDK: %s", valueOrUnset(sdk.AndroidSDK)),
		fmt.Sprintf("Java: %s", valueOrUnset(sdk.Java)),
		fmt.Sprintf("Gradle: %s", valueOrUnset(sdk.Gradle)),
		fmt.Sprintf("Build system: %s", valueOrUnset(sdk.BuildSystem)),
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Android toolchain: %s", readiness)},
		ResultsHeading: "Toolchain",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s system metrics", support.CLIName),
			fmt.Sprintf("%s status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runMetrics(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("system metrics")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics", nil)
	if err != nil {
		return err
	}

	var m support.BuildMetrics
	if err := support.Decode(body, &m); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Total builds: %d", m.TotalBuilds),
		fmt.Sprintf("Successful: %d", m.SuccessfulBuilds),
		fmt.Sprintf("Failed: %d", m.FailedBuilds),
		fmt.Sprintf("Active: %d", m.ActiveBuilds),
		fmt.Sprintf("Success rate: %.1f%%", m.SuccessRate),
		fmt.Sprintf("Average duration: %.1fs", m.AverageDuration),
		fmt.Sprintf("Uptime: %.0fs", m.Uptime),
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Builds: total=%d success=%d failed=%d active=%d", m.TotalBuilds, m.SuccessfulBuilds, m.FailedBuilds, m.ActiveBuilds)},
		ResultsHeading: "Metrics",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s system status", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func valueOrUnset(value string) string {
	if value == "" {
		return "(unset)"
	}
	return value
}
