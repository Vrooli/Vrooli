package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdStatus() error {
	body, resp, err := a.services.Health.Status()
	if err != nil {
		return err
	}

	fmt.Printf("Status: %s\n", defaultValue(resp.Status, "unknown"))
	if resp.Service != "" {
		fmt.Printf("Service: %s\n", resp.Service)
	}
	if resp.Version != "" {
		fmt.Printf("Version: %s\n", resp.Version)
	}

	if resp.Operations.LastExecution != nil {
		icon := "✓"
		if !resp.Operations.LastExecution.Success {
			icon = "✗"
		}
		fmt.Printf("Last execution: %s %s (phases=%d failed=%d) completed %s\n",
			icon,
			resp.Operations.LastExecution.Scenario,
			resp.Operations.LastExecution.PhaseSummary.Total,
			resp.Operations.LastExecution.PhaseSummary.Failed,
			defaultValue(resp.Operations.LastExecution.CompletedAt, "n/a"),
		)
	}

	q := resp.Operations.Queue
	fmt.Printf("Queue: pending=%d queued=%d delegated=%d running=%d failed=%d\n", q.Pending, q.Queued, q.Delegated, q.Running, q.Failed)
	if q.OldestQueuedAgeSecs > 0 {
		fmt.Printf("       oldest queued: %ds\n", q.OldestQueuedAgeSecs)
	}

	if len(resp.Dependencies) > 0 {
		fmt.Println("Dependencies:")
		cliutil.PrintJSONMap(resp.Dependencies, 2)
	}

	if resp.Status == "" && len(body) > 0 {
		cliutil.PrintJSON(body)
	}
	return nil
}

func (a *App) cmdConfigure(args []string) error {
	if len(args) == 0 {
		payload, _ := json.MarshalIndent(a.config, "", "  ")
		fmt.Println(string(payload))
		return nil
	}
	if len(args) != 2 {
		return usageError("usage: configure <api_base|token> <value>")
	}
	key := args[0]
	value := args[1]
	switch key {
	case "api_base":
		a.config.APIBase = value
	case "token", "api_token":
		a.config.Token = value
	default:
		return usageError("unknown configuration key: " + key)
	}
	if err := a.saveConfig(); err != nil {
		return err
	}
	fmt.Printf("Updated %s\n", key)
	return nil
}

func (a *App) cmdGenerate(args []string) error {
	if len(args) == 0 {
		return usageError("usage: generate <scenario> [--types unit,integration] [--coverage 95] [--priority normal] [--notes text] [--notes-file path] [--json]")
	}
	scenario := args[0]
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	types := fs.String("types", "", "Comma-separated types to request")
	coverage := fs.Int("coverage", 0, "Coverage target (1-100)")
	priority := fs.String("priority", "", "Priority (low|normal|high|urgent)")
	notes := fs.String("notes", "", "Notes for this request")
	notesFile := fs.String("notes-file", "", "Path to notes file")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if *coverage < 0 || *coverage > 100 {
		return usageError("coverage must be between 0 and 100")
	}
	if *priority != "" && !isAllowedPriority(*priority) {
		return usageError("priority must be one of: low, normal, high, urgent")
	}

	payload := GenerateRequest{
		ScenarioName:   scenario,
		RequestedTypes: parseCSV(*types),
		Priority:       strings.ToLower(*priority),
		Notes:          *notes,
	}
	if *coverage > 0 {
		val := *coverage
		payload.CoverageTarget = &val
	}
	if *notesFile != "" {
		content, err := readFile(*notesFile)
		if err != nil {
			return fmt.Errorf("read notes file: %w", err)
		}
		payload.Notes = content
	}

	resp, raw, err := a.services.Suite.Generate(payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Suite request queued for %s\n", resp.ScenarioName)
	if resp.ID != "" {
		fmt.Printf("  Request ID : %s\n", resp.ID)
	}
	if resp.Status != "" {
		fmt.Printf("  Status     : %s\n", resp.Status)
	}
	if len(resp.RequestedTypes) > 0 {
		fmt.Printf("  Types      : %s\n", strings.Join(resp.RequestedTypes, ", "))
	}
	if resp.CoverageTarget != nil {
		fmt.Printf("  Coverage   : %d%%\n", *resp.CoverageTarget)
	}
	if resp.Priority != "" {
		fmt.Printf("  Priority   : %s\n", resp.Priority)
	}
	if resp.EstimatedQueueSec > 0 {
		fmt.Printf("  ETA        : ~%ds\n", resp.EstimatedQueueSec)
	}
	return nil
}

func (a *App) cmdExecute(args []string) error {
	if len(args) == 0 {
		return usageError("usage: execute <scenario> [phases...] [--preset quick] [--skip performance] [--request-id id] [--fail-fast] [--json]")
	}
	scenario := args[0]
	fs := flag.NewFlagSet("execute", flag.ContinueOnError)
	preset := fs.String("preset", "", "Preset name")
	phasesFlag := fs.String("phases", "", "Comma-separated phases to run")
	skipFlag := fs.String("skip", "", "Comma-separated phases to skip")
	requestID := fs.String("request-id", "", "Link to suite request")
	failFast := fs.Bool("fail-fast", false, "Stop on first failure")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	phases := parseMultiArgs(parseCSV(*phasesFlag), fs.Args())
	skip := parseCSV(*skipFlag)

	req := ExecuteRequest{
		ScenarioName:   scenario,
		Preset:         *preset,
		Phases:         phases,
		Skip:           skip,
		FailFast:       *failFast,
		SuiteRequestID: *requestID,
	}

	resp, raw, err := a.services.Suite.Execute(req)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	icon := "⚠"
	if resp.Success {
		icon = "✓"
	}
	fmt.Printf("%s Suite execution for %s\n", icon, scenario)
	if resp.ExecutionID != "" {
		fmt.Printf("  Execution : %s\n", resp.ExecutionID)
	}
	if resp.SuiteRequest != "" {
		fmt.Printf("  Request   : %s\n", resp.SuiteRequest)
	}
	if resp.PresetUsed != "" {
		fmt.Printf("  Preset    : %s\n", resp.PresetUsed)
	}
	if resp.StartedAt != "" {
		fmt.Printf("  Started   : %s\n", resp.StartedAt)
	}
	if resp.CompletedAt != "" {
		fmt.Printf("  Finished  : %s\n", resp.CompletedAt)
	}

	fmt.Println()
	fmt.Println("Phase results:")
	for _, phase := range resp.Phases {
		phaseIcon := "✓"
		if !strings.EqualFold(phase.Status, "passed") {
			phaseIcon = "✗"
		}
		fmt.Printf("  %s %-12s status=%-8s duration=%gs\n", phaseIcon, phase.Name, phase.Status, phase.DurationSeconds)
		if phase.LogPath != "" {
			fmt.Printf("      log: %s\n", phase.LogPath)
		}
		if phase.Error != "" {
			fmt.Printf("      error: %s\n", phase.Error)
		}
	}

	if resp.Error != "" {
		fmt.Printf("\nError: %s\n", resp.Error)
	}

	if resp.Success {
		return nil
	}
	return fmt.Errorf("suite execution completed with failures")
}

func (a *App) cmdRunTests(args []string) error {
	if len(args) == 0 {
		return usageError("usage: run-tests <scenario> [--type phased] [--json]")
	}
	scenario := args[0]
	fs := flag.NewFlagSet("run-tests", flag.ContinueOnError)
	testType := fs.String("type", "", "Test runner type to request")
	jsonOutput := fs.Bool("json", false, "Output raw JSON")
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	req := RunTestsRequest{}
	if *testType != "" {
		req.Type = *testType
	}

	resp, raw, err := a.services.RunTests.Run(scenario, req)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Scenario tests completed via %s\n", defaultValue(resp.Type, "unknown"))
	if len(resp.Command.Command) > 0 {
		fmt.Printf("  Command : %s\n", strings.Join(resp.Command.Command, " "))
	}
	if resp.Command.WorkingDir != "" {
		fmt.Printf("  Workdir : %s\n", resp.Command.WorkingDir)
	}
	if resp.LogPath != "" {
		fmt.Printf("  Logs    : %s\n", resp.LogPath)
	}
	if resp.Status != "" {
		fmt.Printf("  Status  : %s\n", resp.Status)
	}
	return nil
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func isAllowedPriority(priority string) bool {
	switch strings.ToLower(priority) {
	case "low", "normal", "high", "urgent":
		return true
	default:
		return false
	}
}
